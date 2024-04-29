# create a rule
resource "authsignal_rule" "test" {
  action_code = "test-rules"
  name        = "made-with-terraform"
  description = "hello world"
  priority    = 2
  type        = "CHALLENGE"
  is_active   = false
  verification_methods = [
    "AUTHENTICATOR_APP",
    "EMAIL_MAGIC_LINK",
    "EMAIL_OTP"
  ]
  prompt_to_enroll_verification_methods = [
    "PASSKEY"
  ]
  default_verification_method = "EMAIL_OTP"
  conditions = jsonencode({
    "and" : [
      {
        "==" : [
          {
            "var" : "ip.isAnonymous"
          },
          false
        ]
      }
    ]
  })
}
