package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"strings"
	"testing"

	"github.com/authsignal/authsignal-management-go/v6"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func credentialObjectOf(t *testing.T, block credentialBlock, members map[string]attr.Value) types.Object {
	t.Helper()

	values := map[string]attr.Value{}

	for _, member := range block.Members {
		if value, ok := members[member.Name]; ok {
			values[member.Name] = value
			continue
		}

		values[member.Name] = nullValueFor(credentialMemberType(member.Kind))
	}

	return types.ObjectValueMust(block.attributeTypes(), values)
}

func marshalBody(t *testing.T, body any) map[string]any {
	t.Helper()

	encoded, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshalling the request body: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("decoding the request body: %v", err)
	}

	return decoded
}

func rateLimitListOf(t *testing.T, entries ...[2]float64) types.List {
	t.Helper()

	elements := make([]attr.Value, 0, len(entries))
	for _, entry := range entries {
		elements = append(elements, types.ObjectValueMust(rateLimitAttributeTypes, map[string]attr.Value{
			"rate_limit":        types.Int64Value(int64(entry[0])),
			"window_in_minutes": types.Float64Value(entry[1]),
		}))
	}

	return types.ListValueMust(rateLimitObjectType(), elements)
}

func smsModelFor(provider string) smsAuthenticatorConfigurationResourceModel {
	model := emptySmsModel()
	model.SmsProvider = types.StringValue(provider)

	return model
}

func twilioConfigured(t *testing.T, model *smsAuthenticatorConfigurationResourceModel) {
	t.Helper()

	model.TwilioCredentials = credentialObjectOf(t, twilioBlock, map[string]attr.Value{
		"auth_token":            types.StringValue("secret-token"),
		"messaging_service_sid": types.StringValue("MG123"),
		"account_sid":           types.StringValue("AC123"),
	})
}

func TestSmsCreateOmitsWhatTheConfigurationDoesNotSet(t *testing.T) {
	plan := smsModelFor("TWILIO")

	config := plan
	twilioConfigured(t, &config)

	body, diags := smsCreateBody(context.Background(), plan, config)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	encoded := marshalBody(t, body)

	if encoded["smsProvider"] != "TWILIO" {
		t.Errorf("smsProvider is %v, want TWILIO", encoded["smsProvider"])
	}

	for _, absent := range []string{"isActive", "smsCountryCodes", "defaultCountryCode", "submissionRateLimitConfiguration", "sendingRateLimitConfigurations", "prefixRateLimitConfigurations", "allowedCustomSmsVariables"} {
		if _, ok := encoded[absent]; ok {
			t.Errorf("create body carries %q, which the configuration does not set", absent)
		}
	}

	credentials, ok := encoded["twilioCredentials"].(map[string]any)
	if !ok {
		t.Fatal("create body has no twilioCredentials")
	}

	if credentials["authToken"] != "secret-token" {
		t.Errorf("authToken is %v, want the configured secret", credentials["authToken"])
	}
}

func TestSmsCreateRequiresTheMatchingCredentialBlock(t *testing.T) {
	plan := smsModelFor("TWILIO")

	_, diags := smsCreateBody(context.Background(), plan, plan)
	if !diags.HasError() {
		t.Fatal("expected a diagnostic for the missing twilio_credentials block")
	}
}

func TestSmsCreateRejectsAnIncompleteCredentialBlock(t *testing.T) {
	plan := smsModelFor("TWILIO")

	config := plan
	config.TwilioCredentials = credentialObjectOf(t, twilioBlock, map[string]attr.Value{
		"messaging_service_sid": types.StringValue("MG123"),
		"account_sid":           types.StringValue("AC123"),
	})

	_, diags := smsCreateBody(context.Background(), plan, config)
	if !diags.HasError() {
		t.Fatal("expected a diagnostic for the missing auth_token")
	}
}

func smsUpdateFixture(t *testing.T) (smsAuthenticatorConfigurationResourceModel, smsAuthenticatorConfigurationResourceModel) {
	t.Helper()

	state := smsModelFor("TWILIO")
	state.DefaultCountryCode = types.StringValue("NZ")
	state.SendingRateLimitConfigurations = rateLimitListOf(t, [2]float64{3, 60})
	twilioConfigured(t, &state)

	plan := smsModelFor("TWILIO")
	plan.SubmissionRateLimitConfiguration = types.ObjectValueMust(rateLimitAttributeTypes, map[string]attr.Value{
		"rate_limit":        types.Int64Value(5),
		"window_in_minutes": types.Float64Value(1.5),
	})
	twilioConfigured(t, &plan)

	return plan, state
}

func smsUpdatePatch(t *testing.T) map[string]any {
	t.Helper()

	plan, state := smsUpdateFixture(t)

	body, diags := smsUpdateBody(context.Background(), plan, plan, state)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	return marshalBody(t, body)
}

func TestSmsUpdateSendsTheValuesTheConfigurationSets(t *testing.T) {
	submission, ok := smsUpdatePatch(t)["submissionRateLimitConfiguration"].(map[string]any)
	if !ok {
		t.Fatal("submissionRateLimitConfiguration should be sent as a value")
	}

	if submission["windowInMinutes"] != 1.5 {
		t.Errorf("windowInMinutes is %v, want 1.5 — the window is a decimal", submission["windowInMinutes"])
	}
}

func TestSmsUpdateOmitsTheFieldsTheConfigurationStoppedSetting(t *testing.T) {
	encoded := smsUpdatePatch(t)

	for _, omitted := range []string{"defaultCountryCode", "sendingRateLimitConfigurations"} {
		if value, ok := encoded[omitted]; ok {
			t.Errorf("%q is present as %v; omitting it from the configuration has to leave the stored value alone", omitted, value)
		}
	}
}

func TestSmsUpdateOmitsTheProviderWhenItHasNotChanged(t *testing.T) {
	if _, ok := smsUpdatePatch(t)["smsProvider"]; ok {
		t.Error("smsProvider should be omitted when it has not changed")
	}
}

func smsCollectionPatch(t *testing.T, countryCodes types.Set, variables types.Set) map[string]any {
	t.Helper()

	state := smsModelFor("TWILIO")
	state.SmsCountryCodes = stringSetValue([]string{"AU", "NZ"})
	state.AllowedCustomSmsVariables = stringSetValue([]string{"firstName"})
	twilioConfigured(t, &state)

	config := smsModelFor("TWILIO")
	config.SmsCountryCodes = countryCodes
	config.AllowedCustomSmsVariables = variables
	twilioConfigured(t, &config)

	body, diags := smsUpdateBody(context.Background(), config, config, state)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	return marshalBody(t, body)
}

