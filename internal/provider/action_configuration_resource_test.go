package provider

import (
	"fmt"
	"strconv"
	"strings"
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

// testAccFlowConfig is a MULTI_STEP_AUTHENTICATION action with two RULE nodes and a COMPLETE node,
// each RULE node defining the rule its arm references. The country code is the one thing the
// update step changes.
func testAccFlowConfig(countryCode string) string {
	return fmt.Sprintf(`
		resource "authsignal_action_configuration" "flow" {
			action_code                = "terraform-acceptance-test-flow"
			action_type                = "MULTI_STEP_AUTHENTICATION"
			default_user_action_result = "CHALLENGE"
			flow = jsonencode([
				{
					nodeId           = "rule-country"
					nodeType         = "RULE"
					parentNodeIds    = []
					ruleChildNodeIds = [["from-country", "rule-anonymous"]]
					elseChildNodeId  = "complete"
					rules = [
						{
							ruleId     = "from-country"
							name       = "From %[1]s"
							conditions = { and = [{ in = [{ var = "ip.location.country.countryCode" }, ["%[1]s"]] }] }
						}
					]
				},
				{
					nodeId           = "rule-anonymous"
					nodeType         = "RULE"
					parentNodeIds    = ["rule-country"]
					ruleChildNodeIds = [["anonymous-ip", "complete"]]
					elseChildNodeId  = "complete"
					rules = [
						{
							ruleId     = "anonymous-ip"
							name       = "Anonymous IP"
							conditions = { and = [{ "==" = [{ var = "ip.isAnonymous" }, true] }] }
						}
					]
				},
				{
					nodeId        = "complete"
					nodeType      = "COMPLETE"
					parentNodeIds = ["rule-country", "rule-anonymous"]
				}
			])
		}
	`, countryCode)
}

// The flow version continues from where an archived action of the same code left off, so the test
// asserts the bump between steps rather than absolute versions.
func TestAccActionConfigurationResource_flow(t *testing.T) {
	var firstVersion string

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccFlowConfig("NZ"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("authsignal_action_configuration.flow", "action_code", "terraform-acceptance-test-flow"),
					resource.TestCheckResourceAttr("authsignal_action_configuration.flow", "action_type", "MULTI_STEP_AUTHENTICATION"),
					resource.TestCheckResourceAttr("authsignal_action_configuration.flow", "default_user_action_result", "CHALLENGE"),
					resource.TestCheckResourceAttrSet("authsignal_action_configuration.flow", "flow_version"),
					resource.TestCheckResourceAttrWith("authsignal_action_configuration.flow", "flow_version", func(value string) error {
						firstVersion = value
						return nil
					}),
					resource.TestCheckResourceAttrWith("authsignal_action_configuration.flow", "flow", func(value string) error {
						if !strings.Contains(value, `"ruleId":"from-country"`) || !strings.Contains(value, `["NZ"]`) {
							return fmt.Errorf("expected the embedded rule in the flow, got %s", value)
						}
						return nil
					}),
				),
			},
			{
				Config: testAccFlowConfig("AU"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrWith("authsignal_action_configuration.flow", "flow_version", func(value string) error {
						first, err := strconv.Atoi(firstVersion)
						if err != nil {
							return err
						}
						second, err := strconv.Atoi(value)
						if err != nil {
							return err
						}
						if second != first+1 {
							return fmt.Errorf("expected flow_version %d after changing a condition, got %d", first+1, second)
						}
						return nil
					}),
					resource.TestCheckResourceAttrWith("authsignal_action_configuration.flow", "flow", func(value string) error {
						if !strings.Contains(value, `["AU"]`) {
							return fmt.Errorf("expected the changed condition in the flow, got %s", value)
						}
						return nil
					}),
				),
			},
			{
				ResourceName:                         "authsignal_action_configuration.flow",
				ImportState:                          true,
				ImportStateId:                        "terraform-acceptance-test-flow",
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "action_code",
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}
