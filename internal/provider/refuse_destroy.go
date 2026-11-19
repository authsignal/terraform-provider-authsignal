package provider

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

// The Management API can't delete a tenant's settings or theme, so a destroy fails the plan rather
// than leaving them in place unannounced.
func refuseDestroy(typeName string, what string, kept []unmanagedSetting) diag.Diagnostic {
	detail := "The Management API can't delete a tenant's " + what + ". " +
		"To stop Terraform managing this resource, use this removed block in place of its resource block:\n\n" +
		removedBlock(typeName) + "\n\n" +
		"The tenant keeps its current " + what + "."

	if len(kept) > 0 {
		detail += " These differ from a new tenant's:\n\n" +
			strings.Join(hclLines(kept), "\n") + "\n\n" +
			"To change one, set it in the configuration and apply before removing the resource."
	}

	return diag.NewErrorDiagnostic("Can't Destroy "+typeName, detail)
}

func removedBlock(typeName string) string {
	return "  removed {\n" +
		"    from = " + typeName + ".<name>\n\n" +
		"    lifecycle {\n" +
		"      destroy = false\n" +
		"    }\n" +
		"  }"
}

func refuseTenantSettingsDestroy(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse, typeName string, what string) {
	resourceSchema, ok := req.State.Schema.(schema.Schema)
	if !ok {
		resp.Diagnostics.Append(refuseDestroy(typeName, what, nil))
		return
	}

	defaults, diags := storedDefaults(ctx, req)
	resp.Diagnostics.Append(diags...)

	stateAttributes, err := objectAttributes(ctx, resourceSchema, req.State.Raw)
	if err != nil {
		resp.Diagnostics.AddError("Unable to Read the Settings the Tenant Keeps", err.Error())
	}

	kept := unconfiguredSettings("", resourceSchema.Attributes, map[string]attr.Value{}, stateAttributes, defaults)
	resp.Diagnostics.Append(refuseDestroy(typeName, what, kept))
}