func TestSmsUpdateOmitsACollectionTheConfigurationStoppedSetting(t *testing.T) {
	encoded := smsCollectionPatch(t, types.SetNull(types.StringType), types.SetNull(types.StringType))

	for _, omitted := range []string{"smsCountryCodes", "allowedCustomSmsVariables"} {
		if value, ok := encoded[omitted]; ok {
			t.Errorf("%q is present as %v; omitting a collection must not empty it", omitted, value)
		}
	}
}

func TestSmsUpdateSendsAnEmptyListForACollectionSetToEmpty(t *testing.T) {
	encoded := smsCollectionPatch(t, stringSetValue(nil), stringSetValue(nil))

	for _, emptied := range []string{"smsCountryCodes", "allowedCustomSmsVariables"} {
		value, ok := encoded[emptied]
		if !ok {
			t.Errorf("%q should be sent so an explicit [] actually empties it", emptied)
			continue
		}

		list, isList := value.([]any)
		if !isList {
			t.Errorf("%q is %v, want an empty list — the API does not accept a null here", emptied, value)
			continue
		}

		if len(list) != 0 {
			t.Errorf("%q is %v, want an empty list", emptied, list)
		}
	}
}

func TestSmsProviderSwitchOmitsTheSupersededBlockEntirely(t *testing.T) {
	state := smsModelFor("TWILIO")
	twilioConfigured(t, &state)

	plan := smsModelFor("MODICA_GROUP")
	plan.ModicaGroupCredentials = credentialObjectOf(t, modicaGroupBlock, map[string]attr.Value{
		"username": types.StringValue("modica-user"),
		"password": types.StringValue("modica-password"),
	})

	body, diags := smsUpdateBody(context.Background(), plan, plan, state)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	encoded := marshalBody(t, body)

	if encoded["smsProvider"] != "MODICA_GROUP" {
		t.Errorf("smsProvider is %v, want MODICA_GROUP", encoded["smsProvider"])
	}

	if _, ok := encoded["twilioCredentials"]; ok {
		t.Errorf("the switch patch carries twilioCredentials (%v); the superseded block has to be absent, not null",
			encoded["twilioCredentials"])
	}

	credentials, ok := encoded["modicaGroupCredentials"].(map[string]any)
	if !ok {
		t.Fatal("the switch patch should carry the new provider's credentials")
	}

	if credentials["password"] != "modica-password" {
		t.Errorf("password is %v; a switch has to send the new provider's secrets", credentials["password"])
	}
}

func TestSmsUpdateOmitsADroppedCredentialBlockRatherThanClearingIt(t *testing.T) {
	state := smsModelFor("TWILIO")
	twilioConfigured(t, &state)
	state.ModicaGroupCredentials = credentialObjectOf(t, modicaGroupBlock, map[string]attr.Value{
		"username": types.StringValue("modica-user"),
	})

	config := smsModelFor("TWILIO")
	twilioConfigured(t, &config)

	body, diags := smsUpdateBody(context.Background(), config, config, state)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	encoded := marshalBody(t, body)

	if value, ok := encoded["modicaGroupCredentials"]; ok {
		t.Errorf("modicaGroupCredentials is present as %v; a dropped block has to be absent so the stored credentials survive", value)
	}

	if _, ok := encoded["smsProvider"]; ok {
		t.Error("the patch must not name the provider when the provider has not changed")
	}
}

func TestSmsSecretsAreOnlySentWhenTheVersionChanges(t *testing.T) {
	state := smsModelFor("TWILIO")
	state.TwilioCredentialsVersion = types.StringValue("v1")
	state.TwilioCredentials = credentialObjectOf(t, twilioBlock, map[string]attr.Value{
		"auth_token":            types.StringNull(),
		"messaging_service_sid": types.StringValue("MG123"),
		"account_sid":           types.StringValue("AC123"),
	})

	plan := smsModelFor("TWILIO")
	plan.TwilioCredentialsVersion = types.StringValue("v1")
	plan.TwilioCredentials = credentialObjectOf(t, twilioBlock, map[string]attr.Value{
		"auth_token":            types.StringValue("secret-token"),
		"messaging_service_sid": types.StringValue("MG999"),
		"account_sid":           types.StringValue("AC123"),
	})

	body, diags := smsUpdateBody(context.Background(), plan, plan, state)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	credentials, ok := marshalBody(t, body)["twilioCredentials"].(map[string]any)
	if !ok {
		t.Fatal("the changed metadata should still be sent")
	}

	if credentials["messagingServiceSid"] != "MG999" {
		t.Errorf("messagingServiceSid is %v, want the changed value", credentials["messagingServiceSid"])
	}

	if _, ok := credentials["authToken"]; ok {
		t.Error("authToken was resent although the version did not change")
	}

	rotated := plan
	rotated.TwilioCredentialsVersion = types.StringValue("v2")

	rotatedBody, diags := smsUpdateBody(context.Background(), rotated, rotated, state)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	rotatedCredentials, ok := marshalBody(t, rotatedBody)["twilioCredentials"].(map[string]any)
	if !ok {
		t.Fatal("the rotated block should be sent")
	}

	if rotatedCredentials["authToken"] != "secret-token" {
		t.Errorf("authToken is %v; bumping the version has to resend the secret", rotatedCredentials["authToken"])
	}
}

func TestSmsUpdateSendsNothingWhenNothingChanged(t *testing.T) {
	state := smsModelFor("TWILIO")
	state.TwilioCredentialsVersion = types.StringValue("v1")
	state.TwilioCredentials = credentialObjectOf(t, twilioBlock, map[string]attr.Value{
		"messaging_service_sid": types.StringValue("MG123"),
		"account_sid":           types.StringValue("AC123"),
	})

	plan := state
	plan.TwilioCredentials = credentialObjectOf(t, twilioBlock, map[string]attr.Value{
		"auth_token":            types.StringValue("secret-token"),
		"messaging_service_sid": types.StringValue("MG123"),
		"account_sid":           types.StringValue("AC123"),
	})

	body, diags := smsUpdateBody(context.Background(), plan, plan, state)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	encoded := marshalBody(t, body)
	if len(encoded) != 0 {
		t.Errorf("patch should be empty, got %v", encoded)
	}
}

