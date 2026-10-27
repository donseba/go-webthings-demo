# Atelier Noord — WebThings demo

A bilingual (Dutch/English) website for a fictional ceramics studio, using
most WebThings modules: workshops (Events), work (Gallery), journal
(Publishing), courses and FAQ (Lists), reviews (Testimonials), a contact
form (Forms), and:

| Key | Module | Where |
| --- | --- | --- |
| `signup` | registrations | sign-up form on every workshop, *Praktisch › Mijn aanmelding* to look up or cancel |
| `remarks` | comments | comments below journal posts (moderated) |
| `guestbook` | comments | *Over ons › Gastenboek* (not moderated) |
| `letters` | newsletter | signup in the footer, *Praktisch › Nieuwsbrief* |
| `team` | people | *Over ons › Team*, and the team on the About page |
| `stockists` | directory | *Over ons › Verkooppunten*, with opening hours |
| `downloads` | documents | *Praktisch › Downloads*, and below the courses |
| `poll` | polls | on the home page |
| `notice` | banners | the dismissible announcement bar above the header |
| `stats` | stats | "Popular on the site" next to the journal; `stats.Module()` |
| `redirects` | redirects | old addresses such as `/blog` and `/winkel`; `redirects.Module()` |

The social media profiles in the footer are site settings (*Website-instellingen
› Socialemediaprofielen*), shown with `{{range .Site.Profiles}}`; the share
links below journal posts come from the core `share` function.

`submissions` and the integrations from
[go-webthings-contrib](../go-webthings-contrib) (`mailjet`, `s3`) are not used.
To try them:

- visitors proposing journal posts: add `submissions.Module("tips", "Tips", "journal")`
  to `app.Use`, place it on a page and add `templates/default/tips/form.gohtml`;
- Mailjet: set `Config.Mail` to `mailjet.Sender{APIKey: …, SecretKey: …, From: …}`
  and pass `newsletter.WithProvider(mailjet.List{…, ListID: …})` to the newsletter;
- media on S3 (or MinIO, Spaces): set `Config.Storage` to the store from `s3.New(s3.Config{…})`.

Without a mail server every e-mail (newsletter confirmation, booking
confirmation) is written to the log, which is handy for trying the forms
locally; `WEBTHINGS_SMTP_HOST` and the variables below send it.

```sh
task run
```

Open http://127.0.0.1:8090/ and the admin at http://127.0.0.1:8090/admin/
(local demo account: `admin@example.test` / `demo-only-password`).

The first start fills the site with sample content and photographs
(`seed.go`). After that the seed does nothing, so edits in the admin are kept.
Only the three workshops the seed plans ahead move: once no workshop of the
site is upcoming any more, the next start moves them forward by whole weeks,
with the same weekday and time (`moveWorkshops`), so there is always one to
sign up for.
`task reset` moves `var/` to `var.old-<unix time>/` to start over; delete
those folders yourself.

`task run` serves the site as in production: templates are read once, and
the pages visitors get are kept in memory for a minute (the output cache,
*Website-instellingen › Prestaties*). Changes made in the admin show at
once, and a browser signed in to the admin always gets fresh pages.

While you change the templates, run `task dev` instead. It serves on the
same address with `WEBTHINGS_DEVELOPMENT=true`: the templates are read again
from `templates/` on every request, so an edit shows on the next request
without a restart, a failing page shows the error with its template and
line, and the output cache is off. The site logs
`development mode (WEBTHINGS_DEVELOPMENT): …` when it starts. Never set it
in production: visitors would see the details of errors.

`WEBTHINGS_METRICS_TOKEN` sets `Config.MetricsToken`, which `task run` and
`task dev` make `demo-metrics-token` unless set otherwise, so the metrics,
the output cache's `webthings_output_cache_*` counters among them, are at
`/metrics`:

```sh
curl -H "Authorization: Bearer demo-metrics-token" http://127.0.0.1:8090/metrics | grep output_cache
```

`WEBTHINGS_CONTENT_SECURITY_POLICY` sets `Config.ContentSecurityPolicy`:
`off`, `report-only` or `enforce`, which wins over
*Website-instellingen › Beveiliging › Content-Security-Policy*: that field
then shows the value, disabled, with the note *Ingesteld door de
serverconfiguratie.* `task run` leaves it empty, so the setting decides,
which is off until an administrator changes it. The public pages and the
admin work with `enforce`: no page has a `style` attribute, a `<style>`
element or an inline script, apart from the import map, whose hash the
policy carries, and the structured data, which browsers do not run.

```sh
WEBTHINGS_CONTENT_SECURITY_POLICY=enforce task run
```

E-mail is sent through the mail server in the environment, as with the
clone; without `WEBTHINGS_SMTP_HOST` it is written to the log. The setup
guide's *E-mail* step (*Website-instellingen › Aan de slag*) shows the
server and the sender, and sends a test e-mail to your own address.

