package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccRuleDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `data "authsignal_rule" "terraform-acc-tests" {
					action_code = "terraform-acc-tests"
					rule_id     = "73d8750b-7ddf-4e0e-bf4f-a832d091e835"
				}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					// these fields correspond to a rule that already exists in dev dynamo.
					resource.TestCheckResourceAttr("data.authsignal_rule.terraform-acc-tests", "name", "data-source-test"),
					resource.TestCheckResourceAttr("data.authsignal_rule.terraform-acc-tests", "action_code", "terraform-acc-tests"),
					resource.TestCheckResourceAttr("data.authsignal_rule.terraform-acc-tests", "tenant_id", "680137cd-1bb7-40b3-b3c3-6f0a665e9d94"),
					resource.TestCheckResourceAttr("data.authsignal_rule.terraform-acc-tests", "description", "hello world"),
					resource.TestCheckResourceAttr("data.authsignal_rule.terraform-acc-tests", "is_active", "false"),
					resource.TestCheckResourceAttr("data.authsignal_rule.terraform-acc-tests", "priority", "0"),
					resource.TestCheckResourceAttr("data.authsignal_rule.terraform-acc-tests", "rule_id", "73d8750b-7ddf-4e0e-bf4f-a832d091e835"),
					resource.TestCheckResourceAttr("data.authsignal_rule.terraform-acc-tests", "type", "CHALLENGE"),
					resource.TestCheckResourceAttr("data.authsignal_rule.terraform-acc-tests", "verification_methods.#", "3"),
					resource.TestCheckResourceAttr("data.authsignal_rule.terraform-acc-tests", "verification_methods.0", "AUTHENTICATOR_APP"),
					resource.TestCheckResourceAttr("data.authsignal_rule.terraform-acc-tests", "verification_methods.1", "EMAIL_MAGIC_LINK"),
					resource.TestCheckResourceAttr("data.authsignal_rule.terraform-acc-tests", "verification_methods.2", "EMAIL_OTP"),
					resource.TestCheckResourceAttr("data.authsignal_rule.terraform-acc-tests", "prompt_to_enroll_verification_methods.#", "1"),
					resource.TestCheckResourceAttr("data.authsignal_rule.terraform-acc-tests", "prompt_to_enroll_verification_methods.0", "PASSKEY"),
					resource.TestCheckResourceAttr("data.authsignal_rule.terraform-acc-tests", "default_verification_method", "EMAIL_OTP"),
					resource.TestCheckResourceAttr("data.authsignal_rule.terraform-acc-tests", "conditions", `{"and":[{"==":[{"var":"ip.isAnonymous"},false]}]}`),
				),
			},
		},
	})
}