func passkeyStored() passkeyAuthenticatorConfigurationResourceModel {
	return passkeyAuthenticatorConfigurationResourceModel{
		RelyingParty:                types.StringValue("example.com"),
		ExpectedOrigins:             stringSetValue([]string{"https://example.com"}),
		PasskeyRegistrationHints:    stringListValue([]string{"client-device"}),
		UserVerificationRequirement: types.StringValue("required"),
		AuthenticatorAttachment:     types.StringValue("platform"),
		IsActive:                    types.BoolNull(),
	}
}

func passkeyUpdatePatch(t *testing.T, config passkeyAuthenticatorConfigurationResourceModel) map[string]any {
	t.Helper()

	body, diags := passkeyUpdateBody(context.Background(), config, config, passkeyStored())
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	return marshalBody(t, body)
}

func passkeyDropping() passkeyAuthenticatorConfigurationResourceModel {
	config := passkeyStored()
	config.PasskeyRegistrationHints = types.ListNull(types.StringType)
	config.UserVerificationRequirement = types.StringNull()
	config.AuthenticatorAttachment = types.StringNull()

	return config
}

func TestPasskeyUpdateSendsAnEmptyListWhenTheHintsAreSetToEmpty(t *testing.T) {
	config := passkeyStored()
	config.PasskeyRegistrationHints = stringListValue(nil)

	encoded := passkeyUpdatePatch(t, config)

	hints, ok := encoded["passkeyRegistrationHints"].([]any)
	if !ok || len(hints) != 0 {
		t.Errorf("passkeyRegistrationHints is %v, want an empty list — an explicit [] is what removes the hints", encoded["passkeyRegistrationHints"])
	}
}

func TestPasskeyUpdateOmitsTheHintsTheConfigurationStoppedSetting(t *testing.T) {
	encoded := passkeyUpdatePatch(t, passkeyDropping())

	if value, ok := encoded["passkeyRegistrationHints"]; ok {
		t.Errorf("passkeyRegistrationHints is present as %v; dropping it from the configuration must leave the stored hints alone", value)
	}
}

func TestPasskeyUpdateOmitsTheNullableEnumsTheConfigurationStoppedSetting(t *testing.T) {
	encoded := passkeyUpdatePatch(t, passkeyDropping())

	for _, omitted := range []string{"userVerificationRequirement", "authenticatorAttachment"} {
		if value, ok := encoded[omitted]; ok {
			t.Errorf("%q is present as %v; omitting a nullable field has to preserve it", omitted, value)
		}
	}
}

func TestPasskeyUpdateOmitsTheFieldsThatDidNotChange(t *testing.T) {
	encoded := passkeyUpdatePatch(t, passkeyDropping())

	for _, unchanged := range []string{"relyingParty", "expectedOrigins"} {
		if _, ok := encoded[unchanged]; ok {
			t.Errorf("%q should be omitted when it has not changed", unchanged)
		}
	}
}

func TestPasskeyUpdateSendsAnExplicitlyChangedEnum(t *testing.T) {
	config := passkeyStored()
	config.UserVerificationRequirement = types.StringValue("preferred")

	if encoded := passkeyUpdatePatch(t, config); encoded["userVerificationRequirement"] != "preferred" {
		t.Errorf("userVerificationRequirement is %v, want the configured value", encoded["userVerificationRequirement"])
	}
}

func pushModelFor(provider string) pushAuthenticatorConfigurationResourceModel {
	model := emptyPushModel()
	model.PushProvider = types.StringValue(provider)

	return model
}

func TestPushSwitchToFirebaseOmitsTheApnsBlock(t *testing.T) {
	apnsBlock := apnsBlock
	fcmBlock := fcmBlock

	state := pushAuthenticatorConfigurationResourceModel{
		PushProvider:                   types.StringValue("DEFAULT"),
		IsActive:                       types.BoolNull(),
		WebhookUrl:                     types.StringNull(),
		CredentialLifetimeInMinutes:    types.Int64Null(),
		RequireAppAttestation:          types.BoolNull(),
		AppAttestationFailureMode:      types.StringNull(),
		SendingRateLimitConfigurations: types.ListNull(rateLimitObjectType()),
		ApnsCredentials: credentialObjectOf(t, apnsBlock, map[string]attr.Value{
			"team_id":   types.StringValue("TEAM"),
			"key_id":    types.StringValue("KEY"),
			"bundle_id": types.StringValue("com.example.app"),
		}),
		ApnsCredentialsVersion: types.StringValue("v1"),
		FcmCredentials:         types.ObjectNull(fcmBlock.attributeTypes()),
		FcmCredentialsVersion:  types.StringNull(),
	}

	plan := state
	plan.PushProvider = types.StringValue("FIREBASE")
	plan.ApnsCredentials = types.ObjectNull(apnsBlock.attributeTypes())
	plan.ApnsCredentialsVersion = types.StringNull()
	plan.FcmCredentials = credentialObjectOf(t, fcmBlock, map[string]attr.Value{
		"service_account_key": types.StringValue("{\"type\":\"service_account\"}"),
	})
	plan.FcmCredentialsVersion = types.StringValue("v1")

	body, diags := pushUpdateBody(context.Background(), plan, plan, state)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	encoded := marshalBody(t, body)

	if encoded["pushProvider"] != "FIREBASE" {
		t.Errorf("pushProvider is %v, want FIREBASE", encoded["pushProvider"])
	}

	if _, ok := encoded["apnsCredentials"]; ok {
		t.Errorf("the switch patch carries apnsCredentials (%v); the superseded block has to be absent, not null",
			encoded["apnsCredentials"])
	}

	credentials, ok := encoded["fcmCredentials"].(map[string]any)
	if !ok {
		t.Fatal("the switch patch should carry the Firebase credentials")
	}

	if credentials["serviceAccountKey"] == nil {
		t.Error("a switch has to send the new provider's secret")
	}
}

func TestPushCreateRequiresAtLeastOneCredentialBlockForDefault(t *testing.T) {
	if diags := pushCredentialsSufficient("DEFAULT", false, false); !diags.HasError() {
		t.Error("DEFAULT with neither APNs nor Firebase credentials should be refused")
	}

	if diags := pushCredentialsSufficient("DEFAULT", false, true); diags.HasError() {
		t.Errorf("DEFAULT with Firebase credentials alone should be accepted, got %v", diags)
	}

	if diags := pushCredentialsSufficient("FIREBASE", true, false); !diags.HasError() {
		t.Error("FIREBASE with only APNs credentials should be refused")
	}

	if diags := pushCredentialsSufficient("WEBHOOK", false, false); diags.HasError() {
		t.Errorf("WEBHOOK needs no credentials, got %v", diags)
	}
}

