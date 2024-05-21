package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/authsignal/authsignal-management-go/v2"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
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
	_ resource.Resource                = &ruleResource{}
	_ resource.ResourceWithConfigure   = &ruleResource{}
	_ resource.ResourceWithImportState = &ruleResource{}
)

func NewRuleResource() resource.Resource {
	return &ruleResource{}
}

type ruleResource struct {
	client *authsignal.Client
}

type ruleResourceModel struct {
	Name                              types.String `tfsdk:"name"`
	Description                       types.String `tfsdk:"description"`
	IsActive                          types.Bool   `tfsdk:"is_active"`
	Priority                          types.Int64  `tfsdk:"priority"`
	ActionCode                        types.String `tfsdk:"action_code"`
	RuleId                            types.String `tfsdk:"rule_id"`
	TenantId                          types.String `tfsdk:"tenant_id"`
	Type                              types.String `tfsdk:"type"`
	VerificationMethods               types.List   `tfsdk:"verification_methods"`
	PromptToEnrollVerificationMethods types.List   `tfsdk:"prompt_to_enroll_verification_methods"`
	DefaultVerificationMethod         types.String `tfsdk:"default_verification_method"`
	Conditions                        types.String `tfsdk:"conditions"`
}

func (r *ruleResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_rule"
}

