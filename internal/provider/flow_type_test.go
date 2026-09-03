package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestFlowValueSemanticEquality(t *testing.T) {
	base := `[
	  {"nodeId":"r","nodeType":"RULE","parentNodeIds":[],"ruleChildNodeIds":[["a","c"],["b","c"]],"elseChildNodeId":"c",
	   "rules":[{"ruleId":"a","name":"A","conditions":{"and":[{"==":[{"var":"ip.isAnonymous"},true]}]}},{"ruleId":"b","name":"B"}]},
	  {"nodeId":"c","nodeType":"COMPLETE","parentNodeIds":["r"],"weight":1}
	]`

	testCases := []struct {
		name  string
		other string
		equal bool
	}{
		{
			name:  "identical",
			other: base,
			equal: true,
		},
		{
			name:  "whitespace and key order",
			other: `[{"rules":[{"conditions":{"and":[{"==":[{"var":"ip.isAnonymous"},true]}]},"name":"A","ruleId":"a"},{"name":"B","ruleId":"b"}],"elseChildNodeId":"c","ruleChildNodeIds":[["a","c"],["b","c"]],"parentNodeIds":[],"nodeType":"RULE","nodeId":"r"},{"weight":1,"parentNodeIds":["r"],"nodeType":"COMPLETE","nodeId":"c"}]`,
			equal: true,
		},
		{
			name: "rule order",
			other: `[{"nodeId":"r","nodeType":"RULE","parentNodeIds":[],"ruleChildNodeIds":[["a","c"],["b","c"]],"elseChildNodeId":"c",
			   "rules":[{"ruleId":"b","name":"B"},{"ruleId":"a","name":"A","conditions":{"and":[{"==":[{"var":"ip.isAnonymous"},true]}]}}]},
			  {"nodeId":"c","nodeType":"COMPLETE","parentNodeIds":["r"],"weight":1}]`,
			equal: true,
		},
		{
			name: "null conditions equal absent conditions",
			other: `[{"nodeId":"r","nodeType":"RULE","parentNodeIds":[],"ruleChildNodeIds":[["a","c"],["b","c"]],"elseChildNodeId":"c",
			   "rules":[{"ruleId":"a","name":"A","conditions":{"and":[{"==":[{"var":"ip.isAnonymous"},true]}]}},{"ruleId":"b","name":"B","conditions":null}]},
			  {"nodeId":"c","nodeType":"COMPLETE","parentNodeIds":["r"],"weight":1}]`,
			equal: true,
		},
		{
			name: "1.0 equals 1",
			other: `[{"nodeId":"r","nodeType":"RULE","parentNodeIds":[],"ruleChildNodeIds":[["a","c"],["b","c"]],"elseChildNodeId":"c",
			   "rules":[{"ruleId":"a","name":"A","conditions":{"and":[{"==":[{"var":"ip.isAnonymous"},true]}]}},{"ruleId":"b","name":"B"}]},
			  {"nodeId":"c","nodeType":"COMPLETE","parentNodeIds":["r"],"weight":1.0}]`,
			equal: true,
		},
		{
			name: "html escaping",
			other: `[{"nodeId":"r","nodeType":"RULE","parentNodeIds":[],"ruleChildNodeIds":[["a","c"],["b","c"]],"elseChildNodeId":"c",
			   "rules":[{"ruleId":"a","name":"A","conditions":{"and":[{"==":[{"var":"ip.isAnonymous"},true]}]}},{"ruleId":"b","name":"B"}]},
			  {"nodeId":"c","nodeType":"COMPLETE","parentNodeIds":["r"],"weight":1}]`,
			equal: true,
		},
		{
			name: "a condition changed",
			other: `[{"nodeId":"r","nodeType":"RULE","parentNodeIds":[],"ruleChildNodeIds":[["a","c"],["b","c"]],"elseChildNodeId":"c",
			   "rules":[{"ruleId":"a","name":"A","conditions":{"and":[{"==":[{"var":"ip.isAnonymous"},false]}]}},{"ruleId":"b","name":"B"}]},
			  {"nodeId":"c","nodeType":"COMPLETE","parentNodeIds":["r"],"weight":1}]`,
			equal: false,
		},
		{
			name: "a rule renamed",
			other: `[{"nodeId":"r","nodeType":"RULE","parentNodeIds":[],"ruleChildNodeIds":[["a","c"],["b","c"]],"elseChildNodeId":"c",
			   "rules":[{"ruleId":"a","name":"A","conditions":{"and":[{"==":[{"var":"ip.isAnonymous"},true]}]}},{"ruleId":"b","name":"B renamed"}]},
			  {"nodeId":"c","nodeType":"COMPLETE","parentNodeIds":["r"],"weight":1}]`,
			equal: false,
		},
		{
			name: "node order is the graph",
			other: `[{"nodeId":"c","nodeType":"COMPLETE","parentNodeIds":["r"],"weight":1},
			  {"nodeId":"r","nodeType":"RULE","parentNodeIds":[],"ruleChildNodeIds":[["a","c"],["b","c"]],"elseChildNodeId":"c",
			   "rules":[{"ruleId":"a","name":"A","conditions":{"and":[{"==":[{"var":"ip.isAnonymous"},true]}]}},{"ruleId":"b","name":"B"}]}]`,
			equal: false,
		},
		{
			name: "a node field changed",
			other: `[{"nodeId":"r","nodeType":"RULE","parentNodeIds":[],"ruleChildNodeIds":[["a","c"],["b","c"]],"elseChildNodeId":"c",
			   "rules":[{"ruleId":"a","name":"A","conditions":{"and":[{"==":[{"var":"ip.isAnonymous"},true]}]}},{"ruleId":"b","name":"B"}]},
			  {"nodeId":"c","nodeType":"COMPLETE","parentNodeIds":["r"],"weight":2}]`,
			equal: false,
		},
		{
			name:  "invalid flow is never equal",
			other: `[{"nodeId":"r","nodeType":"RULE","ruleChildNodeIds":[["a","c"]]}]`,
			equal: false,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			equal, diags := NewFlowValue(base).StringSemanticEquals(context.Background(), NewFlowValue(testCase.other))
			if diags.HasError() {
				t.Fatalf("unexpected diagnostics: %v", diags)
			}

			if equal != testCase.equal {
				t.Fatalf("expected equal=%v, got %v", testCase.equal, equal)
			}
		})
	}
}