func TestPushIntegerAndDecimalFieldsKeepTheirTypes(t *testing.T) {
	state := pushAuthenticatorConfigurationResourceModel{
		PushProvider:                   types.StringValue("WEBHOOK"),
		IsActive:                       types.BoolNull(),
		WebhookUrl:                     types.StringValue("https://example.com/hook"),
		CredentialLifetimeInMinutes:    types.Int64Null(),
		RequireAppAttestation:          types.BoolNull(),
		AppAttestationFailureMode:      types.StringNull(),
		SendingRateLimitConfigurations: types.ListNull(rateLimitObjectType()),
		ApnsCredentials:                types.ObjectNull(apnsBlock.attributeTypes()),
		ApnsCredentialsVersion:         types.StringNull(),
		FcmCredentials:                 types.ObjectNull(fcmBlock.attributeTypes()),
		FcmCredentialsVersion:          types.StringNull(),
	}

	plan := state
	plan.CredentialLifetimeInMinutes = types.Int64Value(30)
	plan.SendingRateLimitConfigurations = rateLimitListOf(t, [2]float64{2, 1.5})

	body, diags := pushUpdateBody(context.Background(), plan, plan, state)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	encoded := marshalBody(t, body)

	if encoded["credentialLifetimeInMinutes"] != float64(30) {
		t.Errorf("credentialLifetimeInMinutes is %v, want 30", encoded["credentialLifetimeInMinutes"])
	}

	limits, ok := encoded["sendingRateLimitConfigurations"].([]any)
	if !ok || len(limits) != 1 {
		t.Fatalf("sendingRateLimitConfigurations is %v, want one entry", encoded["sendingRateLimitConfigurations"])
	}

	first := limits[0].(map[string]any)
	if first["windowInMinutes"] != 1.5 {
		t.Errorf("windowInMinutes is %v, want 1.5 — a window may be fractional", first["windowInMinutes"])
	}
}

func TestFullyWriteOnlyBlocksStayOutOfState(t *testing.T) {
	provider := "TNZ"
	response := &authsignal.SmsAuthenticatorConfiguration{
		AuthenticatorId:    "auth_123",
		IsActive:           true,
		VerificationMethod: "SMS",
		SmsProvider:        &provider,
		TnzCredentials:     &authsignal.TnzCredentialsResponse{},
	}

	state := smsStateFromResponse(response, emptySmsModel())

	if !state.TnzCredentials.IsNull() {
		t.Errorf("tnz_credentials is %v, want null — the block is write-only", state.TnzCredentials)
	}

	if state.SmsProvider.ValueString() != "TNZ" {
		t.Errorf("sms_provider is %v, want TNZ", state.SmsProvider)
	}
}

func randomId(prefix string) string {
	return fmt.Sprintf("%s-%d", prefix, rand.Uint64())
}

func emailOtpModelFor(provider string) emailOtpAuthenticatorConfigurationResourceModel {
	model := emptyEmailOtpModel()
	model.EmailProvider = types.StringValue(provider)

	return model
}

func sendgridCredentials(t *testing.T, apiKey string, templateId string) types.Object {
	t.Helper()

	return credentialObjectOf(t, sendgridEmailBlock, map[string]attr.Value{
		"api_key":     types.StringValue(apiKey),
		"template_id": types.StringValue(templateId),
		"from_email":  types.StringValue("noreply@example.com"),
	})
}

func mandrillCredentials(t *testing.T, apiKey string, templateName string) types.Object {
	t.Helper()

	return credentialObjectOf(t, mandrillEmailBlock, map[string]attr.Value{
		"api_key":       types.StringValue(apiKey),
		"template_name": types.StringValue(templateName),
	})
}

func TestEmailOtpCreateSendsOnlyTheSelectedProvidersCredentials(t *testing.T) {
	plan := emailOtpModelFor("SENDGRID")

	config := plan
	config.SendgridEmailCredentials = sendgridCredentials(t, randomId("SG"), randomId("template"))

	body, diags := emailOtpCreateBody(context.Background(), plan, config)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	encoded := marshalBody(t, body)

	if _, ok := encoded["sendgridEmailCredentials"]; !ok {
		t.Error("the create body should carry the selected provider's credentials")
	}

	for _, other := range []string{"birdEmailCredentials", "mailjetEmailCredentials", "mailgunEmailCredentials", "mandrillEmailCredentials", "smtpEmailCredentials"} {
		if _, ok := encoded[other]; ok {
			t.Errorf("the create body carries %q, which does not belong to the chosen provider", other)
		}
	}
}

func TestEmailOtpCreateOmitsOptionalFieldsTheConfigurationLeavesOut(t *testing.T) {
	plan := emailOtpModelFor("MANDRILL")

	config := plan
	config.MandrillEmailCredentials = mandrillCredentials(t, randomId("md"), randomId("template"))

	body, diags := emailOtpCreateBody(context.Background(), plan, config)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	encoded := marshalBody(t, body)

	for _, absent := range []string{"isActive", "submissionRateLimitConfiguration", "sendingRateLimitConfigurations", "allowedCustomEmailVariables"} {
		if _, ok := encoded[absent]; ok {
			t.Errorf("create body carries %q, which the configuration does not set", absent)
		}
	}
}

func TestEmailOtpCreateRequiresTheSelectedProvidersCredentials(t *testing.T) {
	plan := emailOtpModelFor("SENDGRID")

	_, diags := emailOtpCreateBody(context.Background(), plan, plan)
	if !diags.HasError() {
		t.Fatal("expected a diagnostic for the missing sendgrid_email_credentials block")
	}
}

