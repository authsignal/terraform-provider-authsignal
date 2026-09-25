package provider

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func validateResourceConfig(t *testing.T, res resource.ResourceWithValidateConfig, resourceSchema schema.Schema, attributes map[string]tftypes.Value) []string {
	t.Helper()

	resp := &resource.ValidateConfigResponse{}
	res.ValidateConfig(context.Background(), resource.ValidateConfigRequest{
		Config: tfsdk.Config{
			Schema: resourceSchema,
			Raw:    resourceObject(t, resourceSchema, attributes),
		},
	}, resp)

	var summaries []string
	for _, diagnostic := range resp.Diagnostics.Errors() {
		summaries = append(summaries, diagnostic.Summary())
	}

	return summaries
}

func credentialTftypesObject(t *testing.T, resourceSchema schema.Schema, blockName string, members map[string]tftypes.Value) tftypes.Value {
	t.Helper()

	attribute, ok := resourceSchema.Attributes[blockName]
	if !ok {
		t.Fatalf("the schema has no %q attribute", blockName)
	}

	objectType, ok := attribute.GetType().TerraformType(context.Background()).(tftypes.Object)
	if !ok {
		t.Fatalf("%q is not an object", blockName)
	}

	values := make(map[string]tftypes.Value, len(objectType.AttributeTypes))
	for name, attributeType := range objectType.AttributeTypes {
		if value, has := members[name]; has {
			values[name] = value
			continue
		}

		values[name] = tftypes.NewValue(attributeType, nil)
	}

	return tftypes.NewValue(objectType, values)
}

func TestSmsValidateConfigRejectsCredentialsForAnotherProvider(t *testing.T) {
	res := &smsAuthenticatorConfigurationResource{}
	resourceSchema := resourceSchemaOf(t, res)

	summaries := validateResourceConfig(t, res, resourceSchema, map[string]tftypes.Value{
		"sms_provider": tftypes.NewValue(tftypes.String, "TWILIO"),
		"modica_group_credentials": credentialTftypesObject(t, resourceSchema, modicaGroupCredentialsBlock, map[string]tftypes.Value{
			"username": tftypes.NewValue(tftypes.String, "modica-user"),
			"password": tftypes.NewValue(tftypes.String, "modica-password"),
		}),
	})

	if len(summaries) != 1 || !strings.Contains(summaries[0], "Credentials Do Not Match") {
		t.Errorf("expected a mismatched-credentials error, got %v", summaries)
	}
}

func TestSmsValidateConfigAcceptsTheMatchingProvidersCredentials(t *testing.T) {
	res := &smsAuthenticatorConfigurationResource{}
	resourceSchema := resourceSchemaOf(t, res)

	summaries := validateResourceConfig(t, res, resourceSchema, map[string]tftypes.Value{
		"sms_provider": tftypes.NewValue(tftypes.String, "TWILIO"),
		"twilio_credentials": credentialTftypesObject(t, resourceSchema, twilioCredentialsBlock, map[string]tftypes.Value{
			"auth_token":            tftypes.NewValue(tftypes.String, "secret-token"),
			"messaging_service_sid": tftypes.NewValue(tftypes.String, "MG123"),
			"account_sid":           tftypes.NewValue(tftypes.String, "AC123"),
		}),
	})

	if len(summaries) != 0 {
		t.Errorf("expected no diagnostics, got %v", summaries)
	}
}

func TestEmailOtpValidateConfigRejectsCredentialsForAnotherProvider(t *testing.T) {
	res := &emailOtpAuthenticatorConfigurationResource{}
	resourceSchema := resourceSchemaOf(t, res)

	summaries := validateResourceConfig(t, res, resourceSchema, map[string]tftypes.Value{
		"email_provider": tftypes.NewValue(tftypes.String, "MANDRILL"),
		"sendgrid_email_credentials": credentialTftypesObject(t, resourceSchema, sendgridEmailCredentialsBlock, map[string]tftypes.Value{
			"api_key":     tftypes.NewValue(tftypes.String, "SG.key"),
			"template_id": tftypes.NewValue(tftypes.String, "d-123"),
			"from_email":  tftypes.NewValue(tftypes.String, "noreply@example.com"),
		}),
	})

	if len(summaries) != 1 || !strings.Contains(summaries[0], "Credentials Do Not Match") {
		t.Errorf("expected a mismatched-credentials error, got %v", summaries)
	}
}