| Variable | Default | Meaning |
| --- | --- | --- |
| `WEBTHINGS_SMTP_HOST` | | Mail server (`Config.Mail`, a `mail.SMTP`). Without it e-mail is written to the log. |
| `WEBTHINGS_SMTP_PORT` | `587` | Mail server port (STARTTLS when offered; port 465 is not supported). |
| `WEBTHINGS_SMTP_USERNAME`, `WEBTHINGS_SMTP_PASSWORD` | | Login for the mail server, when it needs one. |
| `WEBTHINGS_SMTP_FROM` | | Sender, for example `Atelier Noord <website@example.com>`; required with `WEBTHINGS_SMTP_HOST`. |
| `WEBTHINGS_BASE_URL` | | Public address (`Config.BaseURL`), for example `https://atelier.example.com`, for links in e-mail, feeds, share links and `sitemap.xml`. Without it the admin offers no password reset, except with `WEBTHINGS_INSECURE_HTTP`, which `task run` sets. |

```sh
WEBTHINGS_SMTP_HOST=smtp.example.com WEBTHINGS_SMTP_USERNAME=… WEBTHINGS_SMTP_PASSWORD=… \
WEBTHINGS_SMTP_FROM="Atelier Noord <website@example.com>" task run
```

- `main.go`: configuration and the module list
- `seed.go`: the sample content; `seed_modules.go` the content of the extra
  modules (portraits with focal points on the faces, so crops keep them in
  view, and generated PDFs)
- `templates/default/`: the Atelier theme: templates, `assets/` and
  `theme.json`; `500.gohtml` is the translated page visitors get when a
  template or a module fails. The layout links its stylesheets with
  `{{styles "site.css" "variant.css" "focal-points.css"}}`: go-partial's
  `partial-errors.css` first, which core serves and which styles the box
  that takes the place of a part of a page that failed, then the theme's
  `site.css` and `variant.css`, core's `focal-points.css`, and last the
  theme's values stylesheet, each with the file's Subresource Integrity;
  `{{assetScript "site.js" "type" "module"}}` writes the script tag the
  same way. The layout places the tags of the page in its `<head>` with
  `{{slot "head"}}`; the theme has no `regions/head.gohtml`, so WebThings
  writes them: the title, the description, the canonical and alternate
  links, with the Dutch address as `hreflang="x-default"` on pages in both
  languages, the `og:` tags with `og:image` and `twitter:card`, the
  favicon and the apple-touch-icon, and links to the Atom and RSS feeds of
  the page shown with `{{feedLinks}}`: the journal's at `/nl/journaal/feed.atom`
  and `.rss`, also per category, such as
  `/nl/journaal/category/studio/feed.atom`, and the upcoming workshops'
  at `/nl/workshops/feed.atom` and `.rss`, with as many items as
  *Website-instellingen › Algemeen › Berichten in feeds* says (20 by default).
  `{{structuredData}}` there writes the page's schema.org JSON-LD for search
  engines: the site, Atelier Noord as its organisation, the trail and the
  page, and on a journal post an `Article`, on a workshop an `Event`, with
  the sign-up's `Offer` (price, availability and deadline) while it is
  upcoming, on a member of the team a `Person` and on a stockist a
  `LocalBusiness` with its opening hours; the lists of the journal, the
  workshops, the work, the team and the stockists carry an `ItemList` of
  their items. `{{apiLink}}` there writes nothing, as the demo leaves the
  content API off (`Config.API`). The seed sets the logo mark
  (`brand/favicon.png`) as the logo in the site settings, so the header
  shows it and the organisation has a `logo`. The seed only fills an empty
  site, so on every start the demo also makes the mark the logo of a site
  that has none, when the mark is in the media library already
  (`brandLogo` in `seed.go`). It uploads nothing and never replaces a logo;
  otherwise set the logo under *Website-instellingen › Algemeen* or start
  over with `task reset`.
- `templates/night/`: a second theme with only other colours (`assets/variant.css`)
- `translations/`: the website's own texts
- `importmap.json`, `assets.go` and `assets/vendor/`: htmx, vendored with its
  licence and a lock file by `task vendor`, embedded in the program and served
  at `/assets/` through `Config.ImportMap`; core adds go-partial's browser
  clients to that import map, with their integrity
- `forms.go`: the go-form theme `atelier` (`Config.FormTheme`), which gives
  the visitor forms the classes of `site.css`. Every visitor form comes from
  its module: the comment form, the guestbook, the newsletter signup, the
  booking form, the poll and the cancel form are written with
  `{{form_open .Form}}` and `{{form_field .Form "name" nil}}` per field,
  with their labels and limits, and the contact form with
  `{{form_render .Form .Errors}}`.