func TestEmailOtpUpdateSendsTheValuesTheConfigurationSets(t *testing.T) {
	apiKey := randomId("SG")
	templateId := randomId("template")

	state := emailOtpModelFor("SENDGRID")
	state.SendgridEmailCredentials = sendgridCredentials(t, apiKey, templateId)

	plan := state
	plan.SubmissionRateLimitConfiguration = types.ObjectValueMust(rateLimitAttributeTypes, map[string]attr.Value{
		"rate_limit":        types.Int64Value(4),
		"window_in_minutes": types.Float64Value(2.5),
	})

	body, diags := emailOtpUpdateBody(context.Background(), plan, plan, state)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	submission, ok := marshalBody(t, body)["submissionRateLimitConfiguration"].(map[string]any)
	if !ok {
		t.Fatal("submissionRateLimitConfiguration should be sent as a value")
	}

	if submission["windowInMinutes"] != 2.5 {
		t.Errorf("windowInMinutes is %v, want 2.5 — the window is a decimal", submission["windowInMinutes"])
	}
}

func TestEmailOtpUpdateOmitsTheRateLimitsTheConfigurationStoppedSetting(t *testing.T) {
	state := emailOtpModelFor("SENDGRID")
	state.SendgridEmailCredentials = sendgridCredentials(t, randomId("SG"), randomId("template"))
	state.SubmissionRateLimitConfiguration = types.ObjectValueMust(rateLimitAttributeTypes, map[string]attr.Value{
		"rate_limit":        types.Int64Value(4),
		"window_in_minutes": types.Float64Value(1),
	})
	state.SendingRateLimitConfigurations = rateLimitListOf(t, [2]float64{5, 60})

	config := state
	config.SubmissionRateLimitConfiguration = types.ObjectNull(rateLimitAttributeTypes)
	config.SendingRateLimitConfigurations = types.ListNull(rateLimitObjectType())

	body, diags := emailOtpUpdateBody(context.Background(), config, config, state)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	encoded := marshalBody(t, body)

	for _, omitted := range []string{"submissionRateLimitConfiguration", "sendingRateLimitConfigurations"} {
		if value, ok := encoded[omitted]; ok {
			t.Errorf("%q is present as %v; omitting it has to leave the stored limit alone", omitted, value)
		}
	}
}

func emailOtpVariablesPatch(t *testing.T, variables types.Set) map[string]any {
	t.Helper()

	state := emailOtpModelFor("SENDGRID")
	state.SendgridEmailCredentials = sendgridCredentials(t, randomId("SG"), randomId("template"))
	state.AllowedCustomEmailVariables = stringSetValue([]string{"firstName", "lastName"})

	config := state
	config.AllowedCustomEmailVariables = variables

	body, diags := emailOtpUpdateBody(context.Background(), config, config, state)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	return marshalBody(t, body)
}

func TestEmailOtpUpdateOmitsTheCustomVariablesTheConfigurationStoppedSetting(t *testing.T) {
	if value, ok := emailOtpVariablesPatch(t, types.SetNull(types.StringType))["allowedCustomEmailVariables"]; ok {
		t.Errorf("allowedCustomEmailVariables is present as %v; omitting it must not empty it", value)
	}
}

func TestEmailOtpUpdateSendsAnEmptyListWhenTheCustomVariablesAreSetToEmpty(t *testing.T) {
	value, ok := emailOtpVariablesPatch(t, stringSetValue(nil))["allowedCustomEmailVariables"]
	if !ok {
		t.Fatal("allowedCustomEmailVariables should be sent so an explicit [] actually empties it")
	}

	list, isList := value.([]any)
	if !isList || len(list) != 0 {
		t.Errorf("allowedCustomEmailVariables is %v, want an empty list — the API does not accept a null here", value)
	}
}

func TestEmailOtpUpdateOmitsEverythingWhenNothingChanged(t *testing.T) {
	apiKey := randomId("SG")
	templateId := randomId("template")

	state := emailOtpModelFor("SENDGRID")
	state.SendgridEmailCredentialsVersion = types.StringValue("1")
	state.SendgridEmailCredentials = credentialObjectOf(t, sendgridEmailBlock, map[string]attr.Value{
		"template_id": types.StringValue(templateId),
		"from_email":  types.StringValue("noreply@example.com"),
	})

	plan := state
	plan.SendgridEmailCredentials = sendgridCredentials(t, apiKey, templateId)

	body, diags := emailOtpUpdateBody(context.Background(), plan, plan, state)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if encoded := marshalBody(t, body); len(encoded) != 0 {
		t.Errorf("patch should be empty, got %v", encoded)
	}
}

func TestEmailOtpWithholdsSecretsWhileTheVersionIsUnchanged(t *testing.T) {
	apiKey := randomId("SG")

	state := emailOtpModelFor("SENDGRID")
	state.SendgridEmailCredentialsVersion = types.StringValue("1")
	state.SendgridEmailCredentials = credentialObjectOf(t, sendgridEmailBlock, map[string]attr.Value{
		"template_id": types.StringValue(randomId("template")),
		"from_email":  types.StringValue("noreply@example.com"),
	})

	plan := state
	plan.SendgridEmailCredentials = sendgridCredentials(t, apiKey, randomId("template"))

	body, diags := emailOtpUpdateBody(context.Background(), plan, plan, state)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	credentials, ok := marshalBody(t, body)["sendgridEmailCredentials"].(map[string]any)
	if !ok {
		t.Fatal("the changed metadata should still be sent")
	}

	if _, ok := credentials["apiKey"]; ok {
		t.Error("the API key was resent although the version did not change")
	}
}

func TestEmailOtpResendsSecretsWhenTheVersionChanges(t *testing.T) {
	apiKey := randomId("SG")
	templateId := randomId("template")

	state := emailOtpModelFor("SENDGRID")
	state.SendgridEmailCredentialsVersion = types.StringValue("1")
	state.SendgridEmailCredentials = credentialObjectOf(t, sendgridEmailBlock, map[string]attr.Value{
		"template_id": types.StringValue(templateId),
		"from_email":  types.StringValue("noreply@example.com"),
	})

	plan := state
	plan.SendgridEmailCredentialsVersion = types.StringValue("2")
	plan.SendgridEmailCredentials = sendgridCredentials(t, apiKey, templateId)

	body, diags := emailOtpUpdateBody(context.Background(), plan, plan, state)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	credentials, ok := marshalBody(t, body)["sendgridEmailCredentials"].(map[string]any)
	if !ok {
		t.Fatal("the rotated block should be sent")
	}

	if credentials["apiKey"] != apiKey {
		t.Errorf("apiKey is %v; bumping the version has to resend the secret", credentials["apiKey"])
	}
}

