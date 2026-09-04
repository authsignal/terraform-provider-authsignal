package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

var (
	_ basetypes.StringTypable                    = FlowType{}
	_ basetypes.StringValuable                   = FlowValue{}
	_ basetypes.StringValuableWithSemanticEquals = FlowValue{}
)

// FlowType is the attribute type of an action flow document: a JSON string whose semantic equality
// ignores formatting, key order, rule order and other differences that do not change the flow. It is
// structured like jsontypes.Normalized but implemented here so the provider does not take on another
// dependency.
type FlowType struct {
	basetypes.StringType
}

func (t FlowType) String() string {
	return "authsignal.FlowType"
}

func (t FlowType) ValueType(_ context.Context) attr.Value {
	return FlowValue{}
}

func (t FlowType) Equal(o attr.Type) bool {
	other, ok := o.(FlowType)
	if !ok {
		return false
	}

	return t.StringType.Equal(other.StringType)
}

func (t FlowType) ValueFromString(_ context.Context, in basetypes.StringValue) (basetypes.StringValuable, diag.Diagnostics) {
	return FlowValue{StringValue: in}, nil
}

func (t FlowType) ValueFromTerraform(ctx context.Context, in tftypes.Value) (attr.Value, error) {
	attrValue, err := t.StringType.ValueFromTerraform(ctx, in)
	if err != nil {
		return nil, err
	}

	stringValue, ok := attrValue.(basetypes.StringValue)
	if !ok {
		return nil, fmt.Errorf("unexpected value type of %T", attrValue)
	}

	stringValuable, diags := t.ValueFromString(ctx, stringValue)
	if diags.HasError() {
		return nil, fmt.Errorf("unexpected error converting StringValue to StringValuable: %v", diags)
	}

	return stringValuable, nil
}

// FlowValue is an action flow document held as a JSON string.
type FlowValue struct {
	basetypes.StringValue
}

func NewFlowNull() FlowValue {
	return FlowValue{StringValue: basetypes.NewStringNull()}
}

func NewFlowUnknown() FlowValue {
	return FlowValue{StringValue: basetypes.NewStringUnknown()}
}

func NewFlowValue(value string) FlowValue {
	return FlowValue{StringValue: basetypes.NewStringValue(value)}
}

func (v FlowValue) Type(_ context.Context) attr.Type {
	return FlowType{}
}

func (v FlowValue) Equal(o attr.Value) bool {
	other, ok := o.(FlowValue)
	if !ok {
		return false
	}

	return v.StringValue.Equal(other.StringValue)
}

// StringSemanticEquals canonicalises both flows and deep-compares the result. Two flows are equal
// when their nodes match in order and they carry the same rules, whatever the JSON formatting, key
// order, rule order, or `1.0` versus `1`.
func (v FlowValue) StringSemanticEquals(_ context.Context, newValuable basetypes.StringValuable) (bool, diag.Diagnostics) {
	var diags diag.Diagnostics

	newValue, ok := newValuable.(FlowValue)
	if !ok {
		diags.AddError(
			"Semantic Equality Check Error",
			"An unexpected value type was received while performing semantic equality checks. "+
				"Please report this to the provider developers.\n\n"+
				"Expected Value Type: "+fmt.Sprintf("%T", v)+"\n"+
				"Got Value Type: "+fmt.Sprintf("%T", newValuable),
		)

		return false, diags
	}

	if v.IsNull() || v.IsUnknown() || newValue.IsNull() || newValue.IsUnknown() {
		return v.StringValue.Equal(newValue.StringValue), diags
	}

	return flowsEqual(v.ValueString(), newValue.ValueString()), diags
}