func TestPushValidateConfigAllowsFirebaseCredentialsUnderDefault(t *testing.T) {
	res := &pushAuthenticatorConfigurationResource{}
	resourceSchema := resourceSchemaOf(t, res)

	summaries := validateResourceConfig(t, res, resourceSchema, map[string]tftypes.Value{
		"push_provider": tftypes.NewValue(tftypes.String, "DEFAULT"),
		"fcm_credentials": credentialTftypesObject(t, resourceSchema, fcmCredentialsBlock, map[string]tftypes.Value{
			"service_account_key": tftypes.NewValue(tftypes.String, "{}"),
		}),
		"fcm_credentials_version": tftypes.NewValue(tftypes.String, "1"),
	})

	if len(summaries) != 0 {
		t.Errorf("expected no diagnostics, got %v", summaries)
	}
}

func writeOnlyBlockSummaries(t *testing.T, version tftypes.Value) []string {
	t.Helper()

	res := &pushAuthenticatorConfigurationResource{}
	resourceSchema := resourceSchemaOf(t, res)

	return validateResourceConfig(t, res, resourceSchema, map[string]tftypes.Value{
		"push_provider": tftypes.NewValue(tftypes.String, "FIREBASE"),
		"fcm_credentials": credentialTftypesObject(t, resourceSchema, fcmCredentialsBlock, map[string]tftypes.Value{
			"service_account_key": tftypes.NewValue(tftypes.String, "{}"),
		}),
		"fcm_credentials_version": version,
	})
}

func TestPushValidateConfigRejectsAWriteOnlyBlockWithNoVersionMarker(t *testing.T) {
	summaries := writeOnlyBlockSummaries(t, tftypes.NewValue(tftypes.String, nil))

	if len(summaries) != 1 || !strings.Contains(summaries[0], "Missing Version Marker") {
		t.Errorf("expected a missing-version-marker error, got %v", summaries)
	}
}

func TestPushValidateConfigAcceptsAWriteOnlyBlockCarryingItsVersionMarker(t *testing.T) {
	if summaries := writeOnlyBlockSummaries(t, tftypes.NewValue(tftypes.String, "1")); len(summaries) != 0 {
		t.Errorf("a write-only block with its marker is valid, got %v", summaries)
	}
}

func TestSmsValidateConfigDoesNotRequireAVersionMarkerForABlockWithMetadata(t *testing.T) {
	res := &smsAuthenticatorConfigurationResource{}
	resourceSchema := resourceSchemaOf(t, res)

	summaries := validateResourceConfig(t, res, resourceSchema, map[string]tftypes.Value{
		"sms_provider": tftypes.NewValue(tftypes.String, "TWILIO"),
		"twilio_credentials": credentialTftypesObject(t, resourceSchema, twilioCredentialsBlock, map[string]tftypes.Value{
			"auth_token":            tftypes.NewValue(tftypes.String, "secret-token"),
			"messaging_service_sid": tftypes.NewValue(tftypes.String, "MG123"),
			"account_sid":           tftypes.NewValue(tftypes.String, "AC123"),
		}),
	})

	if len(summaries) != 0 {
		t.Errorf("twilio_credentials has readable metadata and needs no marker, got %v", summaries)
	}
}

func TestSmsValidateConfigRejectsTnzCredentialsWithNoVersionMarker(t *testing.T) {
	res := &smsAuthenticatorConfigurationResource{}
	resourceSchema := resourceSchemaOf(t, res)

	summaries := validateResourceConfig(t, res, resourceSchema, map[string]tftypes.Value{
		"sms_provider": tftypes.NewValue(tftypes.String, "TNZ"),
		"tnz_credentials": credentialTftypesObject(t, resourceSchema, tnzCredentialsBlock, map[string]tftypes.Value{
			"api_key": tftypes.NewValue(tftypes.String, "tnz-key"),
		}),
	})

	if len(summaries) != 1 || !strings.Contains(summaries[0], "Missing Version Marker") {
		t.Errorf("expected a missing-version-marker error, got %v", summaries)
	}
}

