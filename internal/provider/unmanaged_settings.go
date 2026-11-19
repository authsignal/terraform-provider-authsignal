package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/authsignal/authsignal-management-go/v6"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

const defaultsPrivateStateKey = "defaults"

func storeDefaults(ctx context.Context, private interface {
	SetKey(context.Context, string, []byte) diag.Diagnostics
}, defaults authsignal.Defaults) diag.Diagnostics {
	// The framework always sets Private, but a test that calls a resource directly leaves it nil.
	if reflect.ValueOf(private).IsNil() {
		return nil
	}

	// Stored as an empty set rather than null, because null means the defaults haven't been read.
	if defaults == nil {
		defaults = authsignal.Defaults{}
	}

	encoded, err := json.Marshal(defaults)
	if err != nil {
		var diags diag.Diagnostics
		diags.AddError("Unable to Store the Management API's Defaults", err.Error())
		return diags
	}

	return private.SetKey(ctx, defaultsPrivateStateKey, encoded)
}

// Creating tenant settings or pre-built UI settings takes over a tenant that already has values, and
// the create only sends what the configuration sets. The plan reads the tenant so that it fails for
// anything else the create would keep, before anything is written.
func checkUnmanagedSettingsIncludingCreate[M any](ctx context.Context, client *authsignal.Client, modelFrom func(*authsignal.TenantResponse) M, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	creating := req.State.Raw.IsNull() && !req.Plan.Raw.IsNull()
	if !creating {
		checkUnmanagedSettings(ctx, req, resp)
		return
	}

	resourceSchema, ok := req.Plan.Schema.(schema.Schema)
	if !ok {
		return
	}

	tenant, _, err := client.GetTenant()
	if err != nil {
		resp.Diagnostics.AddError("Unable to Read Authsignal Tenant Settings", err.Error())
		return
	}

	existing := tfsdk.State{Schema: resourceSchema}
	resp.Diagnostics.Append(existing.Set(ctx, modelFrom(tenant))...)
	if resp.Diagnostics.HasError() {
		return
	}

	configAttributes, configErr := objectAttributes(ctx, resourceSchema, req.Config.Raw)
	existingAttributes, existingErr := objectAttributes(ctx, resourceSchema, existing.Raw)
	if err := errors.Join(configErr, existingErr); err != nil {
		resp.Diagnostics.AddError("Unable to Check for Unmanaged Settings", err.Error())
		return
	}

	if settings := unconfiguredSettings("", resourceSchema.Attributes, configAttributes, existingAttributes, tenant.Defaults); len(settings) > 0 {
		resp.Diagnostics.Append(unmanagedSettingsDiagnostic(settings))
	}
}

// An Optional+Computed setting the configuration leaves out keeps whatever the tenant has, so no plan
// would show a change made outside Terraform. The plan fails instead when the tenant's value is neither
// blank nor the default the Management API reports, which the last read stored in private state.
func checkUnmanagedSettings(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	resourceSchema, ok := req.Plan.Schema.(schema.Schema)
	if !ok {
		return
	}

	defaults, diags := storedDefaults(ctx, req)
	resp.Diagnostics.Append(diags...)
	if diags.HasError() {
		return
	}

	resp.Diagnostics.Append(unmanagedSettingsDiagnostics(ctx, resourceSchema, req.Config.Raw, req.State.Raw, req.Plan.Raw, defaults)...)
}

// The defaults are nil until a read stores them, so state written by 3.x has none until a plan refreshes.
func storedDefaults(ctx context.Context, req resource.ModifyPlanRequest) (authsignal.Defaults, diag.Diagnostics) {
	encoded, diags := req.Private.GetKey(ctx, defaultsPrivateStateKey)
	if diags.HasError() || len(encoded) == 0 {
		return nil, diags
	}

	var defaults authsignal.Defaults
	if err := json.Unmarshal(encoded, &defaults); err != nil {
		diags.AddError("Unable to Read the Management API's Defaults", err.Error())
	}

	return defaults, diags
}

