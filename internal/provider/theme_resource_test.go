package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// The Management API rejects an exit position under dark mode, so the configuration cannot offer
// one. Terraform turns this away while validating, before the theme is read or written.
func TestAccThemeRejectsAPerModeExitPosition(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `resource "authsignal_theme" "theme" {
  name = "Management-API-Testing"
  dark_mode = {
    container = {
      exit_position = "bottom"
    }
  }
}`,
				ExpectError: regexp.MustCompile("exit_position"),
			},
		},
	})
}
