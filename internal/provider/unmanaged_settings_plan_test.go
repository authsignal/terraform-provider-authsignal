package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestPlanFailsForAnImportedSettingTheConfigurationLeavesOut(t *testing.T) {
	skipUnlessAcceptance(t)

	_, stub := newStubAPI(t, map[string][]stubResponse{
		passkeyGetRoute: okResponse(`{"authenticatorId":"auth_passkey","isActive":false,"verificationMethod":"PASSKEY",` +
			`"relyingParty":"example.com","expectedOrigins":["https://example.com"],` +
			`"userVerificationRequirement":"preferred","authenticatorAttachment":"all-supported",` +
			`"defaults":{"isActive":true,"userVerificationRequirement":"preferred","authenticatorAttachment":"all-supported","preventRemovingLastPasskey":false}}`),
	})
	t.Setenv("AUTHSIGNAL_HOST", stub.url)

	config := `
resource "authsignal_passkey_authenticator_configuration" "passkey" {
  relying_party    = "example.com"
  expected_origins = ["https://example.com"]
%s}
`

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:             fmt.Sprintf(config, ""),
				ResourceName:       "authsignal_passkey_authenticator_configuration.passkey",
				ImportState:        true,
				ImportStateId:      passkeyAuthenticatorSlug,
				ImportStatePersist: true,
			},
			{
				Config:      fmt.Sprintf(config, ""),
				PlanOnly:    true,
				ExpectError: regexp.MustCompile(`Settings Missing From the Configuration[\s\S]*is_active\s+=\s+false\s+#\s+default:\s+true`),
			},
			{
				Config:   fmt.Sprintf(config, "  is_active        = false\n"),
				PlanOnly: true,
			},
		},
	})
}
