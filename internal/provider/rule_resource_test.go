package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccRuleResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: `
				resource "authsignal_rule" "terraform-acc-tests" {
					action_code = "terraform-acc-tests"
					name        = "create-rule-test"
					description = "hello world"
					priority    = 1
					type        = "ALLOW"
					is_active   = false
					verification_methods = [
						"AUTHENTICATOR_APP",
						"EMAIL_MAGIC_LINK",
						"EMAIL_OTP"
					]
					prompt_to_enroll_verification_methods = [
						"PASSKEY"
					]
					default_verification_method = "EMAIL_OTP"
					conditions = jsonencode({
						"and" : [
							{
								"==" : [
									{
										"var" : "ip.isAnonymous"
									},
									false
								]
							}
						]
					})
				}
				`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("authsignal_rule.terraform-acc-tests", "name", "create-rule-test"),
					resource.TestCheckResourceAttr("authsignal_rule.terraform-acc-tests", "action_code", "terraform-acc-tests"),
					resource.TestCheckResourceAttr("authsignal_rule.terraform-acc-tests", "tenant_id", "680137cd-1bb7-40b3-b3c3-6f0a665e9d94"),
					resource.TestCheckResourceAttr("authsignal_rule.terraform-acc-tests", "description", "hello world"),
					resource.TestCheckResourceAttr("authsignal_rule.terraform-acc-tests", "is_active", "false"),
					resource.TestCheckResourceAttr("authsignal_rule.terraform-acc-tests", "priority", "1"),
					resource.TestCheckResourceAttr("authsignal_rule.terraform-acc-tests", "type", "ALLOW"),
					resource.TestCheckResourceAttr("authsignal_rule.terraform-acc-tests", "verification_methods.#", "3"),
					resource.TestCheckResourceAttr("authsignal_rule.terraform-acc-tests", "verification_methods.0", "AUTHENTICATOR_APP"),
					resource.TestCheckResourceAttr("authsignal_rule.terraform-acc-tests", "verification_methods.1", "EMAIL_MAGIC_LINK"),
					resource.TestCheckResourceAttr("authsignal_rule.terraform-acc-tests", "verification_methods.2", "EMAIL_OTP"),
					resource.TestCheckResourceAttr("authsignal_rule.terraform-acc-tests", "prompt_to_enroll_verification_methods.#", "1"),
					resource.TestCheckResourceAttr("authsignal_rule.terraform-acc-tests", "prompt_to_enroll_verification_methods.0", "PASSKEY"),
					resource.TestCheckResourceAttr("authsignal_rule.terraform-acc-tests", "default_verification_method", "EMAIL_OTP"),
					resource.TestCheckResourceAttr("authsignal_rule.terraform-acc-tests", "conditions", `{"and":[{"==":[{"var":"ip.isAnonymous"},false]}]}`),
				),
			},
			// Update and Read testing
			{
				Config: `
				resource "authsignal_rule" "terraform-acc-tests" {
					action_code = "terraform-acc-tests"
					name        = "update-rule-test"
					description = "hello world"
					priority    = 2
					type        = "CHALLENGE"
					is_active   = false
					verification_methods = [
						"AUTHENTICATOR_APP",
						"EMAIL_MAGIC_LINK",
						"EMAIL_OTP"
					]
					prompt_to_enroll_verification_methods = [
						"PASSKEY"
					]
					default_verification_method = "EMAIL_OTP"
					conditions = jsonencode({
						"and" : [
							{
								"==" : [
									{
										"var" : "ip.isAnonymous"
									},
									false
								]
							}
						]
					})
				}
				`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("authsignal_rule.terraform-acc-tests", "name", "update-rule-test"),
					resource.TestCheckResourceAttr("authsignal_rule.terraform-acc-tests", "action_code", "terraform-acc-tests"),
					resource.TestCheckResourceAttr("authsignal_rule.terraform-acc-tests", "tenant_id", "680137cd-1bb7-40b3-b3c3-6f0a665e9d94"),
					resource.TestCheckResourceAttr("authsignal_rule.terraform-acc-tests", "description", "hello world"),
					resource.TestCheckResourceAttr("authsignal_rule.terraform-acc-tests", "is_active", "false"),
					resource.TestCheckResourceAttr("authsignal_rule.terraform-acc-tests", "priority", "2"),
					resource.TestCheckResourceAttr("authsignal_rule.terraform-acc-tests", "type", "CHALLENGE"),
					resource.TestCheckResourceAttr("authsignal_rule.terraform-acc-tests", "verification_methods.#", "3"),
					resource.TestCheckResourceAttr("authsignal_rule.terraform-acc-tests", "verification_methods.0", "AUTHENTICATOR_APP"),
					resource.TestCheckResourceAttr("authsignal_rule.terraform-acc-tests", "verification_methods.1", "EMAIL_MAGIC_LINK"),
					resource.TestCheckResourceAttr("authsignal_rule.terraform-acc-tests", "verification_methods.2", "EMAIL_OTP"),
					resource.TestCheckResourceAttr("authsignal_rule.terraform-acc-tests", "prompt_to_enroll_verification_methods.#", "1"),
					resource.TestCheckResourceAttr("authsignal_rule.terraform-acc-tests", "prompt_to_enroll_verification_methods.0", "PASSKEY"),
					resource.TestCheckResourceAttr("authsignal_rule.terraform-acc-tests", "default_verification_method", "EMAIL_OTP"),
					resource.TestCheckResourceAttr("authsignal_rule.terraform-acc-tests", "conditions", `{"and":[{"==":[{"var":"ip.isAnonymous"},false]}]}`),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}
