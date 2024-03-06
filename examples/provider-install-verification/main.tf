terraform {
  required_providers {
    authsignal = {
      source = "authsignal.com/provider/authsignal"
    }
  }
}

provider "authsignal" {}

data "authsignal_action_configurations" "example" {}