func TestPushValidateConfigRejectsApnsCredentialsUnderFirebase(t *testing.T) {
	res := &pushAuthenticatorConfigurationResource{}
	resourceSchema := resourceSchemaOf(t, res)

	summaries := validateResourceConfig(t, res, resourceSchema, map[string]tftypes.Value{
		"push_provider": tftypes.NewValue(tftypes.String, "FIREBASE"),
		"apns_credentials": credentialTftypesObject(t, resourceSchema, apnsCredentialsBlock, map[string]tftypes.Value{
			"team_id":     tftypes.NewValue(tftypes.String, "TEAM"),
			"key_id":      tftypes.NewValue(tftypes.String, "KEY"),
			"bundle_id":   tftypes.NewValue(tftypes.String, "com.example.app"),
			"private_key": tftypes.NewValue(tftypes.String, "-----BEGIN PRIVATE KEY-----"),
		}),
	})

	if len(summaries) != 1 || !strings.Contains(summaries[0], "Credentials Do Not Match") {
		t.Errorf("expected a mismatched-credentials error, got %v", summaries)
	}
}

func TestPushValidateConfigRequiresAWebhookUrlForTheWebhookProvider(t *testing.T) {
	res := &pushAuthenticatorConfigurationResource{}
	resourceSchema := resourceSchemaOf(t, res)

	summaries := validateResourceConfig(t, res, resourceSchema, map[string]tftypes.Value{
		"push_provider": tftypes.NewValue(tftypes.String, "WEBHOOK"),
	})

	if len(summaries) != 1 || !strings.Contains(summaries[0], "Missing Webhook URL") {
		t.Errorf("expected a missing-webhook-url error, got %v", summaries)
	}
}

func TestImportStateSlugAcceptsOnlyItsOwnSlug(t *testing.T) {
	cases := []struct {
		id       string
		slug     string
		accepted bool
	}{
		{"sms", smsAuthenticatorSlug, true},
		{"SMS", smsAuthenticatorSlug, true},
		{" sms ", smsAuthenticatorSlug, true},
		{"email-otp", emailOtpAuthenticatorSlug, true},
		{"email_otp", emailOtpAuthenticatorSlug, false},
		{"passkey", passkeyAuthenticatorSlug, true},
		{"push", pushAuthenticatorSlug, true},
		{"sms", pushAuthenticatorSlug, false},
		{"", smsAuthenticatorSlug, false},
	}

	for _, testCase := range cases {
		diags := importStateSlug(testCase.id, testCase.slug)

		if diags.HasError() == testCase.accepted {
			t.Errorf("importing %q as %q: accepted=%t, want %t", testCase.id, testCase.slug, !diags.HasError(), testCase.accepted)
		}

		if !testCase.accepted && diags.HasError() && !strings.Contains(diags.Errors()[0].Detail(), testCase.slug) {
			t.Errorf("the error for %q should name the expected slug %q, got %q", testCase.id, testCase.slug, diags.Errors()[0].Detail())
		}
	}
}

func TestCreateConflictDiagnosticNamesTheImportCommand(t *testing.T) {
	diagnostic := createConflictDiagnostic("authsignal_sms_authenticator_configuration", "sms", smsAuthenticatorSlug)

	if !strings.Contains(diagnostic.Detail(), "terraform import authsignal_sms_authenticator_configuration.sms sms") {
		t.Errorf("the conflict message should name the import command, got %q", diagnostic.Detail())
	}
}

