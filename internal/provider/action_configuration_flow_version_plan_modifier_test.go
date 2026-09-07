package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestFlowVersionFollowsFlow(t *testing.T) {
	const stored = `{"actionNodes":[{"nodeId":"c","nodeType":"COMPLETE"}],"rules":[]}`
	const reformatted = `{"rules":[],"actionNodes":[{"nodeType":"COMPLETE","nodeId":"c"}]}`
	const changed = `{"actionNodes":[{"nodeId":"c","nodeType":"BLOCK"}],"rules":[]}`

	testCases := []struct {
		name       string
		actionType string
		stateFlow  string
		planFlow   string
		expected   types.Int64
	}{
		// The framework marks a computed attribute unknown before the plan modifiers run.
		{"the flow changed", actionTypeFlow, stored, changed, types.Int64Unknown()},
		{"the flow is the same document", actionTypeFlow, stored, reformatted, types.Int64Value(4)},
		{"a CLASSIC action has no flow", actionTypeClassic, "", "", types.Int64Null()},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			resourceSchema := actionConfigurationSchema(t)

			stateVersion := tftypes.NewValue(tftypes.Number, nil)
			if testCase.actionType == actionTypeFlow {
				stateVersion = tftypes.NewValue(tftypes.Number, 4)
			}

			state := actionConfigurationObject(t, resourceSchema, map[string]tftypes.Value{
				"action_code":  tftypes.NewValue(tftypes.String, "sign-in"),
				"action_type":  optionalString(testCase.actionType),
				"flow":         optionalString(testCase.stateFlow),
				"flow_version": stateVersion,
			})

			plan := actionConfigurationObject(t, resourceSchema, map[string]tftypes.Value{
				"action_code":  tftypes.NewValue(tftypes.String, "sign-in"),
				"action_type":  optionalString(testCase.actionType),
				"flow":         optionalString(testCase.planFlow),
				"flow_version": tftypes.NewValue(tftypes.Number, tftypes.UnknownValue),
			})

			req := planmodifier.Int64Request{
				Path:       path.Root("flow_version"),
				State:      tfsdk.State{Schema: resourceSchema, Raw: state},
				Plan:       tfsdk.Plan{Schema: resourceSchema, Raw: plan},
				StateValue: types.Int64Null(),
				PlanValue:  types.Int64Unknown(),
			}

			if testCase.actionType == actionTypeFlow {
				req.StateValue = types.Int64Value(4)
			}

			resp := &planmodifier.Int64Response{PlanValue: req.PlanValue}

			flowVersionFollowsFlow{}.PlanModifyInt64(context.Background(), req, resp)

			if resp.Diagnostics.HasError() {
				t.Fatal(resp.Diagnostics)
			}

			if !resp.PlanValue.Equal(testCase.expected) {
				t.Fatalf("expected %v, got %v", testCase.expected, resp.PlanValue)
			}
		})
	}
}