type unmanagedSetting struct {
	block         string
	name          string
	value         attr.Value
	defaultValue  json.RawMessage
	defaultUnread bool
}

func unmanagedSettingsDiagnostics(ctx context.Context, resourceSchema schema.Schema, config tftypes.Value, state tftypes.Value, plan tftypes.Value, defaults authsignal.Defaults) diag.Diagnostics {
	if state.IsNull() || plan.IsNull() {
		return nil
	}

	configAttributes, configErr := objectAttributes(ctx, resourceSchema, config)
	stateAttributes, stateErr := objectAttributes(ctx, resourceSchema, state)
	planAttributes, planErr := objectAttributes(ctx, resourceSchema, plan)
	if err := errors.Join(configErr, stateErr, planErr); err != nil {
		return diag.Diagnostics{diag.NewErrorDiagnostic("Unable to Check for Unmanaged Settings", err.Error())}
	}

	if settings := unmanagedSettingsIn("", resourceSchema.Attributes, configAttributes, stateAttributes, planAttributes, defaults); len(settings) > 0 {
		return diag.Diagnostics{unmanagedSettingsDiagnostic(settings)}
	}

	return nil
}

func objectAttributes(ctx context.Context, resourceSchema schema.Schema, raw tftypes.Value) (map[string]attr.Value, error) {
	value, err := resourceSchema.Type().ValueFromTerraform(ctx, raw)
	if err != nil {
		return nil, err
	}

	object, ok := value.(types.Object)
	if !ok {
		return nil, fmt.Errorf("expected an object, got %T", value)
	}

	return object.Attributes(), nil
}

func unmanagedSettingsIn(block string, attributes map[string]schema.Attribute, config map[string]attr.Value, state map[string]attr.Value, plan map[string]attr.Value, defaults authsignal.Defaults) []unmanagedSetting {
	var settings []unmanagedSetting

	for _, name := range sortedNames(attributes) {
		if nested, ok := attributes[name].(schema.SingleNestedAttribute); ok {
			stateBlock, stateOk := state[name].(types.Object)
			planBlock, planOk := plan[name].(types.Object)

			// A block the plan removes is cleared whole, so nothing inside it is left unmanaged.
			if !stateOk || !planOk || !isKnownAndSet(stateBlock) || !isKnownAndSet(planBlock) {
				continue
			}

			configMembers := map[string]attr.Value{}
			if configBlock, ok := config[name].(types.Object); ok && isKnownAndSet(configBlock) {
				configMembers = configBlock.Attributes()
			}

			settings = append(settings, unmanagedSettingsIn(nestedPath(block, name), nested.Attributes, configMembers, stateBlock.Attributes(), planBlock.Attributes(), defaults)...)
			continue
		}

		if !isUnmanagedWhenLeftOut(attributes[name]) {
			continue
		}

		if configured, ok := config[name]; ok && !configured.IsNull() {
			continue
		}

		value, ok := state[name]
		if !ok || isBlank(value) {
			continue
		}

		// A plan modifier's change, such as clearing a superseded value, already shows in the plan.
		if planned, ok := plan[name]; ok && !planned.IsUnknown() && !planned.Equal(value) {
			continue
		}

		// The API reports defaults for top-level settings only.
		var defaultValue json.RawMessage
		if block == "" {
			defaultValue = defaults[jsonFieldName(name)]
		}

		if matchesDefault(value, defaultValue) {
			continue
		}

		settings = append(settings, unmanagedSetting{block: block, name: name, value: value, defaultValue: defaultValue, defaultUnread: block == "" && defaults == nil})
	}

	return settings
}

func isUnmanagedWhenLeftOut(attribute schema.Attribute) bool {
	return attribute.IsComputed() && isOptionalWithoutDefault(attribute)
}

