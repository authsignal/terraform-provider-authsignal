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

func TestPlanToCreateTenantSettingsFailsForWhatTheTenantAlreadyHas(t *testing.T) {
	skipUnlessAcceptance(t)

	_, stub := newStubAPI(t, map[string][]stubResponse{
		"GET /tenant": okResponse(`{"tokenDurationInMinutes":30,"hideSuccessScreenOnEnrollment":true,"ipWhitelist":[],` +
			`"authenticatorEventsWebhookConfig":{"url":"https://example.com/hook","includeCredentialPublicKey":true},` +
			`"logEventsWebhookConfig":{"endpointUrl":"https://example.com/logs"},` +
			`"defaults":{"hideSuccessScreenOnEnrollment":false,"tokenDurationInMinutes":10}}`),
	})
	t.Setenv("AUTHSIGNAL_HOST", stub.url)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "authsignal_tenant_settings" "tenant" {
  authenticator_events_webhook_config = {
    url = "https://example.com/hook"
  }
}
`,
				PlanOnly: true,
				ExpectError: regexp.MustCompile(`token_duration_in_minutes\s+=\s+30\s+#\s+default:\s+10\s+` +
					`authenticator_events_webhook_config\s+=\s+\{\s+include_credential_public_key\s+=\s+true\s+#\s+no\s+default\s+\}\s+` +
					`log_events_webhook_config\s+=\s+\{\s+endpoint_url\s+=\s+"https://example.com/logs"\s+#\s+no\s+default\s+\}`),
			},
			{
				Config:      `resource "authsignal_pre_built_ui_settings" "ui" {}`,
				PlanOnly:    true,
				ExpectError: regexp.MustCompile(`hide_success_screen_on_enrollment\s+=\s+true\s+#\s+default:\s+false`),
			},
			{
				Config: `
resource "authsignal_tenant_settings" "tenant" {
  token_duration_in_minutes = 30

  authenticator_events_webhook_config = {
    url                           = "https://example.com/hook"
    include_credential_public_key = true
  }

  log_events_webhook_config = {
    endpoint_url = "https://example.com/logs"
  }
}

resource "authsignal_pre_built_ui_settings" "ui" {
  hide_success_screen_on_enrollment = true
}
`,
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
		},
	})

	stub.assertNotCalled(t, "PATCH /tenant")
}
