package provider

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func authenticatorResourceSchemas(t *testing.T) map[string]schema.Schema {
	t.Helper()

	return map[string]schema.Schema{
		"authsignal_sms_authenticator_configuration":       resourceSchemaOf(t, NewSmsAuthenticatorConfigurationResource()),
		"authsignal_email_otp_authenticator_configuration": resourceSchemaOf(t, NewEmailOtpAuthenticatorConfigurationResource()),
		"authsignal_passkey_authenticator_configuration":   resourceSchemaOf(t, NewPasskeyAuthenticatorConfigurationResource()),
		"authsignal_push_authenticator_configuration":      resourceSchemaOf(t, NewPushAuthenticatorConfigurationResource()),
	}
}

func authenticatorCredentialBlocks() map[string][]credentialBlock {
	return map[string][]credentialBlock{
		"sms":       smsCredentialBlocks,
		"email_otp": emailOtpCredentialBlocks,
		"push":      pushCredentialBlocks,
	}
}

func TestSingletonSchemasHaveNoIdentifyingAttributes(t *testing.T) {
	for name, resourceSchema := range authenticatorResourceSchemas(t) {
		for _, forbidden := range []string{"id", "tenant_id"} {
			if _, ok := resourceSchema.Attributes[forbidden]; ok {
				t.Errorf("%s: schema has a %q attribute, but the configuration is a singleton on the provider's tenant", name, forbidden)
			}
		}
	}
}

func TestServerOwnedAttributesAreComputed(t *testing.T) {
	for name, resourceSchema := range authenticatorResourceSchemas(t) {
		for _, computed := range []string{"authenticator_id", "verification_method"} {
			attribute, ok := resourceSchema.Attributes[computed]
			if !ok {
				t.Errorf("%s: schema has no %q attribute", name, computed)
				continue
			}

			if !attribute.IsComputed() || attribute.IsOptional() || attribute.IsRequired() {
				t.Errorf("%s: %q should be computed only, got computed=%t optional=%t required=%t",
					name, computed, attribute.IsComputed(), attribute.IsOptional(), attribute.IsRequired())
			}
		}
	}
}

func TestSecretMembersAreWriteOnly(t *testing.T) {
	schemas := authenticatorResourceSchemas(t)

	for prefix, blocks := range authenticatorCredentialBlocks() {
		resourceSchema := schemas["authsignal_"+prefix+"_authenticator_configuration"]

		for _, block := range blocks {
			attribute, ok := resourceSchema.Attributes[block.Name]
			if !ok {
				t.Errorf("%s: schema has no %q attribute", prefix, block.Name)
				continue
			}

			nested, ok := attribute.(schema.SingleNestedAttribute)
			if !ok {
				t.Errorf("%s: %q should be a single nested attribute, got %T", prefix, block.Name, attribute)
				continue
			}

			if nested.WriteOnly != block.allSecret() {
				t.Errorf("%s: %q write-only is %t but the block returns %s readable metadata",
					prefix, block.Name, nested.WriteOnly, map[bool]string{true: "no", false: "some"}[block.allSecret()])
			}

			for _, member := range block.Members {
				child, ok := nested.Attributes[member.Name]
				if !ok {
					t.Errorf("%s: %q has no %q member", prefix, block.Name, member.Name)
					continue
				}

				wantWriteOnly := member.Secret || block.allSecret()
				if child.IsWriteOnly() != wantWriteOnly {
					t.Errorf("%s: %s.%s write-only is %t, want %t", prefix, block.Name, member.Name, child.IsWriteOnly(), wantWriteOnly)
				}

			}
		}
	}
}

func TestWriteOnlyMembersAreNeverComputed(t *testing.T) {
	schemas := authenticatorResourceSchemas(t)

	for prefix, blocks := range authenticatorCredentialBlocks() {
		resourceSchema := schemas["authsignal_"+prefix+"_authenticator_configuration"]

		for _, block := range blocks {
			nested, ok := resourceSchema.Attributes[block.Name].(schema.SingleNestedAttribute)
			if !ok {
				continue
			}

			for name, child := range nested.Attributes {
				if child.IsWriteOnly() && child.IsComputed() {
					t.Errorf("%s: %s.%s is write-only and computed, which Terraform does not allow", prefix, block.Name, name)
				}
			}
		}
	}
}

