package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccCustomDataPointResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing for action model type
			{
				Config: `
					resource "authsignal_custom_data_point" "terraform_acc_test_action_custom_data_point" {
						name = "Terraform Acc Test Action Model Type"
						data_type = "number"
						model_type = "action"
						description = "A test custom data point."
					}
				`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("authsignal_custom_data_point.terraform_acc_test_action_custom_data_point", "name", "Terraform Acc Test Action Model Type"),
					resource.TestCheckResourceAttr("authsignal_custom_data_point.terraform_acc_test_action_custom_data_point", "data_type", "number"),
					resource.TestCheckResourceAttr("authsignal_custom_data_point.terraform_acc_test_action_custom_data_point", "model_type", "action"),
					resource.TestCheckResourceAttr("authsignal_custom_data_point.terraform_acc_test_action_custom_data_point", "description", "A test custom data point."),
				),
			},
			// Create and Read testing for user model type
			{
				Config: `
					resource "authsignal_custom_data_point" "terraform_acc_test_user_custom_data_point" {
						name = "Terraform Acc Test User Model Type"
						data_type = "text"
						model_type = "user"
						description = "A test custom data point."
					}
				`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("authsignal_custom_data_point.terraform_acc_test_user_custom_data_point", "name", "Terraform Acc Test User Model Type"),
					resource.TestCheckResourceAttr("authsignal_custom_data_point.terraform_acc_test_user_custom_data_point", "data_type", "text"),
					resource.TestCheckResourceAttr("authsignal_custom_data_point.terraform_acc_test_user_custom_data_point", "model_type", "user"),
					resource.TestCheckResourceAttr("authsignal_custom_data_point.terraform_acc_test_user_custom_data_point", "description", "A test custom data point."),
				),
			},
			// Overwrite testing for action model type
			{
				Config: `
					resource "authsignal_custom_data_point" "terraform_acc_test_action_custom_data_point" {
						name = "Terraform Acc Test Action Model Type"
						data_type = "multiselect"
						model_type = "action"
						description = "A test custom data point.!.!.!"
					}
				`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("authsignal_custom_data_point.terraform_acc_test_action_custom_data_point", "name", "Terraform Acc Test Action Model Type"),
					resource.TestCheckResourceAttr("authsignal_custom_data_point.terraform_acc_test_action_custom_data_point", "data_type", "multiselect"),
					resource.TestCheckResourceAttr("authsignal_custom_data_point.terraform_acc_test_action_custom_data_point", "model_type", "action"),
					resource.TestCheckResourceAttr("authsignal_custom_data_point.terraform_acc_test_action_custom_data_point", "description", "A test custom data point.!.!.!"),
				),
			},
			// Overwrite testing for user model type
			{
				Config: `
					resource "authsignal_custom_data_point" "terraform_acc_test_user_custom_data_point" {
						name = "Terraform Acc Test User Model Type"
						data_type = "boolean"
						model_type = "user"
						description = "A test custom data point.!.!.!"
					}
				`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("authsignal_custom_data_point.terraform_acc_test_user_custom_data_point", "name", "Terraform Acc Test User Model Type"),
					resource.TestCheckResourceAttr("authsignal_custom_data_point.terraform_acc_test_user_custom_data_point", "data_type", "boolean"),
					resource.TestCheckResourceAttr("authsignal_custom_data_point.terraform_acc_test_user_custom_data_point", "model_type", "user"),
					resource.TestCheckResourceAttr("authsignal_custom_data_point.terraform_acc_test_user_custom_data_point", "description", "A test custom data point.!.!.!"),
				),
			},
		},
	})
}
