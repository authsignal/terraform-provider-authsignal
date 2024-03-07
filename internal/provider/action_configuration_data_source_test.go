package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccActionConfigurationDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `data "authsignal_action_configuration" "helloworld" {action_code="helloworld"}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					// these fields correspond to an Action Configuration that already exists in dev dynamo.
					resource.TestCheckResourceAttr("data.authsignal_action_configuration.helloworld", "action_code", "helloworld"),
					resource.TestCheckResourceAttr("data.authsignal_action_configuration.helloworld", "tenant_id", "680137cd-1bb7-40b3-b3c3-6f0a665e9d94"),
					resource.TestCheckResourceAttr("data.authsignal_action_configuration.helloworld", "default_user_action_result", "ALLOW"),
				),
			},
		},
	})
}
