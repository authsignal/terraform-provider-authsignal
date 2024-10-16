package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccValueListResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing for strings
			{
				Config: `
					resource "authsignal_value_list" "terraform-acc-test-strings" {
						name = "Terraform Acc Test Strings"
						is_active = true
						value_list_items_strings = ["terraform", "acceptance", "test"]
					}
				`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("authsignal_value_list.terraform-acc-test-strings", "name", "Terraform Acc Test Strings"),
					resource.TestCheckResourceAttr("authsignal_value_list.terraform-acc-test-strings", "alias", "terraform-acc-test-strings"),
					resource.TestCheckResourceAttr("authsignal_value_list.terraform-acc-test-strings", "item_type", "string"),
					resource.TestCheckResourceAttr("authsignal_value_list.terraform-acc-test-strings", "is_active", "true"),
					resource.TestCheckResourceAttr("authsignal_value_list.terraform-acc-test-strings", "value_list_items_strings.0", "terraform"),
					resource.TestCheckResourceAttr("authsignal_value_list.terraform-acc-test-strings", "value_list_items_strings.1", "acceptance"),
					resource.TestCheckResourceAttr("authsignal_value_list.terraform-acc-test-strings", "value_list_items_strings.2", "test"),
				),
			},
			// Create and Read testing for numbers
			{
				Config: `
					resource "authsignal_value_list" "terraform-acc-test-numbers" {
						name = "Terraform Acc Test Numbers"
						is_active = true
						value_list_items_numbers = [1, 2, 3]
					}
				`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("authsignal_value_list.terraform-acc-test-numbers", "name", "Terraform Acc Test Numbers"),
					resource.TestCheckResourceAttr("authsignal_value_list.terraform-acc-test-numbers", "alias", "terraform-acc-test-numbers"),
					resource.TestCheckResourceAttr("authsignal_value_list.terraform-acc-test-numbers", "item_type", "number"),
					resource.TestCheckResourceAttr("authsignal_value_list.terraform-acc-test-numbers", "is_active", "true"),
					resource.TestCheckResourceAttr("authsignal_value_list.terraform-acc-test-numbers", "value_list_items_numbers.0", "1"),
					resource.TestCheckResourceAttr("authsignal_value_list.terraform-acc-test-numbers", "value_list_items_numbers.1", "2"),
					resource.TestCheckResourceAttr("authsignal_value_list.terraform-acc-test-numbers", "value_list_items_numbers.2", "3"),
				),
			},
			// Update testing for strings without recreate
			{
				Config: `
					resource "authsignal_value_list" "terraform-acc-test-strings" {
						name = "Terraform Acc Test Strings"
						is_active = false
						value_list_items_strings = ["updated", "terraform", "test"]
					}
				`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("authsignal_value_list.terraform-acc-test-strings", "name", "Terraform Acc Test Strings"),
					resource.TestCheckResourceAttr("authsignal_value_list.terraform-acc-test-strings", "alias", "terraform-acc-test-strings"),
					resource.TestCheckResourceAttr("authsignal_value_list.terraform-acc-test-strings", "item_type", "string"),
					resource.TestCheckResourceAttr("authsignal_value_list.terraform-acc-test-strings", "is_active", "false"),
					resource.TestCheckResourceAttr("authsignal_value_list.terraform-acc-test-strings", "value_list_items_strings.0", "updated"),
					resource.TestCheckResourceAttr("authsignal_value_list.terraform-acc-test-strings", "value_list_items_strings.1", "terraform"),
					resource.TestCheckResourceAttr("authsignal_value_list.terraform-acc-test-strings", "value_list_items_strings.2", "test"),
				),
			},
			// Update testing for numbers without recreate
			{
				Config: `
					resource "authsignal_value_list" "terraform-acc-test-numbers" {
						name = "Terraform Acc Test Numbers"
						is_active = false
						value_list_items_numbers = [4, 5, 6]
					}
				`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("authsignal_value_list.terraform-acc-test-numbers", "name", "Terraform Acc Test Numbers"),
					resource.TestCheckResourceAttr("authsignal_value_list.terraform-acc-test-numbers", "alias", "terraform-acc-test-numbers"),
					resource.TestCheckResourceAttr("authsignal_value_list.terraform-acc-test-numbers", "item_type", "number"),
					resource.TestCheckResourceAttr("authsignal_value_list.terraform-acc-test-numbers", "is_active", "false"),
					resource.TestCheckResourceAttr("authsignal_value_list.terraform-acc-test-numbers", "value_list_items_numbers.0", "4"),
					resource.TestCheckResourceAttr("authsignal_value_list.terraform-acc-test-numbers", "value_list_items_numbers.1", "5"),
					resource.TestCheckResourceAttr("authsignal_value_list.terraform-acc-test-numbers", "value_list_items_numbers.2", "6"),
				),
			},
			// Update testing for strings with recreate
			{
				Config: `
					resource "authsignal_value_list" "terraform-acc-test-strings" {
						name = "Terraform Acc Test Strings Updated"
						is_active = false
						value_list_items_strings = ["foo", "bar", "baz"]
					}
				`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("authsignal_value_list.terraform-acc-test-strings", "name", "Terraform Acc Test Strings Updated"),
					resource.TestCheckResourceAttr("authsignal_value_list.terraform-acc-test-strings", "alias", "terraform-acc-test-strings-updated"),
					resource.TestCheckResourceAttr("authsignal_value_list.terraform-acc-test-strings", "item_type", "string"),
					resource.TestCheckResourceAttr("authsignal_value_list.terraform-acc-test-strings", "is_active", "false"),
					resource.TestCheckResourceAttr("authsignal_value_list.terraform-acc-test-strings", "value_list_items_strings.0", "foo"),
					resource.TestCheckResourceAttr("authsignal_value_list.terraform-acc-test-strings", "value_list_items_strings.1", "bar"),
					resource.TestCheckResourceAttr("authsignal_value_list.terraform-acc-test-strings", "value_list_items_strings.2", "baz"),
				),
			},
			// Update testing for numbers with recreate
			{
				Config: `
					resource "authsignal_value_list" "terraform-acc-test-numbers" {
						name = "Terraform Acc Test Numbers Updated"
						is_active = false
						value_list_items_numbers = [7, 8, 9]
					}
				`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("authsignal_value_list.terraform-acc-test-numbers", "name", "Terraform Acc Test Numbers Updated"),
					resource.TestCheckResourceAttr("authsignal_value_list.terraform-acc-test-numbers", "alias", "terraform-acc-test-numbers-updated"),
					resource.TestCheckResourceAttr("authsignal_value_list.terraform-acc-test-numbers", "item_type", "number"),
					resource.TestCheckResourceAttr("authsignal_value_list.terraform-acc-test-numbers", "is_active", "false"),
					resource.TestCheckResourceAttr("authsignal_value_list.terraform-acc-test-numbers", "value_list_items_numbers.0", "7"),
					resource.TestCheckResourceAttr("authsignal_value_list.terraform-acc-test-numbers", "value_list_items_numbers.1", "8"),
					resource.TestCheckResourceAttr("authsignal_value_list.terraform-acc-test-numbers", "value_list_items_numbers.2", "9"),
				),
			},
		},
	})
}
