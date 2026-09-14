package provider

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func mismatchedCredentialDiagnostic(blockName string, providerAttribute string, provider string) diag.Diagnostic {
	return diag.NewErrorDiagnostic(
		"Credentials Do Not Match the Configured Provider",
		fmt.Sprintf("`%s` is set but `%s` is %q. A configuration may only carry the credentials of the provider it names. "+
			"Remove `%s`, or change `%s` to the provider it belongs to.", blockName, providerAttribute, provider, blockName, providerAttribute),
	)
}

func missingCredentialDiagnostic(blockName string, providerAttribute string, provider string) diag.Diagnostic {
	return diag.NewErrorDiagnostic(
		"Missing Credentials for the Configured Provider",
		fmt.Sprintf("`%s` is %q, so `%s` has to be set when the configuration is created.", providerAttribute, provider, blockName),
	)
}

func incompleteCredentialDiagnostic(blockName string, missing []string) diag.Diagnostic {
	return diag.NewErrorDiagnostic(
		"Incomplete Credentials",
		fmt.Sprintf("`%s` is missing %s, which the Management API requires when the credentials are first stored.",
			blockName, strings.Join(quoteAll(missing), ", ")),
	)
}

func clearedCountryCodesDiagnostic() diag.Diagnostic {
	return diag.NewErrorDiagnostic(
		"Default Country Code Conflicts With an Empty Country Restriction",
		"`sms_country_codes` is set to `[]`, which clears the tenant's country restriction and the stored "+
			"`default_country_code` along with it, so the Management API rejects a request that sends both. "+
			"Remove `default_country_code`, or list the country codes you want to allow in `sms_country_codes` "+
			"and keep the default among them.",
	)
}

func missingVersionMarkerDiagnostic(block credentialBlock) diag.Diagnostic {
	return diag.NewErrorDiagnostic(
		"Missing Version Marker for a Write-Only Credential Block",
		fmt.Sprintf("`%s` is set but `%s` is not. Set `%s` to an opaque counter or date, never the credential itself. "+
			"Change it whenever you want the credentials sent again.", block.Name, block.versionName(), block.versionName()),
	)
}

func writeOnlyBlockDiagnostics(blocks []credentialBlock, object func(string) types.Object, version func(string) types.String) diag.Diagnostics {
	var diags diag.Diagnostics

	for _, block := range blocks {
		if !block.allSecret() {
			continue
		}

		if isKnownAndSet(object(block.Name)) && version(block.Name).IsNull() {
			diags.Append(missingVersionMarkerDiagnostic(block))
		}
	}

	return diags
}

func defaultCountryCodeOutsideRestrictionDiagnostic(defaultCode string, allowed []string) diag.Diagnostic {
	return diag.NewErrorDiagnostic(
		"Default Country Code Is Not an Allowed Country",
		fmt.Sprintf("`default_country_code` is %q but `sms_country_codes` allows only %s. A number entered without a country code "+
			"is assumed to be in the default country, so the default has to be one users may receive an SMS at. Add %q to "+
			"`sms_country_codes`, or choose a default from the codes already listed.",
			defaultCode, strings.Join(quoteAll(allowed), ", "), defaultCode),
	)
}

func quoteAll(values []string) []string {
	quoted := make([]string, 0, len(values))
	for _, value := range values {
		quoted = append(quoted, strconv.Quote(value))
	}

	return quoted
}

type greaterThanZeroValidator struct{}

func (v greaterThanZeroValidator) Description(_ context.Context) string {
	return "must be greater than 0"
}

func (v greaterThanZeroValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v greaterThanZeroValidator) ValidateFloat64(_ context.Context, req validator.Float64Request, resp *validator.Float64Response) {
	if !isKnownAndSet(req.ConfigValue) {
		return
	}

	if req.ConfigValue.ValueFloat64() > 0 {
		return
	}

	resp.Diagnostics.AddAttributeError(
		req.Path,
		"Invalid Window",
		fmt.Sprintf("Expected a window greater than 0 minutes, got %v.", req.ConfigValue.ValueFloat64()),
	)
}