// A schema default already plans a change when the configuration leaves the setting out.
func isOptionalWithoutDefault(attribute schema.Attribute) bool {
	if !attribute.IsOptional() {
		return false
	}

	switch typed := attribute.(type) {
	case schema.BoolAttribute:
		return typed.Default == nil
	case schema.Int64Attribute:
		return typed.Default == nil
	case schema.Float64Attribute:
		return typed.Default == nil
	case schema.StringAttribute:
		return typed.Default == nil
	case schema.SetAttribute:
		return typed.Default == nil
	case schema.ListAttribute:
		return typed.Default == nil
	default:
		return false
	}
}

// The settings the tenant has that the configuration doesn't set, and that differ from a new tenant's.
func unconfiguredSettings(block string, attributes map[string]schema.Attribute, config map[string]attr.Value, existing map[string]attr.Value, defaults authsignal.Defaults) []unmanagedSetting {
	var settings []unmanagedSetting

	for _, name := range sortedNames(attributes) {
		value, ok := existing[name]
		if !ok || isBlank(value) {
			continue
		}

		configured, ok := config[name]
		isConfigured := ok && !configured.IsNull()

		if nested, ok := attributes[name].(schema.SingleNestedAttribute); ok {
			existingBlock, ok := value.(types.Object)
			if !ok {
				continue
			}

			if !isConfigured {
				settings = append(settings, wholeBlock(nestedPath(block, name), nested.Attributes, existingBlock.Attributes())...)
				continue
			}

			if configBlock, ok := configured.(types.Object); ok && !configBlock.IsUnknown() {
				settings = append(settings, unconfiguredSettings(nestedPath(block, name), nested.Attributes, configBlock.Attributes(), existingBlock.Attributes(), defaults)...)
			}
			continue
		}

		if isConfigured || !isOptionalWithoutDefault(attributes[name]) {
			continue
		}

		var defaultValue json.RawMessage
		if block == "" {
			defaultValue = defaults[jsonFieldName(name)]
		}

		if matchesDefault(value, defaultValue) {
			continue
		}

		settings = append(settings, unmanagedSetting{block: block, name: name, value: value, defaultValue: defaultValue})
	}

	return settings
}

func wholeBlock(block string, attributes map[string]schema.Attribute, existing map[string]attr.Value) []unmanagedSetting {
	var settings []unmanagedSetting

	for _, name := range sortedNames(attributes) {
		if value, ok := existing[name]; ok && !isBlank(value) {
			settings = append(settings, unmanagedSetting{block: block, name: name, value: value})
		}
	}

	return settings
}

func nestedPath(block string, name string) string {
	if block == "" {
		return name
	}

	return block + "." + name
}

func isBlank(value attr.Value) bool {
	if value.IsNull() || value.IsUnknown() {
		return true
	}

	switch typed := value.(type) {
	case types.String:
		return typed.ValueString() == ""
	case types.Set:
		return len(typed.Elements()) == 0
	case types.List:
		return len(typed.Elements()) == 0
	default:
		return false
	}
}

func matchesDefault(value attr.Value, defaultValue json.RawMessage) bool {
	if defaultValue == nil {
		return false
	}

	var decoded any
	if err := json.Unmarshal(defaultValue, &decoded); err != nil {
		return false
	}

	switch typed := value.(type) {
	case types.Bool:
		expected, ok := decoded.(bool)
		return ok && expected == typed.ValueBool()
	case types.Int64:
		expected, ok := decoded.(float64)
		return ok && expected == float64(typed.ValueInt64())
	case types.String:
		expected, ok := decoded.(string)
		return ok && expected == typed.ValueString()
	default:
		return false
	}
}

func jsonFieldName(name string) string {
	words := strings.Split(name, "_")
	for index := 1; index < len(words); index++ {
		if words[index] != "" {
			words[index] = strings.ToUpper(words[index][:1]) + words[index][1:]
		}
	}

	return strings.Join(words, "")
}

func sortedNames(attributes map[string]schema.Attribute) []string {
	names := make([]string, 0, len(attributes))
	for name := range attributes {
		names = append(names, name)
	}
	sort.Strings(names)

	return names
}

