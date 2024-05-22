package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccActionConfigurationResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: `
					resource "authsignal_action_configuration" "test" {
						action_code = "terraform-acceptance-test"
						default_user_action_result = "ALLOW"
					}
				`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("authsignal_action_configuration.test", "action_code", "terraform-acceptance-test"),
					resource.TestCheckResourceAttr("authsignal_action_configuration.test", "default_user_action_result", "ALLOW"),
					resource.TestCheckResourceAttr("authsignal_action_configuration.test", "tenant_id", "ec3910e7-ab32-479e-b58b-36a122631d58"),
				),
			},
			{
				Config: `
					resource "authsignal_action_configuration" "test-templates" {
						action_code = "terraform-acceptance-test-templates"
						default_user_action_result = "ALLOW"
						messaging_templates = jsonencode({
							"en": {
								"defaultTemplate": "hello world"
							}
						})
					}
				`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("authsignal_action_configuration.test-templates", "action_code", "terraform-acceptance-test-templates"),
					resource.TestCheckResourceAttr("authsignal_action_configuration.test-templates", "default_user_action_result", "ALLOW"),
					resource.TestCheckResourceAttr("authsignal_action_configuration.test-templates", "tenant_id", "ec3910e7-ab32-479e-b58b-36a122631d58"),
					resource.TestCheckResourceAttr("authsignal_action_configuration.test-templates", "messaging_templates", "{\"en\":{\"defaultTemplate\":\"hello world\"}}"),
				),
			},
			{
				Config: `
					resource "authsignal_action_configuration" "test-verification-methods" {
						action_code = "terraform-acceptance-test-verification-methods"
						default_user_action_result = "ALLOW"
						verification_methods = ["EMAIL_OTP", "EMAIL_MAGIC_LINK"]
						default_verification_method = "EMAIL_MAGIC_LINK"
					}
				`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("authsignal_action_configuration.test-verification-methods", "action_code", "terraform-acceptance-test-verification-methods"),
					resource.TestCheckResourceAttr("authsignal_action_configuration.test-verification-methods", "default_user_action_result", "ALLOW"),
					resource.TestCheckResourceAttr("authsignal_action_configuration.test-verification-methods", "tenant_id", "ec3910e7-ab32-479e-b58b-36a122631d58"),
					resource.TestCheckResourceAttr("authsignal_action_configuration.test-verification-methods", "verification_methods.0", "EMAIL_OTP"),
					resource.TestCheckResourceAttr("authsignal_action_configuration.test-verification-methods", "verification_methods.1", "EMAIL_MAGIC_LINK"),
					resource.TestCheckResourceAttr("authsignal_action_configuration.test-verification-methods", "default_verification_method", "EMAIL_MAGIC_LINK"),
				),
			},
			// Update and Read testing
			{
				Config: `
					resource "authsignal_action_configuration" "test" {
						action_code = "terraform-acceptance-test"
						default_user_action_result = "BLOCK"
					}
				`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("authsignal_action_configuration.test", "action_code", "terraform-acceptance-test"),
					resource.TestCheckResourceAttr("authsignal_action_configuration.test", "default_user_action_result", "BLOCK"),
					resource.TestCheckResourceAttr("authsignal_action_configuration.test", "tenant_id", "ec3910e7-ab32-479e-b58b-36a122631d58"),
				),
			},
			{
				Config: `
					resource "authsignal_action_configuration" "test-templates" {
						action_code = "terraform-acceptance-test-templates"
						default_user_action_result = "ALLOW"
						messaging_templates = jsonencode({
							"fr": {
								"defaultTemplate": "bonjour"
							}
						})
					}
				`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("authsignal_action_configuration.test-templates", "action_code", "terraform-acceptance-test-templates"),
					resource.TestCheckResourceAttr("authsignal_action_configuration.test-templates", "default_user_action_result", "ALLOW"),
					resource.TestCheckResourceAttr("authsignal_action_configuration.test-templates", "tenant_id", "ec3910e7-ab32-479e-b58b-36a122631d58"),
					resource.TestCheckResourceAttr("authsignal_action_configuration.test-templates", "messaging_templates", "{\"fr\":{\"defaultTemplate\":\"bonjour\"}}"),
				),
			},
			{
				Config: `
					resource "authsignal_action_configuration" "test-verification-methods" {
						action_code = "terraform-acceptance-test-verification-methods"
						default_user_action_result = "ALLOW"
						verification_methods = ["AUTHENTICATOR_APP", "EMAIL_MAGIC_LINK"]
						default_verification_method = "AUTHENTICATOR_APP"
					}
				`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("authsignal_action_configuration.test-verification-methods", "action_code", "terraform-acceptance-test-verification-methods"),
					resource.TestCheckResourceAttr("authsignal_action_configuration.test-verification-methods", "default_user_action_result", "ALLOW"),
					resource.TestCheckResourceAttr("authsignal_action_configuration.test-verification-methods", "tenant_id", "ec3910e7-ab32-479e-b58b-36a122631d58"),
					resource.TestCheckResourceAttr("authsignal_action_configuration.test-verification-methods", "verification_methods.0", "AUTHENTICATOR_APP"),
					resource.TestCheckResourceAttr("authsignal_action_configuration.test-verification-methods", "verification_methods.1", "EMAIL_MAGIC_LINK"),
					resource.TestCheckResourceAttr("authsignal_action_configuration.test-verification-methods", "default_verification_method", "AUTHENTICATOR_APP"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}
