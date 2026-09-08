package provider

import (
	"context"
	"fmt"

	"github.com/authsignal/authsignal-management-go/v6"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &flowResource{}
	_ resource.ResourceWithConfigure   = &flowResource{}
	_ resource.ResourceWithImportState = &flowResource{}
	_ resource.ResourceWithModifyPlan  = &flowResource{}
)

func NewFlowResource() resource.Resource {
	return &flowResource{}
}

type flowResource struct {
	client *authsignal.Client
}

type flowResourceModel struct {
	ActionCode          types.String `tfsdk:"action_code"`
	Flow                FlowValue    `tfsdk:"flow"`
	FlowVersion         types.Int64  `tfsdk:"flow_version"`
	TenantId            types.String `tfsdk:"tenant_id"`
	LastActionCreatedAt types.String `tfsdk:"last_action_created_at"`
}

func (r *flowResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_flow"
}

func (r *flowResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a `FLOW` action. Use `authsignal_action_configuration` for `CLASSIC` actions.",
		Attributes: map[string]schema.Attribute{
			"action_code": schema.StringAttribute{
				Description: "The name of the action that users perform which you will track. (e.g 'login')",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"flow": schema.StringAttribute{
				CustomType: FlowType{},
				Description: "The flow document as JSON. It must contain `actionNodes` and `rules`. " +
					"Use `file()` to load a flow exported from the Authsignal Portal, or `jsonencode()` to define it inline. " +
					"Before using a verification method in a flow, enable its authenticator configuration in Authsignal.",
				Required: true,
				Validators: []validator.String{
					flowValidator{},
				},
				PlanModifiers: []planmodifier.String{
					flowKeepsStateWhenEqual{},
				},
			},
			"flow_version": schema.Int64Attribute{
				Description: "The version of the published flow.",
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					flowVersionFollowsFlow{},
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

// The framework marks every computed attribute with a null configuration value unknown
// before the attribute plan modifiers run, whenever the proposed plan differs from the
// prior state. An imported flow is the composed document rather than the configured
// text, so the proposal always differs and last_action_created_at stays unknown forever.
func (r *flowResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.State.Raw.IsNull() || req.Plan.Raw.IsNull() || len(resp.RequiresReplace) > 0 {
		return
	}

	var plan flowResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	var state flowResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	flowEqual, diags := state.Flow.StringSemanticEquals(ctx, plan.Flow)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// last_action_created_at and flow_version follow the action, so only the configurable attributes decide.
	if flowEqual && plan.ActionCode.Equal(state.ActionCode) {
		resp.Plan.Raw = req.State.Raw
	}
}

func (r *flowResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan flowResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	actionCode := plan.ActionCode.ValueString()

	// Creating over an existing CLASSIC action would adopt an action this resource does not
	// model, and the create request carries the action type, so it could convert one. The
	// check runs before the first write rather than after it, when nothing can be undone,
	// and it fails closed: an action whose type cannot be established is not written to.
	actionType, found, err := existingActionType(r.client, actionCode)
	if err != nil {
		resp.Diagnostics.Append(preflightFailedDiagnostics(actionCode, err)...)
		return
	}

	if found && !isFlowActionType(actionType) {
		resp.Diagnostics.Append(classicActionDiagnostics(actionCode)...)
		return
	}

	var actionConfigurationToCreate = authsignal.ActionConfiguration{
		ActionCode: authsignal.SetValue(actionCode),
		// Recreated archived actions retain fields omitted from the request, so an archived
		// CLASSIC action would revive as a CLASSIC action unless the type is sent explicitly.
		ActionType: authsignal.SetValue(actionTypeFlow),
	}

	actionConfiguration, _, err := r.client.CreateActionConfiguration(actionConfigurationToCreate)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating action configuration",
			"Could not create action configuration "+actionCode+", unexpected error: "+err.Error(),
		)
		return
	}

	plan.TenantId = types.StringValue(actionConfiguration.TenantId)
	plan.LastActionCreatedAt = types.StringValue(actionConfiguration.LastActionCreatedAt)
	plan.FlowVersion = types.Int64Null()

	// The action exists from here on, so every failure below records it first: Terraform
	// keeps the state of a failed create, and that state is the only handle on the action.
	if !isFlowActionType(actionConfiguration.ActionType) {
		resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
		resp.Diagnostics.Append(wrongTypeAfterCreateDiagnostics(
			actionCode, actionConfiguration.ActionType, actionTypeFlow, "authsignal_action_configuration",
		)...)
		return
	}

	// The action now exists with no flow published. Recording it before the second phase
	// means a failed publish leaves a resource that the next refresh reconciles, rather
	// than an action the configuration has no way to reach again.
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(publishFlow(r.client, actionCode, plan.Flow, nil)...)
	if resp.Diagnostics.HasError() {
		return
	}

	publishedActionConfiguration, _, err := r.client.GetActionConfiguration(actionCode)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading action configuration",
			"Could not read action configuration code "+actionCode+" after publishing its flow: "+err.Error(),
		)
		return
	}

	fields, diags := readFlowFields(ctx, r.client, publishedActionConfiguration, plan.Flow)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	plan.Flow = fields.Flow
	plan.FlowVersion = fields.FlowVersion
	plan.TenantId = types.StringValue(publishedActionConfiguration.TenantId)
	plan.LastActionCreatedAt = types.StringValue(publishedActionConfiguration.LastActionCreatedAt)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *flowResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state flowResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	actionCode := state.ActionCode.ValueString()

	actionConfiguration, statusCode, err := r.client.GetActionConfiguration(actionCode)

	if statusCode == 404 {
		resp.State.RemoveResource(ctx)
		return
	}

	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading action configuration",
			"Could not read action configuration code "+actionCode+": "+err.Error(),
		)
		return
	}

	if !isFlowActionType(actionConfiguration.ActionType) {
		resp.Diagnostics.Append(classicActionDiagnostics(actionCode)...)
		return
	}

	fields, diags := readFlowFields(ctx, r.client, actionConfiguration, state.Flow)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	state.Flow = fields.Flow
	state.FlowVersion = fields.FlowVersion
	state.TenantId = types.StringValue(actionConfiguration.TenantId)
	state.LastActionCreatedAt = types.StringValue(actionConfiguration.LastActionCreatedAt)

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// The graph is only ever published through UpdateActionFlow, which is the one endpoint
// that takes the flow's rules and the version the change is based on.
func (r *flowResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan flowResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state flowResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	actionCode := plan.ActionCode.ValueString()

	changed, diags := flowChanged(ctx, plan.Flow, state.Flow)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if changed {
		var expectedFlowVersion *int64
		if !state.FlowVersion.IsNull() && !state.FlowVersion.IsUnknown() {
			version := state.FlowVersion.ValueInt64()
			expectedFlowVersion = &version
		}

		resp.Diagnostics.Append(publishFlow(r.client, actionCode, plan.Flow, expectedFlowVersion)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	updatedActionConfiguration, _, err := r.client.GetActionConfiguration(actionCode)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Authsignal action configuration",
			"Could not read action configuration ID "+actionCode+": "+err.Error(),
		)
		return
	}

	if !isFlowActionType(updatedActionConfiguration.ActionType) {
		resp.Diagnostics.Append(classicActionDiagnostics(actionCode)...)
		return
	}

	fields, diags := readFlowFields(ctx, r.client, updatedActionConfiguration, plan.Flow)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	plan.ActionCode = types.StringValue(updatedActionConfiguration.ActionCode)
	plan.Flow = fields.Flow
	plan.FlowVersion = fields.FlowVersion
	plan.TenantId = types.StringValue(updatedActionConfiguration.TenantId)
	plan.LastActionCreatedAt = types.StringValue(updatedActionConfiguration.LastActionCreatedAt)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// A flow cannot be unpublished on its own, so this resource owns the whole action
// configuration and deleting it deletes the action.
func (r *flowResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state flowResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, _, err := r.client.DeleteActionConfiguration(state.ActionCode.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting Authsignal action configuration",
			"Could not delete action configuration, unexpected error: "+err.Error(),
		)
		return
	}
}

func (r *flowResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("action_code"), req, resp)
}

func (r *flowResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
