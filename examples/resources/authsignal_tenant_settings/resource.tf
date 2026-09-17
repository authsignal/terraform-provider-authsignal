resource "authsignal_tenant_settings" "tenant_settings" {
  token_duration_in_minutes = 15

  authenticator_events_webhook_config = {
    url                           = "https://example.com/authsignal/authenticator-events"
    include_credential_public_key = true
  }

  log_events_webhook_config = {
    endpoint_url = "https://example.com/authsignal/log-events"
  }

  ip_whitelist = ["203.0.113.0/24", "198.51.100.7/32"]
}