func TestBlockSizeAcceptsCountryAndPositiveDigits(t *testing.T) {
	for _, accepted := range []string{"country", "1", "6"} {
		if _, diags := blockSizeFrom(accepted); diags.HasError() {
			t.Errorf("%q should be a valid block size, got %v", accepted, diags)
		}
	}

	for _, rejected := range []string{"", "0", "-1", "region", "1.5"} {
		if _, diags := blockSizeFrom(rejected); !diags.HasError() {
			t.Errorf("%q should be rejected as a block size", rejected)
		}
	}
}

func TestExpectedOriginAcceptsEveryFormTheApiAccepts(t *testing.T) {
	accepted := []string{
		"https://example.com",
		"https://app.example.com:8443",
		"http://localhost:3000",
		"android:apk-key-hash:abc123",
	}

	for _, origin := range accepted {
		if !expectedOriginPattern.MatchString(origin) {
			t.Errorf("%q should be a valid expected origin", origin)
		}
	}
}

func TestExpectedOriginRejectsAnOriginWithNoRecognisedScheme(t *testing.T) {
	for _, origin := range []string{"example.com", "ftp://example.com", ""} {
		if expectedOriginPattern.MatchString(origin) {
			t.Errorf("%q should be rejected as an expected origin", origin)
		}
	}
}

func TestPushValidateConfigRejectsAWebhookUrlForANonWebhookProvider(t *testing.T) {
	for _, provider := range []string{"DEFAULT", "FIREBASE"} {
		res := &pushAuthenticatorConfigurationResource{}
		resourceSchema := resourceSchemaOf(t, res)

		summaries := validateResourceConfig(t, res, resourceSchema, map[string]tftypes.Value{
			"push_provider": tftypes.NewValue(tftypes.String, provider),
			"webhook_url":   tftypes.NewValue(tftypes.String, "https://example.com/hook"),
		})

		found := false
		for _, summary := range summaries {
			if strings.Contains(summary, "Webhook URL Does Not Apply") {
				found = true
			}
		}

		if !found {
			t.Errorf("%s with a webhook_url should be refused, got %v", provider, summaries)
		}
	}
}

func TestPushValidateConfigAcceptsAWebhookUrlForTheWebhookProvider(t *testing.T) {
	res := &pushAuthenticatorConfigurationResource{}
	resourceSchema := resourceSchemaOf(t, res)

	summaries := validateResourceConfig(t, res, resourceSchema, map[string]tftypes.Value{
		"push_provider": tftypes.NewValue(tftypes.String, "WEBHOOK"),
		"webhook_url":   tftypes.NewValue(tftypes.String, "https://example.com/hook"),
	})

	if len(summaries) != 0 {
		t.Errorf("expected no diagnostics, got %v", summaries)
	}
}

func countryCodeSet(t *testing.T, codes ...string) tftypes.Value {
	t.Helper()

	values := make([]tftypes.Value, 0, len(codes))
	for _, code := range codes {
		values = append(values, tftypes.NewValue(tftypes.String, code))
	}

	return tftypes.NewValue(tftypes.Set{ElementType: tftypes.String}, values)
}

func smsCountryCodeConfig(t *testing.T, resourceSchema schema.Schema, countryCodes tftypes.Value, defaultCode tftypes.Value) map[string]tftypes.Value {
	t.Helper()

	return map[string]tftypes.Value{
		"sms_provider":         tftypes.NewValue(tftypes.String, "TWILIO"),
		"sms_country_codes":    countryCodes,
		"default_country_code": defaultCode,
		"twilio_credentials": credentialTftypesObject(t, resourceSchema, twilioCredentialsBlock, map[string]tftypes.Value{
			"auth_token":            tftypes.NewValue(tftypes.String, "secret-token"),
			"messaging_service_sid": tftypes.NewValue(tftypes.String, "MG123"),
			"account_sid":           tftypes.NewValue(tftypes.String, "AC123"),
		}),
	}
}

func smsCountryCodeSummaries(t *testing.T, countryCodes tftypes.Value, defaultCode tftypes.Value) []string {
	t.Helper()

	res := &smsAuthenticatorConfigurationResource{}
	resourceSchema := resourceSchemaOf(t, res)

	return validateResourceConfig(t, res, resourceSchema, smsCountryCodeConfig(t, resourceSchema, countryCodes, defaultCode))
}

