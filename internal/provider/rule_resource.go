package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

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
	_ resource.Resource                = &ruleResource{}
	_ resource.ResourceWithConfigure   = &ruleResource{}
	_ resource.ResourceWithImportState = &ruleResource{}
)

// NewActionConfigurationResource is a helper function to simplify the provider implementation.
func NewRuleResource() resource.Resource {
	return &ruleResource{}
}

// actionConfigurationResource is the resource implementation.
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

// Metadata returns the resource type name.
func (r *ruleResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_rule"
}

// Schema defines the schema for the resource.
func (d *ruleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
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
				Description: "Determines the order which the rules are applied in.",
				Required:    true,
			},
			"action_code": schema.StringAttribute{
				Description: "The action that the rule is applied to.",
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
				Description: "The ID of the tenant that the rule belongs to.",
				Computed:    true,
			},
			"type": schema.StringAttribute{
				Description: "The result that the rule should return (e.g. allow, challenge).",
				Required:    true,
			},
			"verification_methods": schema.ListAttribute{
				ElementType: types.StringType,
				Description: "Determines the order which the rules are applied in.",
				Optional:    true,
			},
			"prompt_to_enroll_verification_methods": schema.ListAttribute{
				ElementType: types.StringType,
				Description: "<description here>",
				Optional:    true,
			},
			"default_verification_method": schema.StringAttribute{
				Description: "The default verification method that users should be prompted with.",
				Optional:    true,
			},
			"conditions": schema.StringAttribute{
				Description: "The conditions of the rule.",
				Required:    true,
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
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
			"Unable to marshall conditions",
			err.Error(),
		)
		return
	}

	var ruleToCreate = authsignal.Rule{
		Name:                              plan.Name.ValueString(),
		Description:                       plan.Description.ValueString(),
		IsActive:                          plan.IsActive.ValueBool(),
		Priority:                          plan.Priority.ValueInt64(),
		Type:                              plan.Type.ValueString(),
		VerificationMethods:               verificationMethodsSlice,
		PromptToEnrollVerificationMethods: promptToEnrollVerificationMethodsSlice,
		DefaultVerificationMethod:         plan.DefaultVerificationMethod.ValueString(),
		Conditions:                        conditionsJson,
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

// Read refreshes the Terraform state with the latest data.
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
			"Unable to marshall conditions",
			err.Error(),
		)
		return
	}

	state.Name = types.StringValue(rule.Name)
	state.Description = types.StringValue(rule.Description)
	state.IsActive = types.BoolValue(rule.IsActive)
	state.Priority = types.Int64Value(rule.Priority)
	state.Type = types.StringValue(rule.Type)
	state.VerificationMethods = verificationMethodsList
	state.PromptToEnrollVerificationMethods = promptToEnrollVerificationMethodsList
	state.DefaultVerificationMethod = types.StringValue(rule.DefaultVerificationMethod)
	state.Conditions = types.StringValue(string(conditionsJson))
	state.TenantId = types.StringValue(rule.TenantId)

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update updates the resource and sets the updated Terraform state on success.
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
			"Unable to marshall conditions",
			err2.Error(),
		)
		return
	}

	var ruleToUpdate = authsignal.Rule{
		Name:                              plan.Name.ValueString(),
		Description:                       plan.Description.ValueString(),
		IsActive:                          plan.IsActive.ValueBool(),
		Priority:                          plan.Priority.ValueInt64(),
		Type:                              plan.Type.ValueString(),
		VerificationMethods:               verificationMethodsSlice,
		PromptToEnrollVerificationMethods: promptToEnrollVerificationMethodsSlice,
		DefaultVerificationMethod:         plan.DefaultVerificationMethod.ValueString(),
		Conditions:                        conditionsJson,
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
			"Could not read rule code "+plan.ActionCode.ValueString()+"-"+plan.RuleId.ValueString()+": "+err.Error(),
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

// Delete deletes the resource and removes the Terraform state on success.
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