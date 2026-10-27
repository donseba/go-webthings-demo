// HTMX, the mobile menu, the gallery lightbox and dismissible announcements.
// The site works without this script: links load whole pages and forms return
// to their anchor through the back field that {{form_open .Form}} writes.
// With it, the hx-* attributes of the theme swap only what changes;
// go-importmap serves htmx and
// go-partial's interactions client, and the layout's {{importmap}} maps their
// names.
import "htmx";
// go-partial's exp/interactions client: a link that gets a 5xx answer, or
// none, loads as a whole page, and a form shows the server's error page (the
// body's clientAttrs "reloadOnError"), so the visitor sees what went wrong
// rather than nothing.
import "partial-interactions";

document.addEventListener("click", (event) => {
    const toggle = event.target.closest(".menu-toggle");
    if (toggle) {
        const menu = document.getElementById(toggle.getAttribute("aria-controls"));
        const open = menu.classList.toggle("open");
        toggle.setAttribute("aria-expanded", String(open));
        return;
    }

    // Navigation keeps the header and updates only the menu's items: a link
    // in the header, in the menu or beside it such as search or a language,
    // closes the menu.
    if (event.target.closest(".site-header a")) {
        for (const toggle of document.querySelectorAll(".menu-toggle")) {
            closeMenu(toggle);
        }
    }

    const photo = event.target.closest("[data-lightbox] img");
    if (photo) {
        const dialog = document.querySelector("dialog.lightbox");
        const image = dialog.querySelector("img");
        // The album's large copy; the shown one without it.
        image.src = photo.dataset.full || photo.currentSrc || photo.src;
        image.alt = photo.alt;
        dialog.showModal();
        return;
    }

    if (event.target.matches("dialog.lightbox")) {
        event.target.close();
        return;
    }

    const close = event.target.closest(".announcement-close");
    if (close) {
        const announcement = close.closest("[data-dismiss]");
        announcement.hidden = true;
        remember("dismissed:" + announcement.dataset.dismiss);
    }
});

// Escape closes an open menu and puts the focus back on its button.
document.addEventListener("keydown", (event) => {
    if (event.key !== "Escape") {
        return;
    }

    for (const toggle of document.querySelectorAll('.menu-toggle[aria-expanded="true"]')) {
        closeMenu(toggle);
        toggle.focus();
    }
});

function closeMenu(toggle) {
    document.getElementById(toggle.getAttribute("aria-controls"))?.classList.remove("open");
    toggle.setAttribute("aria-expanded", "false");
}

// Announcements stay away once closed; storage may be unavailable.
function remember(key) {
    try {
        localStorage.setItem(key, "1");
    } catch {
        // The announcement comes back on the next page.
    }
}

function remembered(key) {
    try {
        return localStorage.getItem(key) === "1";
    } catch {
        return false;
    }
}

// A navigation that replaces the whole menu closes it; the button follows.
document.addEventListener("htmx:after:swap", () => {
    for (const toggle of document.querySelectorAll(".menu-toggle")) {
        const menu = document.getElementById(toggle.getAttribute("aria-controls"));
        toggle.setAttribute("aria-expanded", String(menu?.classList.contains("open") ?? false));
    }
});

for (const announcement of document.querySelectorAll("[data-dismiss]")) {
    if (remembered("dismissed:" + announcement.dataset.dismiss)) {
        announcement.hidden = true;
        continue;
    }

    announcement.querySelector(".announcement-close").hidden = false;
}
