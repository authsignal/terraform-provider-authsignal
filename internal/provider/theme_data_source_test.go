package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccThemeDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `data "authsignal_theme" "theme" {}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.authsignal_theme.theme", "name", "Management-API-Testing"),
					resource.TestCheckResourceAttr("data.authsignal_theme.theme", "logo_url", "https://res.cloudinary.com/authsignal/image/upload/v1716774524/m7bvgqjzr29tp69qcogr.jpg"),
					resource.TestCheckResourceAttr("data.authsignal_theme.theme", "favicon_url", "https://res.cloudinary.com/authsignal/image/upload/v1716774530/nvjzmbqax2jiiwse3a40.jpg"),
					resource.TestCheckResourceAttr("data.authsignal_theme.theme", "watermark_url", "https://res.cloudinary.com/authsignal/image/upload/v1716774530/nvjzmbqax2jiiwse3a40.jpg"),
					resource.TestCheckResourceAttr("data.authsignal_theme.theme", "primary_color", "#121282"),
					resource.TestCheckResourceAttr("data.authsignal_theme.theme", "borders.button_border_radius", "4"),
					resource.TestCheckResourceAttr("data.authsignal_theme.theme", "borders.button_border_width", "4"),
					resource.TestCheckResourceAttr("data.authsignal_theme.theme", "borders.card_border_radius", "8"),
					resource.TestCheckResourceAttr("data.authsignal_theme.theme", "borders.card_border_width", "5"),
					resource.TestCheckResourceAttr("data.authsignal_theme.theme", "borders.input_border_radius", "8"),
					resource.TestCheckResourceAttr("data.authsignal_theme.theme", "borders.input_border_width", "4"),
					resource.TestCheckResourceAttr("data.authsignal_theme.theme", "borders.container_border_radius", "7"),
					resource.TestCheckResourceAttr("data.authsignal_theme.theme", "colors.button_primary_text", "#7b5654"),
					resource.TestCheckResourceAttr("data.authsignal_theme.theme", "colors.button_primary_border", "#9595dd"),
					resource.TestCheckResourceAttr("data.authsignal_theme.theme", "colors.button_secondary_text", "#8e8ee2"),
					resource.TestCheckResourceAttr("data.authsignal_theme.theme", "colors.button_secondary_background", "#e88285"),
					resource.TestCheckResourceAttr("data.authsignal_theme.theme", "colors.button_secondary_border", "#6969cd"),
					resource.TestCheckResourceAttr("data.authsignal_theme.theme", "colors.card_background", "#9c2b2d"),
					resource.TestCheckResourceAttr("data.authsignal_theme.theme", "colors.card_border", "#7d98cd"),
					resource.TestCheckResourceAttr("data.authsignal_theme.theme", "colors.input_background", "#c55e5d"),
					resource.TestCheckResourceAttr("data.authsignal_theme.theme", "colors.input_border", "#83a1cd"),
					resource.TestCheckResourceAttr("data.authsignal_theme.theme", "colors.link", "#3e568d"),
					resource.TestCheckResourceAttr("data.authsignal_theme.theme", "colors.heading_text", "#7373bd"),
					resource.TestCheckResourceAttr("data.authsignal_theme.theme", "colors.body_text", "#bcbcf1"),
					resource.TestCheckResourceAttr("data.authsignal_theme.theme", "colors.container_background", "#a7767d"),
					resource.TestCheckResourceAttr("data.authsignal_theme.theme", "colors.container_border", "#9a191d"),
					resource.TestCheckResourceAttr("data.authsignal_theme.theme", "colors.divider", "#6c8fd3"),
					resource.TestCheckResourceAttr("data.authsignal_theme.theme", "colors.icon", "#5959ad"),
					resource.TestCheckResourceAttr("data.authsignal_theme.theme", "colors.loader", "#7474bd"),
					resource.TestCheckResourceAttr("data.authsignal_theme.theme", "colors.positive", "#85e5bd"),
					resource.TestCheckResourceAttr("data.authsignal_theme.theme", "colors.critical", "#c45f5d"),
					resource.TestCheckResourceAttr("data.authsignal_theme.theme", "colors.information", "#3b62ad"),
					resource.TestCheckResourceAttr("data.authsignal_theme.theme", "colors.hover", "#1a5fed"),
					resource.TestCheckResourceAttr("data.authsignal_theme.theme", "colors.focus", "#8f82cd"),
					resource.TestCheckResourceAttr("data.authsignal_theme.theme", "page_background.background_color", "#427ab2"),
					resource.TestCheckResourceAttr("data.authsignal_theme.theme", "page_background.background_image_url", "https://res.cloudinary.com/authsignal/image/upload/v1716774650/scn2dj1eqvpg9uxuffsr.jpg"),
					resource.TestCheckResourceAttr("data.authsignal_theme.theme", "container.content_alignment", "right"),
					resource.TestCheckResourceAttr("data.authsignal_theme.theme", "container.padding", "61"),
					resource.TestCheckResourceAttr("data.authsignal_theme.theme", "container.logo_alignment", "right"),
					resource.TestCheckResourceAttr("data.authsignal_theme.theme", "container.logo_position", "inside"),
					resource.TestCheckResourceAttr("data.authsignal_theme.theme", "container.logo_height", "113"),
					resource.TestCheckResourceAttr("data.authsignal_theme.theme", "container.position", "outside"),

					resource.TestCheckResourceAttr("data.authsignal_theme.theme", "dark_mode.logo_url", "https://res.cloudinary.com/authsignal/image/upload/v1716774524/m7bvgqjzr29tp69qcogr.png"),
					resource.TestCheckResourceAttr("data.authsignal_theme.theme", "dark_mode.favicon_url", "https://res.cloudinary.com/authsignal/image/upload/v1716774530/nvjzmbqax2jiiwse3a40.png"),
					resource.TestCheckResourceAttr("data.authsignal_theme.theme", "dark_mode.watermark_url", "https://res.cloudinary.com/authsignal/image/upload/v1716774530/nvjzmbqax2jiiwse3a40.jpg"),
					resource.TestCheckResourceAttr("data.authsignal_theme.theme", "dark_mode.primary_color", "#a7a7d7"),
					resource.TestCheckResourceAttr("data.authsignal_theme.theme", "dark_mode.borders.button_border_radius", "6"),
					resource.TestCheckResourceAttr("data.authsignal_theme.theme", "dark_mode.borders.button_border_width", "6"),
					resource.TestCheckResourceAttr("data.authsignal_theme.theme", "dark_mode.borders.card_border_radius", "3"),
					resource.TestCheckResourceAttr("data.authsignal_theme.theme", "dark_mode.borders.card_border_width", "4"),
					resource.TestCheckResourceAttr("data.authsignal_theme.theme", "dark_mode.borders.input_border_radius", "2"),
					resource.TestCheckResourceAttr("data.authsignal_theme.theme", "dark_mode.borders.input_border_width", "1"),
					resource.TestCheckResourceAttr("data.authsignal_theme.theme", "dark_mode.borders.container_border_radius", "5"),
					resource.TestCheckResourceAttr("data.authsignal_theme.theme", "dark_mode.colors.button_primary_text", "#a2818a"),
					resource.TestCheckResourceAttr("data.authsignal_theme.theme", "dark_mode.colors.button_primary_border", "#8c8caa"),
					resource.TestCheckResourceAttr("data.authsignal_theme.theme", "dark_mode.colors.button_secondary_text", "#7a7a9a"),
					resource.TestCheckResourceAttr("data.authsignal_theme.theme", "dark_mode.colors.button_secondary_background", "#9f7a7b"),
					resource.TestCheckResourceAttr("data.authsignal_theme.theme", "dark_mode.colors.button_secondary_border", "#7979aa"),
					resource.TestCheckResourceAttr("data.authsignal_theme.theme", "dark_mode.colors.card_background", "#91555a"),
					resource.TestCheckResourceAttr("data.authsignal_theme.theme", "dark_mode.colors.card_border", "#97a1ba"),
					resource.TestCheckResourceAttr("data.authsignal_theme.theme", "dark_mode.colors.input_background", "#ac8b8a"),
					resource.TestCheckResourceAttr("data.authsignal_theme.theme", "dark_mode.colors.input_border", "#6d747a"),
					resource.TestCheckResourceAttr("data.authsignal_theme.theme", "dark_mode.colors.link", "#899bba"),
					resource.TestCheckResourceAttr("data.authsignal_theme.theme", "dark_mode.colors.heading_text", "#52526a"),
					resource.TestCheckResourceAttr("data.authsignal_theme.theme", "dark_mode.colors.body_text", "#4040ba"),
					resource.TestCheckResourceAttr("data.authsignal_theme.theme", "dark_mode.colors.container_background", "#723b3a"),
					resource.TestCheckResourceAttr("data.authsignal_theme.theme", "dark_mode.colors.container_border", "#ec939a"),
					resource.TestCheckResourceAttr("data.authsignal_theme.theme", "dark_mode.colors.divider", "#8994aa"),
					resource.TestCheckResourceAttr("data.authsignal_theme.theme", "dark_mode.colors.icon", "#21215a"),
					resource.TestCheckResourceAttr("data.authsignal_theme.theme", "dark_mode.colors.loader", "#4d4d5a"),
					resource.TestCheckResourceAttr("data.authsignal_theme.theme", "dark_mode.colors.positive", "#4b585a"),
					resource.TestCheckResourceAttr("data.authsignal_theme.theme", "dark_mode.colors.critical", "#6a585a"),
					resource.TestCheckResourceAttr("data.authsignal_theme.theme", "dark_mode.colors.information", "#686e7a"),
					resource.TestCheckResourceAttr("data.authsignal_theme.theme", "dark_mode.colors.hover", "#5f646a"),
					resource.TestCheckResourceAttr("data.authsignal_theme.theme", "dark_mode.colors.focus", "#55535a"),
					resource.TestCheckResourceAttr("data.authsignal_theme.theme", "dark_mode.page_background.background_color", "#89add1"),
					resource.TestCheckResourceAttr("data.authsignal_theme.theme", "dark_mode.page_background.background_image_url", "https://res.cloudinary.com/authsignal/image/upload/v1716774650/scn2dj1eqvpg9uxuffsr.png"),
					resource.TestCheckResourceAttr("data.authsignal_theme.theme", "dark_mode.container.content_alignment", "left"),
					resource.TestCheckResourceAttr("data.authsignal_theme.theme", "dark_mode.container.padding", "58"),
					resource.TestCheckResourceAttr("data.authsignal_theme.theme", "dark_mode.container.logo_alignment", "right"),
					resource.TestCheckResourceAttr("data.authsignal_theme.theme", "dark_mode.container.logo_position", "outside"),
					resource.TestCheckResourceAttr("data.authsignal_theme.theme", "dark_mode.container.logo_height", "99"),
					resource.TestCheckResourceAttr("data.authsignal_theme.theme", "dark_mode.container.position", "outside"),
				),
			},
		},
	})
}
