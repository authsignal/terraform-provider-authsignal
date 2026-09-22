package provider

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/authsignal/authsignal-management-go/v6"
	"github.com/hashicorp/terraform-plugin-framework-validators/float64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

const (
	smsAuthenticatorSlug      = "sms"
	emailOtpAuthenticatorSlug = "email-otp"
	passkeyAuthenticatorSlug  = "passkey"
	pushAuthenticatorSlug     = "push"
)

const webhookProvider = "WEBHOOK"

var allowedSmsProviders = []string{"BIRD", "MESSAGE_MEDIA", "MODICA_GROUP", "TNZ", "TWILIO", webhookProvider}

var allowedEmailProviders = []string{"BIRD", "MAILGUN", "MAILJET", "MANDRILL", "SENDGRID", "SES", "SMTP", webhookProvider}

var allowedPushProviders = []string{"DEFAULT", "FIREBASE", "WEBHOOK"}

var allowedUserVerificationRequirements = []string{"preferred", "required"}

var allowedAuthenticatorAttachments = []string{"all-supported", "cross-platform", "platform"}

var allowedPasskeyRegistrationHints = []string{"client-device", "hybrid", "security-key"}

var allowedAppAttestationFailureModes = []string{"ALLOW_WITH_WARNING", "BLOCK"}

func allowedValues(values []string) string {
	quoted := make([]string, 0, len(values))
	for _, value := range values {
		quoted = append(quoted, fmt.Sprintf("`%s`", value))
	}

	return " One of " + strings.Join(quoted, ", ") + "."
}

func importStateSlug(id string, slug string) diag.Diagnostics {
	var diags diag.Diagnostics

	if !strings.EqualFold(strings.TrimSpace(id), slug) {
		diags.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Expected the import identifier to be %q, got %q. "+
				"Each authenticator configuration is a per-tenant singleton, so the slug of its Management API path is the only valid identifier.", slug, id),
		)
	}

	return diags
}

func createConflictDiagnostic(resourceType string, resourceName string, slug string) diag.Diagnostic {
	return diag.NewErrorDiagnostic(
		"Authenticator Configuration Already Exists",
		fmt.Sprintf("The tenant already has a %s authenticator configuration, and a tenant may only have one. "+
			"Import it instead of creating it:\n\n  terraform import %s.%s %s", slug, resourceType, resourceName, slug),
	)
}

var rateLimitAttributeTypes = map[string]attr.Type{
	"rate_limit":        types.Int64Type,
	"window_in_minutes": types.Float64Type,
}

type rateLimitModel struct {
	RateLimit       types.Int64   `tfsdk:"rate_limit"`
	WindowInMinutes types.Float64 `tfsdk:"window_in_minutes"`
}

func rateLimitAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"rate_limit": schema.Int64Attribute{
			Description: "Maximum attempts in the window. Must be between 1 and 100.",
			Required:    true,
			Validators: []validator.Int64{
				int64validator.Between(1, 100),
			},
		},
		"window_in_minutes": schema.Float64Attribute{
			Description: "Window length in minutes. Must be between 1 and 1440; fractional values are supported.",
			Required:    true,
			Validators: []validator.Float64{
				float64validator.Between(1, 1440),
			},
		},
	}
}

func rateLimitObjectType() types.ObjectType {
	return types.ObjectType{AttrTypes: rateLimitAttributeTypes}
}

func rateLimitValue(rateLimit *authsignal.RateLimitConfiguration) types.Object {
	if rateLimit == nil {
		return types.ObjectNull(rateLimitAttributeTypes)
	}

	return types.ObjectValueMust(rateLimitAttributeTypes, map[string]attr.Value{
		"rate_limit":        types.Int64Value(rateLimit.RateLimit),
		"window_in_minutes": types.Float64Value(rateLimit.WindowInMinutes),
	})
}