func TestFlowValueAbsentRulesEqualEmptyRules(t *testing.T) {
	withEmpty := `[{"nodeId":"r","nodeType":"RULE","ruleChildNodeIds":[],"elseChildNodeId":"c","rules":[]},{"nodeId":"c","nodeType":"COMPLETE"}]`
	without := `[{"nodeId":"r","nodeType":"RULE","ruleChildNodeIds":[],"elseChildNodeId":"c"},{"nodeId":"c","nodeType":"COMPLETE"}]`

	equal, diags := NewFlowValue(withEmpty).StringSemanticEquals(context.Background(), NewFlowValue(without))
	if diags.HasError() || !equal {
		t.Fatalf("expected an empty rules array to equal no rules array, equal=%v diags=%v", equal, diags)
	}
}

func TestFlowValueNullAndUnknownCompareStrictly(t *testing.T) {
	ctx := context.Background()

	if equal, _ := NewFlowNull().StringSemanticEquals(ctx, NewFlowNull()); !equal {
		t.Fatal("null must equal null")
	}

	if equal, _ := NewFlowNull().StringSemanticEquals(ctx, NewFlowValue("[]")); equal {
		t.Fatal("null must not equal a value")
	}

	if equal, _ := NewFlowUnknown().StringSemanticEquals(ctx, NewFlowValue("[]")); equal {
		t.Fatal("unknown must not equal a value")
	}

	_, diags := NewFlowValue("[]").StringSemanticEquals(ctx, types.StringValue("[]"))
	if !diags.HasError() {
		t.Fatal("a plain string is the wrong value type and must be reported")
	}
}

func TestFlowTypeConvertsToFlowValue(t *testing.T) {
	ctx := context.Background()

	fromTerraform, err := FlowType{}.ValueFromTerraform(ctx, tftypes.NewValue(tftypes.String, "[]"))
	if err != nil {
		t.Fatal(err)
	}

	if converted, ok := fromTerraform.(FlowValue); !ok || converted.ValueString() != "[]" {
		t.Fatalf("expected a FlowValue holding [], got %T %v", fromTerraform, fromTerraform)
	}

	valuable, diags := FlowType{}.ValueFromString(ctx, basetypes.NewStringValue("[]"))
	if diags.HasError() {
		t.Fatal(diags)
	}

	flow, ok := valuable.(FlowValue)
	if !ok || flow.ValueString() != "[]" {
		t.Fatalf("expected a FlowValue holding [], got %T %v", valuable, valuable)
	}

	if !flow.Type(ctx).Equal(FlowType{}) {
		t.Fatalf("expected FlowType, got %v", flow.Type(ctx))
	}

	if (FlowType{}).Equal(types.StringType) {
		t.Fatal("FlowType must not equal the plain string type")
	}

	if _, ok := (FlowType{}).ValueType(ctx).(FlowValue); !ok {
		t.Fatal("FlowType's value type must be FlowValue")
	}
}

func TestFlowValidatorReportsEveryBreachAtTheAttribute(t *testing.T) {
	ctx := context.Background()
	flow := `[{"nodeId":"c","nodeType":"COMPLETE","rules":[]},{"nodeId":"c","nodeType":"COMPLETE"}]`

	request := validator.StringRequest{
		Path:        path.Root("flow"),
		ConfigValue: types.StringValue(flow),
	}
	response := &validator.StringResponse{}

	flowValidator{}.ValidateString(ctx, request, response)

	if response.Diagnostics.ErrorsCount() != 2 {
		t.Fatalf("expected one diagnostic per broken invariant, got %v", response.Diagnostics)
	}

	for _, diagnostic := range response.Diagnostics {
		if diagnostic.Summary() != "Invalid action flow" {
			t.Fatalf("unexpected summary %q", diagnostic.Summary())
		}
	}

	if !containsDetail(response, "[0].rules: only RULE nodes carry rules") || !containsDetail(response, `[1].nodeId: duplicates [0].nodeId ("c")`) {
		t.Fatalf("expected JSON paths in the details, got %v", response.Diagnostics)
	}

	valid := &validator.StringResponse{}
	flowValidator{}.ValidateString(ctx, validator.StringRequest{Path: path.Root("flow"), ConfigValue: types.StringValue(contractFlow)}, valid)
	if valid.Diagnostics.HasError() {
		t.Fatalf("the contract flow must validate: %v", valid.Diagnostics)
	}

	skipped := &validator.StringResponse{}
	flowValidator{}.ValidateString(ctx, validator.StringRequest{Path: path.Root("flow"), ConfigValue: types.StringUnknown()}, skipped)
	flowValidator{}.ValidateString(ctx, validator.StringRequest{Path: path.Root("flow"), ConfigValue: types.StringNull()}, skipped)
	if skipped.Diagnostics.HasError() {
		t.Fatalf("null and unknown values are not validated: %v", skipped.Diagnostics)
	}
}

func containsDetail(response *validator.StringResponse, detail string) bool {
	for _, diagnostic := range response.Diagnostics {
		if diagnostic.Detail() == detail {
			return true
		}
	}

	return false
}
