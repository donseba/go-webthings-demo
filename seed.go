package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"fmt"
	"time"

	webthings "github.com/donseba/go-webthings"
	"github.com/donseba/go-webthings/content"
	"github.com/donseba/go-webthings/media"
	"github.com/donseba/go-webthings/modules/events"
	"github.com/donseba/go-webthings/modules/forms"
	"github.com/donseba/go-webthings/modules/gallery"
	"github.com/donseba/go-webthings/modules/lists"
	"github.com/donseba/go-webthings/modules/publishing"
	"github.com/donseba/go-webthings/modules/testimonials"
	"github.com/donseba/go-webthings/pages"
	"github.com/donseba/go-webthings/site"
)

// seed fills an empty site with the Atelier Noord sample content. It does
// nothing once any page besides Home exists.
func seed(ctx context.Context, app *webthings.App) error {
	existing, err := app.Pages.ListLatest(ctx)
	if err != nil || len(existing) > 1 {
		return err
	}

	s := seeder{ctx: ctx, app: app}

	steps := []func() error{
		s.settings,
		s.images,
		s.services,
		s.faq,
		s.journal,
		s.workshops,
		s.work,
		s.reviews,
		s.contact,
		s.team,
		s.stockists,
		s.downloads,
		s.signups,
		s.remarks,
		s.guestbook,
		s.letters,
		s.poll,
		s.notice,
		s.redirects,
		s.pages,
	}

	for _, step := range steps {
		if err := step(); err != nil {
			return fmt.Errorf("seed: %w", err)
		}
	}

	return nil
}

type seeder struct {
	ctx    context.Context
	app    *webthings.App
	photos []string
	// postIDs and events are the IDs of the sample journal posts and
	// workshops, in the order they are seeded, for comments and sign-ups.
	postIDs []string
	// postPaths are the public URLs of the sample journal posts per locale,
	// in the same order as postIDs, so a seeded comment records the page it
	// was written on, like a visitor's comment does.
	postPaths []map[string]string
	events    []string
}

// both is a text in Dutch and English.
func both[T any](nl, en T) []content.Translation[T] {
	return []content.Translation[T]{{Locale: "nl", Text: nl}, {Locale: "en", Text: en}}
}

// add saves and publishes one item.
func add[F, T any](
	ctx context.Context,
	store content.Store[F, T],
	fields F,
	translations []content.Translation[T],
) (string, error) {
	item, err := store.Save(ctx, content.Draft[F, T]{Fields: fields, Translations: translations, Note: "Sample content"})
	if err != nil {
		return "", err
	}

	return item.ID, store.Publish(ctx, item.ID, item.Revision.ID, "nl")
}

func (s *seeder) settings() error {
	settings, err := s.app.Site.Settings(s.ctx)
	if err != nil {
		return err
	}

	settings.Timezone = "Europe/Amsterdam"
	settings.Organisation = "Atelier Noord"
	settings.Email = "hallo@ateliernoord.example"
	settings.Phone = "+31 20 123 45 67"
	settings.Address = "Noorderstraat 12\n1015 AB Amsterdam"
	settings.Profiles = []site.Profile{
		{Network: "instagram", URL: "https://www.instagram.com/atelier.noord.demo"},
		{Network: "facebook", URL: "https://www.facebook.com/atelier.noord.demo"},
		{Network: "mastodon", URL: "https://mastodon.social/@ateliernoorddemo"},
	}
	// The logo mark, also the favicon, and the image shown when a page is
	// shared; all stay editable under Site settings. The logo is in the
	// header and the Organization of the structured data.
	favicon, err := s.brand("favicon.png", "Logo van Atelier Noord", "Atelier Noord logo")
	if err != nil {
		return err
	}

	share, err := s.brand(
		"share.jpg",
		"Atelier Noord, keramiekstudio en werkplaats in Amsterdam-Noord",
		"Atelier Noord, ceramics studio and workshop in Amsterdam-Noord",
	)
	if err != nil {
		return err
	}

	settings.LogoMediaID = favicon
	settings.FaviconMediaID = favicon
	settings.SocialImageMediaID = share
	settings.Translations = []site.Translation{
		{
			Locale:         "nl",
			Title:          "Atelier Noord",
			Tagline:        "Keramiekstudio en werkplaats in Amsterdam-Noord",
			SEODescription: "Draaicursussen, workshops en handgemaakt servies uit Amsterdam-Noord.",
			Footer:         "Open woensdag tot en met zaterdag, 10:00–17:00.",
		},
		{
			Locale:         "en",
			Title:          "Atelier Noord",
			Tagline:        "Ceramics studio and workshop in Amsterdam-Noord",
			SEODescription: "Wheel-throwing courses, workshops and handmade tableware from Amsterdam-Noord.",
			Footer:         "Open Wednesday to Saturday, 10:00–17:00.",
		},
	}

	return s.app.Site.Save(s.ctx, settings)
}

