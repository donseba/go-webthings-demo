package main

import (
	"bytes"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/donseba/go-webthings/db"
	"github.com/donseba/go-webthings/media"
	"github.com/donseba/go-webthings/modules/banners"
	"github.com/donseba/go-webthings/modules/comments"
	"github.com/donseba/go-webthings/modules/directory"
	"github.com/donseba/go-webthings/modules/documents"
	"github.com/donseba/go-webthings/modules/newsletter"
	"github.com/donseba/go-webthings/modules/people"
	"github.com/donseba/go-webthings/modules/polls"
	"github.com/donseba/go-webthings/modules/redirects"
	"github.com/donseba/go-webthings/modules/registrations"
)

// The sample content of the newer modules (comments, newsletter, team, …). Like the rest
// of the seed it only runs on an empty site (see seed).

// upload stores a generated file in the media library with a title in both
// languages and returns its ID.
func (s *seeder) upload(name string, data []byte, nl, en string) (string, error) {
	asset, err := s.app.Media.Upload(s.ctx, name, bytes.NewReader(data))
	if err != nil {
		return "", err
	}

	for locale, title := range map[string]string{"nl": nl, "en": en} {
		translation := media.Translation{Locale: locale, Title: title, AltText: title}
		if err := s.app.Media.SaveTranslation(s.ctx, asset.ID, translation); err != nil {
			return "", err
		}
	}

	return asset.ID, nil
}

func (s *seeder) team() error {
	store := people.Of(s.app, "team").Store()
	members := []struct {
		name, slug, email string
		nl, en            people.Text
	}{
		{
			"Marit Visser",
			"marit-visser",
			"marit@ateliernoord.example",
			people.Text{
				Group: "Oprichters",
				Role:  "Keramist en docent",
				Bio:   "Marit draait al twintig jaar en ontwerpt het servies voor onze restaurants. Ze geeft de draaicursus op dinsdag.",
			},
			people.Text{
				Group: "Founders",
				Role:  "Ceramicist and teacher",
				Bio:   "Marit has been throwing for twenty years and designs the tableware for our restaurants. She teaches the Tuesday wheel course.",
			},
		},
		{
			"Daan Mulder",
			"daan-mulder",
			"daan@ateliernoord.example",
			people.Text{
				Group: "Oprichters",
				Role:  "Glazuur en ovens",
				Bio:   "Daan mengt al onze glazuren en houdt de ovens draaiende. Vraag hem gerust naar celadon.",
			},
			people.Text{
				Group: "Founders",
				Role:  "Glazes and kilns",
				Bio:   "Daan mixes all our glazes and keeps the kilns running. Feel free to ask him about celadon.",
			},
		},
		{
			"Yara Haddad",
			"yara-haddad",
			"",
			people.Text{
				Group: "Docenten",
				Role:  "Docent handvormen",
				Bio:   "Yara bouwt beelden en vazen uit rollen en platen, en leert je hoe je zonder draaischijf kunt werken.",
			},
			people.Text{
				Group: "Teachers",
				Role:  "Hand-building teacher",
				Bio:   "Yara builds sculptures and vases from coils and slabs, and shows you how to work without a wheel.",
			},
		},
		{
			"Joris de Boer",
			"joris-de-boer",
			"",
			people.Text{
				Group: "Docenten",
				Role:  "Docent draaicursus",
				Bio:   "Joris geeft de avondcursussen en de kennismakingsavonden. Geduldig, en altijd met koffie.",
			},
			people.Text{
				Group: "Teachers",
				Role:  "Wheel-throwing teacher",
				Bio:   "Joris teaches the evening courses and the taster evenings. Patient, and always with coffee.",
			},
		},
	}

	// The faces in the portraits: the team page shows square crops, and the
	// focal points keep the faces in view instead of the middle of the
	// upright photos.
	faces := map[string]media.FocalPoint{
		"marit-visser":  {X: 0.45, Y: 0.37},
		"daan-mulder":   {X: 0.51, Y: 0.35},
		"yara-haddad":   {X: 0.56, Y: 0.35},
		"joris-de-boer": {X: 0.44, Y: 0.32},
	}

	for _, member := range members {
		// Each member's portrait is photos/<slug>.jpg.
		data, err := photoFiles.ReadFile("photos/" + member.slug + ".jpg")
		if err != nil {
			return err
		}

		photo, err := s.upload(member.slug+".jpg", data, member.name, member.name)
		if err != nil {
			return err
		}

		if err := s.app.Media.SetFocalPoint(s.ctx, photo, faces[member.slug]); err != nil {
			return err
		}

		fields := people.Fields{Slug: member.slug, Name: member.name, Photo: photo, Email: member.email}
		if _, err := add(s.ctx, store, fields, both(member.nl, member.en)); err != nil {
			return err
		}
	}

	return nil
}