func TestOnlyBlocksWithNoReturnedMetadataAreFullyWriteOnly(t *testing.T) {
	expected := map[string]bool{
		tnzCredentialsBlock: true,
		fcmCredentialsBlock: true,
	}

	for prefix, blocks := range authenticatorCredentialBlocks() {
		for _, block := range blocks {
			if block.allSecret() != expected[block.Name] {
				t.Errorf("%s: %q fully write-only is %t, want %t", prefix, block.Name, block.allSecret(), expected[block.Name])
			}
		}
	}
}

func TestEveryCredentialBlockHasAnOptionalVersionAttribute(t *testing.T) {
	schemas := authenticatorResourceSchemas(t)

	for prefix, blocks := range authenticatorCredentialBlocks() {
		resourceSchema := schemas["authsignal_"+prefix+"_authenticator_configuration"]

		for _, block := range blocks {
			attribute, ok := resourceSchema.Attributes[block.versionName()]
			if !ok {
				t.Errorf("%s: schema has no %q attribute", prefix, block.versionName())
				continue
			}

			if !attribute.IsOptional() || attribute.IsComputed() || attribute.IsWriteOnly() {
				t.Errorf("%s: %q should be plain optional, got optional=%t computed=%t writeOnly=%t",
					prefix, block.versionName(), attribute.IsOptional(), attribute.IsComputed(), attribute.IsWriteOnly())
			}
		}
	}
}

func TestProviderEnumsMatchTheContract(t *testing.T) {
	cases := []struct {
		name     string
		values   []string
		expected []string
	}{
		{"sms", allowedSmsProviders, []string{"BIRD", "MESSAGE_MEDIA", "MODICA_GROUP", "TNZ", "TWILIO", "WEBHOOK"}},
		{"email", allowedEmailProviders, []string{"BIRD", "MAILGUN", "MAILJET", "MANDRILL", "SENDGRID", "SMTP", "WEBHOOK"}},
		{"push", allowedPushProviders, []string{"DEFAULT", "FIREBASE", "WEBHOOK"}},
	}

	for _, testCase := range cases {
		if strings.Join(testCase.values, ",") != strings.Join(testCase.expected, ",") {
			t.Errorf("%s providers are %v, want %v", testCase.name, testCase.values, testCase.expected)
		}
	}
}

func TestSchemasCarryNothingOutOfScope(t *testing.T) {
	forbiddenSubstrings := []string{
		"sms_channel",
		"is_editable_by_user",
		"is_hidden_to_user",
		"recovery_methods",
		"whatsapp",
		"airship",
		"atomic",
		"biometric",
		"uplift",
		"totp",
		"issuer_name",
		"allowed_custom_push_variables",
	}

	forbiddenNames := []string{
		"rate_limit_configuration",
		"sending_rate_limit_configuration",
		"additional_sending_rate_limit_configurations",
	}

	for name, resourceSchema := range authenticatorResourceSchemas(t) {
		for attributeName, attribute := range resourceSchema.Attributes {
			names := []string{attributeName}

			if nested, ok := attribute.(schema.SingleNestedAttribute); ok {
				for memberName := range nested.Attributes {
					names = append(names, attributeName+"."+memberName)
				}
			}

			for _, candidate := range names {
				for _, banned := range forbiddenSubstrings {
					if strings.Contains(candidate, banned) {
						t.Errorf("%s: attribute %q matches out-of-scope name %q", name, candidate, banned)
					}
				}

				for _, banned := range forbiddenNames {
					if candidate == banned {
						t.Errorf("%s: attribute %q is a name the typed contract replaced", name, candidate)
					}
				}
			}
		}
	}
}

