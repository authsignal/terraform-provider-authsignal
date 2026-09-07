package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
)

type flowKeepsStateWhenEqual struct{}

func (m flowKeepsStateWhenEqual) Description(_ context.Context) string {
	return "Keeps the stored flow when the configured flow is the same document."
}

func (m flowKeepsStateWhenEqual) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

// The framework applies StringSemanticEquals to responses but not to the plan.
func (m flowKeepsStateWhenEqual) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if req.StateValue.IsNull() || req.StateValue.IsUnknown() {
		return
	}

	if req.PlanValue.IsNull() || req.PlanValue.IsUnknown() {
		return
	}

	equal, diags := FlowValue{StringValue: req.StateValue}.StringSemanticEquals(ctx, FlowValue{StringValue: req.PlanValue})
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if equal {
		resp.PlanValue = req.StateValue
	}
}
