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
					resource.TestCheckResourceAttr("data.authsignal_action_configuration.helloworld", "tenant_id", "ec3910e7-ab32-479e-b58b-36a122631d58"),
					resource.TestCheckResourceAttr("data.authsignal_action_configuration.helloworld", "default_user_action_result", "ALLOW"),
					resource.TestCheckResourceAttr("data.authsignal_action_configuration.helloworld", "messaging_templates", `{"en":{"defaultTemplate":"hello world 123"}}`),
					resource.TestCheckResourceAttr("data.authsignal_action_configuration.helloworld", "verification_methods.0", "EMAIL_OTP"),
					resource.TestCheckResourceAttr("data.authsignal_action_configuration.helloworld", "verification_methods.1", "AUTHENTICATOR_APP"),
					resource.TestCheckResourceAttr("data.authsignal_action_configuration.helloworld", "default_verification_method", "AUTHENTICATOR_APP"),
				),
			},
		},
	})
}
