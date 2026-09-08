# Load a flow exported from the Authsignal Portal.
resource "authsignal_flow" "sign_in" {
  action_code = "sign-in"
  flow        = file("${path.module}/flow-sign-in.json")
}
