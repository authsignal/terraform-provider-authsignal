package provider

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/authsignal/authsignal-management-go/v6"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func authenticatorEventsWebhook(url attr.Value, includePublicKey attr.Value) types.Object {
	return types.ObjectValueMust(authenticatorEventsWebhookConfigAttributeTypes, map[string]attr.Value{
		"url":                           url,
		"include_credential_public_key": includePublicKey,
	})
}

func requestBodyOf(t *testing.T, model tenantSettingsResourceModel) map[string]any {
	t.Helper()

	settings, diags := tenantSettingsRequestFrom(context.Background(), model)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	encoded, err := json.Marshal(settings)
	if err != nil {
		t.Fatalf("failed to marshal settings: %v", err)
	}

	var body map[string]any
	if err := json.Unmarshal(encoded, &body); err != nil {
		t.Fatalf("failed to unmarshal settings: %v", err)
	}

	return body
}

func TestTenantSettingsOmitsUnmanagedSettings(t *testing.T) {
	body := requestBodyOf(t, tenantSettingsResourceModel{
		TokenDurationInMinutes:           types.Int64Null(),
		AuthenticatorEventsWebhookConfig: types.ObjectNull(authenticatorEventsWebhookConfigAttributeTypes),
		LogEventsWebhookConfig:           types.ObjectNull(logEventsWebhookConfigAttributeTypes),
		IpWhitelist:                      types.SetNull(types.StringType),
	})

	if len(body) != 0 {
		t.Errorf("expected an empty request body, got %v", body)
	}
}

func TestTenantSettingsSendsAConfiguredWebhook(t *testing.T) {
	body := requestBodyOf(t, tenantSettingsResourceModel{
		TokenDurationInMinutes: types.Int64Null(),
		AuthenticatorEventsWebhookConfig: authenticatorEventsWebhook(
			types.StringValue("https://example.com/events"),
			types.BoolValue(true),
		),
		LogEventsWebhookConfig: types.ObjectValueMust(logEventsWebhookConfigAttributeTypes, map[string]attr.Value{
			"endpoint_url": types.StringValue("https://example.com/logs"),
		}),
		IpWhitelist: types.SetNull(types.StringType),
	})

	events, ok := body["authenticatorEventsWebhookConfig"].(map[string]any)
	if !ok || events["url"] != "https://example.com/events" || events["includeCredentialPublicKey"] != true {
		t.Errorf("expected the authenticator events webhook to be sent, got %v", body)
	}

	logs, ok := body["logEventsWebhookConfig"].(map[string]any)
	if !ok || logs["endpointUrl"] != "https://example.com/logs" {
		t.Errorf("expected the log events webhook to be sent, got %v", body)
	}
}

func TestTenantSettingsSendsAFalsePublicKeyFlag(t *testing.T) {
	body := requestBodyOf(t, tenantSettingsResourceModel{
		TokenDurationInMinutes: types.Int64Null(),
		AuthenticatorEventsWebhookConfig: authenticatorEventsWebhook(
			types.StringValue("https://example.com/events"),
			types.BoolValue(false),
		),
		LogEventsWebhookConfig: types.ObjectNull(logEventsWebhookConfigAttributeTypes),
		IpWhitelist:            types.SetNull(types.StringType),
	})

	events := body["authenticatorEventsWebhookConfig"].(map[string]any)

	if events["includeCredentialPublicKey"] != false {
		t.Errorf("expected includeCredentialPublicKey to be sent as false, got %v", events)
	}
}

func TestTenantSettingsOmitsAnUnknownNestedValue(t *testing.T) {
	body := requestBodyOf(t, tenantSettingsResourceModel{
		TokenDurationInMinutes: types.Int64Null(),
		AuthenticatorEventsWebhookConfig: authenticatorEventsWebhook(
			types.StringValue("https://example.com/events"),
			types.BoolUnknown(),
		),
		LogEventsWebhookConfig: types.ObjectNull(logEventsWebhookConfigAttributeTypes),
		IpWhitelist:            types.SetNull(types.StringType),
	})

	events := body["authenticatorEventsWebhookConfig"].(map[string]any)

	if _, present := events["includeCredentialPublicKey"]; present {
		t.Errorf("expected an unknown flag to be omitted, got %v", events)
	}
}

