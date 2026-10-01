resource "authsignal_custom_data_point" "risk_score" {
  name       = "risk_score"
  data_type  = "number"
  model_type = "action"
}

resource "authsignal_value_list" "blocked_countries" {
  name                     = "Blocked countries"
  is_active                = true
  value_list_items_strings = ["KP", "IR"]
}

# Load a flow exported from the Authsignal Portal.
resource "authsignal_flow" "sign_in" {
  action_code = "sign-in"
  flow        = file("${path.module}/flow-sign-in.json")

  depends_on = [
    authsignal_custom_data_point.risk_score,
    authsignal_value_list.blocked_countries,
  ]
}
