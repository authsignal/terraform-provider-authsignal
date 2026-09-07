package provider

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/authsignal/authsignal-management-go/v6"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                   = &actionConfigurationResource{}
	_ resource.ResourceWithConfigure      = &actionConfigurationResource{}
	_ resource.ResourceWithImportState    = &actionConfigurationResource{}
	_ resource.ResourceWithValidateConfig = &actionConfigurationResource{}
)

func NewActionConfigurationResource() resource.Resource {
	return &actionConfigurationResource{}
}

type actionConfigurationResource struct {
	client *authsignal.Client
}

type actionConfigurationResourceModel struct {
	ActionCode                        types.String `tfsdk:"action_code"`
	ActionType                        types.String `tfsdk:"action_type"`
	LastActionCreatedAt               types.String `tfsdk:"last_action_created_at"`
	TenantId                          types.String `tfsdk:"tenant_id"`
	DefaultUserActionResult           types.String `tfsdk:"default_user_action_result"`
	MessagingTemplates                types.String `tfsdk:"messaging_templates"`
	VerificationMethods               types.List   `tfsdk:"verification_methods"`
	PromptToEnrollVerificationMethods types.List   `tfsdk:"prompt_to_enroll_verification_methods"`
	DefaultVerificationMethod         types.String `tfsdk:"default_verification_method"`
	Flow                              FlowValue    `tfsdk:"flow"`
	FlowVersion                       types.Int64  `tfsdk:"flow_version"`
}

func actionTypeRequiresReplace() planmodifier.String {
	return stringplanmodifier.RequiresReplaceIf(
		func(_ context.Context, req planmodifier.StringRequest, resp *stringplanmodifier.RequiresReplaceIfFuncResponse) {
			if req.StateValue.IsNull() || req.StateValue.IsUnknown() {
				return
			}

			if req.ConfigValue.IsNull() && isFlowActionType(req.StateValue.ValueString()) {
				resp.Diagnostics.AddAttributeError(
					req.Path,
					"Action type not set for a flow action",
					"This action is a "+actionTypeFlow+" action on the server but action_type is not set in the configuration, which means "+actionTypeClassic+". "+
						"Set action_type = \""+actionTypeFlow+"\" and flow to manage the flow, or set action_type = \""+actionTypeClassic+"\" explicitly to replace the action with a classic one.",
				)
				return
			}

			resp.RequiresReplace = true
		},
		"The action type cannot be changed once the action exists.",
		"The action type cannot be changed once the action exists.",
	)
}

func defaultUserActionResultValue(defaultUserActionResult string) types.String {
	if len(defaultUserActionResult) == 0 {
		return types.StringNull()
	}

	return types.StringValue(defaultUserActionResult)
}

func (r *actionConfigurationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_action_configuration"
}