// brandFiles are the site's logo mark, which is its favicon too, and the
// image shown when a page is shared.
//
//go:embed brand/favicon.png brand/share.jpg
var brandFiles embed.FS

// brandLogo makes the brand's logo mark, brand/favicon.png, the site's logo
// when the site has none, such as a site seeded before the seed set the
// logo. The mark must be in the media library already, found by its
// checksum: brandLogo uploads nothing and changes no other setting, so it
// can run at every start.
func brandLogo(ctx context.Context, app *webthings.App) error {
	settings, err := app.Site.Settings(ctx)
	if err != nil || settings.LogoMediaID != "" {
		return err
	}

	data, err := brandFiles.ReadFile("brand/favicon.png")
	if err != nil {
		return err
	}

	sum := sha256.Sum256(data)
	checksum := hex.EncodeToString(sum[:])

	assets, err := app.Media.List(ctx)
	if err != nil {
		return err
	}

	for _, asset := range assets {
		if asset.Checksum == checksum {
			settings.LogoMediaID = asset.ID

			return app.Site.Save(ctx, settings)
		}
	}

	return nil
}

// brand uploads one of brandFiles to the media library.
func (s *seeder) brand(name, nl, en string) (string, error) {
	data, err := brandFiles.ReadFile("brand/" + name)
	if err != nil {
		return "", err
	}

	return s.upload(name, data, nl, en)
}

// photoFiles are the sample photographs; photos lists them with their titles.
//
//go:embed photos/*.jpg
var photoFiles embed.FS

var photos = []struct {
	file   string
	nl, en string
}{
	{"tableware-on-table.jpg", "Servies op een houten tafel", "Tableware on a wooden table"},
	{"celadon-bowl.jpg", "Kom in celadonglazuur", "Bowl in celadon glaze"},
	{"throwing-on-the-wheel.jpg", "Draaien aan de schijf", "Throwing on the wheel"},
	{"terracotta-vase.jpg", "Vaas van terracotta", "Terracotta vase"},
	{"vase-and-bowl-rust.jpg", "Vaas en kommetje in roestglazuur", "Vase and bowl in rust glaze"},
	{"slate-blue-vase.jpg", "Vaas in leiblauw glazuur", "Vase in slate-blue glaze"},
	{"kiln.jpg", "Onze nieuwe elektrische oven", "Our new electric kiln"},
	{"cutting-clay.jpg", "Klei snijden aan de werktafel", "Cutting clay at the workbench"},
	{"teapot.jpg", "Theepot in celadon en zand", "Teapot in celadon and sand"},
	{"studio-shelves.jpg", "De werkplaats met planken vol werk", "The studio with shelves of work"},
	{"working-together.jpg", "Samen aan het werk in het atelier", "Working together in the studio"},
	{"shop-interior.jpg", "Keramiek in de winkel", "Ceramics in the shop"},
	{"market-stall.jpg", "Kraam op de Noordermarkt", "Stall at the Noordermarkt"},
	{"restaurant-table.jpg", "Ons servies in het restaurant", "Our tableware in the restaurant"},
	{"coffee-cup.jpg", "Koffie in een beker uit het atelier", "Coffee in a cup from the studio"},
}

// The sample photographs by their place in photos.
const (
	photoTable = iota
	photoCeladon
	photoWheel
	photoTerracotta
	photoRust
	photoSlate
	photoKiln
	photoClay
	photoTeapot
	photoStudio
	photoTogether
	photoShop
	photoMarket
	photoRestaurant
	photoCoffee
)

