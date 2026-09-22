# A button font that draws small, corrected to 125% of the glyphs it would otherwise draw.
resource "authsignal_theme" "theme" {
  name = "My Tenant"
  typography = {
    button = {
      faces = [
        { url = "<url to a font file>", weight = "500" },
      ]
      size_adjust = 125
    }
  }
}
