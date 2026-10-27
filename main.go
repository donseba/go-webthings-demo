// Atelier Noord is the WebThings demo: a bilingual website for a fictional
// ceramics studio using most of the bundled modules. The first start fills it
// with sample content; after that, everything is edited in the admin at /admin/.
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	webthings "github.com/donseba/go-webthings"
	"github.com/donseba/go-webthings/frontend"
	"github.com/donseba/go-webthings/locales"
	"github.com/donseba/go-webthings/mail"
	"github.com/donseba/go-webthings/modules/banners"
	"github.com/donseba/go-webthings/modules/comments"
	"github.com/donseba/go-webthings/modules/directory"
	"github.com/donseba/go-webthings/modules/documents"
	"github.com/donseba/go-webthings/modules/events"
	"github.com/donseba/go-webthings/modules/forms"
	"github.com/donseba/go-webthings/modules/gallery"
	"github.com/donseba/go-webthings/modules/lists"
	"github.com/donseba/go-webthings/modules/newsletter"
	"github.com/donseba/go-webthings/modules/people"
	"github.com/donseba/go-webthings/modules/polls"
	"github.com/donseba/go-webthings/modules/publishing"
	"github.com/donseba/go-webthings/modules/redirects"
	"github.com/donseba/go-webthings/modules/registrations"
	"github.com/donseba/go-webthings/modules/stats"
	"github.com/donseba/go-webthings/modules/testimonials"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := run(ctx); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context) error {
	translations, err := frontend.Translations("translations")
	if err != nil {
		return err
	}

	imports, err := importMap()
	if err != nil {
		return err
	}

	if err := registerFormTheme(); err != nil {
		return err
	}

	sender, err := smtpSender()
	if err != nil {
		return err
	}

	config := webthings.Config{
		Name:          "Atelier Noord",
		StateDir:      env("WEBTHINGS_STATE_DIR", "var"),
		Driver:        env("WEBTHINGS_DB_DRIVER", "sqlite"),
		DSN:           os.Getenv("WEBTHINGS_DB_DSN"),
		Locales:       locales.Config{Default: "nl", Enabled: []string{"nl", "en"}},
		AdminLanguage: env("WEBTHINGS_ADMIN_LANGUAGE", "nl"),
		AdminEmail:    os.Getenv("WEBTHINGS_ADMIN_EMAIL"),
		AdminPassword: os.Getenv("WEBTHINGS_ADMIN_PASSWORD"),
		Insecure:      os.Getenv("WEBTHINGS_INSECURE_HTTP") == "true",
		// The public address, for links in e-mail, feeds and sitemap.xml,
		// and the mail server; without one, e-mail is written to the log.
		BaseURL: os.Getenv("WEBTHINGS_BASE_URL"),
		Mail:    sender,
		// /metrics, such as the output cache's counters, answers requests
		// with "Authorization: Bearer <token>"; without a token it is 404.
		MetricsToken: os.Getenv("WEBTHINGS_METRICS_TOKEN"),
		Templates:    os.DirFS("templates"),
		// The theme's JavaScript packages; see importMap.
		ImportMap:    imports,
		FormTheme:    formTheme,
		Translations: translations,
		// off, report-only or enforce decides the Content-Security-Policy
		// here; empty leaves it to the admin's Security settings.
		ContentSecurityPolicy: os.Getenv("WEBTHINGS_CONTENT_SECURITY_POLICY"),
	}

	developing := development(&config)
	app := webthings.New(config)

	// Groups order the admin's navigation. Agenda, Content, Visitors and
	// Website are translated by the admin; other names show as written.
	err = app.Use(
		events.Module("workshops", "Workshops").Group("Agenda"),
		registrations.Module("signup", "Sign-ups", registrations.ForEvents("workshops")).Group("Agenda"),

		publishing.Module("journal", "Journal").Group("Content"),
		gallery.Module("work", "Work").Group("Content"),
		lists.Module("services", "Services").Group("Content"),
		lists.Module("faq", "FAQ").Group("Content"),
		testimonials.Module("reviews", "Reviews").Group("Content"),
		people.Module("team", "Team").Group("Content"),
		directory.Module("stockists", "Stockists").Group("Content"),
		documents.Module("downloads", "Downloads").Group("Content"),

		forms.Module("contact", "Contact").Group("Visitors"),
		comments.Module("remarks", "Comments").Group("Visitors"),
		comments.Module("guestbook", "Guestbook").Group("Visitors"),
		newsletter.Module("letters", "Newsletter").Group("Visitors"),
		polls.Module("poll", "Polls").Group("Visitors"),

		banners.Module("notice", "Notices").Group("Website"),
		stats.Module().Group("Website"),
		redirects.Module().Group("Website"),
	)
	if err != nil {
		return err
	}

	// "go run . translations" writes the texts of the templates to
	// translations/translations.pot; "go run . translations check" lists the
	// texts each language's .po file still misses (add --csv for a CSV).
	if len(os.Args) > 1 && os.Args[1] == "translations" {
		return app.Translations("translations", os.Args[2:])
	}

	if err := app.Start(ctx); err != nil {
		return err
	}

	// The seed only fills an empty site; later starts keep all edits.
	if err := seed(ctx, app); err != nil {
		_ = app.Close()

		return err
	}

	// A site seeded before the seed set the logo gets the brand's mark as
	// its logo; the site runs without it when that fails.
	if err := brandLogo(ctx, app); err != nil {
		log.Printf("brand logo: %v", err)
	}

	// The seeded workshops move forward by whole weeks once they are all
	// over, so there is always a workshop to sign up for.
	if err := moveWorkshops(ctx, app, time.Now()); err != nil {
		log.Printf("workshops: %v", err)
	}

	if developing {
		log.Println("development mode (WEBTHINGS_DEVELOPMENT): templates reload from disk, detailed errors, no page cache")
	}

	return app.ListenAndServe(ctx, env("WEBTHINGS_LISTEN", "127.0.0.1:8090"))
}

// development turns on the development mode when WEBTHINGS_DEVELOPMENT is
// "true", as task dev sets it: the templates are read again from the
// templates folder on every request, so an edit shows on the next request
// without a restart; a failing page shows the detailed error page with the
// error and its template; and the output cache is off, so no page is served
// from before the edit. It is off by default: visitors must never see the
// errors' details, and run logs a line when it is on.
func development(config *webthings.Config) bool {
	if os.Getenv("WEBTHINGS_DEVELOPMENT") != "true" {
		return false
	}

	config.ReloadTemplates = true
	config.Development = true

	return true
}

// smtpSender sends e-mail through the server in WEBTHINGS_SMTP_HOST. Without
// a host it returns nil, so messages are written to the log instead.
func smtpSender() (mail.Sender, error) {
	host := os.Getenv("WEBTHINGS_SMTP_HOST")
	if host == "" {
		return nil, nil
	}

	from := os.Getenv("WEBTHINGS_SMTP_FROM")
	if from == "" {
		return nil, errors.New("WEBTHINGS_SMTP_FROM is required with WEBTHINGS_SMTP_HOST")
	}

	port := 587

	if value := os.Getenv("WEBTHINGS_SMTP_PORT"); value != "" {
		number, err := strconv.Atoi(value)
		if err != nil {
			return nil, fmt.Errorf("WEBTHINGS_SMTP_PORT: %w", err)
		}

		port = number
	}

	return mail.SMTP{
		Host:     host,
		Port:     port,
		Username: os.Getenv("WEBTHINGS_SMTP_USERNAME"),
		Password: os.Getenv("WEBTHINGS_SMTP_PASSWORD"),
		From:     from,
	}, nil
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	return fallback
}