func (r *actionConfigurationResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages `CLASSIC` and `FLOW` action configurations.",
		Attributes: map[string]schema.Attribute{
			"action_code": schema.StringAttribute{
				Description: "The name of the action that users perform which you will track. (e.g 'login')",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"action_type": schema.StringAttribute{
				Description: "The action type: `CLASSIC` (default) or `FLOW`. Changing this value replaces the action.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(actionTypeClassic),
				Validators: []validator.String{
					stringvalidator.OneOf(actionTypeClassic, actionTypeFlow),
				},
				PlanModifiers: []planmodifier.String{
					actionTypeRequiresReplace(),
				},
			},
			"default_user_action_result": schema.StringAttribute{
				Description: "The result when no rule matches a `CLASSIC` action. Required for `CLASSIC`; must not be set for `FLOW`. Allowed values: `ALLOW`, `CHALLENGE`, `REVIEW`, `BLOCK`.",
				Optional:    true,
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
			"messaging_templates": schema.StringAttribute{
				Description: "Optional messaging templates to be shown in Authsignal's pre-built UI.",
				Optional:    true,
			},
			"verification_methods": schema.ListAttribute{
				ElementType: types.StringType,
				Description: "A list of permitted authenticators that can be used if the result of the action is 'CHALLENGE'. Allowed values: `SMS`, `AUTHENTICATOR_APP`, `EMAIL_MAGIC_LINK`, `EMAIL_OTP`, `DEVICE`, `PUSH`, `QR_CODE`, `IN_APP`, `SECURITY_KEY`, `PASSKEY`, `VERIFF`, `IPROOV`, `PALM_BIOMETRICS_RR`, `IDVERSE`, `ONFIDO`, `APPLE_ID_TOKEN`, `GOOGLE_ID_TOKEN`, `WHATSAPP`, `DIGITAL_CREDENTIAL`, `OIDC_PROVIDER`.",
				Optional:    true,
				Validators: []validator.List{
					listvalidator.ValueStringsAre(stringvalidator.OneOf(allowedVerificationMethods...)),
				},
			},
			"prompt_to_enroll_verification_methods": schema.ListAttribute{
				ElementType: types.StringType,
				Description: "If this is set then users will be prompted to add a passkey after a challenge is completed. Allowed values: `[PASSKEY]`.",
				Optional:    true,
				Validators: []validator.List{
					listvalidator.ValueStringsAre(stringvalidator.OneOf([]string{"PASSKEY"}...)),
				},
			},
			"default_verification_method": schema.StringAttribute{
				Description: "Ignore the user's preference and choose which authenticator the Pre-built UI will present by default. Allowed values: `SMS`, `AUTHENTICATOR_APP`, `EMAIL_MAGIC_LINK`, `EMAIL_OTP`, `DEVICE`, `PUSH`, `QR_CODE`, `IN_APP`, `SECURITY_KEY`, `PASSKEY`, `VERIFF`, `IPROOV`, `PALM_BIOMETRICS_RR`, `IDVERSE`, `ONFIDO`, `APPLE_ID_TOKEN`, `GOOGLE_ID_TOKEN`, `WHATSAPP`, `DIGITAL_CREDENTIAL`, `OIDC_PROVIDER`.",
				Optional:    true,
				Validators: []validator.String{
					stringvalidator.OneOf(allowedVerificationMethods...),
				},
			},
			"flow": schema.StringAttribute{
				CustomType: FlowType{},
				Description: "A JSON object string with `actionNodes` and `rules`, whose `actionNodes` may encode to at most 300,000 bytes. Required for `FLOW`; must not be set for `CLASSIC`. " +
					"Use `file()` with the JSON the Authsignal portal exports for a flow, or use `jsonencode()`. " +
					"Formatting, key order, and rule order do not cause changes. " +
					"Flow rules belong in this document. Using `authsignal_rule` for the same action causes a permanent diff.",
				Optional: true,
				Validators: []validator.String{
					flowValidator{},
				},
				PlanModifiers: []planmodifier.String{
					flowKeepsStateWhenEqual{},
				},
			},
			"flow_version": schema.Int64Attribute{
				Description: "The computed version of the published flow. Null for `CLASSIC` actions and unpublished `FLOW` actions.",
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					flowVersionFollowsFlow{},
				},
			},
		},
	}
}

func (r *actionConfigurationResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config actionConfigurationResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if config.ActionType.IsUnknown() || config.Flow.IsUnknown() || config.DefaultUserActionResult.IsUnknown() {
		return
	}

	// Schema defaults are not applied to raw configuration values.
	actionType := actionTypeClassic
	if !config.ActionType.IsNull() {
		actionType = config.ActionType.ValueString()
	}

	if isFlowActionType(actionType) && config.Flow.IsNull() {
		resp.Diagnostics.AddAttributeError(
			path.Root("flow"),
			"Missing action flow",
			"A `"+actionTypeFlow+"` action needs a flow. Set `flow` to the flow exported from the admin portal, for example `flow = file(\"${path.module}/flow-sign-in.json\")`.",
		)
	}

	if !isFlowActionType(actionType) && !config.Flow.IsNull() {
		resp.Diagnostics.AddAttributeError(
			path.Root("flow"),
			"Unexpected action flow",
			"`flow` can only be set when `action_type` is `"+actionTypeFlow+"`. A `"+actionTypeClassic+"` action evaluates the rules managed with `authsignal_rule` instead.",
		)
	}

	if isFlowActionType(actionType) && !config.DefaultUserActionResult.IsNull() {
		resp.Diagnostics.AddAttributeError(
			path.Root("default_user_action_result"),
			"Unexpected default user action result",
			"`default_user_action_result` can only be set when `action_type` is `"+actionTypeClassic+"`. A `"+actionTypeFlow+"` action decides its outcome in the flow, so the API ignores the default result.",
		)
	}

	if !isFlowActionType(actionType) && config.DefaultUserActionResult.IsNull() {
		resp.Diagnostics.AddAttributeError(
			path.Root("default_user_action_result"),
			"Missing default user action result",
			"A `"+actionTypeClassic+"` action needs a default result for when no rule matches. Set `default_user_action_result` to one of `ALLOW`, `CHALLENGE`, `REVIEW` or `BLOCK`.",
		)
	}
}

