package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccMessageOverridesResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: `
					resource "authsignal_message_overrides" "terraform-acc-test" {
						overrides = {
							en = {
								"sms-code-entry.heading" = "Enter your code"
							}
						}
					}
				`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("authsignal_message_overrides.terraform-acc-test", "overrides.en.sms-code-entry.heading", "Enter your code"),
				),
			},
			// Update testing: change a value and add a locale
			{
				Config: `
					resource "authsignal_message_overrides" "terraform-acc-test" {
						overrides = {
							en = {
								"sms-code-entry.heading" = "Enter the code we sent you"
							}
							"pt-br" = {
								"sms-code-entry.heading" = "Insira seu código"
							}
						}
					}
				`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("authsignal_message_overrides.terraform-acc-test", "overrides.en.sms-code-entry.heading", "Enter the code we sent you"),
					resource.TestCheckResourceAttr("authsignal_message_overrides.terraform-acc-test", "overrides.pt-br.sms-code-entry.heading", "Insira seu código"),
				),
			},
		},
	})
}
