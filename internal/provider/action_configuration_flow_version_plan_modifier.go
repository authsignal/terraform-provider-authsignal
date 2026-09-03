package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
)

// flow_version only moves when the flow is published, and the flow is only published when it
// changed semantically. So the plan keeps the stored version whenever the configured flow means the
// same thing as the stored one, and a formatting-only edit never shows a version change. On create,
// destroy, or a real flow change the version is left unknown.
type flowVersionFollowsFlow struct{}

func (m flowVersionFollowsFlow) Description(_ context.Context) string {
	return "Keeps the stored flow_version unless the flow changes semantically."
}

func (m flowVersionFollowsFlow) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m flowVersionFollowsFlow) PlanModifyInt64(ctx context.Context, req planmodifier.Int64Request, resp *planmodifier.Int64Response) {
	if req.State.Raw.IsNull() || req.Plan.Raw.IsNull() {
		return
	}

	var planFlow, stateFlow FlowValue
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("flow"), &planFlow)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("flow"), &stateFlow)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if planFlow.IsUnknown() {
		return
	}

	if planFlow.IsNull() || stateFlow.IsNull() {
		if planFlow.IsNull() && stateFlow.IsNull() {
			resp.PlanValue = req.StateValue
		}
		return
	}

	equal, diags := stateFlow.StringSemanticEquals(ctx, planFlow)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if equal {
		resp.PlanValue = req.StateValue
	}
}
