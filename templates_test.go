package main

import (
	"io/fs"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// imagePattern, cropPattern, focalPattern and aspectPattern find an <img>
// element, its server crop ({{image .ImageURL "720x540 crop"}}), its focal
// class ({{focalClass .ImageURL "4:3"}}) and a CSS aspect-ratio.
var (
	imagePattern  = regexp.MustCompile(`<img[^>]*>`)
	cropPattern   = regexp.MustCompile(`image (\S+) "(\d+)x(\d+) crop"`)
	focalPattern  = regexp.MustCompile(`focalClass (\S+) "(\d+):(\d+)"`)
	aspectPattern = regexp.MustCompile(`aspect-ratio:\s*(\d+)\s*/\s*(\d+)`)
)

// TestFocalCrops checks that every image with a focal class is cropped by
// the server to the proportions focalClass is given, and that those are the
// proportions of the box site.css crops the image to: the browser shows the
// same part of a photo whether it loads the cropped src or a copy from the
// srcset, and the focal point editors placed stays in view.
func TestFocalCrops(t *testing.T) {
	templates := os.DirFS("templates")
	checked := 0

	err := fs.WalkDir(templates, ".", func(name string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(name, ".gohtml") {
			return err
		}

		data, err := fs.ReadFile(templates, name)
		if err != nil {
			return err
		}

		for _, tag := range imagePattern.FindAllString(string(data), -1) {
			focal := focalPattern.FindStringSubmatch(tag)
			if focal == nil {
				continue
			}

			crop := cropPattern.FindStringSubmatch(tag)
			if crop == nil {
				t.Errorf("%s: a focal class without a crop: %s", name, tag)

				continue
			}

			if crop[1] != focal[1] || number(t, crop[2])*number(t, focal[3]) != number(t, crop[3])*number(t, focal[2]) {
				t.Errorf("%s: crop %sx%s of %s does not have the proportions %s:%s of focalClass %s",
					name, crop[2], crop[3], crop[1], focal[2], focal[3], focal[1])
			}

			checked++
		}

		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	if checked == 0 {
		t.Fatal("no image with a focal class")
	}

	css, err := os.ReadFile("templates/default/assets/site.css")
	if err != nil {
		t.Fatal(err)
	}

	for _, card := range []struct{ partial, selector, ratio string }{
		{"partials/post-card.gohtml", ".card > img", "4:3"},
		{"partials/stockist-card.gohtml", ".card > img", "4:3"},
		{"partials/person-card.gohtml", ".person-card img", "4:5"},
	} {
		data, err := fs.ReadFile(templates, "default/"+card.partial)
		if err != nil {
			t.Fatal(err)
		}

		if !strings.Contains(string(data), `"`+card.ratio+`"`) {
			t.Errorf("%s: no focalClass %q", card.partial, card.ratio)
		}

		aspect := aspectOf(string(css), card.selector)
		if aspect != strings.Replace(card.ratio, ":", "/", 1) {
			t.Errorf("site.css: %s has aspect-ratio %q, want %s", card.selector, aspect, card.ratio)
		}
	}
}

// aspectOf is the aspect-ratio, as "4/3", of the rule for selector in css,
// or "".
func aspectOf(css, selector string) string {
	start := strings.Index(css, "\n"+selector+" {")
	if start < 0 {
		return ""
	}

	rule := css[start:]
	if end := strings.Index(rule, "}"); end >= 0 {
		rule = rule[:end]
	}

	match := aspectPattern.FindStringSubmatch(rule)
	if match == nil {
		return ""
	}

	return match[1] + "/" + match[2]
}

func number(t *testing.T, value string) int {
	t.Helper()

	n, err := strconv.Atoi(value)
	if err != nil {
		t.Fatal(err)
	}

	return n
}
