package provider

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/authsignal/authsignal-management-go/v6"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func isFlowActionType(actionType string) bool {
	return actionType == actionTypeMultiStep
}

// flowFields are the attributes an action configuration gains from its flow. The resource and the
// data source map them the same way.
type flowFields struct {
	ActionType  types.String
	Flow        FlowValue
	FlowVersion types.Int64
}

// readFlowFields maps a read action configuration to its flow attributes. A legacy action (any type
// other than MULTI_STEP_AUTHENTICATION, including the empty string an older API returns) has no
// flow and no rules are listed for it. A flow action's rules are listed and embedded into its
// graph; when the result means the same as the prior value, the prior string is kept so a
// pretty-printed file never drifts. Server rules no node references cannot live in the graph and
// are reported as a warning.
func readFlowFields(ctx context.Context, client *authsignal.Client, actionConfiguration *authsignal.ActionConfigurationResponse, prior FlowValue) (flowFields, diag.Diagnostics) {
	var diags diag.Diagnostics

	if !isFlowActionType(actionConfiguration.ActionType) {
		return flowFields{
			ActionType:  types.StringValue(actionTypeLegacy),
			Flow:        NewFlowNull(),
			FlowVersion: types.Int64Null(),
		}, diags
	}

	rules, _, err := client.ListRules(actionConfiguration.ActionCode)
	if err != nil {
		diags.AddError(
			"Error reading action flow rules",
			"Could not list the rules of action configuration "+actionConfiguration.ActionCode+": "+err.Error(),
		)
		return flowFields{}, diags
	}

	flowJson, unreferenced, err := embedFlow(actionConfiguration.ActionNodes, rules)
	if err != nil {
		diags.AddError(
			"Error reading action flow",
			"Could not marshal the flow of action configuration "+actionConfiguration.ActionCode+": "+err.Error(),
		)
		return flowFields{}, diags
	}

	flow := NewFlowValue(flowJson)

	if !prior.IsNull() && !prior.IsUnknown() {
		equal, equalDiags := prior.StringSemanticEquals(ctx, flow)
		diags.Append(equalDiags...)
		if diags.HasError() {
			return flowFields{}, diags
		}

		if equal {
			flow = prior
		}
	}

	if len(unreferenced) > 0 {
		diags.AddWarning(
			"Unreferenced rules on action "+actionConfiguration.ActionCode,
			fmt.Sprintf("%d rule(s) exist on action %s that no RULE node of its flow references: %s. They cannot be represented in the flow attribute and will be removed on the next publish of the flow.",
				len(unreferenced), actionConfiguration.ActionCode, strings.Join(unreferenced, ", ")),
		)
	}

	flowVersion := types.Int64Null()
	if actionConfiguration.FlowVersion != nil {
		flowVersion = types.Int64Value(*actionConfiguration.FlowVersion)
	}

	return flowFields{
		ActionType:  types.StringValue(actionTypeMultiStep),
		Flow:        flow,
		FlowVersion: flowVersion,
	}, diags
}

// flowChanged reports whether the planned flow means something different from the stored one.
func flowChanged(ctx context.Context, plan FlowValue, state FlowValue) (bool, diag.Diagnostics) {
	if plan.IsNull() || plan.IsUnknown() || state.IsNull() || state.IsUnknown() {
		return !plan.Equal(state), nil
	}

	equal, diags := state.StringSemanticEquals(ctx, plan)

	return !equal, diags
}

// publishFlow lifts the embedded rules out of the flow and publishes `{actionNodes, rules}`. With an
// expected version the API refuses (409) to overwrite a flow that moved on since Terraform read it.
func publishFlow(client *authsignal.Client, actionCode string, flow FlowValue, expectedFlowVersion *int64) diag.Diagnostics {
	var diags diag.Diagnostics

	doc, errs := liftFlow(flow.ValueString())
	if len(errs) > 0 {
		messages := make([]string, len(errs))
		for i, err := range errs {
			messages[i] = err.Error()
		}

		diags.AddError(
			"Invalid action flow",
			"The flow of action configuration "+actionCode+" could not be published:\n"+strings.Join(messages, "\n"),
		)
		return diags
	}

	_, statusCode, err := client.UpdateActionFlow(actionCode, authsignal.ActionFlow{
		ActionNodes:         doc.actionNodes(),
		Rules:               doc.Rules,
		ExpectedFlowVersion: expectedFlowVersion,
	})

	if statusCode == http.StatusConflict {
		expected := "none"
		if expectedFlowVersion != nil {
			expected = fmt.Sprintf("%d", *expectedFlowVersion)
		}

		diags.AddError(
			"Flow changed outside Terraform",
			fmt.Sprintf("The flow of action configuration %s was published by something else since Terraform last read it (Terraform expected flow version %s). Run terraform plan again to refresh the flow and review the difference before applying.", actionCode, expected),
		)
		return diags
	}

	if err != nil {
		diags.AddError(
			"Error publishing action flow",
			"Could not publish the flow of action configuration "+actionCode+": "+err.Error(),
		)
		return diags
	}

	return diags
}
