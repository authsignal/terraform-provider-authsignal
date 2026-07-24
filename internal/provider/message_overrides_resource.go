package provider

import (
	"context"
	"fmt"
	"reflect"

	"github.com/authsignal/authsignal-management-go/v4"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &messageOverridesResource{}
	_ resource.ResourceWithConfigure   = &messageOverridesResource{}
	_ resource.ResourceWithImportState = &messageOverridesResource{}
)

// messageOverridesElemType is the element type of the `overrides` attribute: a map of message
// override ID to override copy, nested under a locale key.
var messageOverridesElemType = types.MapType{ElemType: types.StringType}

func NewMessageOverridesResource() resource.Resource {
	return &messageOverridesResource{}
}

type messageOverridesResource struct {
	client *authsignal.Client
}

type messageOverridesResourceModel struct {
	Overrides types.Map `tfsdk:"overrides"`
}

func (r *messageOverridesResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_message_overrides"
}

func (r *messageOverridesResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a tenant's pre-built UI message overrides. This is a full-replacement, tenant-wide singleton: the configured value is the complete set of overrides, and applying removes any override not present. If the tenant already has overrides configured outside Terraform (e.g. in the admin portal), import the resource first (`terraform import authsignal_message_overrides.<name> \"\"`) instead of creating it, so the plan shows what will change. Use the `authsignal_message_overrides_catalog` data source to discover valid override IDs and locales.",
		Attributes: map[string]schema.Attribute{
			"overrides": schema.MapAttribute{
				Description: "Override copy keyed by locale (e.g. `en`, `pt-br`), then by message override ID (e.g. `sms-code-entry.heading`). Omit to clear all overrides.",
				ElementType: messageOverridesElemType,
				Optional:    true,
			},
		},
	}
}

func planToMessageOverrides(ctx context.Context, overridesMap types.Map) (map[string]map[string]string, diag.Diagnostics) {
	overrides := map[string]map[string]string{}
	if overridesMap.IsNull() || overridesMap.IsUnknown() {
		return overrides, nil
	}
	diags := overridesMap.ElementsAs(ctx, &overrides, false)
	return overrides, diags
}

func messageOverridesToMapValue(ctx context.Context, overrides map[string]map[string]string) (types.Map, diag.Diagnostics) {
	if len(overrides) == 0 {
		return types.MapNull(messageOverridesElemType), nil
	}
	return types.MapValueFrom(ctx, messageOverridesElemType, overrides)
}

func (r *messageOverridesResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan messageOverridesResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	overrides, diags := planToMessageOverrides(ctx, plan.Overrides)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Message overrides are a tenant-wide singleton, so a create is a full replacement. Guard against
	// silently wiping overrides configured outside Terraform (e.g. in the admin portal): if the tenant
	// already has overrides that differ from the plan, require an import first so the plan shows the diff.
	existing, _, err := r.client.GetMessageOverrides()
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read Authsignal Message Overrides",
			err.Error(),
		)
		return
	}

	if len(existing.MessageOverrides) > 0 && !reflect.DeepEqual(existing.MessageOverrides, overrides) {
		resp.Diagnostics.AddError(
			"Tenant already has message overrides configured",
			"This tenant already has message overrides set, and creating this resource would replace them. "+
				"Import the existing resource first so the plan shows what will change:\n\n"+
				"  terraform import authsignal_message_overrides.<name> \"\"",
		)
		return
	}

	_, _, err = r.client.UpdateMessageOverrides(authsignal.MessageOverridesBody{MessageOverrides: overrides})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating message overrides",
			"Could not create message overrides, unexpected error: "+err.Error(),
		)
		return
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *messageOverridesResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	messageOverrides, statusCode, err := r.client.GetMessageOverrides()

	if statusCode == 404 {
		resp.State.RemoveResource(ctx)
		return
	}

	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read Authsignal Message Overrides",
			err.Error(),
		)
		return
	}

	overridesValue, diags := messageOverridesToMapValue(ctx, messageOverrides.MessageOverrides)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, messageOverridesResourceModel{Overrides: overridesValue})
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *messageOverridesResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan messageOverridesResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	overrides, diags := planToMessageOverrides(ctx, plan.Overrides)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, _, err := r.client.UpdateMessageOverrides(authsignal.MessageOverridesBody{MessageOverrides: overrides})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating message overrides",
			"Could not update message overrides, unexpected error: "+err.Error(),
		)
		return
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *messageOverridesResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	_, _, err := r.client.UpdateMessageOverrides(authsignal.MessageOverridesBody{MessageOverrides: map[string]map[string]string{}})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting Authsignal Message Overrides",
			"Could not delete message overrides, unexpected error: "+err.Error(),
		)
		return
	}
}

func (r *messageOverridesResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Message overrides are a tenant-wide singleton keyed by the API secret, so there is nothing to
	// key the import on. Seed empty state; the subsequent Read populates it from the API.
	resp.Diagnostics.Append(resp.State.Set(ctx, messageOverridesResourceModel{Overrides: types.MapNull(messageOverridesElemType)})...)
}

func (r *messageOverridesResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*authsignal.Client)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *authsignal.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.client = client
}