type blockSizeValidator struct{}

func (v blockSizeValidator) Description(_ context.Context) string {
	return "must be the string \"country\" or a positive number of digits"
}

func (v blockSizeValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v blockSizeValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if !isKnownAndSet(req.ConfigValue) {
		return
	}

	value := req.ConfigValue.ValueString()
	if value == "country" {
		return
	}

	if digits, err := strconv.ParseInt(value, 10, 64); err == nil && digits > 0 {
		return
	}

	resp.Diagnostics.AddAttributeError(
		req.Path,
		"Invalid Block Size",
		fmt.Sprintf("Expected the string \"country\" or a positive number of digits, got %q.", value),
	)
}

type expectedOriginValidator struct{}

func (v expectedOriginValidator) Description(_ context.Context) string {
	return "must begin with http://, https:// or android:"
}

func (v expectedOriginValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v expectedOriginValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if !isKnownAndSet(req.ConfigValue) {
		return
	}

	if expectedOriginPattern.MatchString(req.ConfigValue.ValueString()) {
		return
	}

	resp.Diagnostics.AddAttributeError(
		req.Path,
		"Invalid Expected Origin",
		fmt.Sprintf("Expected an origin beginning with \"http://\", \"https://\" or \"android:\", got %q.", req.ConfigValue.ValueString()),
	)
}

type httpsUrlValidator struct{}

func (v httpsUrlValidator) Description(_ context.Context) string {
	return "must be an https URL"
}

func (v httpsUrlValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v httpsUrlValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if !isKnownAndSet(req.ConfigValue) {
		return
	}

	if httpsUrlPattern.MatchString(req.ConfigValue.ValueString()) {
		return
	}

	resp.Diagnostics.AddAttributeError(
		req.Path,
		"Invalid Webhook URL",
		fmt.Sprintf("Expected an https URL, got %q.", req.ConfigValue.ValueString()),
	)
}

func stringSetExcludes(set types.Set, value string) bool {
	for _, element := range set.Elements() {
		text, ok := element.(types.String)
		if !ok || text.IsUnknown() {
			return false
		}

		if text.ValueString() == value {
			return false
		}
	}

	return true
}

type excludedDefaultCountryCode struct{}

func (m excludedDefaultCountryCode) Description(_ context.Context) string {
	return "Plans default_country_code as unknown when the configured sms_country_codes no longer allows the stored default, because the Management API discards the default along with it."
}

func (m excludedDefaultCountryCode) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m excludedDefaultCountryCode) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if req.Plan.Raw.IsNull() {
		return
	}

	if !req.ConfigValue.IsNull() {
		return
	}

	if !isKnownAndSet(req.StateValue) {
		return
	}

	var countryCodes types.Set
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("sms_country_codes"), &countryCodes)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !isKnownAndSet(countryCodes) {
		return
	}

	if stringSetExcludes(countryCodes, req.StateValue.ValueString()) {
		resp.PlanValue = types.StringUnknown()
	}
}

type supersededWebhookUrl struct{}

func (m supersededWebhookUrl) Description(_ context.Context) string {
	return "Plans webhook_url as unknown when push_provider moves away from WEBHOOK, because the Management API discards the stored endpoint."
}

func (m supersededWebhookUrl) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m supersededWebhookUrl) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if req.Plan.Raw.IsNull() || req.State.Raw.IsNull() {
		return
	}

	if !req.ConfigValue.IsNull() {
		return
	}

	if !isKnownAndSet(req.StateValue) {
		return
	}

	var storedProvider types.String
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("push_provider"), &storedProvider)...)

	var configuredProvider types.String
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("push_provider"), &configuredProvider)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if storedProvider.ValueString() != pushWebhookProvider {
		return
	}

	if !isKnownAndSet(configuredProvider) || configuredProvider.ValueString() == pushWebhookProvider {
		return
	}

	resp.PlanValue = types.StringUnknown()
}