func TestRateLimitAttributesUseTheContractNames(t *testing.T) {
	schemas := authenticatorResourceSchemas(t)

	expected := map[string][]string{
		"authsignal_sms_authenticator_configuration":       {"submission_rate_limit_configuration", "sending_rate_limit_configurations", "prefix_rate_limit_configurations"},
		"authsignal_email_otp_authenticator_configuration": {"submission_rate_limit_configuration", "sending_rate_limit_configurations"},
		"authsignal_push_authenticator_configuration":      {"sending_rate_limit_configurations"},
	}

	for name, attributeNames := range expected {
		for _, attributeName := range attributeNames {
			if _, ok := schemas[name].Attributes[attributeName]; !ok {
				t.Errorf("%s: schema has no %q attribute", name, attributeName)
			}
		}
	}

	for _, name := range []string{"authsignal_email_otp_authenticator_configuration", "authsignal_passkey_authenticator_configuration", "authsignal_push_authenticator_configuration"} {
		if _, ok := schemas[name].Attributes["prefix_rate_limit_configurations"]; ok {
			t.Errorf("%s: prefix rate limits are an SMS-only field", name)
		}
	}
}

func TestPasskeyExpectedOriginsIsRequired(t *testing.T) {
	passkeySchema := resourceSchemaOf(t, NewPasskeyAuthenticatorConfigurationResource())

	attribute, ok := passkeySchema.Attributes["expected_origins"]
	if !ok {
		t.Fatal("passkey schema has no expected_origins attribute")
	}

	if !attribute.IsRequired() {
		t.Error("expected_origins should be required")
	}
}

func rateLimitWindowDiagnostics(t *testing.T, window float64) diag.Diagnostics {
	t.Helper()

	attribute, ok := rateLimitAttributes()["window_in_minutes"].(schema.Float64Attribute)
	if !ok {
		t.Fatal("window_in_minutes should be a float64 attribute")
	}

	resp := &validator.Float64Response{}
	for _, declared := range attribute.Validators {
		declared.ValidateFloat64(context.Background(), validator.Float64Request{
			Path:        path.Root("window_in_minutes"),
			ConfigValue: types.Float64Value(window),
		}, resp)
	}

	return resp.Diagnostics
}

func TestRateLimitWindowAcceptsTheSmallestWindowTheApiAllows(t *testing.T) {
	if diags := rateLimitWindowDiagnostics(t, 1); diags.HasError() {
		t.Errorf("a window of 1 minute is the API's minimum and should be accepted, got %v", diags)
	}
}

func TestRateLimitWindowAcceptsAFractionalWindowAboveTheMinimum(t *testing.T) {
	for _, window := range []float64{1.5, 90.25, 1440} {
		if diags := rateLimitWindowDiagnostics(t, window); diags.HasError() {
			t.Errorf("a window of %v should be accepted, got %v", window, diags)
		}
	}
}

func TestRateLimitWindowRejectsAWindowShorterThanTheMinimum(t *testing.T) {
	for _, window := range []float64{0.5, 0, -1} {
		if diags := rateLimitWindowDiagnostics(t, window); !diags.HasError() {
			t.Errorf("a window of %v is below the API's minimum of 1 and should be rejected", window)
		}
	}
}

func TestRateLimitWindowRejectsAWindowLongerThanTheMaximum(t *testing.T) {
	if diags := rateLimitWindowDiagnostics(t, 1441); !diags.HasError() {
		t.Error("a window longer than 1440 minutes should be rejected")
	}
}

func TestPrefixRateLimitWindowAllowsAWindowShorterThanAMinute(t *testing.T) {
	attribute, ok := prefixRateLimitAttributes()["window_in_minutes"].(schema.Float64Attribute)
	if !ok {
		t.Fatal("window_in_minutes should be a float64 attribute")
	}

	resp := &validator.Float64Response{}
	for _, declared := range attribute.Validators {
		declared.ValidateFloat64(context.Background(), validator.Float64Request{
			Path:        path.Root("window_in_minutes"),
			ConfigValue: types.Float64Value(0.5),
		}, resp)
	}

	if resp.Diagnostics.HasError() {
		t.Errorf("a prefix window of 0.5 is above the API's exclusive minimum of 0 and should be accepted, got %v", resp.Diagnostics)
	}
}

func TestPasskeyRegistrationHintsArePlannedAsAnOrderedList(t *testing.T) {
	passkeySchema := resourceSchemaOf(t, NewPasskeyAuthenticatorConfigurationResource())

	attribute, ok := passkeySchema.Attributes["passkey_registration_hints"]
	if !ok {
		t.Fatal("passkey schema has no passkey_registration_hints attribute")
	}

	if _, isList := attribute.(schema.ListAttribute); !isList {
		t.Errorf("passkey_registration_hints is %T, want a list so its order is part of the plan", attribute)
	}
}

