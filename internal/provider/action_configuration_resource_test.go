package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccActionConfigurationResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: `
					resource "authsignal_action_configuration" "test" {
						action_code = "terraform-acceptance-test"
						default_user_action_result = "ALLOW"
					}
				`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("authsignal_action_configuration.test", "action_code", "terraform-acceptance-test"),
					resource.TestCheckResourceAttr("authsignal_action_configuration.test", "default_user_action_result", "ALLOW"),
					resource.TestCheckResourceAttr("authsignal_action_configuration.test", "tenant_id", "680137cd-1bb7-40b3-b3c3-6f0a665e9d94"),
				),
			},
			// Update and Read testing
			{
				Config: `
					resource "authsignal_action_configuration" "test" {
						action_code = "terraform-acceptance-test"
						default_user_action_result = "BLOCK"
					}
				`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("authsignal_action_configuration.test", "action_code", "terraform-acceptance-test"),
					resource.TestCheckResourceAttr("authsignal_action_configuration.test", "default_user_action_result", "BLOCK"),
					resource.TestCheckResourceAttr("authsignal_action_configuration.test", "tenant_id", "680137cd-1bb7-40b3-b3c3-6f0a665e9d94"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}