func (s *seeder) stockists() error {
	service := directory.Of(s.app, "stockists")

	shops, err := add(s.ctx, service.Categories(), directory.CategoryFields{Slug: "winkels"}, both(
		directory.CategoryText{Name: "Winkels", Summary: "Hier koop je ons servies en losse stukken."},
		directory.CategoryText{Name: "Shops", Summary: "Where you can buy our tableware and one-off pieces."},
	))
	if err != nil {
		return err
	}

	tables, err := add(s.ctx, service.Categories(), directory.CategoryFields{Slug: "horeca"}, both(
		directory.CategoryText{Name: "Cafés en restaurants", Summary: "Hier eet en drink je uit ons servies."},
		directory.CategoryText{Name: "Cafés and restaurants", Summary: "Where you eat and drink from our tableware."},
	))
	if err != nil {
		return err
	}

	// open gives the same opening time to several weekdays.
	open := func(from, until string, days ...time.Weekday) directory.OpeningHours {
		var hours directory.OpeningHours

		for _, day := range days {
			hours[day] = directory.Day{Open: from, Close: until}
		}

		return hours
	}

	daily := open("08:00", "17:00", time.Monday, time.Tuesday, time.Wednesday, time.Thursday, time.Friday)
	daily[time.Saturday] = directory.Day{Open: "09:00", Close: "17:00"}
	daily[time.Sunday] = directory.Day{Open: "09:00", Close: "17:00"}

	entries := []struct {
		fields directory.EntryFields
		nl, en directory.EntryText
	}{
		{
			directory.EntryFields{
				Category: shops, Slug: "het-kleihuis", Name: "Het Kleihuis", Image: s.photos[photoShop],
				Website: "https://kleihuis.example", Phone: "+31 20 765 43 21",
				Street: "NDSM-Plein 28", Postcode: "1033 WB", City: "Amsterdam",
				Latitude: coordinate(52.4012), Longitude: coordinate(4.8935),
				Hours: open("11:00", "18:00", time.Wednesday, time.Thursday, time.Friday, time.Saturday, time.Sunday),
			},
			directory.EntryText{
				Summary:     "Conceptstore op de NDSM-werf met werk van tien Amsterdamse makers.",
				Description: "Het Kleihuis verkoopt onze kommen en bekers, en elk seizoen een kleine serie losse stukken.",
				Keywords:    "servies, kommen, cadeau",
			},
			directory.EntryText{
				Summary:     "Concept store at the NDSM wharf with work by ten Amsterdam makers.",
				Description: "Het Kleihuis sells our bowls and cups, and a small series of one-off pieces every season.",
				Keywords:    "tableware, bowls, gift",
			},
		},
		{
			directory.EntryFields{
				Category: shops, Slug: "noordermarkt-kraam", Name: "Kraam op de Noordermarkt", Image: s.photos[photoMarket],
				Street: "Noordermarkt", Postcode: "1015 MV", City: "Amsterdam",
				Hours: open("09:00", "16:00", time.Saturday),
			},
			directory.EntryText{
				Summary:   "Onze eigen kraam: tweede keus, proefstukken en nieuw werk.",
				Keywords:  "markt, tweede keus",
				HoursNote: "Alleen op zaterdag, niet bij regen.",
			},
			directory.EntryText{
				Summary:   "Our own stall: seconds, test pieces and new work.",
				Keywords:  "market, seconds",
				HoursNote: "Saturdays only, not when it rains.",
			},
		},
		{
			directory.EntryFields{
				Category: tables, Slug: "restaurant-kade", Name: "Restaurant Kade", Image: s.photos[photoRestaurant],
				Website: "https://restaurantkade.example", Email: "reserveren@restaurantkade.example",
				Street: "Buiksloterkade 4", Postcode: "1031 KA", City: "Amsterdam",
				Latitude: coordinate(52.3841), Longitude: coordinate(4.9025),
				Hours: open("17:30", "23:00", time.Tuesday, time.Wednesday, time.Thursday, time.Friday, time.Saturday),
			},
			directory.EntryText{Summary: "Seizoensmenu op borden en kommen in roestglazuur.", Keywords: "diner, restaurant"},
			directory.EntryText{Summary: "A seasonal menu on plates and bowls in rust glaze.", Keywords: "dinner, restaurant"},
		},
		{
			directory.EntryFields{
				Category: tables, Slug: "koffiebar-pont", Name: "Koffiebar Pont", Image: s.photos[photoCoffee],
				Street: "Pontplein 2", Postcode: "1021 JX", City: "Amsterdam",
				Latitude: coordinate(52.3825), Longitude: coordinate(4.9002),
				Hours: daily,
			},
			directory.EntryText{
				Summary:  "Koffie uit onze celadon bekers, met uitzicht op het IJ.",
				Keywords: "koffie, ontbijt, bekers",
			},
			directory.EntryText{
				Summary:  "Coffee from our celadon cups, with a view of the IJ.",
				Keywords: "coffee, breakfast, cups",
			},
		},
	}

	for _, entry := range entries {
		if _, err := add(s.ctx, service.Store(), entry.fields, both(entry.nl, entry.en)); err != nil {
			return err
		}
	}

	return nil
}