func smsCountryCodeDetails(t *testing.T, countryCodes tftypes.Value, defaultCode tftypes.Value) []string {
	t.Helper()

	res := &smsAuthenticatorConfigurationResource{}
	resourceSchema := resourceSchemaOf(t, res)

	resp := &resource.ValidateConfigResponse{}
	res.ValidateConfig(context.Background(), resource.ValidateConfigRequest{
		Config: tfsdk.Config{
			Schema: resourceSchema,
			Raw:    resourceObject(t, resourceSchema, smsCountryCodeConfig(t, resourceSchema, countryCodes, defaultCode)),
		},
	}, resp)

	var details []string
	for _, diagnostic := range resp.Diagnostics.Errors() {
		details = append(details, diagnostic.Detail())
	}

	return details
}

func TestSmsValidateConfigRejectsADefaultCountryCodeAlongsideAClearedRestriction(t *testing.T) {
	summaries := smsCountryCodeSummaries(t, countryCodeSet(t), tftypes.NewValue(tftypes.String, "NZ"))

	if len(summaries) != 1 || !strings.Contains(summaries[0], "Default Country Code Conflicts") {
		t.Errorf("expected a cleared-restriction error, got %v", summaries)
	}
}

func TestSmsValidateConfigAcceptsAClearedRestrictionWithNoDefault(t *testing.T) {
	summaries := smsCountryCodeSummaries(t, countryCodeSet(t), tftypes.NewValue(tftypes.String, nil))

	if len(summaries) != 0 {
		t.Errorf("clearing the restriction on its own is how it is lifted, got %v", summaries)
	}
}

func TestSmsValidateConfigAcceptsADefaultCountryCodeWithANonEmptyRestriction(t *testing.T) {
	summaries := smsCountryCodeSummaries(t, countryCodeSet(t, "AU", "NZ"), tftypes.NewValue(tftypes.String, "NZ"))

	if len(summaries) != 0 {
		t.Errorf("a default inside a non-empty restriction is valid, got %v", summaries)
	}
}

func TestSmsValidateConfigAcceptsADefaultCountryCodeWhenTheRestrictionIsOmitted(t *testing.T) {
	summaries := smsCountryCodeSummaries(t,
		tftypes.NewValue(tftypes.Set{ElementType: tftypes.String}, nil),
		tftypes.NewValue(tftypes.String, "NZ"),
	)

	if len(summaries) != 0 {
		t.Errorf("omitting the restriction leaves it unmanaged and says nothing about the default, got %v", summaries)
	}
}

func TestSmsValidateConfigDefersOnAnUnknownCountryRestriction(t *testing.T) {
	summaries := smsCountryCodeSummaries(t,
		tftypes.NewValue(tftypes.Set{ElementType: tftypes.String}, tftypes.UnknownValue),
		tftypes.NewValue(tftypes.String, "NZ"),
	)

	if len(summaries) != 0 {
		t.Errorf("an unknown restriction cannot be judged yet, got %v", summaries)
	}
}

func TestSmsValidateConfigRejectsADefaultCountryCodeTheRestrictionDoesNotAllow(t *testing.T) {
	summaries := smsCountryCodeSummaries(t, countryCodeSet(t, "AU", "GB"), tftypes.NewValue(tftypes.String, "NZ"))

	if len(summaries) != 1 || !strings.Contains(summaries[0], "Default Country Code Is Not an Allowed Country") {
		t.Errorf("expected a default-outside-restriction error, got %v", summaries)
	}
}

func TestSmsDefaultCountryCodeDiagnosticNamesTheAllowedCodes(t *testing.T) {
	details := smsCountryCodeDetails(t, countryCodeSet(t, "AU", "GB"), tftypes.NewValue(tftypes.String, "NZ"))
	if len(details) != 1 {
		t.Fatalf("expected one diagnostic, got %v", details)
	}

	for _, expected := range []string{`"AU"`, `"GB"`, `"NZ"`} {
		if !strings.Contains(details[0], expected) {
			t.Errorf("the diagnostic should name %s, got %s", expected, details[0])
		}
	}
}

