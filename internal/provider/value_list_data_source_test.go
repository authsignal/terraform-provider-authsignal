package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccValueListDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `data "authsignal_value_list" "hello_world_strings" {alias="hello-world-strings"}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					// these fields correspond to an Value List that already exists in dev.
					resource.TestCheckResourceAttr("data.authsignal_value_list.hello_world_strings", "alias", "hello-world-strings"),
					resource.TestCheckResourceAttr("data.authsignal_value_list.hello_world_strings", "item_type", "string"),
					resource.TestCheckResourceAttr("data.authsignal_value_list.hello_world_strings", "name", "Hello World Strings"),
					resource.TestCheckResourceAttr("data.authsignal_value_list.hello_world_strings", "is_active", "true"),
					resource.TestCheckResourceAttr("data.authsignal_value_list.hello_world_strings", "value_list_items_strings.0", "hello"),
					resource.TestCheckResourceAttr("data.authsignal_value_list.hello_world_strings", "value_list_items_strings.1", "world"),
					resource.TestCheckNoResourceAttr("data.authsignal_value_list.hello_world_strings", "value_list_items_numbers"),
				),
			},
			{
				Config: `data "authsignal_value_list" "hello_world_numbers" {alias="hello-world-numbers"}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					// these fields correspond to an Value List that already exists in dev.
					resource.TestCheckResourceAttr("data.authsignal_value_list.hello_world_numbers", "alias", "hello-world-numbers"),
					resource.TestCheckResourceAttr("data.authsignal_value_list.hello_world_numbers", "item_type", "number"),
					resource.TestCheckResourceAttr("data.authsignal_value_list.hello_world_numbers", "name", "Hello World Numbers"),
					resource.TestCheckResourceAttr("data.authsignal_value_list.hello_world_numbers", "is_active", "true"),
					resource.TestCheckResourceAttr("data.authsignal_value_list.hello_world_numbers", "value_list_items_numbers.0", "123"),
					resource.TestCheckResourceAttr("data.authsignal_value_list.hello_world_numbers", "value_list_items_numbers.1", "456"),
					resource.TestCheckResourceAttr("data.authsignal_value_list.hello_world_numbers", "value_list_items_numbers.2", "789"),
					resource.TestCheckNoResourceAttr("data.authsignal_value_list.hello_world_numbers", "value_list_items_strings"),
				),
			},
		},
	})
}