func TestTenantSettingsSendsAnEmptyAllowlist(t *testing.T) {
	body := requestBodyOf(t, tenantSettingsResourceModel{
		TokenDurationInMinutes:           types.Int64Null(),
		AuthenticatorEventsWebhookConfig: types.ObjectNull(authenticatorEventsWebhookConfigAttributeTypes),
		LogEventsWebhookConfig:           types.ObjectNull(logEventsWebhookConfigAttributeTypes),
		IpWhitelist:                      types.SetValueMust(types.StringType, []attr.Value{}),
	})

	allowlist, ok := body["ipWhitelist"].([]any)
	if !ok || len(allowlist) != 0 {
		t.Errorf("expected an empty ipWhitelist to be sent, got %v", body)
	}
}

func TestTenantSettingsSendsTheAllowlistSorted(t *testing.T) {
	body := requestBodyOf(t, tenantSettingsResourceModel{
		TokenDurationInMinutes:           types.Int64Value(15),
		AuthenticatorEventsWebhookConfig: types.ObjectNull(authenticatorEventsWebhookConfigAttributeTypes),
		LogEventsWebhookConfig:           types.ObjectNull(logEventsWebhookConfigAttributeTypes),
		IpWhitelist: types.SetValueMust(types.StringType, []attr.Value{
			types.StringValue("198.51.100.7/32"),
			types.StringValue("203.0.113.0/24"),
		}),
	})

	allowlist := body["ipWhitelist"].([]any)

	if allowlist[0] != "198.51.100.7/32" || allowlist[1] != "203.0.113.0/24" {
		t.Errorf("expected a sorted ipWhitelist, got %v", allowlist)
	}

	if body["tokenDurationInMinutes"] != float64(15) {
		t.Errorf("expected tokenDurationInMinutes to be sent, got %v", body["tokenDurationInMinutes"])
	}
}

func TestTenantSettingsModelReadsBackAConfiguredTenant(t *testing.T) {
	url := "https://example.com/events"
	endpointUrl := "https://example.com/logs"
	includePublicKey := false
	duration := int64(15)
	allowlist := []string{"203.0.113.0/24"}

	model := tenantSettingsModelFromResponse(&authsignal.TenantResponse{
		TokenDurationInMinutes: &duration,
		AuthenticatorEventsWebhookConfig: &authsignal.AuthenticatorEventsWebhookConfigResponse{
			Url:                        &url,
			IncludeCredentialPublicKey: &includePublicKey,
		},
		LogEventsWebhookConfig: &authsignal.LogEventsWebhookConfigResponse{EndpointUrl: &endpointUrl},
		IpWhitelist:            &allowlist,
	})

	if model.TokenDurationInMinutes.ValueInt64() != 15 {
		t.Errorf("expected the token duration to be read back, got %v", model.TokenDurationInMinutes)
	}

	if model.AuthenticatorEventsWebhookConfig.IsNull() || model.LogEventsWebhookConfig.IsNull() {
		t.Errorf("expected both webhooks to be read back, got %v and %v",
			model.AuthenticatorEventsWebhookConfig, model.LogEventsWebhookConfig)
	}

	if model.IpWhitelist.IsNull() || len(model.IpWhitelist.Elements()) != 1 {
		t.Errorf("expected the allowlist to be read back, got %v", model.IpWhitelist)
	}
}

func TestTenantSettingsModelLeavesUnconfiguredSettingsNull(t *testing.T) {
	model := tenantSettingsModelFromResponse(&authsignal.TenantResponse{})

	if !model.AuthenticatorEventsWebhookConfig.IsNull() {
		t.Errorf("expected an unconfigured authenticator webhook to stay null")
	}

	if !model.LogEventsWebhookConfig.IsNull() {
		t.Errorf("expected an unconfigured log webhook to stay null")
	}

	if !model.IpWhitelist.IsNull() {
		t.Errorf("expected an unconfigured allowlist to stay null")
	}

	if !model.TokenDurationInMinutes.IsNull() {
		t.Errorf("expected an unconfigured token duration to stay null")
	}
}

