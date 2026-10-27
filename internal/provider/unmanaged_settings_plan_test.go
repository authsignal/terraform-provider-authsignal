package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
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

func TestPlanSwitchingAwayFromTheWebhookProviderRemovesTheWebhookUrl(t *testing.T) {
	skipUnlessAcceptance(t)

	_, stub := newStubAPI(t, map[string][]stubResponse{
		emailOtpGetRoute: okResponse(`{"authenticatorId":"auth_email","isActive":true,"verificationMethod":"EMAIL_OTP",` +
			`"emailProvider":"WEBHOOK","webhookUrl":"https://example.com/otp","defaults":{"isActive":true}}`),
	})
	t.Setenv("AUTHSIGNAL_HOST", stub.url)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "authsignal_email_otp_authenticator_configuration" "email_otp" {
  email_provider = "WEBHOOK"
  webhook_url    = "https://example.com/otp"
}
`,
				ResourceName:       emailOtpAddress,
				ImportState:        true,
				ImportStateId:      emailOtpAuthenticatorSlug,
				ImportStatePersist: true,
			},
			{
				Config: `
resource "authsignal_email_otp_authenticator_configuration" "email_otp" {
  email_provider = "SENDGRID"

  sendgrid_email_credentials = {
    api_key     = "SG.test"
    template_id = "d-test"
    from_email  = "noreply@example.com"
  }
}
`,
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectKnownValue(emailOtpAddress, tfjsonpath.New("webhook_url"), knownvalue.Null()),
					},
				},
			},
		},
	})

	stub.assertNotCalled(t, emailOtpPatchRoute)
}

func TestPlanClearingTheSmsCountryRestrictionRemovesTheDefaultCountryCode(t *testing.T) {
	skipUnlessAcceptance(t)

	_, stub := newStubAPI(t, map[string][]stubResponse{
		smsGetRoute: okResponse(`{"authenticatorId":"auth_sms","isActive":true,"verificationMethod":"SMS",` +
			`"smsProvider":"WEBHOOK","webhookUrl":"https://example.com/sms",` +
			`"smsCountryCodes":["AU","NZ"],"defaultCountryCode":"NZ","defaults":{"isActive":true}}`),
	})
	t.Setenv("AUTHSIGNAL_HOST", stub.url)

	const address = "authsignal_sms_authenticator_configuration.sms"

	config := `
resource "authsignal_sms_authenticator_configuration" "sms" {
  sms_provider = "WEBHOOK"
  webhook_url  = "https://example.com/sms"
%s}
`

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:             fmt.Sprintf(config, "  sms_country_codes    = [\"AU\", \"NZ\"]\n  default_country_code = \"NZ\"\n"),
				ResourceName:       address,
				ImportState:        true,
				ImportStateId:      smsAuthenticatorSlug,
				ImportStatePersist: true,
			},
			{
				Config:             fmt.Sprintf(config, "  sms_country_codes = []\n"),
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectKnownValue(address, tfjsonpath.New("default_country_code"), knownvalue.Null()),
					},
				},
			},
		},
	})

	stub.assertNotCalled(t, smsPatchRoute)
}