func TestEmailOtpProviderSwitchCarriesTheNewProvidersBlock(t *testing.T) {
	mandrillKey := randomId("md")

	state := emailOtpModelFor("SENDGRID")
	state.SendgridEmailCredentials = sendgridCredentials(t, randomId("SG"), randomId("template"))

	plan := emailOtpModelFor("MANDRILL")
	plan.MandrillEmailCredentials = mandrillCredentials(t, mandrillKey, randomId("template"))

	body, diags := emailOtpUpdateBody(context.Background(), plan, plan, state)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	encoded := marshalBody(t, body)

	if encoded["emailProvider"] != "MANDRILL" {
		t.Errorf("emailProvider is %v, want MANDRILL", encoded["emailProvider"])
	}

	credentials, ok := encoded["mandrillEmailCredentials"].(map[string]any)
	if !ok {
		t.Fatal("the switch patch should carry the new provider's credentials")
	}

	if credentials["apiKey"] != mandrillKey {
		t.Errorf("apiKey is %v; a switch has to send the new provider's secret", credentials["apiKey"])
	}
}

func TestEmailOtpProviderSwitchSendsNoKeyForTheSupersededBlock(t *testing.T) {
	state := emailOtpModelFor("SENDGRID")
	state.SendgridEmailCredentials = sendgridCredentials(t, randomId("SG"), randomId("template"))

	plan := emailOtpModelFor("MANDRILL")
	plan.MandrillEmailCredentials = mandrillCredentials(t, randomId("md"), randomId("template"))

	body, diags := emailOtpUpdateBody(context.Background(), plan, plan, state)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	encoded := marshalBody(t, body)

	if value, ok := encoded["sendgridEmailCredentials"]; ok {
		t.Errorf("the switch patch carries sendgridEmailCredentials (%v); the superseded block has to be absent, not null", value)
	}
}

func TestEmailOtpProviderSwitchDoesNotLeakTheSupersededProvidersSecret(t *testing.T) {
	supersededKey := randomId("SG-superseded")

	state := emailOtpModelFor("SENDGRID")
	state.SendgridEmailCredentials = sendgridCredentials(t, supersededKey, randomId("template"))

	plan := emailOtpModelFor("MANDRILL")
	plan.MandrillEmailCredentials = mandrillCredentials(t, randomId("md"), randomId("template"))

	body, diags := emailOtpUpdateBody(context.Background(), plan, plan, state)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	encoded, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshalling the request body: %v", err)
	}

	if strings.Contains(string(encoded), supersededKey) {
		t.Errorf("the superseded provider's secret appears in the switch patch: %s", encoded)
	}
}

func TestEmailOtpMailjetTemplateIdIsSentAsAnInteger(t *testing.T) {
	plan := emailOtpModelFor("MAILJET")

	config := plan
	config.MailjetEmailCredentials = credentialObjectOf(t, mailjetEmailBlock, map[string]attr.Value{
		"private_key": types.StringValue(randomId("mj-private")),
		"public_key":  types.StringValue(randomId("mj-public")),
		"template_id": types.Int64Value(4815162342),
	})

	body, diags := emailOtpCreateBody(context.Background(), plan, config)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	credentials, ok := marshalBody(t, body)["mailjetEmailCredentials"].(map[string]any)
	if !ok {
		t.Fatal("the create body should carry the Mailjet credentials")
	}

	if credentials["templateId"] != float64(4815162342) {
		t.Errorf("templateId is %v, want the configured integer", credentials["templateId"])
	}
}

func TestEmailOtpSmtpCredentialsKeepTheirMemberTypes(t *testing.T) {
	plan := emailOtpModelFor("SMTP")

	config := plan
	config.SmtpEmailCredentials = credentialObjectOf(t, smtpEmailBlock, map[string]attr.Value{
		"host":     types.StringValue("smtp.example.com"),
		"port":     types.Int64Value(587),
		"secure":   types.BoolValue(true),
		"user":     types.StringValue(randomId("user")),
		"password": types.StringValue(randomId("password")),
		"from":     types.StringValue("noreply@example.com"),
	})

	body, diags := emailOtpCreateBody(context.Background(), plan, config)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	credentials, ok := marshalBody(t, body)["smtpEmailCredentials"].(map[string]any)
	if !ok {
		t.Fatal("the create body should carry the SMTP credentials")
	}

	if credentials["port"] != float64(587) {
		t.Errorf("port is %v, want the configured integer", credentials["port"])
	}

	if credentials["secure"] != true {
		t.Errorf("secure is %v, want the configured boolean", credentials["secure"])
	}
}

func TestSmsProviderSwitchDoesNotLeakTheSupersededProvidersSecret(t *testing.T) {
	state := smsModelFor("TWILIO")
	twilioConfigured(t, &state)

	plan := smsModelFor("MODICA_GROUP")
	plan.ModicaGroupCredentials = credentialObjectOf(t, modicaGroupBlock, map[string]attr.Value{
		"username": types.StringValue(randomId("modica-user")),
		"password": types.StringValue(randomId("modica-password")),
	})

	body, diags := smsUpdateBody(context.Background(), plan, plan, state)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	encoded, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshalling the request body: %v", err)
	}

	if strings.Contains(string(encoded), "secret-token") {
		t.Errorf("the superseded provider's secret appears in the switch patch: %s", encoded)
	}
}

func TestPushSwitchAwayFromWebhookSendsNoKeyForTheEndpoint(t *testing.T) {
	state := pushModelFor("WEBHOOK")
	state.WebhookUrl = types.StringValue("https://example.com/hook")

	plan := pushModelFor("FIREBASE")
	plan.FcmCredentials = credentialObjectOf(t, fcmBlock, map[string]attr.Value{
		"service_account_key": types.StringValue(randomId("fcm")),
	})
	plan.FcmCredentialsVersion = types.StringValue("1")

	body, diags := pushUpdateBody(context.Background(), plan, plan, state)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	encoded := marshalBody(t, body)

	if value, present := encoded["webhookUrl"]; present {
		t.Errorf("the switch patch carries webhookUrl (%v); it has to be absent, not null", value)
	}

	if encoded["pushProvider"] != "FIREBASE" {
		t.Errorf("pushProvider is %v, want FIREBASE", encoded["pushProvider"])
	}
}

