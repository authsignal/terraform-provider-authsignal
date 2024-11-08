package provider

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/authsignal/authsignal-management-go/v3"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
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
	ActionCode                        types.String `tfsdk:"action_code"`
	LastActionCreatedAt               types.String `tfsdk:"last_action_created_at"`
	TenantId                          types.String `tfsdk:"tenant_id"`
	DefaultUserActionResult           types.String `tfsdk:"default_user_action_result"`
	MessagingTemplates                types.String `tfsdk:"messaging_templates"`
	VerificationMethods               types.List   `tfsdk:"verification_methods"`
	PromptToEnrollVerificationMethods types.List   `tfsdk:"prompt_to_enroll_verification_methods"`
	DefaultVerificationMethod         types.String `tfsdk:"default_verification_method"`
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
				Description: "The default action behavior if no rules match. Allowed values: `ALLOW`, `CHALLENGE`, `REVIEW`, `BLOCK`.",
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
			"messaging_templates": schema.StringAttribute{
				Description: "Optional messaging templates to be shown in Authsignal's pre-built UI.",
				Optional:    true,
			},
			"verification_methods": schema.ListAttribute{
				ElementType: types.StringType,
				Description: "A list of permitted authenticators that can be used if the result of the action is 'CHALLENGE'. Allowed values: `SMS`, `AUTHENTICATOR_APP`, `EMAIL_MAGIC_LINK`, `EMAIL_OTP`, `PUSH`, `SECURITY_KEY`, `PASSKEY`, `VERIFF`, `IPROOV`, `REDROCK`, `IDVERSE`.",
				Optional:    true,
				Validators: []validator.List{
					listvalidator.ValueStringsAre(stringvalidator.OneOf([]string{"SMS", "AUTHENTICATOR_APP", "EMAIL_MAGIC_LINK", "EMAIL_OTP", "PUSH", "SECURITY_KEY", "PASSKEY", "VERIFF", "IPROOV", "REDROCK", "IDVERSE"}...)),
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
				Description: "Ignore the user's preference and choose which authenticator the Pre-built UI will present by default. Allowed values: `SMS`, `AUTHENTICATOR_APP`, `EMAIL_MAGIC_LINK`, `EMAIL_OTP`, `PUSH`, `SECURITY_KEY`, `PASSKEY`, `VERIFF`, `IPROOV`, `REDROCK`, `IDVERSE`.",
				Optional:    true,
				Validators: []validator.String{
					stringvalidator.OneOf([]string{"SMS", "AUTHENTICATOR_APP", "EMAIL_MAGIC_LINK", "EMAIL_OTP", "PUSH", "SECURITY_KEY", "PASSKEY", "VERIFF", "IPROOV", "REDROCK", "IDVERSE"}...),
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

	state.DefaultUserActionResult = types.StringValue(actionConfiguration.DefaultUserActionResult)
	state.LastActionCreatedAt = types.StringValue(actionConfiguration.LastActionCreatedAt)
	state.TenantId = types.StringValue(actionConfiguration.TenantId)
	state.VerificationMethods = verificationMethodsList
	state.PromptToEnrollVerificationMethods = promptToEnrollVerificationMethodsList

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

	var actionConfigurationDefaultUserActionResult = plan.DefaultUserActionResult.ValueString()
	if len(actionConfigurationDefaultUserActionResult) > 0 {
		actionConfigurationToUpdate.DefaultUserActionResult = authsignal.SetValue(actionConfigurationDefaultUserActionResult)
	} else {
		actionConfigurationToUpdate.DefaultUserActionResult = authsignal.SetNull(actionConfigurationDefaultUserActionResult)
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

	updatedActionConfiguration, _, err := r.client.GetActionConfiguration(plan.ActionCode.ValueString())
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
