package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccCustomDataPointDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `data "authsignal_custom_data_point" "terraform_data_source_testing" {id="45930709-396e-440c-893e-8f67794dc345"}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.authsignal_custom_data_point.terraform_data_source_testing", "name", "Terraform_Data_Source_Testing"),
					resource.TestCheckResourceAttr("data.authsignal_custom_data_point.terraform_data_source_testing", "data_type", "text"),
					resource.TestCheckResourceAttr("data.authsignal_custom_data_point.terraform_data_source_testing", "model_type", "action"),
					resource.TestCheckResourceAttr("data.authsignal_custom_data_point.terraform_data_source_testing", "description", "hello world"),
				),
			},
		},
	})
}
