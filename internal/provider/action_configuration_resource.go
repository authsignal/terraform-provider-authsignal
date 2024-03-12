package provider

import (
	"context"
	"fmt"

	authsignal "authsignal.com/authsignal-management-go"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &actionConfigurationResource{}
	_ resource.ResourceWithConfigure   = &actionConfigurationResource{}
	_ resource.ResourceWithImportState = &actionConfigurationResource{}
)

// NewActionConfigurationResource is a helper function to simplify the provider implementation.
func NewActionConfigurationResource() resource.Resource {
	return &actionConfigurationResource{}
}

// actionConfigurationResource is the resource implementation.
type actionConfigurationResource struct {
	client *authsignal.Client
}

type actionConfigurationResourceModel struct {
	ActionCode              types.String `tfsdk:"action_code"`
	LastActionCreatedAt     types.String `tfsdk:"last_action_created_at"`
	TenantId                types.String `tfsdk:"tenant_id"`
	DefaultUserActionResult types.String `tfsdk:"default_user_action_result"`
}

// Metadata returns the resource type name.
func (r *actionConfigurationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_action_configuration"
}

// Schema defines the schema for the resource.
func (r *actionConfigurationResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"action_code": schema.StringAttribute{
				Description: "A string used to define the action to track against.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"last_action_created_at": schema.StringAttribute{
				Description: "The date of when an action was last tracked.",
				Computed:    true,
			},
			"tenant_id": schema.StringAttribute{
				Description: "The ID of your tenant.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"default_user_action_result": schema.StringAttribute{
				Description: "The default action behaviour if no rules match.",
				Required:    true,
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *actionConfigurationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan actionConfigurationResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var actionConfigurationToCreate = authsignal.ActionConfiguration{ActionCode: plan.ActionCode.ValueString(), DefaultUserActionResult: plan.DefaultUserActionResult.ValueString()}

	actionConfiguration, err := r.client.CreateActionConfiguration(actionConfigurationToCreate)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating action configuration",
			"Could not create action configuration, unexpected error: "+err.Error(),
		)
		return
	}

	tflog.Info(ctx, fmt.Sprintf("%+v", actionConfiguration))

	plan.DefaultUserActionResult = types.StringValue(actionConfiguration.DefaultUserActionResult)
	plan.TenantId = types.StringValue(actionConfiguration.TenantId)
	plan.LastActionCreatedAt = types.StringValue(actionConfiguration.LastActionCreatedAt)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *actionConfigurationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state actionConfigurationResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	actionConfiguration, err := r.client.GetActionConfiguration(state.ActionCode.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading action configuration",
			"Could not read action configuration code "+state.ActionCode.ValueString()+": "+err.Error(),
		)
		return
	}

	state.DefaultUserActionResult = types.StringValue(actionConfiguration.DefaultUserActionResult)
	state.LastActionCreatedAt = types.StringValue(actionConfiguration.LastActionCreatedAt)
	state.TenantId = types.StringValue(actionConfiguration.TenantId)

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *actionConfigurationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan actionConfigurationResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var actionConfigurationToUpdate = authsignal.ActionConfiguration{ActionCode: plan.ActionCode.ValueString(), DefaultUserActionResult: plan.DefaultUserActionResult.ValueString()}

	_, err := r.client.UpdateActionConfiguration(actionConfigurationToUpdate)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Authsignal action configuration",
			"Could not update action configuration, unexpected error: "+err.Error(),
		)
		return
	}

	updatedActionConfiguration, err := r.client.GetActionConfiguration(plan.ActionCode.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Authsignal action configuration",
			"Could not read action configuration ID "+plan.ActionCode.ValueString()+": "+err.Error(),
		)
		return
	}

	plan.ActionCode = types.StringValue(updatedActionConfiguration.ActionCode)
	plan.DefaultUserActionResult = types.StringValue(updatedActionConfiguration.DefaultUserActionResult)
	plan.TenantId = types.StringValue(updatedActionConfiguration.TenantId)
	plan.LastActionCreatedAt = types.StringValue(updatedActionConfiguration.LastActionCreatedAt)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *actionConfigurationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state actionConfigurationResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.DeleteActionConfiguration(state.ActionCode.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting HashiCups Order",
			"Could not delete order, unexpected error: "+err.Error(),
		)
		return
	}
}

func (r *actionConfigurationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Retrieve import ID and save to id attribute
	resource.ImportStatePassthroughID(ctx, path.Root("action_code"), req, resp)
}

func (r *actionConfigurationResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
