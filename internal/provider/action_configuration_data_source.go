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
	_ datasource.DataSource              = &actionConfigurationDataSource{}
	_ datasource.DataSourceWithConfigure = &actionConfigurationDataSource{}
)

func NewActionConfigurationDataSource() datasource.DataSource {
	return &actionConfigurationDataSource{}
}

type actionConfigurationDataSource struct {
	client *authsignal.Client
}

type actionConfigurationDataSourceModel struct {
	ActionCode                        types.String `tfsdk:"action_code"`
	TenantId                          types.String `tfsdk:"tenant_id"`
	DefaultUserActionResult           types.String `tfsdk:"default_user_action_result"`
	LastActionCreatedAt               types.String `tfsdk:"last_action_created_at"`
	MessagingTemplates                types.String `tfsdk:"messaging_templates"`
	VerificationMethods               types.List   `tfsdk:"verification_methods"`
	PromptToEnrollVerificationMethods types.List   `tfsdk:"prompt_to_enroll_verification_methods"`
	DefaultVerificationMethod         types.String `tfsdk:"default_verification_method"`
}

func (d *actionConfigurationDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_action_configuration"
}

func (d *actionConfigurationDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"action_code": schema.StringAttribute{
				Description: "The name of the action that users perform which you will track. (e.g 'login')",
				Required:    true,
			},
			"default_user_action_result": schema.StringAttribute{
				Description: "The default action behavior if no rules match. (i.e 'CHALLENGE').",
				Computed:    true,
			},
			"last_action_created_at": schema.StringAttribute{
				Description: "The date of when an action was last tracked for any user.",
				Computed:    true,
			},
			"tenant_id": schema.StringAttribute{
				Description: "The ID of your tenant. This can be found in the admin portal.",
				Computed:    true,
			},
			"messaging_templates": schema.StringAttribute{
				Description: "Optional messaging templates to be shown in Authsignal's pre-built UI.",
				Computed:    true,
			},
			"verification_methods": schema.ListAttribute{
				ElementType: types.StringType,
				Description: "A list of permitted authenticators that can be used if the result of the action is 'CHALLENGE'.",
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
		},
	}
}

func (d *actionConfigurationDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {

	var data actionConfigurationDataSourceModel
	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	actionConfiguration, _, err := d.client.GetActionConfiguration(data.ActionCode.ValueString())

	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read Authsignal ActionConfiguration",
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

	messagingTemplatesJson, err := json.Marshal(actionConfiguration.MessagingTemplates)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to marshal messaging templates",
			err.Error(),
		)
		return
	}

	actionConfigurationState := actionConfigurationDataSourceModel{
		ActionCode:                        types.StringValue(actionConfiguration.ActionCode),
		TenantId:                          types.StringValue(actionConfiguration.TenantId),
		DefaultUserActionResult:           types.StringValue(actionConfiguration.DefaultUserActionResult),
		LastActionCreatedAt:               types.StringValue(actionConfiguration.LastActionCreatedAt),
		VerificationMethods:               verificationMethodsList,
		PromptToEnrollVerificationMethods: promptToEnrollVerificationMethodsList,
	}

	if actionConfiguration.MessagingTemplates != nil {
		actionConfigurationState.MessagingTemplates = types.StringValue(string(messagingTemplatesJson))
	} else {
		actionConfigurationState.MessagingTemplates = types.StringNull()
	}

	if len(actionConfiguration.DefaultVerificationMethod) > 0 {
		actionConfigurationState.DefaultVerificationMethod = types.StringValue(actionConfiguration.DefaultVerificationMethod)
	} else {
		actionConfigurationState.DefaultVerificationMethod = types.StringNull()
	}

	diags2 := resp.State.Set(ctx, &actionConfigurationState)

	resp.Diagnostics.Append(diags2...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (d *actionConfigurationDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
