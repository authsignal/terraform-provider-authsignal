# A padding per axis. Neither axis falls back to `padding`, so it can be left out.
resource "authsignal_theme" "theme" {
  name = "My Tenant"
  container = {
    padding_horizontal = 24
    padding_vertical   = 48
  }
}
