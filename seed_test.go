package main

import (
	"bytes"
	"testing"
	"time"

	"github.com/donseba/go-webthings/modules/events"
	"github.com/donseba/go-webthings/sitetest"
)

// TestBrandLogo checks that a site without a logo gets the brand's mark
// from its media library as its logo, and that a site with a logo, or
// without the mark, keeps its settings.
func TestBrandLogo(t *testing.T) {
	ctx := t.Context()
	s := sitetest.Start(t)
	app := s.App

	// Without the mark in the library, nothing changes.
	if err := brandLogo(ctx, app); err != nil {
		t.Fatal(err)
	}

	settings, err := app.Site.Settings(ctx)
	if err != nil || settings.LogoMediaID != "" {
		t.Fatalf("logo without the mark: %q %v", settings.LogoMediaID, err)
	}

	data, err := brandFiles.ReadFile("brand/favicon.png")
	if err != nil {
		t.Fatal(err)
	}

	other := s.Upload("other.png", sitetest.PNG(t, 4, 4))

	mark, err := app.Media.Upload(ctx, "mark.png", bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}

	settings.Organisation = "Kept"
	if err := app.Site.Save(ctx, settings); err != nil {
		t.Fatal(err)
	}

	if err := brandLogo(ctx, app); err != nil {
		t.Fatal(err)
	}

	settings, err = app.Site.Settings(ctx)
	if err != nil || settings.LogoMediaID != mark.ID || settings.Organisation != "Kept" {
		t.Fatalf("logo with the mark: %+v %v", settings, err)
	}

	// A logo the site has stays.
	settings.LogoMediaID = other.ID
	if err := app.Site.Save(ctx, settings); err != nil {
		t.Fatal(err)
	}

	if err := brandLogo(ctx, app); err != nil {
		t.Fatal(err)
	}

	if settings, _ = app.Site.Settings(ctx); settings.LogoMediaID != other.ID {
		t.Fatalf("the site's own logo was replaced by %q", settings.LogoMediaID)
	}
}

// TestMoveWorkshops checks that the seeded workshops move forward by whole
// weeks once none is upcoming, keeping their weekday, time of day and
// spacing across a change of summer time; that other past workshops stay;
// and that an upcoming workshop of the site's own keeps them in place.
func TestMoveWorkshops(t *testing.T) {
	ctx := t.Context()
	s := sitetest.Start(t, events.Module("workshops", "Workshops"))
	store := events.Of(s.App, "workshops").Store()

	zone, err := time.LoadLocation("Europe/Amsterdam")
	if err != nil {
		t.Fatal(err)
	}

	// October is summer time in Amsterdam, December is not.
	october := func(day, hour int) time.Time {
		return time.Date(2026, 10, day, hour, 0, 0, 0, zone)
	}

	workshop := func(slug string, start time.Time) string {
		t.Helper()

		end := start.Add(3 * time.Hour)
		fields := events.EventFields{Start: start, End: &end, Timezone: "Europe/Amsterdam"}

		id, err := add(ctx, store, fields, both(
			events.EventText{Slug: slug, Title: slug},
			events.EventText{Slug: slug + "-en", Title: slug},
		))
		if err != nil {
			t.Fatal(err)
		}

		return id
	}

	start := func(id string) time.Time {
		t.Helper()

		item, err := store.Load(ctx, id, "")
		if err != nil {
			t.Fatal(err)
		}

		return item.Fields.Start.In(zone)
	}

	wheel := workshop("kennismaken-met-de-draaischijf", october(11, 19))
	glaze := workshop("glazuurdag", october(18, 10))
	studio := workshop("open-studio", october(1, 10))

	now := time.Date(2026, 12, 15, 12, 0, 0, 0, zone)

	if err := moveWorkshops(ctx, s.App, now); err != nil {
		t.Fatal(err)
	}

	moved := start(wheel)
	if !moved.After(now) || moved.Sub(now) > 7*24*time.Hour ||
		moved.Weekday() != october(11, 19).Weekday() || moved.Hour() != 19 {
		t.Fatalf("first workshop at %v", moved)
	}

	if next := start(glaze); next.YearDay()-moved.YearDay() != 7 || next.Hour() != 10 {
		t.Fatalf("second workshop at %v after %v", next, moved)
	}

	if past := start(studio); !past.Equal(october(1, 10)) {
		t.Fatalf("a workshop the seed does not plan moved to %v", past)
	}

	// A month later they are over again, but the site's own workshop is
	// still to come.
	workshop("eigen-workshop", now.AddDate(0, 0, 60))

	if err := moveWorkshops(ctx, s.App, now.AddDate(0, 0, 30)); err != nil {
		t.Fatal(err)
	}

	if again := start(wheel); !again.Equal(moved) {
		t.Fatalf("moved with an upcoming workshop: %v", again)
	}
}