func pushCredentialLifetimeDiagnostics(t *testing.T, minutes int64) diag.Diagnostics {
	t.Helper()

	pushSchema := resourceSchemaOf(t, NewPushAuthenticatorConfigurationResource())

	attribute, ok := pushSchema.Attributes["credential_lifetime_in_minutes"].(schema.Int64Attribute)
	if !ok {
		t.Fatal("credential_lifetime_in_minutes should be an int64 attribute")
	}

	resp := &validator.Int64Response{}
	for _, declared := range attribute.Validators {
		declared.ValidateInt64(context.Background(), validator.Int64Request{
			Path:        path.Root("credential_lifetime_in_minutes"),
			ConfigValue: types.Int64Value(minutes),
		}, resp)
	}

	return resp.Diagnostics
}

func TestPushCredentialLifetimeAcceptsTheBoundsTheApiAllows(t *testing.T) {
	for _, minutes := range []int64{0, 1, 60, 525600} {
		if diags := pushCredentialLifetimeDiagnostics(t, minutes); diags.HasError() {
			t.Errorf("a lifetime of %d minutes should be accepted, got %v", minutes, diags)
		}
	}
}

func TestPushCredentialLifetimeRejectsANegativeLifetime(t *testing.T) {
	if diags := pushCredentialLifetimeDiagnostics(t, -1); !diags.HasError() {
		t.Error("a negative credential lifetime should be rejected")
	}
}

func TestPushCredentialLifetimeRejectsALifetimeBeyondAYear(t *testing.T) {
	if diags := pushCredentialLifetimeDiagnostics(t, 525601); !diags.HasError() {
		t.Error("a credential lifetime beyond 525600 minutes should be rejected")
	}
}

func TestCredentialBlocksAreOptionalAndNotComputed(t *testing.T) {
	schemas := authenticatorResourceSchemas(t)

	for prefix, blocks := range authenticatorCredentialBlocks() {
		resourceSchema := schemas["authsignal_"+prefix+"_authenticator_configuration"]

		for _, block := range blocks {
			attribute, ok := resourceSchema.Attributes[block.Name]
			if !ok {
				continue
			}

			if !attribute.IsOptional() {
				t.Errorf("%s: %q should be optional", prefix, block.Name)
			}

			if attribute.IsComputed() {
				t.Errorf("%s: %q is computed and holds write-only members, which Terraform rejects at schema validation", prefix, block.Name)
			}
		}
	}
}

func responseReadableOptionals() map[string][]string {
	return map[string][]string{
		"sms": {
			"is_active", "sms_country_codes", "default_country_code",
			"submission_rate_limit_configuration", "sending_rate_limit_configurations",
			"prefix_rate_limit_configurations", "allowed_custom_sms_variables",
		},
		"email_otp": {
			"is_active", "submission_rate_limit_configuration",
			"sending_rate_limit_configurations", "allowed_custom_email_variables",
		},
		"passkey": {
			"is_active", "passkey_registration_hints",
			"user_verification_requirement", "authenticator_attachment",
		},
		"push": {
			"is_active", "webhook_url", "credential_lifetime_in_minutes",
			"require_app_attestation", "app_attestation_failure_mode",
			"sending_rate_limit_configurations",
		},
	}
}

func TestResponseReadableOptionalAttributesAreAlsoComputed(t *testing.T) {
	schemas := authenticatorResourceSchemas(t)

	for method, names := range responseReadableOptionals() {
		resourceSchema := schemas["authsignal_"+method+"_authenticator_configuration"]

		for _, name := range names {
			attribute, ok := resourceSchema.Attributes[name]
			if !ok {
				t.Errorf("%s: the schema has no attribute %q", method, name)
				continue
			}

			if !attribute.IsOptional() {
				t.Errorf("%s: %q should stay optional; the practitioner has to be able to leave it out", method, name)
			}

			if !attribute.IsComputed() {
				t.Errorf("%s: %q is optional but not computed, so leaving it out plans a change against whatever "+
					"the API returns; an omitted attribute has to be unmanaged, not a perpetual diff", method, name)
			}
		}
	}
}
