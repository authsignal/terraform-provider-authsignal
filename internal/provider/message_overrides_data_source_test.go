package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccMessageOverridesDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
					resource "authsignal_message_overrides" "test" {
						overrides = {
							en = {
								"sms-code-entry.heading" = "Enter your code"
							}
						}
					}

					data "authsignal_message_overrides" "current" {
						depends_on = [authsignal_message_overrides.test]
					}
				`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.authsignal_message_overrides.current", "overrides.en.sms-code-entry.heading", "Enter your code"),
				),
			},
		},
	})
}
