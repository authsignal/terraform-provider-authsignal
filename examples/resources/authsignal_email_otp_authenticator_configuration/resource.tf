variable "sendgrid_api_key" {
  type      = string
  ephemeral = true
  sensitive = true
}

resource "authsignal_email_otp_authenticator_configuration" "email_otp" {
  email_provider = "SENDGRID"

  sendgrid_email_credentials = {
    api_key     = var.sendgrid_api_key
    template_id = "d-00000000000000000000000000000000"
    from_email  = "noreply@example.com"
    from_name   = "Example"
  }

  sendgrid_email_credentials_version = "1"
}
