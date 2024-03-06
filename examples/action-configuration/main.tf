terraform {
  required_providers {
    authsignal = {
      source = "authsignal.com/provider/authsignal"
    }
  }
}

resource "authsignal_action_configuration" "terraform-provider-test" {
  action_code                = "terraform-provider-test"
  default_user_action_result = "ALLOW"
}

output "terraform_provider_test_action_configuration" {
  value = resource.authsignal_action_configuration.terraform-provider-test
}