func TestTenantSettingsModelDistinguishesAnEmptyAllowlistFromNone(t *testing.T) {
	empty := []string{}

	model := tenantSettingsModelFromResponse(&authsignal.TenantResponse{IpWhitelist: &empty})

	if model.IpWhitelist.IsNull() || len(model.IpWhitelist.Elements()) != 0 {
		t.Errorf("expected an empty allowlist to read back as empty, got %v", model.IpWhitelist)
	}
}

func planModifiedWebhookUrlFor(
	t *testing.T,
	res resource.Resource,
	providerAttribute string,
	storedProvider string,
	configuredProvider string,
	configuredUrl types.String,
) types.String {
	t.Helper()

	resourceSchema := resourceSchemaOf(t, res)
	storedUrl := types.StringValue("https://example.com/stored")

	configuredEndpoint := tftypes.NewValue(tftypes.String, nil)
	if !configuredUrl.IsNull() {
		configuredEndpoint = tftypes.NewValue(tftypes.String, configuredUrl.ValueString())
	}

	configValue := resourceObject(t, resourceSchema, map[string]tftypes.Value{
		providerAttribute: tftypes.NewValue(tftypes.String, configuredProvider),
		"webhook_url":     configuredEndpoint,
	})

	planValue := resourceObject(t, resourceSchema, map[string]tftypes.Value{
		providerAttribute: tftypes.NewValue(tftypes.String, configuredProvider),
		"webhook_url":     tftypes.NewValue(tftypes.String, storedUrl.ValueString()),
	})

	stateValue := resourceObject(t, resourceSchema, map[string]tftypes.Value{
		providerAttribute: tftypes.NewValue(tftypes.String, storedProvider),
		"webhook_url":     tftypes.NewValue(tftypes.String, storedUrl.ValueString()),
	})

	resp := &planmodifier.StringResponse{PlanValue: storedUrl}
	supersededWebhookUrl{providerAttribute: providerAttribute}.PlanModifyString(context.Background(), planmodifier.StringRequest{
		Config:      tfsdk.Config{Schema: resourceSchema, Raw: configValue},
		Plan:        tfsdk.Plan{Schema: resourceSchema, Raw: planValue},
		State:       tfsdk.State{Schema: resourceSchema, Raw: stateValue},
		ConfigValue: configuredUrl,
		PlanValue:   storedUrl,
		StateValue:  storedUrl,
	}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatal(detailsOf(resp.Diagnostics))
	}

	return resp.PlanValue
}

func TestSmsWebhookUrlIsPlannedUnknownWhenLeavingTheWebhookProvider(t *testing.T) {
	planned := planModifiedWebhookUrlFor(
		t, &smsAuthenticatorConfigurationResource{}, "sms_provider", "WEBHOOK", "TNZ", types.StringNull(),
	)

	if !planned.IsUnknown() {
		t.Errorf("expected the endpoint to be planned as unknown, got %v", planned)
	}
}

func TestEmailOtpWebhookUrlIsPlannedUnknownWhenLeavingTheWebhookProvider(t *testing.T) {
	planned := planModifiedWebhookUrlFor(
		t, &emailOtpAuthenticatorConfigurationResource{}, "email_provider", "WEBHOOK", "SENDGRID", types.StringNull(),
	)

	if !planned.IsUnknown() {
		t.Errorf("expected the endpoint to be planned as unknown, got %v", planned)
	}
}

func TestSmsWebhookUrlSurvivesStayingOnTheWebhookProvider(t *testing.T) {
	planned := planModifiedWebhookUrlFor(
		t, &smsAuthenticatorConfigurationResource{}, "sms_provider", "WEBHOOK", "WEBHOOK", types.StringNull(),
	)

	if planned.IsUnknown() || planned.ValueString() != "https://example.com/stored" {
		t.Errorf("expected the stored endpoint to survive, got %v", planned)
	}
}