func unmanagedSettingsDiagnostic(settings []unmanagedSetting) diag.Diagnostic {
	return diag.NewErrorDiagnostic(
		"Settings Missing From the Configuration",
		defaultsUnreadNotice(settings)+
			"The tenant has values for these settings, but the configuration doesn't set them:\n\n"+
			strings.Join(hclLines(settings), "\n")+"\n\n"+
			"Changes made to them outside Terraform won't show in a plan, and a tenant built from this configuration "+
			"won't get these values. Add them to the configuration.",
	)
}

func defaultsUnreadNotice(settings []unmanagedSetting) string {
	for _, setting := range settings {
		if setting.defaultUnread {
			return "The provider hasn't read this resource's defaults yet, so it can't tell which of these settings are at their default. " +
				"It reads them when a plan refreshes, and state written by version 3 of the provider doesn't have them. " +
				"Run the plan again without `-refresh=false`.\n\n"
		}
	}

	return ""
}

func hclLines(settings []unmanagedSetting) []string {
	var topLevel []unmanagedSetting
	nested := map[string][]unmanagedSetting{}
	var blocks []string

	for _, setting := range settings {
		if setting.block == "" {
			topLevel = append(topLevel, setting)
			continue
		}

		if _, seen := nested[setting.block]; !seen {
			blocks = append(blocks, setting.block)
		}
		nested[setting.block] = append(nested[setting.block], setting)
	}

	var lines []string
	lines = append(lines, hclSettingLines(topLevel, "  ")...)

	for _, block := range blocks {
		if len(lines) > 0 {
			lines = append(lines, "")
		}

		lines = append(lines, "  "+block+" = {")
		lines = append(lines, hclSettingLines(nested[block], "    ")...)
		lines = append(lines, "  }")
	}

	return lines
}

func hclSettingLines(settings []unmanagedSetting, indent string) []string {
	nameWidth := 0
	for _, setting := range settings {
		nameWidth = max(nameWidth, len(setting.name))
	}

	assignments := make([]string, len(settings))
	assignmentWidth := 0
	for index, setting := range settings {
		assignments[index] = fmt.Sprintf("%-*s = %s", nameWidth, setting.name, hclValue(setting.value))
		assignmentWidth = max(assignmentWidth, len(assignments[index]))
	}

	lines := make([]string, len(settings))
	for index, setting := range settings {
		lines[index] = fmt.Sprintf("%s%-*s  # %s", indent, assignmentWidth, assignments[index], defaultComment(setting))
	}

	return lines
}

func defaultComment(setting unmanagedSetting) string {
	if setting.defaultUnread {
		return "default not read yet"
	}

	var compact bytes.Buffer
	if setting.defaultValue == nil || json.Compact(&compact, setting.defaultValue) != nil {
		return "no default"
	}

	return "default: " + compact.String()
}

func hclValue(value attr.Value) string {
	switch typed := value.(type) {
	case types.Bool:
		return strconv.FormatBool(typed.ValueBool())
	case types.Int64:
		return strconv.FormatInt(typed.ValueInt64(), 10)
	case types.String:
		return hclString(typed.ValueString())
	case types.Set:
		return hclStrings(typed.Elements(), true)
	case types.List:
		return hclStrings(typed.Elements(), false)
	default:
		return value.String()
	}
}

func hclStrings(elements []attr.Value, sorted bool) string {
	values := make([]string, 0, len(elements))
	for _, element := range elements {
		if typed, ok := element.(types.String); ok {
			values = append(values, typed.ValueString())
		}
	}

	if sorted {
		sort.Strings(values)
	}

	quoted := make([]string, len(values))
	for index, value := range values {
		quoted[index] = hclString(value)
	}

	return "[" + strings.Join(quoted, ", ") + "]"
}

func hclString(value string) string {
	quoted := strconv.Quote(value)
	quoted = strings.ReplaceAll(quoted, "${", "$${")

	return strings.ReplaceAll(quoted, "%{", "%%{")
}
