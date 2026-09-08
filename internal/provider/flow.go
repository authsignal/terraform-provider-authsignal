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
	return actionType == actionTypeFlow
}

// The two resources own the same remote action configuration, so each one reports the
// other by name when the server disagrees with the type it manages.
func flowActionDiagnostics(actionCode string) diag.Diagnostics {
	var diags diag.Diagnostics

	diags.AddError(
		"Action configuration is a "+actionTypeFlow+" action",
		"Action configuration "+actionCode+" is a "+actionTypeFlow+" action on the server, and `authsignal_action_configuration` manages "+actionTypeClassic+" actions only. "+
			"Manage this action with `authsignal_flow` instead.",
	)

	return diags
}

func classicActionDiagnostics(actionCode string) diag.Diagnostics {
	var diags diag.Diagnostics

	diags.AddError(
		"Action configuration is a "+actionTypeClassic+" action",
		"Action configuration "+actionCode+" is a "+actionTypeClassic+" action on the server, and `authsignal_flow` manages "+actionTypeFlow+" actions only. "+
			"Manage this action with `authsignal_action_configuration`, and its rules with `authsignal_rule`, instead.",
	)

	return diags
}

// existingActionType reports the type of an action configuration that already exists, so
// a create can refuse to write across the CLASSIC/FLOW boundary before it mutates
// anything. Only a 404 means the action is absent; every other failure is returned as an
// error, because a create request states the action type and could convert an action that
// this lookup failed to see. The caller is expected to fail closed on that error.
func existingActionType(client *authsignal.Client, actionCode string) (actionType string, found bool, err error) {
	actionConfiguration, statusCode, err := client.GetActionConfiguration(actionCode)

	if statusCode == http.StatusNotFound {
		return "", false, nil
	}

	if err != nil {
		return "", false, err
	}

	if actionConfiguration == nil {
		return "", false, fmt.Errorf("the API returned neither an action configuration nor an error")
	}

	return actionConfiguration.ActionType, true, nil
}

// wrongTypeAfterCreateDiagnostics reports an action the API created with a type the
// resource does not manage. The action exists and Terraform is holding it, but the
// resource's Read rejects the type, so an ordinary plan, apply or destroy fails while
// refreshing. The recovery therefore has to skip the refresh or drop the resource from
// state; the action is not deleted here because the create may have revived an archived
// action rather than made a new one.
func wrongTypeAfterCreateDiagnostics(actionCode string, createdType string, wantedType string, otherResource string) diag.Diagnostics {
	var diags diag.Diagnostics

	diags.AddError(
		"Action was not created as a "+wantedType+" action",
		"The API created action configuration "+actionCode+" with action type "+createdType+" instead of "+wantedType+". "+
			"Check which action types the Management API in this region supports.\n\n"+
			"Terraform has recorded the action so that it is not lost. This resource only reads a "+wantedType+" action, "+
			"so an ordinary `terraform plan`, `terraform apply` and `terraform destroy` will now all fail while refreshing it. Recover in one of two ways:\n\n"+
			"  * Delete the action: `terraform destroy -refresh=false -target=ADDRESS`, where ADDRESS is this resource's address.\n"+
			"  * Keep the action and stop managing it here: `terraform state rm ADDRESS`, then archive the action in the Authsignal portal "+
			"or manage it with `"+otherResource+"`.",
	)

	return diags
}

// preflightFailedDiagnostics reports a create abandoned because the action type could not
// be established. Nothing has been written when this is returned.
func preflightFailedDiagnostics(actionCode string, err error) diag.Diagnostics {
	var diags diag.Diagnostics

	diags.AddError(
		"Could not check the action configuration before creating it",
		"Terraform could not read action configuration "+actionCode+" to establish whether it already exists and which type it is: "+err.Error()+"\n\n"+
			"A create states the action type, so it could convert an action that this check failed to see. Nothing has been created. "+
			"Resolve the API error and apply again.",
	)

	return diags
}

type flowFields struct {
	ActionType  types.String
	Flow        FlowValue
	FlowVersion types.Int64
}

func readFlowFields(ctx context.Context, client *authsignal.Client, actionConfiguration *authsignal.ActionConfigurationResponse, prior FlowValue) (flowFields, diag.Diagnostics) {
	var diags diag.Diagnostics

	if !isFlowActionType(actionConfiguration.ActionType) {
		return flowFields{
			ActionType:  types.StringValue(actionTypeClassic),
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

	var preferredRuleIds []string
	if !prior.IsNull() && !prior.IsUnknown() {
		preferredRuleIds = flowRuleIds(prior.ValueString())
	}

	flowJson, err := composeFlowWithRuleOrder(actionConfiguration.ActionNodes, rules, preferredRuleIds)
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

	flowVersion := types.Int64Null()
	if actionConfiguration.FlowVersion != nil {
		flowVersion = types.Int64Value(*actionConfiguration.FlowVersion)
	}

	return flowFields{
		ActionType:  types.StringValue(actionTypeFlow),
		Flow:        flow,
		FlowVersion: flowVersion,
	}, diags
}

func flowChanged(ctx context.Context, plan FlowValue, state FlowValue) (bool, diag.Diagnostics) {
	if plan.IsNull() || plan.IsUnknown() || state.IsNull() || state.IsUnknown() {
		return !plan.Equal(state), nil
	}

	equal, diags := state.StringSemanticEquals(ctx, plan)

	return !equal, diags
}

func publishFlow(client *authsignal.Client, actionCode string, flow FlowValue, expectedFlowVersion *int64) diag.Diagnostics {
	var diags diag.Diagnostics

	doc, errs := parseFlow(flow.ValueString())
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
		ActionNodes:         doc.ActionNodes,
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
