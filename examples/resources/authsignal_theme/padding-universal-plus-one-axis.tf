# `padding_horizontal` covers the left and right; `padding` covers the top and bottom, since no
# `padding_vertical` overrides it there.
resource "authsignal_theme" "theme" {
  name = "My Tenant"
  container = {
    padding            = 32
    padding_horizontal = 24
  }
}
