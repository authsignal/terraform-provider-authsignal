package provider

import (
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// A theme is imported with an empty identifier. The test framework reads an empty ImportStateId as
// no identifier at all and goes looking for one in the state, so it has to come from a function.
func importTheEmptyThemeId(*terraform.State) (string, error) {
	return "", nil
}

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

// Needs AUTHSIGNAL_DESTRUCTIVE_THEME_ACC=1. The test leaves the tenant with an empty theme, because
// Terraform destroys what it owns and a theme delete clears every field. Seed the theme again
// afterwards, or TestAccThemeDataSource fails.
//
// Terraform owns the axis paddings: a configured value is written, a zero is a padding, and an axis
// dropped from the configuration clears the stored override. Each apply is followed by an empty
// plan, since a value that does not survive the round trip shows up as a perpetual diff.
func TestAccThemeAxisPadding(t *testing.T) {
	if os.Getenv("AUTHSIGNAL_DESTRUCTIVE_THEME_ACC") == "" {
		t.Skip("set AUTHSIGNAL_DESTRUCTIVE_THEME_ACC=1 to run. The test clears the tenant's theme when it finishes.")
	}

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
				ImportStateIdFunc:  importTheEmptyThemeId,
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
			// Leave the tenant on the fixture padding, so the theme matches what the data source
			// test expects.
			{
				Config: testAccThemeAxisPaddingConfig(`
					padding = 61
				`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("authsignal_theme.theme", "container.padding", "61"),
				),
			},
			// What was read back is what an import gets.
			{
				ResourceName:      "authsignal_theme.theme",
				ImportState:       true,
				ImportStateIdFunc: importTheEmptyThemeId,
				ImportStateVerify: true,
				// A theme is identified by its name. It has no id.
				ImportStateVerifyIdentifierAttribute: "name",
			},
		},
	})
}

// The provider clears every theme field the configuration leaves out, so a partial configuration
// would wipe the test tenant's theme. The configuration below carries the whole fixture theme and
// varies only the container padding. The values match theme_data_source_test.go.
func testAccThemeAxisPaddingConfig(padding string) string {
	return `resource "authsignal_theme" "theme" {
  name          = "Management-API-Testing"
  logo_url      = "https://res.cloudinary.com/authsignal/image/upload/v1716774524/m7bvgqjzr29tp69qcogr.jpg"
  favicon_url   = "https://res.cloudinary.com/authsignal/image/upload/v1716774530/nvjzmbqax2jiiwse3a40.jpg"
  watermark_url = "https://res.cloudinary.com/authsignal/image/upload/v1716774530/nvjzmbqax2jiiwse3a40.jpg"
  primary_color = "#121282"
  borders = {
    button_border_radius    = 4
    button_border_width     = 4
    card_border_radius      = 8
    card_border_width       = 5
    input_border_radius     = 8
    input_border_width      = 4
    container_border_radius = 7
  }
  colors = {
    button_primary_text         = "#7b5654"
    button_primary_border       = "#9595dd"
    button_secondary_text       = "#8e8ee2"
    button_secondary_background = "#e88285"
    button_secondary_border     = "#6969cd"
    card_background             = "#9c2b2d"
    card_border                 = "#7d98cd"
    input_background            = "#c55e5d"
    input_border                = "#83a1cd"
    link                        = "#3e568d"
    heading_text                = "#7373bd"
    body_text                   = "#bcbcf1"
    container_background        = "#a7767d"
    container_border            = "#9a191d"
    divider                     = "#6c8fd3"
    icon                        = "#5959ad"
    loader                      = "#7474bd"
    positive                    = "#85e5bd"
    critical                    = "#c45f5d"
    information                 = "#3b62ad"
    hover                       = "#1a5fed"
    focus                       = "#8f82cd"
  }
  page_background = {
    background_color     = "#427ab2"
    background_image_url = "https://res.cloudinary.com/authsignal/image/upload/v1716774650/scn2dj1eqvpg9uxuffsr.jpg"
  }
  container = {
    content_alignment = "right"
    logo_alignment    = "right"
    logo_position     = "inside"
    logo_height       = 113
` + padding + `
  }
  dark_mode = {
    logo_url      = "https://res.cloudinary.com/authsignal/image/upload/v1716774524/m7bvgqjzr29tp69qcogr.png"
    favicon_url   = "https://res.cloudinary.com/authsignal/image/upload/v1716774530/nvjzmbqax2jiiwse3a40.png"
    watermark_url = "https://res.cloudinary.com/authsignal/image/upload/v1716774530/nvjzmbqax2jiiwse3a40.jpg"
    primary_color = "#a7a7d7"
    borders = {
      button_border_radius    = 6
      button_border_width     = 6
      card_border_radius      = 3
      card_border_width       = 4
      input_border_radius     = 2
      input_border_width      = 1
      container_border_radius = 5
    }
    colors = {
      button_primary_text         = "#a2818a"
      button_primary_border       = "#8c8caa"
      button_secondary_text       = "#7a7a9a"
      button_secondary_background = "#9f7a7b"
      button_secondary_border     = "#7979aa"
      card_background             = "#91555a"
      card_border                 = "#97a1ba"
      input_background            = "#ac8b8a"
      input_border                = "#6d747a"
      link                        = "#899bba"
      heading_text                = "#52526a"
      body_text                   = "#4040ba"
      container_background        = "#723b3a"
      container_border            = "#ec939a"
      divider                     = "#8994aa"
      icon                        = "#21215a"
      loader                      = "#4d4d5a"
      positive                    = "#4b585a"
      critical                    = "#6a585a"
      information                 = "#686e7a"
      hover                       = "#5f646a"
      focus                       = "#55535a"
    }
    page_background = {
      background_color     = "#89add1"
      background_image_url = "https://res.cloudinary.com/authsignal/image/upload/v1716774650/scn2dj1eqvpg9uxuffsr.png"
    }
    container = {
      content_alignment = "left"
      padding           = 58
      logo_alignment    = "right"
      logo_position     = "outside"
      logo_height       = 99
    }
  }
}`
}
