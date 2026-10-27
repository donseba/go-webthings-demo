package main

import (
	"bytes"
	"testing"

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
