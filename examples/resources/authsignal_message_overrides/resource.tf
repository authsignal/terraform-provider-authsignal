resource "authsignal_message_overrides" "message_overrides" {
  overrides = {
    en = {
      "sms-code-entry.heading" = "Enter your code"
    }
    "pt-br" = {
      "sms-code-entry.heading" = "Insira seu código"
    }
  }
}