func TestSmsValidateConfigAcceptsADefaultCountryCodeTheRestrictionAllows(t *testing.T) {
	if summaries := smsCountryCodeSummaries(t, countryCodeSet(t, "AU", "NZ"), tftypes.NewValue(tftypes.String, "NZ")); len(summaries) != 0 {
		t.Errorf("a default listed in the restriction is valid, got %v", summaries)
	}
}

func TestSmsValidateConfigDefersOnAnUnknownDefaultCountryCode(t *testing.T) {
	summaries := smsCountryCodeSummaries(t, countryCodeSet(t, "AU", "GB"), tftypes.NewValue(tftypes.String, tftypes.UnknownValue))

	if len(summaries) != 0 {
		t.Errorf("an unknown default cannot be judged yet, got %v", summaries)
	}
}

func summariesOf(diagnostics diag.Diagnostics) []string {
	var summaries []string
	for _, diagnostic := range diagnostics.Errors() {
		summaries = append(summaries, diagnostic.Summary())
	}

	return summaries
}

func float64ValidatorErrors(t *testing.T, v validator.Float64, value types.Float64) []string {
	t.Helper()

	resp := &validator.Float64Response{}
	v.ValidateFloat64(context.Background(), validator.Float64Request{
		Path:        path.Root("window_in_minutes"),
		ConfigValue: value,
	}, resp)

	return summariesOf(resp.Diagnostics)
}

func stringValidatorErrors(t *testing.T, v validator.String, value types.String) []string {
	t.Helper()

	resp := &validator.StringResponse{}
	v.ValidateString(context.Background(), validator.StringRequest{
		Path:        path.Root("value"),
		ConfigValue: value,
	}, resp)

	return summariesOf(resp.Diagnostics)
}

func TestGreaterThanZeroValidatorAcceptsAFractionalWindow(t *testing.T) {
	if errors := float64ValidatorErrors(t, greaterThanZeroValidator{}, types.Float64Value(0.5)); len(errors) != 0 {
		t.Errorf("a window of 0.5 was refused: %v", errors)
	}
}

func TestGreaterThanZeroValidatorRejectsZeroAndBelow(t *testing.T) {
	for _, value := range []float64{0, -1} {
		errors := float64ValidatorErrors(t, greaterThanZeroValidator{}, types.Float64Value(value))
		if len(errors) == 0 {
			t.Errorf("a window of %v was accepted; the API's exclusive minimum is 0", value)
		}
	}
}

func TestGreaterThanZeroValidatorDefersOnUnknownAndNull(t *testing.T) {
	for name, value := range map[string]types.Float64{"unknown": types.Float64Unknown(), "null": types.Float64Null()} {
		if errors := float64ValidatorErrors(t, greaterThanZeroValidator{}, value); len(errors) != 0 {
			t.Errorf("a %s window produced %v; there is nothing to judge yet", name, errors)
		}
	}
}

func TestBlockSizeValidatorAcceptsCountryAndPositiveDigits(t *testing.T) {
	for _, value := range []string{"country", "1", "7"} {
		if errors := stringValidatorErrors(t, blockSizeValidator{}, types.StringValue(value)); len(errors) != 0 {
			t.Errorf("block_size %q was refused: %v", value, errors)
		}
	}
}

func TestBlockSizeValidatorRejectsAnythingElse(t *testing.T) {
	for _, value := range []string{"", "0", "-3", "COUNTRY", "three", "1.5"} {
		if errors := stringValidatorErrors(t, blockSizeValidator{}, types.StringValue(value)); len(errors) == 0 {
			t.Errorf("block_size %q was accepted; it is neither \"country\" nor a positive number of digits", value)
		}
	}
}

func TestBlockSizeValidatorDefersOnUnknownAndNull(t *testing.T) {
	for name, value := range map[string]types.String{"unknown": types.StringUnknown(), "null": types.StringNull()} {
		if errors := stringValidatorErrors(t, blockSizeValidator{}, value); len(errors) != 0 {
			t.Errorf("a %s block_size produced %v; there is nothing to judge yet", name, errors)
		}
	}
}