func (r *actionConfigurationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan actionConfigurationResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	verificationMethodsSlice := make([]string, 0, len(plan.VerificationMethods.Elements()))
	diags1 := plan.VerificationMethods.ElementsAs(ctx, &verificationMethodsSlice, false)
	resp.Diagnostics.Append(diags1...)
	if resp.Diagnostics.HasError() {
		return
	}

	promptToEnrollVerificationMethodsSlice := make([]string, 0, len(plan.PromptToEnrollVerificationMethods.Elements()))
	diags2 := plan.PromptToEnrollVerificationMethods.ElementsAs(ctx, &promptToEnrollVerificationMethodsSlice, false)
	resp.Diagnostics.Append(diags2...)
	if resp.Diagnostics.HasError() {
		return
	}

	var messagingTemplatesJson authsignal.MessagingTemplates

	if len(string(plan.MessagingTemplates.ValueString())) > 0 {
		err := json.Unmarshal([]byte(plan.MessagingTemplates.ValueString()), &messagingTemplatesJson)
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to unmarshal messaging templates",
				err.Error(),
			)
			return
		}
	}

	var actionConfigurationToCreate = authsignal.ActionConfiguration{}

	var actionConfigurationActionCode = plan.ActionCode.ValueString()
	if len(actionConfigurationActionCode) > 0 {
		actionConfigurationToCreate.ActionCode = authsignal.SetValue(actionConfigurationActionCode)
	}

	// Recreated archived actions retain fields omitted from the request.
	actionType := plan.ActionType.ValueString()
	if len(actionType) == 0 {
		actionType = actionTypeClassic
	}
	actionConfigurationToCreate.ActionType = authsignal.SetValue(actionType)

	var actionConfigurationDefaultUserActionResult = plan.DefaultUserActionResult.ValueString()
	if len(actionConfigurationDefaultUserActionResult) > 0 {
		actionConfigurationToCreate.DefaultUserActionResult = authsignal.SetValue(actionConfigurationDefaultUserActionResult)
	}

	if len(string(plan.MessagingTemplates.ValueString())) > 0 {
		actionConfigurationToCreate.MessagingTemplates = authsignal.SetValue(messagingTemplatesJson)
	}

	var actionConfigurationDefaultVerificationMethod = plan.DefaultVerificationMethod.ValueString()
	if len(actionConfigurationDefaultVerificationMethod) > 0 {
		actionConfigurationToCreate.DefaultVerificationMethod = authsignal.SetValue(actionConfigurationDefaultVerificationMethod)
	}

	if len(verificationMethodsSlice) > 0 {
		actionConfigurationToCreate.VerificationMethods = authsignal.SetValue(verificationMethodsSlice)
	}

	if len(promptToEnrollVerificationMethodsSlice) > 0 {
		actionConfigurationToCreate.PromptToEnrollVerificationMethods = authsignal.SetValue(promptToEnrollVerificationMethodsSlice)
	}

	actionConfiguration, _, err := r.client.CreateActionConfiguration(actionConfigurationToCreate)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating action configuration",
			"Could not create action configuration, unexpected error: "+err.Error(),
		)
		return
	}

	plan.DefaultUserActionResult = defaultUserActionResultValue(actionConfiguration.DefaultUserActionResult)
	plan.TenantId = types.StringValue(actionConfiguration.TenantId)
	plan.LastActionCreatedAt = types.StringValue(actionConfiguration.LastActionCreatedAt)

	if !isFlowActionType(actionType) {
		plan.ActionType = types.StringValue(actionTypeClassic)
		plan.Flow = NewFlowNull()
		plan.FlowVersion = types.Int64Null()

		diags = resp.State.Set(ctx, plan)
		resp.Diagnostics.Append(diags...)
		return
	}

	if !isFlowActionType(actionConfiguration.ActionType) {
		resp.Diagnostics.AddError(
			"Action was not created as a flow action",
			fmt.Sprintf("The API created action configuration %s with action type %q instead of %q. Check that the Management API in this region supports flow-based actions.", plan.ActionCode.ValueString(), actionConfiguration.ActionType, actionTypeFlow),
		)
	}

	plan.FlowVersion = types.Int64Null()
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(publishFlow(r.client, plan.ActionCode.ValueString(), plan.Flow, nil)...)
	if resp.Diagnostics.HasError() {
		return
	}

	publishedActionConfiguration, _, err := r.client.GetActionConfiguration(plan.ActionCode.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading action configuration",
			"Could not read action configuration code "+plan.ActionCode.ValueString()+" after publishing its flow: "+err.Error(),
		)
		return
	}

	fields, diags := readFlowFields(ctx, r.client, publishedActionConfiguration, plan.Flow)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	plan.ActionType = fields.ActionType
	plan.Flow = fields.Flow
	plan.FlowVersion = fields.FlowVersion

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

	actionConfiguration, statusCode, err := r.client.GetActionConfiguration(state.ActionCode.ValueString())

	if statusCode == 404 {
		resp.State.RemoveResource(ctx)
		return
	}

	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading action configuration",
			"Could not read action configuration code "+state.ActionCode.ValueString()+": "+err.Error(),
		)
		return
	}

	messagingTemplatesJson, err := json.Marshal(actionConfiguration.MessagingTemplates)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to marshal messaging templates",
			err.Error(),
		)
		return
	}

	verificationMethodsList, diags := types.ListValueFrom(ctx, types.StringType, actionConfiguration.VerificationMethods)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	promptToEnrollVerificationMethodsList, diags := types.ListValueFrom(ctx, types.StringType, actionConfiguration.PromptToEnrollVerificationMethods)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	fields, diags := readFlowFields(ctx, r.client, actionConfiguration, state.Flow)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	state.DefaultUserActionResult = defaultUserActionResultValue(actionConfiguration.DefaultUserActionResult)
	state.LastActionCreatedAt = types.StringValue(actionConfiguration.LastActionCreatedAt)
	state.TenantId = types.StringValue(actionConfiguration.TenantId)
	state.VerificationMethods = verificationMethodsList
	state.PromptToEnrollVerificationMethods = promptToEnrollVerificationMethodsList
	state.ActionType = fields.ActionType
	state.Flow = fields.Flow
	state.FlowVersion = fields.FlowVersion

	if actionConfiguration.MessagingTemplates != nil {
		state.MessagingTemplates = types.StringValue(string(messagingTemplatesJson))
	} else {
		state.MessagingTemplates = types.StringNull()
	}

	if len(actionConfiguration.DefaultVerificationMethod) > 0 {
		state.DefaultVerificationMethod = types.StringValue(actionConfiguration.DefaultVerificationMethod)
	} else {
		state.DefaultVerificationMethod = types.StringNull()
	}

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

	var state actionConfigurationResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var messagingTemplatesJson authsignal.MessagingTemplates

	verificationMethodsSlice := make([]string, 0, len(plan.VerificationMethods.Elements()))
	diags1 := plan.VerificationMethods.ElementsAs(ctx, &verificationMethodsSlice, false)
	resp.Diagnostics.Append(diags1...)
	if resp.Diagnostics.HasError() {
		return
	}

	promptToEnrollVerificationMethodsSlice := make([]string, 0, len(plan.PromptToEnrollVerificationMethods.Elements()))
	diags2 := plan.PromptToEnrollVerificationMethods.ElementsAs(ctx, &promptToEnrollVerificationMethodsSlice, false)
	resp.Diagnostics.Append(diags2...)
	if resp.Diagnostics.HasError() {
		return
	}

	var actionConfigurationToUpdate = authsignal.ActionConfiguration{}

	// The update endpoint does not accept a null default result.
	var actionConfigurationDefaultUserActionResult = plan.DefaultUserActionResult.ValueString()
	if len(actionConfigurationDefaultUserActionResult) > 0 {
		actionConfigurationToUpdate.DefaultUserActionResult = authsignal.SetValue(actionConfigurationDefaultUserActionResult)
	}

	var actionConfigurationDefaultVerificationMethod = plan.DefaultVerificationMethod.ValueString()
	if len(actionConfigurationDefaultVerificationMethod) > 0 {
		actionConfigurationToUpdate.DefaultVerificationMethod = authsignal.SetValue(actionConfigurationDefaultVerificationMethod)
	} else {
		actionConfigurationToUpdate.DefaultVerificationMethod = authsignal.SetNull(actionConfigurationDefaultVerificationMethod)
	}

	if len(verificationMethodsSlice) > 0 {
		actionConfigurationToUpdate.VerificationMethods = authsignal.SetValue(verificationMethodsSlice)
	} else {
		actionConfigurationToUpdate.VerificationMethods = authsignal.SetNull(verificationMethodsSlice)
	}

	if len(promptToEnrollVerificationMethodsSlice) > 0 {
		actionConfigurationToUpdate.PromptToEnrollVerificationMethods = authsignal.SetValue(promptToEnrollVerificationMethodsSlice)
	} else {
		actionConfigurationToUpdate.PromptToEnrollVerificationMethods = authsignal.SetNull(promptToEnrollVerificationMethodsSlice)
	}

	if len(string(plan.MessagingTemplates.ValueString())) > 0 {
		err := json.Unmarshal([]byte(plan.MessagingTemplates.ValueString()), &messagingTemplatesJson)
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to unmarshal messaging templates",
				err.Error(),
			)
			return
		}

		actionConfigurationToUpdate.MessagingTemplates = authsignal.SetValue(messagingTemplatesJson)
	} else {
		actionConfigurationToUpdate.MessagingTemplates = authsignal.SetNull(messagingTemplatesJson)
	}

	_, _, err2 := r.client.UpdateActionConfiguration(plan.ActionCode.ValueString(), actionConfigurationToUpdate)
	if err2 != nil {
		resp.Diagnostics.AddError(
			"Error Updating Authsignal action configuration",
			"Could not update action configuration, unexpected error: "+err2.Error(),
		)
		return
	}

	if isFlowActionType(plan.ActionType.ValueString()) {
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

			resp.Diagnostics.Append(publishFlow(r.client, plan.ActionCode.ValueString(), plan.Flow, expectedFlowVersion)...)
			if resp.Diagnostics.HasError() {
				return
			}
		}
	}

	updatedActionConfiguration, _, err := r.client.GetActionConfiguration(plan.ActionCode.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Authsignal action configuration",
			"Could not read action configuration ID "+plan.ActionCode.ValueString()+": "+err.Error(),
		)
		return
	}

	fields, diags := readFlowFields(ctx, r.client, updatedActionConfiguration, plan.Flow)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	plan.ActionCode = types.StringValue(updatedActionConfiguration.ActionCode)
	plan.DefaultUserActionResult = defaultUserActionResultValue(updatedActionConfiguration.DefaultUserActionResult)
	plan.TenantId = types.StringValue(updatedActionConfiguration.TenantId)
	plan.LastActionCreatedAt = types.StringValue(updatedActionConfiguration.LastActionCreatedAt)
	plan.ActionType = fields.ActionType
	plan.Flow = fields.Flow
	plan.FlowVersion = fields.FlowVersion

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

	_, _, err := r.client.DeleteActionConfiguration(state.ActionCode.ValueString())
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
