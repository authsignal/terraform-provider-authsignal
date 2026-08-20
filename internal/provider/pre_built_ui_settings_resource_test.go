package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccPreBuiltUiSettingsResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: `
					resource "authsignal_pre_built_ui_settings" "terraform-acc-test" {
						hide_success_screen_on_enrollment = true
					}
				`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("authsignal_pre_built_ui_settings.terraform-acc-test", "hide_success_screen_on_enrollment", "true"),
				),
			},
			// Update testing: a setting turned back off has to reach the API, not be dropped as empty
			{
				Config: `
					resource "authsignal_pre_built_ui_settings" "terraform-acc-test" {
						hide_success_screen_on_enrollment = false
					}
				`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("authsignal_pre_built_ui_settings.terraform-acc-test", "hide_success_screen_on_enrollment", "false"),
				),
			},
		},
	})
}