Module templates live in a folder named after the module's key
(`templates/default/workshops/list.gohtml` for `events.Module("workshops", ...)`).
The FAQ and the course list share `templates/default/lists/list.gohtml`: a
folder named after the module type is used when a key has no folder of its own.
The same goes for the two comments instances, which share
`templates/default/comments/list.gohtml`. Shared pieces, such as a comment, a
poll or a person card, are in `templates/default/partials/`.

The theme's `site.js` imports htmx and go-partial's `partial-interactions`,
which `{{importmap}}` in the layout maps.
With it, links and forms swap only `<main id="content">`, and the head
tags WebThings writes and then the regions in `regions/` follow out of band,
in the order of their file names (editor, flash, languages, menu), the same
on every request; the head tags replace those of the page left, so the journal's feed links
and a post's `Article` do not stay behind on a page without them. The
fragments in `fragments/`, such as the comments, the booking form and the
search results, update on their own. Those with a visitor form are named
after its module's key and carry it as their id: `remarks.gohtml`,
`signup.gohtml` and `letters.gohtml`, with `#remarks`, `#signup` and
`#letters`, where a form's way back lands also without the script.
Templates place them with `{{requiredSlot "remarks"}}`, and the journal's
most read list with
`{{deferredSlot "popular" "reveal"}}`, which loads it once it scrolls into
view; a fragment with a visitor form is never deferred, as a deferred one
would not show a post with mistakes again. The theme writes these htmx
attributes with go-partial's helpers:
`clientAttrs` on the body for the boosted links and forms, `submitAttrs`,
given to `form_open`, on forms that swap only their own part, such as the
comment form and the poll, `liveAttrs` on the search box, and `noBoost` on
links that must load a whole page, such as the language switch. Without the
script every link and form loads a whole page.

Switch themes in the admin under Theme, preview them first, and change the
accent colour and heading font there. The accent is a colour picker,
`#8aa598` in the default theme and `#e0a458` in the night theme until an
editor chooses another. `theme.json` gives it `"css": "--accent"`, so core
serves the colour as `--accent` in the theme's values stylesheet,
`/theme/<theme>/values.css`, which `{{styles}}` links last, after
`site.css`, `variant.css` and `focal-points.css`. The layout writes the heading font as
`data-headings` on `<html>`, which `site.css` reads. The photos of the
cards keep their focal point in view with `{{focalClass}}`, and the poll's
results are `<meter>` elements, so no public page has a `style`
attribute, which a Content-Security-Policy without `'unsafe-inline'`,
such as the site's own with `enforce`, blocks. The server crops each card's
photo to the proportions of its box in `site.css`, which are also the
proportions its `{{focalClass}}` gets: 720x540 and 4:3 for the cards of
journal posts and stockists, 600x750 and 4:5 for the team's. So a photo shows the
same part whether the browser loads the cropped `src` or a copy from the
`srcset`, and the focal point stays in view. `TestFocalCrops` in
`templates_test.go` checks this for every image with a focal class. The
night theme shows how small a theme can be: everything it lacks comes from
`default/`.

## Building

`task build` writes a static binary to `bin/demo`
(`CGO_ENABLED=0 go build -tags nodynamic`). Keep `-tags nodynamic` for Alpine
or `scratch` images: without it the WebP encoder of the image variants loads
a system libwebp when one is installed (through purego), which makes the
Linux binary dynamically linked against glibc's loader, so it does not start
there. `file bin/demo` says "statically linked" for a Linux build.

`go.mod` requires a released go-webthings. To run the demo on a change of
go-webthings, build it in a Go workspace with a go-webthings checkout; see
[Development](../go-webthings/README.md#development) in go-webthings'
README.

## Translations

Template texts are written in English with `tl`, for example
`{{tl localizer "Read more"}}` (`tn` for plurals, `ctl`/`ctn` with a context),
and translated in `translations/<language>.po`. `main.go` loads the
catalogues with `frontend.Translations("translations")` and sets them as
`Config.Translations`; core translates its own texts with them too, such as
the visitor forms' labels and messages and the names of months and days.
English and Dutch have those names and the date patterns built in, so
`en.po` and `nl.po` leave them out and the check below does not ask for
them there; a translation in them would replace the built-in one.

1. `task translations` scans every template, including the modules' default
   templates and e-mail, adds the texts core and the modules' forms
   translate in Go and the calendar's names and patterns, and writes them to
   `translations/translations.pot`.
2. Open a language's `.po` file in [Poedit](https://poedit.net/), choose
   *Translation › Update from POT File…* and pick `translations/translations.pot`.
3. Translate the new texts and save. Restart the site to load them.

`task translations:check` lists the texts each language still misses and fails
when any are missing, so it can run in CI (`task translations:check -- --csv`
for a spreadsheet). Texts that are not written literally, such as
`{{tl localizer .Weekday.String}}`, are reported as warnings: name each possible
text in a literal `tl` call (for example in a `{{define}}` that is never used),
or updating from the POT file marks their translations as obsolete.