func rateLimitListValue(rateLimits *[]authsignal.RateLimitConfiguration) types.List {
	if rateLimits == nil {
		return types.ListNull(rateLimitObjectType())
	}

	elements := make([]attr.Value, 0, len(*rateLimits))
	for _, rateLimit := range *rateLimits {
		limit := rateLimit
		elements = append(elements, rateLimitValue(&limit))
	}

	return types.ListValueMust(rateLimitObjectType(), elements)
}

func rateLimitFrom(ctx context.Context, object types.Object) (*authsignal.RateLimitConfiguration, diag.Diagnostics) {
	if !isKnownAndSet(object) {
		return nil, nil
	}

	var model rateLimitModel
	diags := object.As(ctx, &model, basetypes.ObjectAsOptions{})
	if diags.HasError() {
		return nil, diags
	}

	return &authsignal.RateLimitConfiguration{
		RateLimit:       model.RateLimit.ValueInt64(),
		WindowInMinutes: model.WindowInMinutes.ValueFloat64(),
	}, diags
}

func rateLimitsFrom(ctx context.Context, list types.List) (*[]authsignal.RateLimitConfiguration, diag.Diagnostics) {
	if !isKnownAndSet(list) {
		return nil, nil
	}

	var diags diag.Diagnostics
	rateLimits := make([]authsignal.RateLimitConfiguration, 0, len(list.Elements()))

	for _, element := range list.Elements() {
		object, ok := element.(types.Object)
		if !ok {
			continue
		}

		rateLimit, elementDiags := rateLimitFrom(ctx, object)
		diags.Append(elementDiags...)
		if rateLimit != nil {
			rateLimits = append(rateLimits, *rateLimit)
		}
	}

	return &rateLimits, diags
}

var prefixRateLimitAttributeTypes = map[string]attr.Type{
	"enabled":           types.BoolType,
	"block_size":        types.StringType,
	"rate_limit":        types.Int64Type,
	"window_in_minutes": types.Float64Type,
	"country_codes":     types.SetType{ElemType: types.StringType},
}

type prefixRateLimitModel struct {
	Enabled         types.Bool    `tfsdk:"enabled"`
	BlockSize       types.String  `tfsdk:"block_size"`
	RateLimit       types.Int64   `tfsdk:"rate_limit"`
	WindowInMinutes types.Float64 `tfsdk:"window_in_minutes"`
	CountryCodes    types.Set     `tfsdk:"country_codes"`
}

func prefixRateLimitObjectType() types.ObjectType {
	return types.ObjectType{AttrTypes: prefixRateLimitAttributeTypes}
}

func blockSizeFrom(value string) (*authsignal.BlockSize, diag.Diagnostics) {
	var diags diag.Diagnostics

	if value == "country" {
		return &authsignal.BlockSize{Country: true}, diags
	}

	digits, err := strconv.ParseInt(value, 10, 64)
	if err != nil || digits <= 0 {
		diags.AddError(
			"Invalid Block Size",
			fmt.Sprintf("block_size must be the string \"country\" or a positive number of digits, got %q.", value),
		)
		return nil, diags
	}

	return &authsignal.BlockSize{Numeric: digits}, diags
}

func blockSizeValue(blockSize *authsignal.BlockSize) types.String {
	if blockSize == nil {
		return types.StringNull()
	}

	if blockSize.Country {
		return types.StringValue("country")
	}

	return types.StringValue(strconv.FormatInt(blockSize.Numeric, 10))
}

func prefixRateLimitListValue(rateLimits *[]authsignal.PrefixRateLimitConfiguration) types.List {
	if rateLimits == nil {
		return types.ListNull(prefixRateLimitObjectType())
	}

	elements := make([]attr.Value, 0, len(*rateLimits))

	for _, rateLimit := range *rateLimits {
		countryCodes := types.SetNull(types.StringType)
		if rateLimit.CountryCodes != nil {
			countryCodes = stringSetValue(*rateLimit.CountryCodes)
		}

		elements = append(elements, types.ObjectValueMust(prefixRateLimitAttributeTypes, map[string]attr.Value{
			"enabled":           types.BoolPointerValue(rateLimit.Enabled),
			"block_size":        blockSizeValue(rateLimit.BlockSize),
			"rate_limit":        types.Int64PointerValue(rateLimit.RateLimit),
			"window_in_minutes": types.Float64PointerValue(rateLimit.WindowInMinutes),
			"country_codes":     countryCodes,
		}))
	}

	return types.ListValueMust(prefixRateLimitObjectType(), elements)
}