func (d *ruleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"action_code": schema.StringAttribute{
				Description: "The name of the action that users perform which you will track. (e.g 'login')",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Description: "A string used to name the rule.",
				Required:    true,
			},
			"description": schema.StringAttribute{
				Description: "A description of the rule.",
				Optional:    true,
			},
			"is_active": schema.BoolAttribute{
				Description: "Toggles whether or not the rule is actively applied.",
				Required:    true,
			},
			"priority": schema.Int64Attribute{
				Description: "Determines the order which the rules are applied in, where 0 is applied first, 1 is applied second...",
				Required:    true,
				Validators: []validator.Int64{
					int64validator.Between(0, 99),
				},
			},
			"type": schema.StringAttribute{
				Description: "The result that the rule should return when the conditions are met. (e.g. ALLOW, CHALLENGE)",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.OneOf([]string{"ALLOW", "CHALLENGE", "REVIEW", "BLOCK"}...),
				},
			},
			"verification_methods": schema.ListAttribute{
				ElementType: types.StringType,
				Description: "A list of permitted authenticators that can be used if the type of the rule is 'CHALLENGE'.",
				Optional:    true,
				Validators: []validator.List{
					listvalidator.ValueStringsAre(stringvalidator.OneOf([]string{"SMS", "AUTHENTICATOR_APP", "EMAIL_MAGIC_LINK", "EMAIL_OTP", "PUSH", "SECURITY_KEY", "PASSKEY", "VERIFF", "IPROOV", "REDROCK", "IDVERSE"}...)),
				},
			},
			"prompt_to_enroll_verification_methods": schema.ListAttribute{
				ElementType: types.StringType,
				Description: "If this is set then users will be prompted to add a passkey after a challenge is completed.",
				Optional:    true,
				Validators: []validator.List{
					listvalidator.ValueStringsAre(stringvalidator.OneOf([]string{"PASSKEY"}...)),
				},
			},
			"default_verification_method": schema.StringAttribute{
				Description: "Ignore the user's preference and choose which authenticator the Pre-built UI will present by default.",
				Optional:    true,
				Validators: []validator.String{
					stringvalidator.OneOf([]string{"SMS", "AUTHENTICATOR_APP", "EMAIL_MAGIC_LINK", "EMAIL_OTP", "PUSH", "SECURITY_KEY", "PASSKEY", "VERIFF", "IPROOV", "REDROCK", "IDVERSE"}...),
				},
			},
			"conditions": schema.StringAttribute{
				Description: "The logical conditions to match tracked actions against. If the conditions are met then the rule's type will be returned in the track action response.",
				Required:    true,
			},
			"rule_id": schema.StringAttribute{
				Description: "The ID of the rule.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
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

func (r *ruleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ruleResourceModel
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

	var conditionsJson authsignal.Condition

	err := json.Unmarshal([]byte(plan.Conditions.ValueString()), &conditionsJson)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to marshal conditions",
			err.Error(),
		)
		return
	}

	var ruleToCreate = authsignal.Rule{
		IsActive: authsignal.SetValue(plan.IsActive.ValueBool()),
		Priority: authsignal.SetValue(plan.Priority.ValueInt64()),
		Type:     authsignal.SetValue(plan.Type.ValueString()),
	}

	var ruleName = plan.Name.ValueString()
	if len(ruleName) > 0 {
		ruleToCreate.Name = authsignal.SetValue(ruleName)
	}
	var ruleDescription = plan.Description.ValueString()
	if len(ruleDescription) > 0 {
		ruleToCreate.Description = authsignal.SetValue(ruleDescription)
	}

	var ruleDefaultVerificationMethod = plan.DefaultVerificationMethod.ValueString()
	if len(ruleDefaultVerificationMethod) > 0 {
		ruleToCreate.DefaultVerificationMethod = authsignal.SetValue(ruleDefaultVerificationMethod)
	}

	if len(verificationMethodsSlice) > 0 {
		ruleToCreate.VerificationMethods = authsignal.SetValue(verificationMethodsSlice)
	}

	if len(promptToEnrollVerificationMethodsSlice) > 0 {
		ruleToCreate.PromptToEnrollVerificationMethods = authsignal.SetValue(promptToEnrollVerificationMethodsSlice)
	}

	if len(string(plan.Conditions.ValueString())) > 0 {
		ruleToCreate.Conditions = authsignal.SetValue(conditionsJson)
	}

	rule, err := r.client.CreateRule(plan.ActionCode.ValueString(), ruleToCreate)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating rule",
			"Could not create rule, unexpected error: "+err.Error(),
		)
		return
	}

	tflog.Info(ctx, fmt.Sprintf("%+v", rule))

	plan.RuleId = types.StringValue(rule.RuleId)
	plan.TenantId = types.StringValue(rule.TenantId)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *ruleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ruleResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	rule, err := r.client.GetRule(state.ActionCode.ValueString(), state.RuleId.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading rule",
			"Could not read rule code "+state.ActionCode.ValueString()+"-"+state.RuleId.ValueString()+": "+err.Error(),
		)
		return
	}

	verificationMethodsList, diags := types.ListValueFrom(ctx, types.StringType, rule.VerificationMethods)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	promptToEnrollVerificationMethodsList, diags := types.ListValueFrom(ctx, types.StringType, rule.PromptToEnrollVerificationMethods)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	conditionsJson, err := json.Marshal(rule.Conditions)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to marshal conditions",
			err.Error(),
		)
		return
	}

	state.Name = types.StringValue(rule.Name)
	state.IsActive = types.BoolValue(rule.IsActive)
	state.Priority = types.Int64Value(rule.Priority)
	state.Type = types.StringValue(rule.Type)
	state.VerificationMethods = verificationMethodsList
	state.PromptToEnrollVerificationMethods = promptToEnrollVerificationMethodsList

	state.TenantId = types.StringValue(rule.TenantId)

	if len(rule.Description) > 0 {
		state.Description = types.StringValue(rule.Description)
	} else {
		state.Description = types.StringNull()
	}

	if len(rule.DefaultVerificationMethod) > 0 {
		state.DefaultVerificationMethod = types.StringValue(rule.DefaultVerificationMethod)
	} else {
		state.DefaultVerificationMethod = types.StringNull()
	}

	if rule.Conditions != nil {
		state.Conditions = types.StringValue(string(conditionsJson))
	} else {
		state.Conditions = types.StringNull()
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *ruleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ruleResourceModel
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

	var conditionsJson authsignal.Condition

	err2 := json.Unmarshal([]byte(plan.Conditions.ValueString()), &conditionsJson)
	if err2 != nil {
		resp.Diagnostics.AddError(
			"Unable to marshal conditions",
			err2.Error(),
		)
		return
	}

	var ruleToUpdate = authsignal.Rule{
		IsActive: authsignal.SetValue(plan.IsActive.ValueBool()),
		Priority: authsignal.SetValue(plan.Priority.ValueInt64()),
		Type:     authsignal.SetValue(plan.Type.ValueString()),
	}

	var ruleName = plan.Name.ValueString()
	if len(ruleName) > 0 {
		ruleToUpdate.Name = authsignal.SetValue(ruleName)
	} else {
		ruleToUpdate.Name = authsignal.SetNull(ruleName)
	}

	var ruleDescription = plan.Description.ValueString()
	if len(ruleDescription) > 0 {
		ruleToUpdate.Description = authsignal.SetValue(ruleDescription)
	} else {
		ruleToUpdate.Description = authsignal.SetNull(ruleDescription)
	}

	var ruleDefaultVerificationMethod = plan.DefaultVerificationMethod.ValueString()
	if len(ruleDefaultVerificationMethod) > 0 {
		ruleToUpdate.DefaultVerificationMethod = authsignal.SetValue(ruleDefaultVerificationMethod)
	} else {
		ruleToUpdate.DefaultVerificationMethod = authsignal.SetNull(ruleDefaultVerificationMethod)
	}

	if len(verificationMethodsSlice) > 0 {
		ruleToUpdate.VerificationMethods = authsignal.SetValue(verificationMethodsSlice)
	} else {
		ruleToUpdate.VerificationMethods = authsignal.SetNull(verificationMethodsSlice)
	}

	if len(promptToEnrollVerificationMethodsSlice) > 0 {
		ruleToUpdate.PromptToEnrollVerificationMethods = authsignal.SetValue(promptToEnrollVerificationMethodsSlice)
	} else {
		ruleToUpdate.PromptToEnrollVerificationMethods = authsignal.SetNull(promptToEnrollVerificationMethodsSlice)
	}

	if len(string(plan.Conditions.ValueString())) > 0 {
		ruleToUpdate.Conditions = authsignal.SetValue(conditionsJson)
	} else {
		ruleToUpdate.Conditions = authsignal.SetNull(conditionsJson)
	}

	_, err := r.client.UpdateRule(plan.ActionCode.ValueString(), plan.RuleId.ValueString(), ruleToUpdate)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Authsignal rule",
			"Could not update rule, unexpected error: "+err.Error(),
		)
		return
	}

	updatedRule, err := r.client.GetRule(plan.ActionCode.ValueString(), plan.RuleId.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Authsignal rule",
			"Could not read rule using values: "+plan.ActionCode.ValueString()+"-"+plan.RuleId.ValueString()+":\n"+err.Error(),
		)
		return
	}

	plan.RuleId = types.StringValue(updatedRule.RuleId)
	plan.TenantId = types.StringValue(updatedRule.TenantId)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *ruleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ruleResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.DeleteRule(state.ActionCode.ValueString(), state.RuleId.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting Authsignal rule",
			"Could not delete rule, unexpected error: "+err.Error(),
		)
		return
	}
}

func (r *ruleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	ruleIdentifiers := strings.Split(req.ID, "/")

	if len(ruleIdentifiers) != 2 || ruleIdentifiers[0] == "" || ruleIdentifiers[1] == "" {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Expected import identifier with format: action_code/rule_id. Got: %q", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("action_code"), ruleIdentifiers[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("rule_id"), ruleIdentifiers[1])...)
}

func (r *ruleResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