func TestSmsWebhookUrlIsKeptWhenSwitchingToTheWebhookProvider(t *testing.T) {
	planned := planModifiedWebhookUrlFor(
		t, &smsAuthenticatorConfigurationResource{}, "sms_provider", "TNZ", "WEBHOOK",
		types.StringValue("https://example.com/new"),
	)

	if planned.IsUnknown() || planned.ValueString() != "https://example.com/stored" {
		t.Errorf("expected a configured endpoint to be left alone, got %v", planned)
	}
}

func TestEmailOtpWebhookUrlIsKeptWhenSwitchingToTheWebhookProvider(t *testing.T) {
	planned := planModifiedWebhookUrlFor(
		t, &emailOtpAuthenticatorConfigurationResource{}, "email_provider", "SENDGRID", "WEBHOOK",
		types.StringValue("https://example.com/new"),
	)

	if planned.IsUnknown() || planned.ValueString() != "https://example.com/stored" {
		t.Errorf("expected a configured endpoint to be left alone, got %v", planned)
	}
}

func TestTenantSettingsFalsePublicKeyFlagRoundTrips(t *testing.T) {
	body := requestBodyOf(t, tenantSettingsResourceModel{
		TokenDurationInMinutes: types.Int64Null(),
		AuthenticatorEventsWebhookConfig: authenticatorEventsWebhook(
			types.StringValue("https://example.com/events"),
			types.BoolValue(false),
		),
		LogEventsWebhookConfig: types.ObjectNull(logEventsWebhookConfigAttributeTypes),
		IpWhitelist:            types.SetNull(types.StringType),
	})

	if body["authenticatorEventsWebhookConfig"].(map[string]any)["includeCredentialPublicKey"] != false {
		t.Fatalf("expected the request to carry a false flag, got %v", body)
	}

	url := "https://example.com/events"
	includePublicKey := false

	model := tenantSettingsModelFromResponse(&authsignal.TenantResponse{
		AuthenticatorEventsWebhookConfig: &authsignal.AuthenticatorEventsWebhookConfigResponse{
			Url:                        &url,
			IncludeCredentialPublicKey: &includePublicKey,
		},
	})

	var config authenticatorEventsWebhookConfigModel
	if diags := model.AuthenticatorEventsWebhookConfig.As(context.Background(), &config, basetypes.ObjectAsOptions{}); diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if config.IncludeCredentialPublicKey.IsNull() || config.IncludeCredentialPublicKey.ValueBool() {
		t.Errorf("expected a false flag to be read back as false, got %v", config.IncludeCredentialPublicKey)
	}
}

func TestTenantSettingsOmittedNestedFieldConvergesOnTheApiValue(t *testing.T) {
	body := requestBodyOf(t, tenantSettingsResourceModel{
		TokenDurationInMinutes: types.Int64Null(),
		AuthenticatorEventsWebhookConfig: authenticatorEventsWebhook(
			types.StringValue("https://example.com/events"),
			types.BoolNull(),
		),
		LogEventsWebhookConfig: types.ObjectNull(logEventsWebhookConfigAttributeTypes),
		IpWhitelist:            types.SetNull(types.StringType),
	})

	events := body["authenticatorEventsWebhookConfig"].(map[string]any)

	if _, present := events["includeCredentialPublicKey"]; present {
		t.Errorf("expected an unset flag to be left to the API, got %v", events)
	}

	url := "https://example.com/events"
	stored := true

	model := tenantSettingsModelFromResponse(&authsignal.TenantResponse{
		AuthenticatorEventsWebhookConfig: &authsignal.AuthenticatorEventsWebhookConfigResponse{
			Url:                        &url,
			IncludeCredentialPublicKey: &stored,
		},
	})

	var config authenticatorEventsWebhookConfigModel
	if diags := model.AuthenticatorEventsWebhookConfig.As(context.Background(), &config, basetypes.ObjectAsOptions{}); diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if !config.IncludeCredentialPublicKey.ValueBool() {
		t.Errorf("expected state to take the stored flag, got %v", config.IncludeCredentialPublicKey)
	}
}