func (s *seeder) images() error {
	for _, photo := range photos {
		data, err := photoFiles.ReadFile("photos/" + photo.file)
		if err != nil {
			return err
		}

		asset, err := s.app.Media.Upload(s.ctx, photo.file, bytes.NewReader(data))
		if err != nil {
			return err
		}

		for locale, title := range map[string]string{"nl": photo.nl, "en": photo.en} {
			translation := media.Translation{Locale: locale, Title: title, AltText: title}
			if err := s.app.Media.SaveTranslation(s.ctx, asset.ID, translation); err != nil {
				return err
			}
		}

		s.photos = append(s.photos, asset.ID)
	}

	return nil
}

func (s *seeder) services() error {
	store := lists.Of(s.app, "services").Store()
	entries := []struct {
		image int
		nl    lists.Text
		en    lists.Text
	}{
		{
			photoTerracotta,
			lists.Text{Title: "Draaicursus", Body: "Acht avonden aan de draaischijf, van centreren tot je eerste set kommen."},
			lists.Text{
				Title: "Wheel-throwing course",
				Body:  "Eight evenings at the wheel, from centring clay to your first set of bowls.",
			},
		},
		{
			photoRust,
			lists.Text{
				Title: "Handvormen",
				Body:  "Duimpotten, rollen en plaatwerk: bouwen zonder draaischijf, in je eigen tempo.",
			},
			lists.Text{Title: "Hand-building", Body: "Pinch pots, coils and slabs: building without a wheel, at your own pace."},
		},
		{
			photoStudio,
			lists.Text{Title: "Studiolidmaatschap", Body: "Een vaste plank, toegang tot de ovens en vrij draaien op zaterdag."},
			lists.Text{Title: "Studio membership", Body: "Your own shelf, access to the kilns and open studio on Saturdays."},
		},
	}

	for _, entry := range entries {
		_, err := add(s.ctx, store, lists.Fields{Image: s.photos[entry.image]}, both(entry.nl, entry.en))
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *seeder) faq() error {
	store := lists.Of(s.app, "faq").Store()
	questions := [][2]lists.Text{
		{
			{
				Title: "Heb ik ervaring nodig?",
				Body:  "Nee. De meeste cursisten raken voor het eerst klei aan; we beginnen bij het begin.",
			},
			{
				Title: "Do I need experience?",
				Body:  "No. Most students touch clay for the first time; we start at the very beginning.",
			},
		},
		{
			{
				Title: "Wanneer kan ik mijn werk ophalen?",
				Body:  "Werk wordt twee keer gestookt. Reken op drie tot vier weken na de laatste les.",
			},
			{
				Title: "When can I collect my work?",
				Body:  "Work is fired twice. Expect three to four weeks after the last lesson.",
			},
		},
		{
			{
				Title: "Is het servies vaatwasserbestendig?",
				Body:  "Ja, al onze glazuren zijn voedselveilig en geschikt voor de vaatwasser.",
			},
			{Title: "Is the tableware dishwasher safe?", Body: "Yes, all our glazes are food safe and dishwasher proof."},
		},
		{
			{
				Title: "Kan ik een cursus cadeau geven?",
				Body:  "Zeker. Stuur ons een bericht via het contactformulier en we maken een cadeaubon.",
			},
			{
				Title: "Can I give a course as a gift?",
				Body:  "Of course. Send us a message through the contact form and we will make a gift card.",
			},
		},
	}

	for _, question := range questions {
		if _, err := add(s.ctx, store, lists.Fields{}, both(question[0], question[1])); err != nil {
			return err
		}
	}

	return nil
}

func (s *seeder) journal() error {
	service := publishing.Of(s.app, "journal")

	process, err := add(s.ctx, service.Categories(), struct{}{}, both(
		publishing.CategoryText{Slug: "werkwijze", Title: "Werkwijze"},
		publishing.CategoryText{Slug: "process", Title: "Process"},
	))
	if err != nil {
		return err
	}

	studio, err := add(s.ctx, service.Categories(), struct{}{}, both(
		publishing.CategoryText{Slug: "studio", Title: "Studio"},
		publishing.CategoryText{Slug: "studio", Title: "Studio"},
	))
	if err != nil {
		return err
	}

	now := time.Now()
	posts := []struct {
		days       int
		image      int
		categories content.IDs
		nl, en     publishing.PostText
	}{
		{
			3,
			photoCeladon,
			content.IDs{process},
			publishing.PostText{
				Slug:    "celadon-in-zes-stappen",
				Title:   "Celadon in zes stappen",
				Excerpt: "Hoe we ons groenblauwe glazuur mengen, testen en stoken.",
				Body:    "## Waarom celadon\n\nCeladon is een van de oudste glazuren die er bestaan. Het **ijzer** in het glazuur kleurt groen in een reducerende stook.\n\n## De stappen\n\n1. Afwegen\n2. Zeven\n3. Testtegels\n4. Dompelen\n5. Drogen\n6. Stoken op 1260 °C\n\n> Geduld is het belangrijkste ingrediënt.",
			},
			publishing.PostText{
				Slug:    "celadon-in-six-steps",
				Title:   "Celadon in six steps",
				Excerpt: "How we mix, test and fire our blue-green glaze.",
				Body:    "## Why celadon\n\nCeladon is one of the oldest glazes in existence. The **iron** in the glaze turns green in a reduction firing.\n\n## The steps\n\n1. Weighing\n2. Sieving\n3. Test tiles\n4. Dipping\n5. Drying\n6. Firing at 1260 °C\n\n> Patience is the main ingredient.",
			},
		},
		{
			12,
			photoKiln,
			content.IDs{studio},
			publishing.PostText{
				Slug:    "een-nieuwe-oven",
				Title:   "Een nieuwe oven",
				Excerpt: "Onze tweede oven is binnen: meer ruimte voor cursistenwerk.",
				Body:    "Na een jaar sparen staat hij er: een elektrische oven van 120 liter. Daarmee stoken we het werk van cursisten voortaan elke week.",
			},
			publishing.PostText{
				Slug:    "a-new-kiln",
				Title:   "A new kiln",
				Excerpt: "Our second kiln has arrived: more room for student work.",
				Body:    "After a year of saving it is here: a 120 litre electric kiln. From now on we fire student work every week.",
			},
		},
		{
			26,
			photoClay,
			content.IDs{process, studio},
			publishing.PostText{
				Slug:    "klei-uit-de-buurt",
				Title:   "Klei uit de buurt",
				Excerpt: "We graven, zeven en testen klei van een boerderij in Waterland.",
				Body:    "Lokale klei is grillig, en daar houden we van. Na drie testrondes hebben we een mengsel dat goed draait en mooi roestbruin bakt.",
			},
			publishing.PostText{
				Slug:    "local-clay",
				Title:   "Local clay",
				Excerpt: "We dig, sieve and test clay from a farm in Waterland.",
				Body:    "Local clay is unpredictable, and we love that. After three rounds of tests we have a blend that throws well and fires a warm rust brown.",
			},
		},
	}

	for _, post := range posts {
		fields := publishing.PostFields{
			Date:       now.AddDate(0, 0, -post.days),
			Image:      s.photos[post.image],
			Categories: post.categories,
		}

		id, err := add(s.ctx, service.Store(), fields, both(post.nl, post.en))
		if err != nil {
			return err
		}

		s.postIDs = append(s.postIDs, id)
		s.postPaths = append(s.postPaths, map[string]string{
			"nl": "/nl/journaal/" + post.nl.Slug,
			"en": "/en/journal/" + post.en.Slug,
		})
	}

	return nil
}

func (s *seeder) workshops() error {
	store := events.Of(s.app, "workshops").Store()
	zone, _ := time.LoadLocation("Europe/Amsterdam")
	day := func(offset, hour int) time.Time {
		date := time.Now().In(zone).AddDate(0, 0, offset)

		return time.Date(date.Year(), date.Month(), date.Day(), hour, 0, 0, 0, zone)
	}

	workshops := []struct {
		start, end time.Time
		image      int
		nl, en     events.EventText
	}{
		{
			day(5, 19),
			day(5, 22),
			photoWheel,
			events.EventText{
				Slug:    "kennismaken-met-de-draaischijf",
				Title:   "Kennismaken met de draaischijf",
				Summary: "Een avond proberen, zonder verplichtingen.",
			},
			events.EventText{
				Slug:    "try-the-wheel",
				Title:   "Try the wheel",
				Summary: "One evening to give it a go, no strings attached.",
			},
		},
		{
			day(12, 10), day(12, 16), photoCeladon,
			events.EventText{Slug: "glazuurdag", Title: "Glazuurdag", Summary: "Een dag lang glazuren mengen en testen."},
			events.EventText{Slug: "glaze-day", Title: "Glaze day", Summary: "A full day of mixing and testing glazes."},
		},
		{
			day(20, 14),
			day(20, 17),
			photoTeapot,
			events.EventText{
				Slug:    "theepot-masterclass",
				Title:   "Masterclass theepot",
				Summary: "Tuit, oor en deksel die passen: voor gevorderden.",
			},
			events.EventText{
				Slug:    "teapot-masterclass",
				Title:   "Teapot masterclass",
				Summary: "Spout, handle and lid that fit: for experienced makers.",
			},
		},
		{
			day(-30, 10),
			day(-30, 16),
			photoTogether,
			events.EventText{
				Slug:    "open-atelier",
				Title:   "Open atelier",
				Summary: "Kijken, proberen en koffie drinken in de werkplaats.",
			},
			events.EventText{
				Slug:    "open-studio",
				Title:   "Open studio",
				Summary: "Look around, try something and have coffee in the studio.",
			},
		},
	}

	for _, workshop := range workshops {
		end := workshop.end
		fields := events.EventFields{
			Start:    workshop.start,
			End:      &end,
			Timezone: "Europe/Amsterdam",
			Image:    s.photos[workshop.image],
		}

		workshop.nl.Body = workshop.nl.Summary + "\n\nMateriaal, koffie en het stoken van je werk zijn inbegrepen."
		workshop.en.Body = workshop.en.Summary + "\n\nMaterials, coffee and firing your work are included."

		for _, text := range []*events.EventText{&workshop.nl, &workshop.en} {
			text.Location = "Atelier Noord"
			text.Address = "Noorderstraat 12, Amsterdam"
		}

		id, err := add(s.ctx, store, fields, both(workshop.nl, workshop.en))
		if err != nil {
			return err
		}

		s.events = append(s.events, id)
	}

	return nil
}

func (s *seeder) work() error {
	store := gallery.Of(s.app, "work").Store()
	albums := []struct {
		images []int
		nl, en gallery.AlbumText
	}{
		{
			[]int{photoRust, photoTable, photoCeladon, photoTeapot, photoRestaurant, photoCoffee},
			gallery.AlbumText{Slug: "servies", Title: "Servies", Description: "Kommen, borden en bekers voor elke dag."},
			gallery.AlbumText{Slug: "tableware", Title: "Tableware", Description: "Bowls, plates and cups for every day."},
		},
		{
			[]int{photoWheel, photoStudio, photoClay, photoKiln, photoTogether, photoTerracotta, photoSlate},
			gallery.AlbumText{
				Slug:        "uit-de-werkplaats",
				Title:       "Uit de werkplaats",
				Description: "Proeven, vazen en een kijkje achter de schermen.",
			},
			gallery.AlbumText{
				Slug:        "from-the-studio",
				Title:       "From the studio",
				Description: "Tests, vases and a look behind the scenes.",
			},
		},
	}

	for _, album := range albums {
		var ids content.IDs
		for _, index := range album.images {
			ids = append(ids, s.photos[index])
		}

		fields := gallery.AlbumFields{Cover: ids[0], Images: ids}
		if _, err := add(s.ctx, store, fields, both(album.nl, album.en)); err != nil {
			return err
		}
	}

	return nil
}

func (s *seeder) reviews() error {
	store := testimonials.Of(s.app, "reviews").Store()
	reviews := []struct {
		author string
		nl, en testimonials.Text
	}{
		{
			"Sanne de Vries",
			testimonials.Text{
				Quote: "Na acht weken draai ik kommen waar ik echt uit eet. De lessen zijn rustig en persoonlijk.",
				Role:  "Cursist draaicursus",
			},
			testimonials.Text{
				Quote: "After eight weeks I throw bowls I actually eat from. The lessons are calm and personal.",
				Role:  "Wheel-throwing student",
			},
		},
		{
			"Ahmed Benali",
			testimonials.Text{
				Quote: "De glazuurdag was een openbaring. Ik heb meer geleerd dan in een jaar zelf proberen.",
				Role:  "Hobbykeramist",
			},
			testimonials.Text{
				Quote: "The glaze day was a revelation. I learnt more than in a year of trying on my own.",
				Role:  "Hobby potter",
			},
		},
		{
			"Restaurant Kade",
			testimonials.Text{
				Quote:        "Ons servies komt uit Atelier Noord. Gasten vragen er elke week naar.",
				Role:         "Chef",
				Organisation: "Restaurant Kade",
			},
			testimonials.Text{
				Quote:        "Our tableware comes from Atelier Noord. Guests ask about it every week.",
				Role:         "Chef",
				Organisation: "Restaurant Kade",
			},
		},
	}

	for _, review := range reviews {
		fields := testimonials.Fields{Author: review.author}
		if _, err := add(s.ctx, store, fields, both(review.nl, review.en)); err != nil {
			return err
		}
	}

	return nil
}

func (s *seeder) contact() error {
	fields, nl, err := forms.ParseFields(`name* | text | Naam
email* | email | E-mailadres
interest | select | Waar gaat het over?
- course | Een cursus
- workshop | Een workshop
- order | Servies op bestelling
- other | Iets anders
message* | textarea | Bericht | Vertel ons waar we je mee kunnen helpen
newsletter | checkbox | Stuur mij de nieuwsbrief
`)
	if err != nil {
		return err
	}

	_, en, err := forms.ParseFields(`name* | text | Name
email* | email | Email address
interest | select | What is it about?
- course | A course
- workshop | A workshop
- order | Tableware to order
- other | Something else
message* | textarea | Message | Tell us how we can help
newsletter | checkbox | Send me the newsletter
`)
	if err != nil {
		return err
	}

	store := forms.Of(s.app, "contact").Store()
	_, err = add(s.ctx, store, forms.Definition{Fields: fields}, both(
		forms.Text{
			Title:   "Contact",
			Intro:   "Vragen over een cursus, een workshop of servies op bestelling? We reageren binnen twee werkdagen.",
			Success: "Dank je wel! We nemen snel contact met je op.",
			Labels:  nl,
		},
		forms.Text{
			Title:   "Contact",
			Intro:   "Questions about a course, a workshop or tableware to order? We reply within two working days.",
			Success: "Thank you! We will be in touch soon.",
			Labels:  en,
		},
	))

	return err
}

// pages builds the menu: each larger module on its own page in the menu, the
// smaller ones below About and Practical. Pages with a module cannot have
// pages below them.
func (s *seeder) pages() error {
	home, err := s.app.Pages.Home(s.ctx)
	if err != nil {
		return err
	}

	_, err = s.page(
		home,
		home,
		"",
		0,
		pages.Text{
			Title:       "Klei, vuur en geduld",
			Description: "Keramiekstudio en werkplaats in Amsterdam-Noord. Leer draaien, glazuren en stoken in kleine groepen.",
			Body:        "Welkom bij **Atelier Noord**. Sinds 2019 maken we servies voor restaurants en geven we les aan iedereen die met klei wil leren werken.",
		},
		pages.Text{
			Title:       "Clay, fire and patience",
			Description: "Ceramics studio and workshop in Amsterdam-Noord. Learn to throw, glaze and fire in small groups.",
			Body:        "Welcome to **Atelier Noord**. Since 2019 we make tableware for restaurants and teach anyone who wants to learn to work with clay.",
		},
	)
	if err != nil {
		return err
	}

	modules := []struct {
		module string
		nl, en pages.Text
	}{
		{"workshops", pages.Text{Slug: "workshops", Title: "Workshops"}, pages.Text{Slug: "workshops", Title: "Workshops"}},
		{"services", pages.Text{Slug: "cursussen", Title: "Cursussen"}, pages.Text{Slug: "courses", Title: "Courses"}},
		{"work", pages.Text{Slug: "werk", Title: "Werk"}, pages.Text{Slug: "work", Title: "Work"}},
		{"journal", pages.Text{Slug: "journaal", Title: "Journaal"}, pages.Text{Slug: "journal", Title: "Journal"}},
	}

	for i, module := range modules {
		if _, err := s.page("", home, module.module, i+1, module.nl, module.en); err != nil {
			return err
		}
	}

	about, err := s.page(
		"",
		home,
		"",
		5,
		pages.Text{
			Slug:        "over-ons",
			Title:       "Over ons",
			Description: "Twee draaischijven, drie ovens en veel geduld.",
			Body:        "Atelier Noord is opgericht door twee keramisten die elkaar ontmoetten op de kunstacademie.\n\n## De werkplaats\n\nIn een oude scheepswerf in Noord hebben we ruimte voor acht cursisten tegelijk.",
		},
		pages.Text{
			Slug:        "about",
			Title:       "About",
			Description: "Two wheels, three kilns and lots of patience.",
			Body:        "Atelier Noord was founded by two ceramicists who met at art school.\n\n## The studio\n\nIn an old shipyard in Noord we have room for eight students at a time.",
		},
	)
	if err != nil {
		return err
	}

	practical, err := s.page(
		"",
		home,
		"",
		6,
		pages.Text{
			Slug:        "praktisch",
			Title:       "Praktisch",
			Description: "Aanmelden, betalen, annuleren en je werk ophalen.",
			Body:        "## Aanmelden en betalen\n\nJe meldt je aan op de pagina van een workshop. Je krijgt meteen een bevestiging met een persoonlijke code en een betaallink.\n\n## Annuleren\n\nTot een week voor de workshop krijg je het volledige bedrag terug. Annuleren doe je zelf onder *Mijn aanmelding*.\n\n## Je werk ophalen\n\nWerk wordt twee keer gestookt. Reken op drie tot vier weken; we sturen je een bericht als het klaar is.",
		},
		pages.Text{
			Slug:        "practical",
			Title:       "Practical",
			Description: "Signing up, paying, cancelling and collecting your work.",
			Body:        "## Signing up and paying\n\nSign up on the page of a workshop. You get a confirmation right away, with a personal code and a payment link.\n\n## Cancelling\n\nUp to a week before the workshop you get the full amount back. Cancel yourself under *My booking*.\n\n## Collecting your work\n\nWork is fired twice. Expect three to four weeks; we send you a message when it is ready.",
		},
	)
	if err != nil {
		return err
	}

	_, err = s.page("", home, "contact", 7,
		pages.Text{Slug: "contact", Title: "Contact"},
		pages.Text{Slug: "contact", Title: "Contact"})
	if err != nil {
		return err
	}

	children := []struct {
		parent   string
		module   string
		position int
		nl, en   pages.Text
	}{
		{
			about, "team", 1,
			pages.Text{Slug: "team", Title: "Team"},
			pages.Text{Slug: "team", Title: "Team"},
		},
		{
			about, "reviews", 2,
			pages.Text{Slug: "ervaringen", Title: "Ervaringen"},
			pages.Text{Slug: "reviews", Title: "Reviews"},
		},
		{
			about, "stockists", 3,
			pages.Text{Slug: "verkooppunten", Title: "Verkooppunten"},
			pages.Text{Slug: "where-to-buy", Title: "Where to buy"},
		},
		{
			about, "guestbook", 4,
			pages.Text{Slug: "gastenboek", Title: "Gastenboek"},
			pages.Text{Slug: "guestbook", Title: "Guestbook"},
		},
		{
			practical, "faq", 1,
			pages.Text{Slug: "veelgestelde-vragen", Title: "Veelgestelde vragen"},
			pages.Text{Slug: "faq", Title: "FAQ"},
		},
		{
			practical, "signup", 2,
			pages.Text{Slug: "mijn-aanmelding", Title: "Mijn aanmelding"},
			pages.Text{Slug: "my-booking", Title: "My booking"},
		},
		{
			practical, "downloads", 3,
			pages.Text{Slug: "downloads", Title: "Downloads"},
			pages.Text{Slug: "downloads", Title: "Downloads"},
		},
		{
			practical, "letters", 4,
			pages.Text{Slug: "nieuwsbrief", Title: "Nieuwsbrief"},
			pages.Text{Slug: "newsletter", Title: "Newsletter"},
		},
	}

	for _, child := range children {
		if _, err := s.page("", child.parent, child.module, child.position, child.nl, child.en); err != nil {
			return err
		}
	}

	return nil
}

// page saves and publishes a page and returns its ID; id is empty for a new page.
func (s *seeder) page(id, parent, module string, position int, nl, en pages.Text) (string, error) {
	fields := pages.Fields{ParentID: parent, Module: module, Position: position}

	saved, err := s.app.Pages.Save(s.ctx, pages.Draft{ID: id, Fields: fields, Translations: both(nl, en)})
	if err != nil {
		return "", err
	}

	return saved.ID, s.app.Pages.Publish(s.ctx, saved.ID, saved.Revision.ID)
}
