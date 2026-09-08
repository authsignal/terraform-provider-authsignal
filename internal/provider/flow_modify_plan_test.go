package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestFlowModifyPlan(t *testing.T) {
	const stored = `{"actionNodes":[{"nodeId":"c","nodeType":"COMPLETE"}],"rules":[]}`
	const reformatted = `{
  "rules": [],
  "actionNodes": [{ "nodeType": "COMPLETE", "nodeId": "c" }]
}`
	const changed = `{"actionNodes":[{"nodeId":"c","nodeType":"BLOCK"}],"rules":[]}`

	testCases := []struct {
		name            string
		planFlow        string
		planActionCode  string
		create          bool
		destroy         bool
		requiresReplace bool
		expectState     bool
	}{
		{name: "the flow is the same document", planFlow: reformatted, expectState: true},
		{name: "the flow changed", planFlow: changed},
		{name: "the action code changed", planFlow: reformatted, planActionCode: "sign-up"},
		{name: "the resource is being created", planFlow: reformatted, create: true},
		{name: "the resource is being destroyed", planFlow: reformatted, destroy: true},
		{name: "the resource is being replaced", planFlow: reformatted, requiresReplace: true},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			resourceSchema := flowSchema(t)

			planActionCode := testCase.planActionCode
			if planActionCode == "" {
				planActionCode = "sign-in"
			}

			state := resourceObject(t, resourceSchema, map[string]tftypes.Value{
				"action_code":            tftypes.NewValue(tftypes.String, "sign-in"),
				"flow":                   tftypes.NewValue(tftypes.String, stored),
				"flow_version":           tftypes.NewValue(tftypes.Number, 4),
				"tenant_id":              tftypes.NewValue(tftypes.String, "tenant"),
				"last_action_created_at": tftypes.NewValue(tftypes.String, "2026-09-07T00:00:00.000Z"),
			})

			plan := resourceObject(t, resourceSchema, map[string]tftypes.Value{
				"action_code":            tftypes.NewValue(tftypes.String, planActionCode),
				"flow":                   tftypes.NewValue(tftypes.String, testCase.planFlow),
				"flow_version":           tftypes.NewValue(tftypes.Number, 4),
				"tenant_id":              tftypes.NewValue(tftypes.String, "tenant"),
				"last_action_created_at": tftypes.NewValue(tftypes.String, tftypes.UnknownValue),
			})

			if testCase.create {
				state = tftypes.NewValue(state.Type(), nil)
			}

			if testCase.destroy {
				plan = tftypes.NewValue(plan.Type(), nil)
			}

			req := resource.ModifyPlanRequest{
				State: tfsdk.State{Schema: resourceSchema, Raw: state},
				Plan:  tfsdk.Plan{Schema: resourceSchema, Raw: plan},
			}

			resp := &resource.ModifyPlanResponse{Plan: tfsdk.Plan{Schema: resourceSchema, Raw: plan}}
			if testCase.requiresReplace {
				resp.RequiresReplace = append(resp.RequiresReplace, path.Root("flow"))
			}

			(&flowResource{}).ModifyPlan(context.Background(), req, resp)

			if resp.Diagnostics.HasError() {
				t.Fatal(resp.Diagnostics)
			}

			expected := plan
			if testCase.expectState {
				expected = state
			}

			if !resp.Plan.Raw.Equal(expected) {
				t.Fatalf("expected the plan to be %v, got %v", expected, resp.Plan.Raw)
			}
		})
	}
}