func coordinate(value float64) sql.NullFloat64 {
	return sql.NullFloat64{Float64: value, Valid: true}
}

func (s *seeder) downloads() error {
	store := documents.Of(s.app, "downloads").Store()
	year := time.Now().Year()

	files := []struct {
		name   string
		lines  []string
		date   time.Time
		nl, en documents.Text
	}{
		{
			name: "prijslijst-price-list.pdf",
			lines: []string{
				fmt.Sprintf("Prijslijst %d / Price list %d", year, year),
				"",
				"Draaicursus, 8 avonden / Wheel-throwing course, 8 evenings ..... EUR 320",
				"Handvormen, 6 lessen / Hand-building, 6 lessons ................ EUR 240",
				"Studiolidmaatschap per maand / Studio membership per month ..... EUR 85",
				"Kennismaken met de draaischijf / Try the wheel ................. EUR 45",
				"Glazuurdag / Glaze day ......................................... EUR 95",
				"Masterclass theepot / Teapot masterclass ....................... EUR 120",
				"",
				"Materiaal, koffie en stoken inbegrepen.",
				"Materials, coffee and firing included.",
				"",
				"Atelier Noord · Noorderstraat 12 · 1015 AB Amsterdam",
			},
			date: time.Date(year, time.January, 6, 0, 0, 0, 0, time.UTC),
			nl: documents.Text{
				Title:       fmt.Sprintf("Prijslijst %d", year),
				Category:    "Cursussen",
				Description: "Alle cursussen, workshops en het studiolidmaatschap op een rij.",
			},
			en: documents.Text{
				Title:       fmt.Sprintf("Price list %d", year),
				Category:    "Courses",
				Description: "All courses, workshops and the studio membership at a glance.",
			},
		},
		{
			name: "glazuren-glazes.pdf",
			lines: []string{
				"Glazuren en voedselveiligheid / Glazes and food safety",
				"",
				"Al onze glazuren zijn loodvrij en getest op voedselveiligheid.",
				"All our glazes are lead-free and tested for food safety.",
				"",
				"Roestbruin / Rust brown ........ 1240 °C, vaatwasserbestendig / dishwasher safe",
				"Celadon ........................ 1260 °C, vaatwasserbestendig / dishwasher safe",
				"Zandwit / Sand white ........... 1240 °C, vaatwasserbestendig / dishwasher safe",
				"",
				"Vragen? hallo@ateliernoord.example",
			},
			date: time.Date(year, time.March, 2, 0, 0, 0, 0, time.UTC),
			nl: documents.Text{
				Title:       "Glazuren en voedselveiligheid",
				Category:    "Werkplaats",
				Description: "Welke glazuren we gebruiken, op welke temperatuur we stoken en wat in de vaatwasser mag.",
			},
			en: documents.Text{
				Title:       "Glazes and food safety",
				Category:    "Studio",
				Description: "Which glazes we use, the temperatures we fire at and what can go in the dishwasher.",
			},
		},
	}

	for _, file := range files {
		asset, err := s.upload(file.name, pdf(file.lines), file.nl.Title, file.en.Title)
		if err != nil {
			return err
		}

		fields := documents.Fields{File: asset, Date: file.date}
		if _, err := add(s.ctx, store, fields, both(file.nl, file.en)); err != nil {
			return err
		}
	}

	return nil
}

