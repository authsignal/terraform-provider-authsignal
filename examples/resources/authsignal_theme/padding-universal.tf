# One padding on every side of the container.
resource "authsignal_theme" "theme" {
  name = "My Tenant"
  container = {
    padding = 32
  }
}
