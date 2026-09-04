package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestFlowValueSemanticEquality(t *testing.T) {
	base := `{
	  "actionNodes": [
	    {"nodeId":"r","nodeType":"RULE","parentNodeIds":[],"ruleChildNodeIds":[["a","c"],["b","c"]],"elseChildNodeId":"c"},
	    {"nodeId":"c","nodeType":"COMPLETE","parentNodeIds":["r"],"weight":1}
	  ],
	  "rules": [
	    {"ruleId":"a","name":"A","conditions":{"and":[{"==":[{"var":"ip.isAnonymous"},true]}]}},
	    {"ruleId":"b","name":"B"}
	  ]
	}`

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
			other: `{"rules":[{"conditions":{"and":[{"==":[{"var":"ip.isAnonymous"},true]}]},"name":"A","ruleId":"a"},{"name":"B","ruleId":"b"}],"actionNodes":[{"elseChildNodeId":"c","ruleChildNodeIds":[["a","c"],["b","c"]],"parentNodeIds":[],"nodeType":"RULE","nodeId":"r"},{"weight":1,"parentNodeIds":["r"],"nodeType":"COMPLETE","nodeId":"c"}]}`,
			equal: true,
		},
		{
			// Flow rules all carry priority 0, so the order the API lists them in is arbitrary and
			// can differ between two reads of an unchanged flow.
			name: "the order the API lists rules in",
			other: `{"actionNodes":[{"nodeId":"r","nodeType":"RULE","parentNodeIds":[],"ruleChildNodeIds":[["a","c"],["b","c"]],"elseChildNodeId":"c"},
			   {"nodeId":"c","nodeType":"COMPLETE","parentNodeIds":["r"],"weight":1}],
			  "rules":[{"ruleId":"b","name":"B"},{"ruleId":"a","name":"A","conditions":{"and":[{"==":[{"var":"ip.isAnonymous"},true]}]}}]}`,
			equal: true,
		},
		{
			name: "null conditions equal absent conditions",
			other: `{"actionNodes":[{"nodeId":"r","nodeType":"RULE","parentNodeIds":[],"ruleChildNodeIds":[["a","c"],["b","c"]],"elseChildNodeId":"c"},
			   {"nodeId":"c","nodeType":"COMPLETE","parentNodeIds":["r"],"weight":1}],
			  "rules":[{"ruleId":"a","name":"A","conditions":{"and":[{"==":[{"var":"ip.isAnonymous"},true]}]}},{"ruleId":"b","name":"B","conditions":null}]}`,
			equal: true,
		},
		{
			name: "1.0 equals 1",
			other: `{"actionNodes":[{"nodeId":"r","nodeType":"RULE","parentNodeIds":[],"ruleChildNodeIds":[["a","c"],["b","c"]],"elseChildNodeId":"c"},
			   {"nodeId":"c","nodeType":"COMPLETE","parentNodeIds":["r"],"weight":1.0}],
			  "rules":[{"ruleId":"a","name":"A","conditions":{"and":[{"==":[{"var":"ip.isAnonymous"},true]}]}},{"ruleId":"b","name":"B"}]}`,
			equal: true,
		},
		{
			name: "a condition changed",
			other: `{"actionNodes":[{"nodeId":"r","nodeType":"RULE","parentNodeIds":[],"ruleChildNodeIds":[["a","c"],["b","c"]],"elseChildNodeId":"c"},
			   {"nodeId":"c","nodeType":"COMPLETE","parentNodeIds":["r"],"weight":1}],
			  "rules":[{"ruleId":"a","name":"A","conditions":{"and":[{"==":[{"var":"ip.isAnonymous"},false]}]}},{"ruleId":"b","name":"B"}]}`,
			equal: false,
		},
		{
			name: "a rule renamed",
			other: `{"actionNodes":[{"nodeId":"r","nodeType":"RULE","parentNodeIds":[],"ruleChildNodeIds":[["a","c"],["b","c"]],"elseChildNodeId":"c"},
			   {"nodeId":"c","nodeType":"COMPLETE","parentNodeIds":["r"],"weight":1}],
			  "rules":[{"ruleId":"a","name":"A","conditions":{"and":[{"==":[{"var":"ip.isAnonymous"},true]}]}},{"ruleId":"b","name":"B renamed"}]}`,
			equal: false,
		},
		{
			name: "a rule dropped",
			other: `{"actionNodes":[{"nodeId":"r","nodeType":"RULE","parentNodeIds":[],"ruleChildNodeIds":[["a","c"],["b","c"]],"elseChildNodeId":"c"},
			   {"nodeId":"c","nodeType":"COMPLETE","parentNodeIds":["r"],"weight":1}],
			  "rules":[{"ruleId":"a","name":"A","conditions":{"and":[{"==":[{"var":"ip.isAnonymous"},true]}]}}]}`,
			equal: false,
		},
		{
			name: "node order is the graph",
			other: `{"actionNodes":[{"nodeId":"c","nodeType":"COMPLETE","parentNodeIds":["r"],"weight":1},
			   {"nodeId":"r","nodeType":"RULE","parentNodeIds":[],"ruleChildNodeIds":[["a","c"],["b","c"]],"elseChildNodeId":"c"}],
			  "rules":[{"ruleId":"a","name":"A","conditions":{"and":[{"==":[{"var":"ip.isAnonymous"},true]}]}},{"ruleId":"b","name":"B"}]}`,
			equal: false,
		},
		{
			name: "a node field changed",
			other: `{"actionNodes":[{"nodeId":"r","nodeType":"RULE","parentNodeIds":[],"ruleChildNodeIds":[["a","c"],["b","c"]],"elseChildNodeId":"c"},
			   {"nodeId":"c","nodeType":"COMPLETE","parentNodeIds":["r"],"weight":2}],
			  "rules":[{"ruleId":"a","name":"A","conditions":{"and":[{"==":[{"var":"ip.isAnonymous"},true]}]}},{"ruleId":"b","name":"B"}]}`,
			equal: false,
		},
		{
			name: "a null node field is not an absent field",
			other: `{"actionNodes":[{"nodeId":"r","nodeType":"RULE","parentNodeIds":[],"ruleChildNodeIds":[["a","c"],["b","c"]],"elseChildNodeId":"c"},
			   {"nodeId":"c","nodeType":"COMPLETE","parentNodeIds":["r"],"weight":1,"message":null}],
			  "rules":[{"ruleId":"a","name":"A","conditions":{"and":[{"==":[{"var":"ip.isAnonymous"},true]}]}},{"ruleId":"b","name":"B"}]}`,
			equal: false,
		},
		{
			name:  "the old embedded shape is not this flow",
			other: `[{"nodeId":"r","nodeType":"RULE","ruleChildNodeIds":[["a","c"]],"rules":[{"ruleId":"a","name":"A"}]}]`,
			equal: false,
		},
		{
			name:  "trailing JSON is not ignored",
			other: base + ` {}`,
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

func TestFlowValueKeepsLargeAdjacentIntegersDistinct(t *testing.T) {
	one := `{"actionNodes":[{"nodeId":"c","nodeType":"COMPLETE","weight":9007199254740992}],"rules":[]}`
	two := `{"actionNodes":[{"nodeId":"c","nodeType":"COMPLETE","weight":9007199254740993}],"rules":[]}`

	equal, diags := NewFlowValue(one).StringSemanticEquals(context.Background(), NewFlowValue(two))
	if diags.HasError() || equal {
		t.Fatalf("expected adjacent integers above 2^53 to differ, equal=%v diags=%v", equal, diags)
	}
}

// jsonencode and Go's json.Marshal both HTML-escape, but a hand-written file need not.
func TestFlowValueHtmlEscapingIsNotAChange(t *testing.T) {
	escaped := `{"actionNodes":[{"nodeId":"r","nodeType":"RULE","ruleChildNodeIds":[["a","c"]],"elseChildNodeId":"c"},{"nodeId":"c","nodeType":"COMPLETE"}],"rules":[{"ruleId":"a","name":"Tom \u0026 Jerry \u003c3"}]}`
	literal := `{"actionNodes":[{"nodeId":"r","nodeType":"RULE","ruleChildNodeIds":[["a","c"]],"elseChildNodeId":"c"},{"nodeId":"c","nodeType":"COMPLETE"}],"rules":[{"ruleId":"a","name":"Tom & Jerry <3"}]}`

	equal, diags := NewFlowValue(escaped).StringSemanticEquals(context.Background(), NewFlowValue(literal))
	if diags.HasError() || !equal {
		t.Fatalf("expected HTML-escaped and literal characters to compare equal, equal=%v diags=%v", equal, diags)
	}
}

// Comparison is structural, so a document that breaches an invariant still compares predictably
// rather than being unequal to everything, including itself.
func TestFlowValueComparesDocumentsThatBreachAnInvariant(t *testing.T) {
	strayRule := `{"actionNodes":[{"nodeId":"c","nodeType":"COMPLETE"}],"rules":[{"ruleId":"stray","name":"Made in the portal"}]}`

	if _, errs := parseFlow(strayRule); len(errs) == 0 {
		t.Fatal("expected the unreferenced rule to breach the reference invariant")
	}

	equal, diags := NewFlowValue(strayRule).StringSemanticEquals(context.Background(), NewFlowValue(strayRule))
	if diags.HasError() || !equal {
		t.Fatalf("a document must equal itself whether or not it validates, equal=%v diags=%v", equal, diags)
	}
}

func TestFlowValueNullAndUnknownCompareStrictly(t *testing.T) {
	ctx := context.Background()

	if equal, _ := NewFlowNull().StringSemanticEquals(ctx, NewFlowNull()); !equal {
		t.Fatal("null must equal null")
	}

	if equal, _ := NewFlowNull().StringSemanticEquals(ctx, NewFlowValue("{}")); equal {
		t.Fatal("null must not equal a value")
	}

	if equal, _ := NewFlowUnknown().StringSemanticEquals(ctx, NewFlowValue("{}")); equal {
		t.Fatal("unknown must not equal a value")
	}

	_, diags := NewFlowValue("{}").StringSemanticEquals(ctx, types.StringValue("{}"))
	if !diags.HasError() {
		t.Fatal("a plain string is the wrong value type and must be reported")
	}
}

func TestFlowTypeConvertsToFlowValue(t *testing.T) {
	ctx := context.Background()

	fromTerraform, err := FlowType{}.ValueFromTerraform(ctx, tftypes.NewValue(tftypes.String, "{}"))
	if err != nil {
		t.Fatal(err)
	}

	if converted, ok := fromTerraform.(FlowValue); !ok || converted.ValueString() != "{}" {
		t.Fatalf("expected a FlowValue holding [], got %T %v", fromTerraform, fromTerraform)
	}

	valuable, diags := FlowType{}.ValueFromString(ctx, basetypes.NewStringValue("{}"))
	if diags.HasError() {
		t.Fatal(diags)
	}

	flow, ok := valuable.(FlowValue)
	if !ok || flow.ValueString() != "{}" {
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
	flow := `{"actionNodes":[{"nodeId":"c","nodeType":"COMPLETE"},{"nodeId":"c","nodeType":"COMPLETE"}],"rules":[{"ruleId":"has space","name":"A"}]}`

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

	if !containsDetail(response, `actionNodes[1].nodeId: duplicates actionNodes[0].nodeId ("c")`) ||
		!containsDetail(response, "rules[0].ruleId: must be 1-64 characters of letters, digits, `_` or `-`") {
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

// The RequiresReplaceIf modifier only reaches the provider's function when plan and state differ
// on an existing resource, so the requests below mimic that with non-null raw plan and state.
func TestActionTypeReplaceDecision(t *testing.T) {
	ctx := context.Background()
	existing := tftypes.NewValue(tftypes.Object{}, map[string]tftypes.Value{})

	testCases := []struct {
		name            string
		config          types.String
		plan            types.String
		state           types.String
		requiresReplace bool
		errors          bool
	}{
		{
			name:   "state from a provider without action_type is not a type change",
			config: types.StringNull(),
			plan:   types.StringValue(actionTypeClassic),
			state:  types.StringNull(),
		},
		{
			name:   "flow action on the server, action_type not configured, fails the plan",
			config: types.StringNull(),
			plan:   types.StringValue(actionTypeClassic),
			state:  types.StringValue(actionTypeFlow),
			errors: true,
		},
		{
			name:            "explicit CLASSIC over a flow action replaces",
			config:          types.StringValue(actionTypeClassic),
			plan:            types.StringValue(actionTypeClassic),
			state:           types.StringValue(actionTypeFlow),
			requiresReplace: true,
		},
		{
			name:            "explicit flow type over a classic action replaces",
			config:          types.StringValue(actionTypeFlow),
			plan:            types.StringValue(actionTypeFlow),
			state:           types.StringValue(actionTypeClassic),
			requiresReplace: true,
		},
		{
			name:   "unchanged type does nothing",
			config: types.StringValue(actionTypeFlow),
			plan:   types.StringValue(actionTypeFlow),
			state:  types.StringValue(actionTypeFlow),
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			request := planmodifier.StringRequest{
				Path:        path.Root("action_type"),
				ConfigValue: testCase.config,
				PlanValue:   testCase.plan,
				StateValue:  testCase.state,
				Plan:        tfsdk.Plan{Raw: existing},
				State:       tfsdk.State{Raw: existing},
			}
			response := &planmodifier.StringResponse{PlanValue: testCase.plan}

			actionTypeRequiresReplace().PlanModifyString(ctx, request, response)

			if response.RequiresReplace != testCase.requiresReplace {
				t.Fatalf("expected requiresReplace=%v, got %v", testCase.requiresReplace, response.RequiresReplace)
			}

			if response.Diagnostics.HasError() != testCase.errors {
				t.Fatalf("expected errors=%v, got %v", testCase.errors, response.Diagnostics)
			}
		})
	}
}