// pdf writes a one-page A4 PDF: the first line as a heading, the others as
// text. Characters outside Windows-1252 are left out.
func pdf(lines []string) []byte {
	var page strings.Builder

	page.WriteString("BT\n/F1 18 Tf\n56 770 Td\n")

	for i, line := range lines {
		if i == 1 {
			page.WriteString("/F2 11 Tf\n")
		}

		fmt.Fprintf(&page, "(%s) Tj\n0 -22 Td\n", pdfText(line))
	}

	page.WriteString("ET\n")

	objects := []string{
		"<< /Type /Catalog /Pages 2 0 R >>",
		"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 595 842] /Contents 4 0 R /Resources << /Font << /F1 5 0 R /F2 6 0 R >> >> >>",
		fmt.Sprintf("<< /Length %d >>\nstream\n%sendstream", page.Len(), page.String()),
		"<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica-Bold /Encoding /WinAnsiEncoding >>",
		"<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica /Encoding /WinAnsiEncoding >>",
	}

	var out bytes.Buffer

	out.WriteString("%PDF-1.4\n")

	offsets := make([]int, len(objects))
	for i, object := range objects {
		offsets[i] = out.Len()
		fmt.Fprintf(&out, "%d 0 obj\n%s\nendobj\n", i+1, object)
	}

	start := out.Len()
	fmt.Fprintf(&out, "xref\n0 %d\n0000000000 65535 f \n", len(objects)+1)

	for _, offset := range offsets {
		fmt.Fprintf(&out, "%010d 00000 n \n", offset)
	}

	fmt.Fprintf(&out, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(objects)+1, start)

	return out.Bytes()
}

// pdfText escapes a line for a PDF string in WinAnsiEncoding.
func pdfText(line string) string {
	var text strings.Builder

	for _, r := range line {
		switch {
		case r == '(' || r == ')' || r == '\\':
			text.WriteByte('\\')
			text.WriteRune(r)
		case r < 0x80:
			text.WriteRune(r)
		case r >= 0xA0 && r <= 0xFF:
			text.WriteByte(byte(r))
		}
	}

	return text.String()
}

