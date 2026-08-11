package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// Dropping a block from the configuration clears it on the next apply, which is what Terraform
// owning the theme means. exit_position is the exception: the Portal's theme editor is where it gets
// set, so the API keeps it and the plan has to say so, or the apply comes back with a value the plan
// did not have. Everything else in the container still clears.
type containerKeepsExitPosition struct{}

func (m containerKeepsExitPosition) Description(_ context.Context) string {
	return "Keeps a stored exit_position when the configuration has no container block."
}

func (m containerKeepsExitPosition) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m containerKeepsExitPosition) PlanModifyObject(ctx context.Context, req planmodifier.ObjectRequest, resp *planmodifier.ObjectResponse) {
	if !req.ConfigValue.IsNull() {
		return
	}

	if req.StateValue.IsNull() {
		resp.PlanValue = types.ObjectNull(containerModel{}.AttributeTypes())
		return
	}

	var state containerModel
	resp.Diagnostics.Append(req.StateValue.As(ctx, &state, basetypes.ObjectAsOptions{})...)
	if resp.Diagnostics.HasError() {
		return
	}

	if state.ExitPosition.IsNull() {
		resp.PlanValue = types.ObjectNull(containerModel{}.AttributeTypes())
		return
	}

	kept := containerModel{ExitPosition: state.ExitPosition}

	object, diags := types.ObjectValue(kept.AttributeTypes(), kept.AttributeValues())
	resp.Diagnostics.Append(diags...)
	resp.PlanValue = object
}
