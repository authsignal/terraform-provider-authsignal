# create an action code
resource "authsignal_action_configuration" "terraform-provider-test" {
  action_code                = "terraform-provider-test"
  default_user_action_result = "BLOCK"
  messaging_templates = jsonencode({
    "en" : {
      "defaultTemplate" : "hello world!"
    }
  })
  verification_methods                  = ["EMAIL_OTP", "PASSKEY"]
  default_verification_method           = "EMAIL_OTP"
  prompt_to_enroll_verification_methods = ["PASSKEY"]
}