// signups opens registration for the upcoming workshops, with a few places
// already taken.
func (s *seeder) signups() error {
	service := registrations.Of(s.app, "signup")
	settings := []registrations.Settings{
		{Capacity: 8, AdultPrice: 4500, ChildPrice: 2500, AllowChildren: true},
		{Capacity: 10, AdultPrice: 9500},
		{Capacity: 6, AdultPrice: 12000},
	}

	for i, setting := range settings {
		setting.SubjectID = s.events[i]
		setting.Open = true
		setting.Texts = []registrations.SubjectText{
			{
				Locale:      "nl",
				Notes:       "Draag kleding die vies mag worden. Annuleren kan tot een week van tevoren.",
				PaymentLink: "https://betalen.ateliernoord.example/workshops",
			},
			{
				Locale:      "en",
				Notes:       "Wear clothes that may get dirty. You can cancel up to a week before.",
				PaymentLink: "https://betalen.ateliernoord.example/workshops?lang=en",
			},
		}

		if err := service.SaveSettings(s.ctx, setting); err != nil {
			return err
		}
	}

	signups := []registrations.Signup{
		{SubjectID: s.events[1], Name: "Sanne de Vries", Email: "sanne@example.test", Adults: 2, Locale: "nl"},
		{SubjectID: s.events[1], Name: "Ahmed Benali", Email: "ahmed@example.test", Adults: 1, Locale: "nl"},
		{SubjectID: s.events[2], Name: "Claire Martin", Email: "claire@example.test", Adults: 2, Locale: "en"},
		{SubjectID: s.events[2], Name: "Pieter Smit", Email: "pieter@example.test", Adults: 2, Locale: "nl"},
		{SubjectID: s.events[2], Name: "Emma Wilson", Email: "emma@example.test", Adults: 1, Locale: "en"},
	}

	for _, signup := range signups {
		if _, err := service.Register(s.ctx, signup); err != nil {
			return err
		}
	}

	return nil
}

// remarks adds approved comments below the first two journal posts.
func (s *seeder) remarks() error {
	service := comments.Of(s.app, "remarks")
	entries := []struct {
		post  int
		entry comments.Entry
		reply string
	}{
		{
			0,
			comments.Entry{
				Name:   "Ahmed",
				Locale: "nl",
				Body:   "Mooi om het hele proces te zien. Hoe lang duurt zo'n reducerende stook?",
			},
			"Ongeveer tien uur, en daarna nog een dag afkoelen. Geduld dus!",
		},
		{
			0,
			comments.Entry{Name: "Claire", Locale: "en", Body: "Lovely write-up. Do you ever sell the test tiles?"},
			"",
		},
		{
			1,
			comments.Entry{Name: "Sanne", Locale: "nl", Body: "Gefeliciteerd! Dan kan ik mijn kommen voortaan sneller ophalen."},
			"",
		},
	}

	for _, item := range entries {
		item.entry.Subject = s.postIDs[item.post]
		item.entry.Path = s.postPaths[item.post][item.entry.Locale]
		item.entry.Approved = true

		comment, err := service.Add(s.ctx, item.entry)
		if err != nil {
			return err
		}

		if item.reply == "" {
			continue
		}

		if _, err := service.Reply(s.ctx, comment.ID, "Atelier Noord", item.reply); err != nil {
			return err
		}
	}

	return nil
}

// guestbook shows new messages right away and adds a few.
func (s *seeder) guestbook() error {
	service := comments.Of(s.app, "guestbook")

	if err := service.SaveSettings(s.ctx, comments.Settings{Moderated: false}); err != nil {
		return err
	}

	entries := []comments.Entry{
		{
			Name:   "Familie Jansen",
			Locale: "nl",
			Body:   "Wat een fijne middag bij het open atelier. De kinderen praten nog steeds over de draaischijf!",
		},
		{Name: "Tom", Locale: "en", Body: "Visited from Rotterdam for the glaze day. Calm, generous teachers and great coffee."},
		{Name: "Lieke", Locale: "nl", Body: "Mijn eerste kom is gelukt! Dank voor het geduld, Joris."},
	}

	for _, entry := range entries {
		entry.Subject = "guestbook"
		entry.Approved = true

		if _, err := service.Add(s.ctx, entry); err != nil {
			return err
		}
	}

	return nil
}

func (s *seeder) letters() error {
	store := newsletter.Of(s.app, "letters").Texts()
	_, err := add(s.ctx, store, struct{}{}, both(
		newsletter.Text{
			Title:  "Nieuwsbrief",
			Intro:  "Eens per maand: nieuwe workshops, open atelierdagen en wat er uit de oven kwam.",
			Footer: "Je krijgt deze e-mail omdat je je hebt ingeschreven voor de nieuwsbrief van Atelier Noord. Uitschrijven:",
		},
		newsletter.Text{
			Title:  "Newsletter",
			Intro:  "Once a month: new workshops, open studio days and what came out of the kiln.",
			Footer: "You receive this e-mail because you subscribed to the Atelier Noord newsletter. Unsubscribe:",
		},
	))

	return err
}

