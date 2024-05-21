# create an action code
resource "authsignal_action_configuration" "terraform-provider-test" {
  action_code                = "terraform-provider-test"
  default_user_action_result = "BLOCK"
  messaging_templates = jsonencode({
    "en" : {
      "defaultTemplate" : "hello world!"
    }
  })
}
