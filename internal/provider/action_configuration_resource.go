package provider

import (
	"context"
	"fmt"

	"github.com/authsignal/authsignal-management-go"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &actionConfigurationResource{}
	_ resource.ResourceWithConfigure   = &actionConfigurationResource{}
	_ resource.ResourceWithImportState = &actionConfigurationResource{}
)

func NewActionConfigurationResource() resource.Resource {
	return &actionConfigurationResource{}
}

type actionConfigurationResource struct {
	client *authsignal.Client
}

type actionConfigurationResourceModel struct {
	ActionCode              types.String `tfsdk:"action_code"`
	LastActionCreatedAt     types.String `tfsdk:"last_action_created_at"`
	TenantId                types.String `tfsdk:"tenant_id"`
	DefaultUserActionResult types.String `tfsdk:"default_user_action_result"`
}

func (r *actionConfigurationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_action_configuration"
}

func (r *actionConfigurationResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"action_code": schema.StringAttribute{
				Description: "The name of the action that users perform which you will track. (e.g 'login')",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"default_user_action_result": schema.StringAttribute{
				Description: "The default action behavior if no rules match. (i.e 'CHALLENGE')",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.OneOf([]string{"ALLOW", "CHALLENGE", "REVIEW", "BLOCK"}...),
				},
			},
			"last_action_created_at": schema.StringAttribute{
				Description: "The date of when an action was last tracked for any user.",
				Computed:    true,
			},
			"tenant_id": schema.StringAttribute{
				Description: "The ID of your tenant. This can be found in the admin portal.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

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

func (r *actionConfigurationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan actionConfigurationResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var actionConfigurationToUpdate = authsignal.ActionConfiguration{ActionCode: plan.ActionCode.ValueString(), DefaultUserActionResult: plan.DefaultUserActionResult.ValueString()}

	_, err := r.client.UpdateActionConfiguration(plan.ActionCode.ValueString(), actionConfigurationToUpdate)
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
			"Error Deleting Authsignal action configuration",
			"Could not delete action configuration, unexpected error: "+err.Error(),
		)
		return
	}
}

func (r *actionConfigurationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
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