func prefixRateLimitsFrom(ctx context.Context, list types.List) ([]authsignal.PrefixRateLimitConfiguration, diag.Diagnostics) {
	var diags diag.Diagnostics

	if !isKnownAndSet(list) {
		return nil, diags
	}

	rateLimits := make([]authsignal.PrefixRateLimitConfiguration, 0, len(list.Elements()))

	for _, element := range list.Elements() {
		object, ok := element.(types.Object)
		if !ok {
			continue
		}

		var model prefixRateLimitModel
		diags.Append(object.As(ctx, &model, basetypes.ObjectAsOptions{})...)
		if diags.HasError() {
			return nil, diags
		}

		rateLimit := authsignal.PrefixRateLimitConfiguration{
			Enabled:         boolPointer(model.Enabled),
			RateLimit:       int64Pointer(model.RateLimit),
			WindowInMinutes: float64Pointer(model.WindowInMinutes),
		}

		if isKnownAndSet(model.BlockSize) {
			blockSize, blockSizeDiags := blockSizeFrom(model.BlockSize.ValueString())
			diags.Append(blockSizeDiags...)
			if diags.HasError() {
				return nil, diags
			}
			rateLimit.BlockSize = blockSize
		}

		if isKnownAndSet(model.CountryCodes) {
			countryCodes, countryCodeDiags := sortedStringsFrom(ctx, model.CountryCodes)
			diags.Append(countryCodeDiags...)
			if diags.HasError() {
				return nil, diags
			}
			rateLimit.CountryCodes = &countryCodes
		}

		rateLimits = append(rateLimits, rateLimit)
	}

	return rateLimits, diags
}

func isKnownAndSet(value attr.Value) bool {
	return !value.IsNull() && !value.IsUnknown()
}

func stringSetValue(values []string) types.Set {
	elements := make([]attr.Value, 0, len(values))
	for _, value := range values {
		elements = append(elements, types.StringValue(value))
	}

	return types.SetValueMust(types.StringType, elements)
}

func stringListValue(values []string) types.List {
	elements := make([]attr.Value, 0, len(values))
	for _, value := range values {
		elements = append(elements, types.StringValue(value))
	}

	return types.ListValueMust(types.StringType, elements)
}

func stringListPointerValue(values *[]string) types.List {
	if values == nil {
		return types.ListNull(types.StringType)
	}

	return stringListValue(*values)
}

func stringsFrom(ctx context.Context, list types.List) ([]string, diag.Diagnostics) {
	var values []string
	diags := list.ElementsAs(ctx, &values, false)

	return values, diags
}

func stringSetPointerValue(values *[]string) types.Set {
	if values == nil {
		return types.SetNull(types.StringType)
	}

	return stringSetValue(*values)
}

func sortedStringsFrom(ctx context.Context, set types.Set) ([]string, diag.Diagnostics) {
	var values []string
	diags := set.ElementsAs(ctx, &values, false)
	sort.Strings(values)

	return values, diags
}

func boolPointer(value types.Bool) *bool {
	if !isKnownAndSet(value) {
		return nil
	}

	return value.ValueBoolPointer()
}

func int64Pointer(value types.Int64) *int64 {
	if !isKnownAndSet(value) {
		return nil
	}

	return value.ValueInt64Pointer()
}

func float64Pointer(value types.Float64) *float64 {
	if !isKnownAndSet(value) {
		return nil
	}

	return value.ValueFloat64Pointer()
}

func stringPointer(value types.String) *string {
	if !isKnownAndSet(value) {
		return nil
	}

	return value.ValueStringPointer()
}

var httpsUrlPattern = regexp.MustCompile(`^https://`)