func TestExpectedOriginValidatorAcceptsEveryFormTheApiTakes(t *testing.T) {
	for _, value := range []string{"https://example.com", "http://localhost:3000", "android:apk-key-hash:abc"} {
		if errors := stringValidatorErrors(t, expectedOriginValidator{}, types.StringValue(value)); len(errors) != 0 {
			t.Errorf("origin %q was refused: %v; the provider must not narrow the API's pattern", value, errors)
		}
	}
}

func TestExpectedOriginValidatorRejectsOtherSchemes(t *testing.T) {
	for _, value := range []string{"example.com", "ftp://example.com", "ios:example", ""} {
		if errors := stringValidatorErrors(t, expectedOriginValidator{}, types.StringValue(value)); len(errors) == 0 {
			t.Errorf("origin %q was accepted; the API's pattern does not allow it", value)
		}
	}
}

func TestExpectedOriginValidatorDefersOnUnknownAndNull(t *testing.T) {
	for name, value := range map[string]types.String{"unknown": types.StringUnknown(), "null": types.StringNull()} {
		if errors := stringValidatorErrors(t, expectedOriginValidator{}, value); len(errors) != 0 {
			t.Errorf("a %s origin produced %v; there is nothing to judge yet", name, errors)
		}
	}
}

func TestHttpsUrlValidatorAcceptsOnlyHttps(t *testing.T) {
	if errors := stringValidatorErrors(t, httpsUrlValidator{}, types.StringValue("https://example.com/hook")); len(errors) != 0 {
		t.Errorf("an https endpoint was refused: %v", errors)
	}

	for _, value := range []string{"http://example.com/hook", "example.com/hook", ""} {
		if errors := stringValidatorErrors(t, httpsUrlValidator{}, types.StringValue(value)); len(errors) == 0 {
			t.Errorf("endpoint %q was accepted; a webhook endpoint has to be https", value)
		}
	}
}

func TestHttpsUrlValidatorDefersOnUnknownAndNull(t *testing.T) {
	for name, value := range map[string]types.String{"unknown": types.StringUnknown(), "null": types.StringNull()} {
		if errors := stringValidatorErrors(t, httpsUrlValidator{}, value); len(errors) != 0 {
			t.Errorf("a %s endpoint produced %v; there is nothing to judge yet", name, errors)
		}
	}
}

func TestSmsValidateConfigRequiresAWebhookUrlForTheWebhookProvider(t *testing.T) {
	res := &smsAuthenticatorConfigurationResource{}
	resourceSchema := resourceSchemaOf(t, res)

	summaries := validateResourceConfig(t, res, resourceSchema, map[string]tftypes.Value{
		"sms_provider": tftypes.NewValue(tftypes.String, "WEBHOOK"),
	})

	if len(summaries) != 1 || !strings.Contains(summaries[0], "Missing Webhook URL") {
		t.Errorf("expected a missing-webhook-url error, got %v", summaries)
	}
}

func TestEmailOtpValidateConfigRequiresAWebhookUrlForTheWebhookProvider(t *testing.T) {
	res := &emailOtpAuthenticatorConfigurationResource{}
	resourceSchema := resourceSchemaOf(t, res)

	summaries := validateResourceConfig(t, res, resourceSchema, map[string]tftypes.Value{
		"email_provider": tftypes.NewValue(tftypes.String, "WEBHOOK"),
	})

	if len(summaries) != 1 || !strings.Contains(summaries[0], "Missing Webhook URL") {
		t.Errorf("expected a missing-webhook-url error, got %v", summaries)
	}
}

