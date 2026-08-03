---
page_title: "Upgrading to v3"
subcategory: ""
description: |-
  Upgrading the Authsignal provider from v2 to v3.
---

# Upgrading to v3

v3 changes how fonts are configured on `authsignal_theme`. Nothing else in the provider changes.

## `dark_mode.typography` has been removed

A typeface is now shared by both colour modes, so the Management API no longer accepts a per-mode
typeface. Requests that send one are rejected, which means any `authsignal_theme` apply on v2 fails
against the current API whether or not you set a dark mode font.

Remove the block:

```diff
 resource "authsignal_theme" "theme" {
   dark_mode = {
-    typography = {
-      display = {
-        font_url = "https://cdn.example.com/display.woff2"
-      }
-    }
     colors = {
       body_text = "#ABCD12"
     }
   }
 }
```

The top-level `typography` block applies to both light and dark mode. If you were relying on a
dark-mode-only font, set it at the top level instead — there is no longer a way to vary the typeface
by mode.

## `typography` takes two roles and a list of faces

`typography` now has a `text` role for body copy and UI labels, alongside the existing `display` role
for headings. Each role takes a `faces` list of up to six font files, each with the weight it covers:

```hcl
resource "authsignal_theme" "theme" {
  typography = {
    text = {
      faces = [
        { url = "https://cdn.example.com/regular.woff2", weight = "400" },
        { url = "https://cdn.example.com/bold.woff2", weight = "700" },
      ]
    }
    display = {
      faces = [
        { url = "https://cdn.example.com/display-variable.woff2", weight = "100 900" },
      ]
    }
  }
}
```

A weight is a single value from 1 to 1000, or an ascending range such as `100 900` for a variable
font. A descending range is rejected.

`font_url` still works and is unchanged, so an existing `typography.display.font_url` needs no edit.
It is deprecated in favour of `faces`, which lets you supply a weight per file rather than a single
file for every weight.