func (s *seeder) poll() error {
	service := polls.Of(s.app, "poll")
	options := polls.Lines{db.NewID(), db.NewID(), db.NewID(), db.NewID()}

	id, err := add(s.ctx, service.Store(), polls.Fields{Options: options}, both(
		polls.Text{
			Question: "Welke glazuur moet terugkomen?",
			Slug:     "welke-glazuur-moet-terugkomen",
			Options:  polls.Lines{"Roestbruin", "Celadon", "Zandwit", "Kobaltblauw"},
		},
		polls.Text{
			Question: "Which glaze should come back?",
			Slug:     "which-glaze-should-come-back",
			Options:  polls.Lines{"Rust brown", "Celadon", "Sand white", "Cobalt blue"},
		},
	))
	if err != nil {
		return err
	}

	votes := []int{9, 14, 5, 7}
	visitor := 0

	for option, count := range votes {
		for range count {
			visitor++

			if err := service.Vote(s.ctx, id, options[option], fmt.Sprintf("seed-%d", visitor)); err != nil {
				return err
			}
		}
	}

	return nil
}

var dutchMonths = []string{
	"", "januari", "februari", "maart", "april", "mei", "juni",
	"juli", "augustus", "september", "oktober", "november", "december",
}

// notice announces an open studio weekend, the first weekend at least five
// days ahead; the banner disappears after that Sunday.
func (s *seeder) notice() error {
	zone, _ := time.LoadLocation("Europe/Amsterdam")
	today := time.Now().In(zone)
	saturday := today.AddDate(0, 0, 5)

	for saturday.Weekday() != time.Saturday {
		saturday = saturday.AddDate(0, 0, 1)
	}

	sunday := saturday.AddDate(0, 0, 1)
	ends := time.Date(sunday.Year(), sunday.Month(), sunday.Day(), 18, 0, 0, 0, zone).UTC()

	dates := func(months []string) string {
		if saturday.Month() == sunday.Month() {
			return fmt.Sprintf("%d–%d %s", saturday.Day(), sunday.Day(), months[sunday.Month()])
		}

		return fmt.Sprintf("%d %s – %d %s", saturday.Day(), months[saturday.Month()], sunday.Day(), months[sunday.Month()])
	}

	english := make([]string, 13)
	for month := time.January; month <= time.December; month++ {
		english[month] = month.String()
	}

	store := banners.Of(s.app, "notice").Store()
	_, err := add(s.ctx, store, banners.Fields{Tone: banners.ToneInfo, Ends: &ends}, both(
		banners.Text{
			Title:       "Open atelier weekend " + dates(dutchMonths),
			Body:        "Kijk mee in de werkplaats, probeer de draaischijf en drink koffie met de makers. Vrij entree, 11:00–17:00.",
			ButtonLabel: "Bekijk de workshops",
			ButtonURL:   "/nl/workshops",
		},
		banners.Text{
			Title:       "Open studio weekend " + dates(english),
			Body:        "Look around the studio, try the wheel and have coffee with the makers. Free entry, 11:00–17:00.",
			ButtonLabel: "See the workshops",
			ButtonURL:   "/en/workshops",
		},
	))

	return err
}

// redirects keeps a few addresses of the studio's previous website working.
func (s *seeder) redirects() error {
	service := redirects.Of(s.app, "redirects")
	rules := []redirects.Redirect{
		{From: "/blog", To: "/nl/journaal", Status: 301},
		{From: "/en/blog", To: "/en/journal", Status: 301},
		{From: "/cursussen-en-workshops", To: "/nl/cursussen", Status: 301},
		{From: "/winkel", To: "/nl/over-ons/verkooppunten", Status: 302},
	}

	for _, rule := range rules {
		if _, err := service.Save(s.ctx, rule); err != nil {
			return err
		}
	}

	return nil
}
