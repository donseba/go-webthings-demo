package main

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	webthings "github.com/donseba/go-webthings"
	"github.com/donseba/go-webthings/mail"
	"github.com/donseba/go-webthings/sitetest"
)

// TestSMTPSender checks that the mail server comes from the environment:
// without WEBTHINGS_SMTP_HOST e-mail is logged, a host needs a sender, the
// port defaults to 587, and a port that is not a number is refused.
func TestSMTPSender(t *testing.T) {
	for _, key := range []string{"HOST", "PORT", "USERNAME", "PASSWORD", "FROM"} {
		t.Setenv("WEBTHINGS_SMTP_"+key, "")
	}

	if sender, err := smtpSender(); sender != nil || err != nil {
		t.Fatalf("without a host: %v %v", sender, err)
	}

	t.Setenv("WEBTHINGS_SMTP_HOST", "smtp.example.test")

	if _, err := smtpSender(); err == nil || !strings.Contains(err.Error(), "WEBTHINGS_SMTP_FROM") {
		t.Fatalf("a host without a sender: %v", err)
	}

	t.Setenv("WEBTHINGS_SMTP_FROM", "Atelier Noord <website@example.test>")
	t.Setenv("WEBTHINGS_SMTP_USERNAME", "website")
	t.Setenv("WEBTHINGS_SMTP_PASSWORD", "test-only")

	sender, err := smtpSender()
	if err != nil {
		t.Fatal(err)
	}

	want := mail.SMTP{
		Host:     "smtp.example.test",
		Port:     587,
		Username: "website",
		Password: "test-only",
		From:     "Atelier Noord <website@example.test>",
	}
	if sender != want {
		t.Fatalf("sender: %#v", sender)
	}

	t.Setenv("WEBTHINGS_SMTP_PORT", "twenty-five")

	if _, err := smtpSender(); err == nil || !strings.Contains(err.Error(), "WEBTHINGS_SMTP_PORT") {
		t.Fatalf("a port that is not a number: %v", err)
	}
}

// TestDevelopment starts a site configured as task dev does, with
// WEBTHINGS_DEVELOPMENT=true and the output cache as run leaves it: an edit
// to a template file shows on the next request without a restart, and a
// failing page shows the error. Without it, neither happens.
func TestDevelopment(t *testing.T) {
	t.Setenv("WEBTHINGS_DEVELOPMENT", "")

	if development(&webthings.Config{}) {
		t.Fatal("development mode without WEBTHINGS_DEVELOPMENT")
	}

	t.Setenv("WEBTHINGS_DEVELOPMENT", "true")

	dir := t.TempDir()
	page := filepath.Join(dir, "page.gohtml")

	write := func(template string) {
		err := os.WriteFile(page, []byte(template), 0o644)
		if err != nil {
			t.Fatal(err)
		}
	}

	write(`<main id="content">first edit</main>`)

	s := sitetest.StartWith(t, sitetest.Options{
		Templates: dir,
		Configure: func(config *webthings.Config) {
			// run leaves the output cache to the site setting.
			config.OutputCacheTTL = 0
			development(config)
		},
	})
	visitor := s.Visitor()

	visitor.Expect("/", http.StatusOK, "first edit")

	write(`<main id="content">second edit</main>`)
	visitor.Expect("/", http.StatusOK, "second edit")

	write(`<main id="content">{{.Missing}}</main>`)

	status, body := visitor.Get("/")
	if status != http.StatusInternalServerError || !strings.Contains(body, "Missing") {
		t.Fatalf("broken page without the error: %d\n%s", status, body)
	}
}