func TestPushUpdateOmitsTheEndpointTheConfigurationStoppedSetting(t *testing.T) {
	state := pushModelFor("WEBHOOK")
	state.WebhookUrl = types.StringValue("https://example.com/hook")

	config := state
	config.WebhookUrl = types.StringNull()

	body, diags := pushUpdateBody(context.Background(), config, config, state)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	encoded := marshalBody(t, body)

	if value, present := encoded["webhookUrl"]; present {
		t.Errorf("webhookUrl is present as %v; omitting it has to leave the stored endpoint alone", value)
	}

	if _, named := encoded["pushProvider"]; named {
		t.Error("the patch must not name the provider when the provider has not changed")
	}
}

func TestPushUpdateSendsAnExplicitlyChangedEndpoint(t *testing.T) {
	state := pushModelFor("WEBHOOK")
	state.WebhookUrl = types.StringValue("https://example.com/hook")

	config := state
	config.WebhookUrl = types.StringValue("https://example.com/moved")

	body, diags := pushUpdateBody(context.Background(), config, config, state)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if encoded := marshalBody(t, body); encoded["webhookUrl"] != "https://example.com/moved" {
		t.Errorf("webhookUrl is %v, want the configured value", encoded["webhookUrl"])
	}
}

func TestPasskeyRegistrationHintsKeepTheirConfiguredOrder(t *testing.T) {
	configured := []string{"security-key", "hybrid", "client-device"}

	state := passkeyAuthenticatorConfigurationResourceModel{
		RelyingParty:             types.StringValue("example.com"),
		ExpectedOrigins:          stringSetValue([]string{"https://example.com"}),
		PasskeyRegistrationHints: types.ListNull(types.StringType),
		IsActive:                 types.BoolNull(),
	}

	plan := state
	plan.PasskeyRegistrationHints = stringListValue(configured)

	body, diags := passkeyUpdateBody(context.Background(), plan, plan, state)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	sent, ok := marshalBody(t, body)["passkeyRegistrationHints"].([]any)
	if !ok {
		t.Fatal("the hints should be sent")
	}

	for i, want := range configured {
		if sent[i] != want {
			t.Fatalf("hints were sent as %v, want %v — the order carries meaning", sent, configured)
		}
	}

	response := &authsignal.PasskeyAuthenticatorConfiguration{
		AuthenticatorId:          "auth_passkey",
		IsActive:                 true,
		VerificationMethod:       "PASSKEY",
		PasskeyRegistrationHints: &configured,
	}

	for i, element := range passkeyStateFromResponse(response).PasskeyRegistrationHints.Elements() {
		stored, isString := element.(types.String)
		if !isString || stored.ValueString() != configured[i] {
			t.Fatalf("hints came back as %v, want %v", passkeyStateFromResponse(response).PasskeyRegistrationHints, configured)
		}
	}
}

func tnzPatch(t *testing.T, priorVersion types.String, plannedVersion types.String, provider string) map[string]any {
	t.Helper()

	state := smsModelFor(provider)
	state.TnzCredentialsVersion = priorVersion

	config := smsModelFor("TNZ")
	config.TnzCredentialsVersion = plannedVersion
	config.TnzCredentials = credentialObjectOf(t, tnzBlock, map[string]attr.Value{
		"api_key": types.StringValue("tnz-secret"),
	})

	body, diags := smsUpdateBody(context.Background(), config, config, state)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	return marshalBody(t, body)
}

func tnzApiKeyIn(t *testing.T, encoded map[string]any) (string, bool) {
	t.Helper()

	credentials, ok := encoded["tnzCredentials"].(map[string]any)
	if !ok {
		return "", false
	}

	key, ok := credentials["apiKey"].(string)

	return key, ok
}

func TestSmsWriteOnlyBlockIsNotResentWhenItsMarkerIsUnchanged(t *testing.T) {
	encoded := tnzPatch(t, types.StringValue("1"), types.StringValue("1"), "TNZ")

	if _, present := encoded["tnzCredentials"]; present {
		t.Errorf("tnzCredentials is present as %v; an unchanged marker means the stored secret is already correct", encoded["tnzCredentials"])
	}
}

func TestSmsWriteOnlyBlockIsResentWhenItsMarkerChanges(t *testing.T) {
	encoded := tnzPatch(t, types.StringValue("1"), types.StringValue("2"), "TNZ")

	key, ok := tnzApiKeyIn(t, encoded)
	if !ok {
		t.Fatalf("tnzCredentials should carry the secret when the marker changes, got %v", encoded)
	}

	if key != "tnz-secret" {
		t.Errorf("apiKey is %q, want the configured secret", key)
	}

	if _, named := encoded["smsProvider"]; named {
		t.Error("rotating in place must not name the provider")
	}
}

func TestSmsWriteOnlyBlockIsSentWhenStateHasNoMarkerYet(t *testing.T) {
	encoded := tnzPatch(t, types.StringNull(), types.StringValue("1"), "TNZ")

	if _, ok := tnzApiKeyIn(t, encoded); !ok {
		t.Errorf("tnzCredentials should carry the secret when state has no marker, got %v", encoded)
	}
}

func TestSmsWriteOnlyBlockIsSentOnAProviderSwitchWithAnUnchangedMarker(t *testing.T) {
	encoded := tnzPatch(t, types.StringValue("1"), types.StringValue("1"), "TWILIO")

	if _, ok := tnzApiKeyIn(t, encoded); !ok {
		t.Errorf("a switch to TNZ has to carry the secret, got %v", encoded)
	}

	if encoded["smsProvider"] != "TNZ" {
		t.Errorf("smsProvider is %v, want TNZ", encoded["smsProvider"])
	}
}

func TestSmsWriteOnlyBlockSendsNoEmptyObjectWhenWithheld(t *testing.T) {
	encoded := tnzPatch(t, types.StringValue("1"), types.StringValue("1"), "TNZ")

	if len(encoded) != 0 {
		t.Errorf("the patch should be empty, got %v", encoded)
	}
}

func TestCredentialVersionMarkersAreNeverSerialised(t *testing.T) {
	patches := map[string]map[string]any{
		"tnz rotation":    tnzPatch(t, types.StringValue("1"), types.StringValue("2"), "TNZ"),
		"tnz first write": tnzPatch(t, types.StringNull(), types.StringValue("1"), "TNZ"),
		"tnz switch":      tnzPatch(t, types.StringValue("1"), types.StringValue("1"), "TWILIO"),
	}

	for name, encoded := range patches {
		body := fmt.Sprintf("%v", encoded)
		for _, spelling := range []string{"version", "Version", "revision", "Revision"} {
			if strings.Contains(body, spelling) {
				t.Errorf("%s: the body contains %q; the marker is Terraform-only and must never reach the API: %s", name, spelling, body)
			}
		}
	}
}

