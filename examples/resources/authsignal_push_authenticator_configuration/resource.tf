variable "fcm_service_account_key" {
  type      = string
  ephemeral = true
  sensitive = true
}

resource "authsignal_push_authenticator_configuration" "push" {
  push_provider = "FIREBASE"

  fcm_credentials = {
    service_account_key = var.fcm_service_account_key
  }

  fcm_credentials_version = "1"
}
