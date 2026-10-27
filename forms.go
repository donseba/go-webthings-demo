package main

import (
	"github.com/donseba/go-form/v2/templates"
)

// formTheme is go-form's theme with the classes of the theme's site.css, in
// which the visitor forms render their fields ({{form_field .Form "name"
// nil}}, webthings.Config.FormTheme).
const formTheme = "atelier"

// formClasses are site.css's classes for go-form's markup: a .form of
// fields, each a .field with its label, a .help text and .error messages,
// and radio buttons as .choices, each a .choice, such as a poll's options.
var formClasses = templates.ThemeClasses{
	Form:         templates.StyleOption{Class: "form"},
	Wrapper:      templates.StyleOption{Class: "field"},
	Description:  templates.StyleOption{Class: "help"},
	Error:        templates.StyleOption{Class: "error"},
	Button:       templates.StyleOption{Class: "button"},
	RadioGroup:   templates.StyleOption{Class: "choices"},
	RadioWrapper: templates.StyleOption{Class: "choice"},
	// The spam trap, hidden by site.css rather than an inline style.
	Honeypot: templates.StyleOption{Class: templates.HoneypotClass},
}

// registerFormTheme registers formTheme with go-form, before the site
// serves.
func registerFormTheme() error {
	_, err := templates.RegisterClassTheme(formTheme, formClasses)

	return err
}
