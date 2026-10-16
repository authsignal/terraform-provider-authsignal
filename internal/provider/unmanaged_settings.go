package provider

import (
	"bytes"
	"context"
	"encoding/json"
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

	encoded, err := json.Marshal(defaults)
	if err != nil {
		var diags diag.Diagnostics
		diags.AddError("Unable to Store the Management API's Defaults", err.Error())
		return diags
	}

	return private.SetKey(ctx, defaultsPrivateStateKey, encoded)
}

// An Optional+Computed setting the configuration leaves out keeps whatever the tenant has, so no plan
// would show a change made outside Terraform. The plan fails instead when the tenant's value is neither
// blank nor the default the Management API reports, which the last read stored in private state.
func checkUnmanagedSettings(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	resourceSchema, ok := req.Plan.Schema.(schema.Schema)
	if !ok {
		return
	}

	encoded, diags := req.Private.GetKey(ctx, defaultsPrivateStateKey)
	resp.Diagnostics.Append(diags...)

	var defaults authsignal.Defaults
	if len(encoded) > 0 {
		if err := json.Unmarshal(encoded, &defaults); err != nil {
			resp.Diagnostics.AddError("Unable to Read the Management API's Defaults", err.Error())
			return
		}
	}

	resp.Diagnostics.Append(unmanagedSettingsDiagnostics(ctx, resourceSchema, req.Config.Raw, req.State.Raw, req.Plan.Raw, defaults)...)
}

type unmanagedSetting struct {
	name         string
	value        attr.Value
	defaultValue json.RawMessage
}

func unmanagedSettingsDiagnostics(ctx context.Context, resourceSchema schema.Schema, config tftypes.Value, state tftypes.Value, plan tftypes.Value, defaults authsignal.Defaults) diag.Diagnostics {
	var diags diag.Diagnostics

	if state.IsNull() || plan.IsNull() {
		return diags
	}

	objects := make([]types.Object, 0, 2)
	for _, raw := range []tftypes.Value{config, state} {
		value, err := resourceSchema.Type().ValueFromTerraform(ctx, raw)
		if err != nil {
			diags.AddError("Unable to Check for Unmanaged Settings", err.Error())
			return diags
		}

		object, ok := value.(types.Object)
		if !ok {
			return diags
		}

		objects = append(objects, object)
	}

	settings := unmanagedSettingsIn(resourceSchema.Attributes, objects[0].Attributes(), objects[1].Attributes(), defaults)
	if len(settings) > 0 {
		diags.Append(unmanagedSettingsDiagnostic(settings))
	}

	return diags
}

func unmanagedSettingsIn(attributes map[string]schema.Attribute, config map[string]attr.Value, state map[string]attr.Value, defaults authsignal.Defaults) []unmanagedSetting {
	var settings []unmanagedSetting

	for _, name := range sortedNames(attributes) {
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

		defaultValue := defaults[jsonFieldName(name)]
		if matchesDefault(value, defaultValue) {
			continue
		}

		settings = append(settings, unmanagedSetting{name: name, value: value, defaultValue: defaultValue})
	}

	return settings
}

// A schema default already plans a change when the configuration leaves the setting out.
func isUnmanagedWhenLeftOut(attribute schema.Attribute) bool {
	if !attribute.IsOptional() || !attribute.IsComputed() {
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
		"The tenant has values for these settings, but the configuration doesn't set them:\n\n"+
			strings.Join(hclSettingLines(settings, "  "), "\n")+"\n\n"+
			"Changes made to them outside Terraform won't show in a plan, and a tenant built from this configuration "+
			"won't get these values. Add them to the configuration.",
	)
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
		lines[index] = fmt.Sprintf("%s%-*s  # %s", indent, assignmentWidth, assignments[index], defaultComment(setting.defaultValue))
	}

	return lines
}

func defaultComment(defaultValue json.RawMessage) string {
	var compact bytes.Buffer
	if defaultValue == nil || json.Compact(&compact, defaultValue) != nil {
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
