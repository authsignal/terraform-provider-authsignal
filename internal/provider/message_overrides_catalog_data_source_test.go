package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccMessageOverridesCatalogDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `data "authsignal_message_overrides_catalog" "catalog" {}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.authsignal_message_overrides_catalog.catalog", "catalog_version"),
					resource.TestCheckResourceAttrSet("data.authsignal_message_overrides_catalog.catalog", "screens.0.id"),
					resource.TestCheckResourceAttrSet("data.authsignal_message_overrides_catalog.catalog", "points.0.public_id"),
					resource.TestCheckResourceAttrSet("data.authsignal_message_overrides_catalog.catalog", "points.0.max_length"),
				),
			},
		},
	})
}
