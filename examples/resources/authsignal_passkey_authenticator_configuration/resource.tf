resource "authsignal_passkey_authenticator_configuration" "passkey" {
  relying_party = "example.com"

  expected_origins = [
    "https://example.com",
    "https://app.example.com",
  ]
}
