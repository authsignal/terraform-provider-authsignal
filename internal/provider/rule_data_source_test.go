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
					rule_id     = "2568fe20-851d-40f6-9c17-448dc484174c"
				}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					// these fields correspond to a rule that already exists in dev dynamo.
					resource.TestCheckResourceAttr("data.authsignal_rule.terraform-acc-tests", "name", "data-source-test"),
					resource.TestCheckResourceAttr("data.authsignal_rule.terraform-acc-tests", "action_code", "terraform-acc-tests"),
					resource.TestCheckResourceAttr("data.authsignal_rule.terraform-acc-tests", "tenant_id", "ec3910e7-ab32-479e-b58b-36a122631d58"),
					resource.TestCheckResourceAttr("data.authsignal_rule.terraform-acc-tests", "description", "hello world"),
					resource.TestCheckResourceAttr("data.authsignal_rule.terraform-acc-tests", "is_active", "false"),
					resource.TestCheckResourceAttr("data.authsignal_rule.terraform-acc-tests", "priority", "0"),
					resource.TestCheckResourceAttr("data.authsignal_rule.terraform-acc-tests", "rule_id", "2568fe20-851d-40f6-9c17-448dc484174c"),
					resource.TestCheckResourceAttr("data.authsignal_rule.terraform-acc-tests", "type", "CHALLENGE"),
					resource.TestCheckResourceAttr("data.authsignal_rule.terraform-acc-tests", "verification_methods.#", "3"),
					resource.TestCheckResourceAttr("data.authsignal_rule.terraform-acc-tests", "verification_methods.0", "EMAIL_OTP"),
					resource.TestCheckResourceAttr("data.authsignal_rule.terraform-acc-tests", "verification_methods.1", "EMAIL_MAGIC_LINK"),
					resource.TestCheckResourceAttr("data.authsignal_rule.terraform-acc-tests", "verification_methods.2", "AUTHENTICATOR_APP"),
					resource.TestCheckResourceAttr("data.authsignal_rule.terraform-acc-tests", "prompt_to_enroll_verification_methods.#", "1"),
					resource.TestCheckResourceAttr("data.authsignal_rule.terraform-acc-tests", "prompt_to_enroll_verification_methods.0", "PASSKEY"),
					resource.TestCheckResourceAttr("data.authsignal_rule.terraform-acc-tests", "default_verification_method", "EMAIL_OTP"),
					resource.TestCheckResourceAttr("data.authsignal_rule.terraform-acc-tests", "conditions", `{"and":[{"==":[{"var":"ip.isAnonymous"},false]}]}`),
				),
			},
		},
	})
}