func TestValidateConfigAcceptsAnUnknownWebhookUrlForTheWebhookProvider(t *testing.T) {
	cases := map[string]struct {
		res               resource.ResourceWithValidateConfig
		providerAttribute string
	}{
		"email-otp": {&emailOtpAuthenticatorConfigurationResource{}, "email_provider"},
		"sms":       {&smsAuthenticatorConfigurationResource{}, "sms_provider"},
		"push":      {&pushAuthenticatorConfigurationResource{}, "push_provider"},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			resourceSchema := resourceSchemaOf(t, tc.res)

			summaries := validateResourceConfig(t, tc.res, resourceSchema, map[string]tftypes.Value{
				tc.providerAttribute: tftypes.NewValue(tftypes.String, "WEBHOOK"),
				"webhook_url":        tftypes.NewValue(tftypes.String, tftypes.UnknownValue),
			})

			if len(summaries) != 0 {
				t.Errorf("an unknown webhook_url produced %v; there is nothing to judge yet", summaries)
			}
		})
	}
}

func TestSmsValidateConfigRejectsAWebhookUrlForACredentialProvider(t *testing.T) {
	for _, provider := range []string{"BIRD", "MESSAGE_MEDIA", "MODICA_GROUP", "TNZ", "TWILIO"} {
		res := &smsAuthenticatorConfigurationResource{}
		resourceSchema := resourceSchemaOf(t, res)

		summaries := validateResourceConfig(t, res, resourceSchema, map[string]tftypes.Value{
			"sms_provider": tftypes.NewValue(tftypes.String, provider),
			"webhook_url":  tftypes.NewValue(tftypes.String, "https://example.com/sms"),
		})

		found := false
		for _, summary := range summaries {
			if strings.Contains(summary, "Webhook URL Does Not Apply") {
				found = true
			}
		}

		if !found {
			t.Errorf("%s: expected a webhook-url-does-not-apply error, got %v", provider, summaries)
		}
	}
}

func TestEmailOtpValidateConfigRejectsAWebhookUrlForACredentialProvider(t *testing.T) {
	res := &emailOtpAuthenticatorConfigurationResource{}
	resourceSchema := resourceSchemaOf(t, res)

	summaries := validateResourceConfig(t, res, resourceSchema, map[string]tftypes.Value{
		"email_provider": tftypes.NewValue(tftypes.String, "SENDGRID"),
		"webhook_url":    tftypes.NewValue(tftypes.String, "https://example.com/email"),
	})

	found := false
	for _, summary := range summaries {
		if strings.Contains(summary, "Webhook URL Does Not Apply") {
			found = true
		}
	}

	if !found {
		t.Errorf("expected a webhook-url-does-not-apply error, got %v", summaries)
	}
}

func TestSmsValidateConfigAcceptsTheWebhookProviderWithItsEndpoint(t *testing.T) {
	res := &smsAuthenticatorConfigurationResource{}
	resourceSchema := resourceSchemaOf(t, res)

	summaries := validateResourceConfig(t, res, resourceSchema, map[string]tftypes.Value{
		"sms_provider": tftypes.NewValue(tftypes.String, "WEBHOOK"),
		"webhook_url":  tftypes.NewValue(tftypes.String, "https://example.com/sms"),
	})

	if len(summaries) != 0 {
		t.Errorf("expected no diagnostics, got %v", summaries)
	}
}

func TestIpv4CidrValidatorAcceptsRangesAndRejectsAnythingElse(t *testing.T) {
	cases := []struct {
		value    string
		accepted bool
	}{
		{"203.0.113.0/24", true},
		{"198.51.100.7/32", true},
		{"0.0.0.0/0", true},
		{"203.0.113.1", false},
		{"203.0.113.0/33", false},
		{"256.0.113.0/24", false},
		{"not-an-address", false},
		{"2001:db8::/32", false},
	}

	for _, testCase := range cases {
		response := &validator.StringResponse{}

		ipv4CidrValidator{}.ValidateString(context.Background(), validator.StringRequest{
			Path:        path.Root("ip_whitelist"),
			ConfigValue: types.StringValue(testCase.value),
		}, response)

		if response.Diagnostics.HasError() == testCase.accepted {
			t.Errorf("%q: accepted=%t, got diagnostics %v", testCase.value, testCase.accepted, response.Diagnostics)
		}
	}
}
