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

// Terraform owns the axis paddings: a configured value is written, a zero is a padding, and an axis
// dropped from the configuration clears the stored override. Each apply is followed by an empty
// plan, since a value that does not survive the round trip shows up as a perpetual diff.
func TestAccThemeAxisPadding(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Themes cannot be created, only updated, so the state has to be imported before the
			// applies below have anything to update.
			{
				Config: testAccThemeAxisPaddingConfig(`
					padding = 32
				`),
				ResourceName:       "authsignal_theme.theme",
				ImportState:        true,
				ImportStateId:      "",
				ImportStatePersist: true,
			},
			// A universal padding on its own leaves both axes unset.
			{
				Config: testAccThemeAxisPaddingConfig(`
					padding = 32
				`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("authsignal_theme.theme", "container.padding", "32"),
					resource.TestCheckNoResourceAttr("authsignal_theme.theme", "container.padding_horizontal"),
					resource.TestCheckNoResourceAttr("authsignal_theme.theme", "container.padding_vertical"),
				),
			},
			{
				Config: testAccThemeAxisPaddingConfig(`
					padding = 32
				`),
				PlanOnly: true,
			},
			// Both axes alongside the universal padding.
			{
				Config: testAccThemeAxisPaddingConfig(`
					padding            = 32
					padding_horizontal = 24
					padding_vertical   = 48
				`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("authsignal_theme.theme", "container.padding", "32"),
					resource.TestCheckResourceAttr("authsignal_theme.theme", "container.padding_horizontal", "24"),
					resource.TestCheckResourceAttr("authsignal_theme.theme", "container.padding_vertical", "48"),
				),
			},
			{
				Config: testAccThemeAxisPaddingConfig(`
					padding            = 32
					padding_horizontal = 24
					padding_vertical   = 48
				`),
				PlanOnly: true,
			},
			// Zero is a padding the configuration asked for, not an absence.
			{
				Config: testAccThemeAxisPaddingConfig(`
					padding            = 32
					padding_horizontal = 0
					padding_vertical   = 48
				`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("authsignal_theme.theme", "container.padding_horizontal", "0"),
					resource.TestCheckResourceAttr("authsignal_theme.theme", "container.padding_vertical", "48"),
				),
			},
			{
				Config: testAccThemeAxisPaddingConfig(`
					padding            = 32
					padding_horizontal = 0
					padding_vertical   = 48
				`),
				PlanOnly: true,
			},
			// An axis dropped from the configuration clears the stored override, and the universal
			// padding it was covering stays.
			{
				Config: testAccThemeAxisPaddingConfig(`
					padding          = 32
					padding_vertical = 48
				`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("authsignal_theme.theme", "container.padding", "32"),
					resource.TestCheckNoResourceAttr("authsignal_theme.theme", "container.padding_horizontal"),
					resource.TestCheckResourceAttr("authsignal_theme.theme", "container.padding_vertical", "48"),
				),
			},
			{
				Config: testAccThemeAxisPaddingConfig(`
					padding          = 32
					padding_vertical = 48
				`),
				PlanOnly: true,
			},
			// Dropping the rest clears both axes.
			{
				Config: testAccThemeAxisPaddingConfig(`
					padding = 32
				`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr("authsignal_theme.theme", "container.padding_horizontal"),
					resource.TestCheckNoResourceAttr("authsignal_theme.theme", "container.padding_vertical"),
				),
			},
			{
				Config: testAccThemeAxisPaddingConfig(`
					padding = 32
				`),
				PlanOnly: true,
			},
			// What was read back is what an import gets.
			{
				ResourceName:      "authsignal_theme.theme",
				ImportState:       true,
				ImportStateId:     "",
				ImportStateVerify: true,
			},
		},
	})
}

func testAccThemeAxisPaddingConfig(padding string) string {
	return `resource "authsignal_theme" "theme" {
  name = "Management-API-Testing"
  container = {
` + padding + `
  }
}`
}
