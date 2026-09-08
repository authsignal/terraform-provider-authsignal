package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestFlowKeepsStateWhenEqual(t *testing.T) {
	const stored = `{"actionNodes":[{"nodeId":"c","nodeType":"COMPLETE"}],"rules":[{"ruleId":"a","name":"A"}]}`
	const reformatted = `{
	  "rules": [{"name": "A", "ruleId": "a"}],
	  "actionNodes": [{"nodeType": "COMPLETE", "nodeId": "c"}]
	}`
	const changed = `{"actionNodes":[{"nodeId":"c","nodeType":"BLOCK"}],"rules":[{"ruleId":"a","name":"A"}]}`

	testCases := []struct {
		name     string
		state    types.String
		plan     types.String
		expected types.String
	}{
		{"the same document formatted differently", types.StringValue(stored), types.StringValue(reformatted), types.StringValue(stored)},
		{"a different document", types.StringValue(stored), types.StringValue(changed), types.StringValue(changed)},
		{"a null state, as on create", types.StringNull(), types.StringValue(stored), types.StringValue(stored)},
		{"a null plan", types.StringValue(stored), types.StringNull(), types.StringNull()},
		{"an unknown plan", types.StringValue(stored), types.StringUnknown(), types.StringUnknown()},
		{"an unknown state", types.StringUnknown(), types.StringValue(stored), types.StringValue(stored)},
		{"a null state and plan", types.StringNull(), types.StringNull(), types.StringNull()},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			req := planmodifier.StringRequest{StateValue: testCase.state, PlanValue: testCase.plan}
			resp := &planmodifier.StringResponse{PlanValue: req.PlanValue}

			flowKeepsStateWhenEqual{}.PlanModifyString(context.Background(), req, resp)

			if resp.Diagnostics.HasError() {
				t.Fatal(resp.Diagnostics)
			}

			if !resp.PlanValue.Equal(testCase.expected) {
				t.Fatalf("expected %v, got %v", testCase.expected, resp.PlanValue)
			}
		})
	}
}