// Reproduces the API's own pattern exactly. http is accepted as well as https, which a local or
// on-premises relying party needs, so narrowing this would refuse configurations the API accepts.
var expectedOriginPattern = regexp.MustCompile(`^(https?://|android:)`)

// A known plan is what the applied state has to match, or Terraform reports an inconsistent result;
// an unknown plan can only be filled from the response. Taking either side unconditionally breaks one
// of those two cases.
func resolvedValue[T attr.Value](planned T, fromResponse T) T {
	if planned.IsUnknown() {
		return fromResponse
	}

	return planned
}

func webhookUrlDiagnostics(providerAttribute string, provider string, webhookUrl types.String) diag.Diagnostics {
	var diags diag.Diagnostics

	if provider == webhookProvider && !isKnownAndSet(webhookUrl) {
		diags.AddError(
			"Missing Webhook URL",
			fmt.Sprintf("`webhook_url` is required when `%s` is %q.", providerAttribute, webhookProvider),
		)
	}

	if provider != webhookProvider && isKnownAndSet(webhookUrl) {
		diags.AddError(
			"Webhook URL Does Not Apply to the Configured Provider",
			fmt.Sprintf("`webhook_url` requires `%s` to be %q, but it is %q. Remove `webhook_url` or change `%s`.",
				providerAttribute, webhookProvider, provider, providerAttribute),
		)
	}

	return diags
}

func missingResponseFieldDiagnostic(method string, field string) diag.Diagnostic {
	return diag.NewErrorDiagnostic(
		"Incomplete Response From the Management API",
		fmt.Sprintf("The Management API returned a %s authenticator configuration without %q, which this resource requires. "+
			"Terraform has left the state as it was rather than recording an incomplete configuration. This is a problem with "+
			"the response rather than with the configuration; try again, and report it if it persists.", method, field),
	)
}

func requiredResponseString(method string, field string, value *string, diags *diag.Diagnostics) {
	if value == nil || *value == "" {
		diags.Append(missingResponseFieldDiagnostic(method, field))
	}
}

func requiredResponseStrings(method string, field string, values *[]string, diags *diag.Diagnostics) {
	if values == nil || len(*values) == 0 {
		diags.Append(missingResponseFieldDiagnostic(method, field))
	}
}

func serverOwnedDiagnostics(method string, authenticatorId string, verificationMethod string) diag.Diagnostics {
	var diags diag.Diagnostics

	requiredResponseString(method, "authenticatorId", &authenticatorId, &diags)
	requiredResponseString(method, "verificationMethod", &verificationMethod, &diags)

	return diags
}

func smsResponseDiagnostics(response *authsignal.SmsAuthenticatorConfiguration) diag.Diagnostics {
	diags := serverOwnedDiagnostics("SMS", response.AuthenticatorId, response.VerificationMethod)
	requiredResponseString("SMS", "smsProvider", response.SmsProvider, &diags)

	return diags
}

func emailOtpResponseDiagnostics(response *authsignal.EmailOtpAuthenticatorConfiguration) diag.Diagnostics {
	diags := serverOwnedDiagnostics("email OTP", response.AuthenticatorId, response.VerificationMethod)
	requiredResponseString("email OTP", "emailProvider", response.EmailProvider, &diags)

	return diags
}

func passkeyResponseDiagnostics(response *authsignal.PasskeyAuthenticatorConfiguration) diag.Diagnostics {
	diags := serverOwnedDiagnostics("passkey", response.AuthenticatorId, response.VerificationMethod)
	requiredResponseString("passkey", "relyingParty", response.RelyingParty, &diags)
	requiredResponseStrings("passkey", "expectedOrigins", response.ExpectedOrigins, &diags)

	return diags
}

func pushResponseDiagnostics(response *authsignal.PushAuthenticatorConfiguration) diag.Diagnostics {
	diags := serverOwnedDiagnostics("push", response.AuthenticatorId, response.VerificationMethod)
	requiredResponseString("push", "pushProvider", response.PushProvider, &diags)

	return diags
}