func memberOf(t *testing.T, object types.Object, name string) attr.Value {
	t.Helper()

	value, ok := object.Attributes()[name]
	if !ok {
		t.Fatalf("the block has no %q member", name)
	}

	return value
}

func TestReadLeavesAnUntrackedOptionalCredentialMemberNull(t *testing.T) {
	prior := credentialObjectOf(t, apnsBlock, map[string]attr.Value{
		"team_id":   types.StringValue("TEAM"),
		"key_id":    types.StringValue("KEY"),
		"bundle_id": types.StringValue("com.example.app"),
	})

	refreshed := credentialReadState(apnsBlock, map[string]attr.Value{
		"team_id":   types.StringValue("TEAM"),
		"key_id":    types.StringValue("KEY"),
		"bundle_id": types.StringValue("com.example.app"),
		"sandbox":   types.BoolValue(false),
	}, true, prior)

	if !memberOf(t, refreshed, "sandbox").IsNull() {
		t.Errorf("sandbox is %v after a read; an optional member state never tracked must stay null, or the API default becomes a permanent diff", memberOf(t, refreshed, "sandbox"))
	}
}

func TestReadRefreshesATrackedOptionalCredentialMember(t *testing.T) {
	prior := credentialObjectOf(t, apnsBlock, map[string]attr.Value{
		"team_id":   types.StringValue("TEAM"),
		"key_id":    types.StringValue("KEY"),
		"bundle_id": types.StringValue("com.example.app"),
		"sandbox":   types.BoolValue(true),
	})

	refreshed := credentialReadState(apnsBlock, map[string]attr.Value{
		"team_id":   types.StringValue("TEAM"),
		"key_id":    types.StringValue("KEY"),
		"bundle_id": types.StringValue("com.example.app"),
		"sandbox":   types.BoolValue(false),
	}, true, prior)

	if !memberOf(t, refreshed, "sandbox").Equal(types.BoolValue(false)) {
		t.Errorf("sandbox is %v after a read; a tracked member has to refresh so drift shows up", memberOf(t, refreshed, "sandbox"))
	}
}

func TestReadRefreshesATrackedOptionalCredentialMemberStoredAsFalse(t *testing.T) {
	prior := credentialObjectOf(t, apnsBlock, map[string]attr.Value{
		"team_id":   types.StringValue("TEAM"),
		"key_id":    types.StringValue("KEY"),
		"bundle_id": types.StringValue("com.example.app"),
		"sandbox":   types.BoolValue(false),
	})

	refreshed := credentialReadState(apnsBlock, map[string]attr.Value{
		"team_id":   types.StringValue("TEAM"),
		"key_id":    types.StringValue("KEY"),
		"bundle_id": types.StringValue("com.example.app"),
		"sandbox":   types.BoolValue(true),
	}, true, prior)

	if !memberOf(t, refreshed, "sandbox").Equal(types.BoolValue(true)) {
		t.Errorf("sandbox is %v after a read; false is a tracked value, not an absent one, so it has to refresh", memberOf(t, refreshed, "sandbox"))
	}
}

func TestReadLeavesAnUntrackedOptionalStringCredentialMemberNull(t *testing.T) {
	prior := credentialObjectOf(t, messageMediaBlock, map[string]attr.Value{})

	refreshed := credentialReadState(messageMediaBlock, map[string]attr.Value{
		"source_number": types.StringValue("+6400000000"),
	}, true, prior)

	if !memberOf(t, refreshed, "source_number").IsNull() {
		t.Errorf("source_number is %v after a read; an optional string state never tracked must stay null", memberOf(t, refreshed, "source_number"))
	}
}

func TestReadRefreshesATrackedOptionalStringCredentialMember(t *testing.T) {
	prior := credentialObjectOf(t, messageMediaBlock, map[string]attr.Value{
		"source_number": types.StringValue("+6400000000"),
	})

	refreshed := credentialReadState(messageMediaBlock, map[string]attr.Value{
		"source_number": types.StringValue("+6499999999"),
	}, true, prior)

	if !memberOf(t, refreshed, "source_number").Equal(types.StringValue("+6499999999")) {
		t.Errorf("source_number is %v after a read; a tracked member has to refresh", memberOf(t, refreshed, "source_number"))
	}
}

func TestReadKeepsSecretCredentialMembersNullEvenWhenTheResponseCarriesThem(t *testing.T) {
	prior := credentialObjectOf(t, messageMediaBlock, map[string]attr.Value{
		"api_key":       types.StringValue("stored-key"),
		"api_secret":    types.StringValue("stored-secret"),
		"source_number": types.StringValue("+6400000000"),
	})

	refreshed := credentialReadState(messageMediaBlock, map[string]attr.Value{
		"api_key":       types.StringValue("leaked-key"),
		"api_secret":    types.StringValue("leaked-secret"),
		"source_number": types.StringValue("+6400000000"),
	}, true, prior)

	for _, secret := range []string{"api_key", "api_secret"} {
		if !memberOf(t, refreshed, secret).IsNull() {
			t.Errorf("%s is %v after a read; a write-only secret must never reach state, whatever the response carries", secret, memberOf(t, refreshed, secret))
		}
	}
}

func TestResolvedValueKeepsAKnownPlanOverTheResponse(t *testing.T) {
	resolved := resolvedValue(types.StringValue("planned"), types.StringValue("from-api"))

	if resolved.ValueString() != "planned" {
		t.Errorf("resolvedValue returned %q; a known plan is what the applied state has to match, or Terraform calls the result inconsistent", resolved.ValueString())
	}
}

func TestResolvedValueKeepsAKnownNullPlanOverTheResponse(t *testing.T) {
	resolved := resolvedValue(types.StringNull(), types.StringValue("from-api"))

	if !resolved.IsNull() {
		t.Errorf("resolvedValue returned %v; a plan that is known to be null is still a known plan", resolved)
	}
}

func TestResolvedValueTakesTheResponseWhenThePlanIsUnknown(t *testing.T) {
	resolved := resolvedValue(types.StringUnknown(), types.StringValue("from-api"))

	if resolved.ValueString() != "from-api" {
		t.Errorf("resolvedValue returned %q; only the response can fill an unknown plan", resolved.ValueString())
	}
}
