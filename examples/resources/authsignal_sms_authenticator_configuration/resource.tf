variable "twilio_auth_token" {
  type      = string
  ephemeral = true
  sensitive = true
}

resource "authsignal_sms_authenticator_configuration" "sms" {
  sms_provider         = "TWILIO"
  sms_country_codes    = ["AU", "NZ"]
  default_country_code = "NZ"

  twilio_credentials = {
    auth_token            = var.twilio_auth_token
    messaging_service_sid = "MG00000000000000000000000000000000"
    account_sid           = "AC00000000000000000000000000000000"
  }

  twilio_credentials_version = "1"
}
