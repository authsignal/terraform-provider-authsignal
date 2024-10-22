package provider

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/authsignal/authsignal-management-go/v3"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ datasource.DataSource              = &ruleDataSource{}
	_ datasource.DataSourceWithConfigure = &ruleDataSource{}
)

func NewRuleDataSource() datasource.DataSource {
	return &ruleDataSource{}
}

type ruleDataSource struct {
	client *authsignal.Client
}

type ruleDataSourceModel struct {
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

func (d *ruleDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_rule"
}

func (d *ruleDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"action_code": schema.StringAttribute{
				Description: "The name of the action that users perform which you will track. (e.g 'login')",
				Required:    true,
			},
			"name": schema.StringAttribute{
				Description: "A string used to name the rule.",
				Computed:    true,
			},
			"description": schema.StringAttribute{
				Description: "A description of the rule.",
				Computed:    true,
			},
			"is_active": schema.BoolAttribute{
				Description: "Toggles whether or not the rule is actively applied.",
				Computed:    true,
			},
			"priority": schema.Int64Attribute{
				Description: "Determines the order which the rules are applied in, where 0 is applied first, 1 is applied second...",
				Computed:    true,
			},
			"type": schema.StringAttribute{
				Description: "The result that the rule should return when the conditions are met. (e.g. ALLOW, CHALLENGE)",
				Computed:    true,
			},
			"verification_methods": schema.ListAttribute{
				ElementType: types.StringType,
				Description: "A list of permitted authenticators that can be used if the type of the rule is 'CHALLENGE'.",
				Computed:    true,
			},
			"prompt_to_enroll_verification_methods": schema.ListAttribute{
				ElementType: types.StringType,
				Description: "If this is set then users will be prompted to add a passkey after a challenge is completed.",
				Computed:    true,
			},
			"default_verification_method": schema.StringAttribute{
				Description: "Ignore the user's preference and choose which authenticator the Pre-built UI will present by default.",
				Computed:    true,
			},
			"conditions": schema.StringAttribute{
				Description: "The logical conditions to match tracked actions against. If the conditions are met then the rule's type will be returned in the track action response.",
				Computed:    true,
			},
			"rule_id": schema.StringAttribute{
				Description: "The ID of the rule. This can be obtained from the Authsignal portal.",
				Required:    true,
			},
			"tenant_id": schema.StringAttribute{
				Description: "The ID of your tenant. This can be found in the admin portal.",
				Computed:    true,
			},
		},
	}
}

func (d *ruleDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {

	var data ruleDataSourceModel
	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	rule, _, err := d.client.GetRule(data.ActionCode.ValueString(), data.RuleId.ValueString())

	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read Authsignal Rule",
			err.Error(),
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

	ruleState := ruleDataSourceModel{
		Name:                              types.StringValue(rule.Name),
		IsActive:                          types.BoolValue(rule.IsActive),
		Priority:                          types.Int64Value(rule.Priority),
		ActionCode:                        types.StringValue(rule.ActionCode),
		RuleId:                            types.StringValue(rule.RuleId),
		TenantId:                          types.StringValue(rule.TenantId),
		Type:                              types.StringValue(rule.Type),
		VerificationMethods:               verificationMethodsList,
		PromptToEnrollVerificationMethods: promptToEnrollVerificationMethodsList,
		Conditions:                        types.StringValue(string(conditionsJson)),
	}

	if len(rule.Description) > 0 {
		ruleState.Description = types.StringValue(rule.Description)
	} else {
		ruleState.Description = types.StringNull()
	}

	if len(rule.DefaultVerificationMethod) > 0 {
		ruleState.DefaultVerificationMethod = types.StringValue(rule.DefaultVerificationMethod)
	} else {
		ruleState.DefaultVerificationMethod = types.StringNull()
	}

	diags2 := resp.State.Set(ctx, &ruleState)
	resp.Diagnostics.Append(diags2...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (d *ruleDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	tflog.Info(ctx, "Configuring Authsignal client")

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

	d.client = client
}
