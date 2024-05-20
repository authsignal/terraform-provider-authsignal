package provider

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/authsignal/authsignal-management-go/v2"
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
	ActionCode              types.String `tfsdk:"action_code"`
	TenantId                types.String `tfsdk:"tenant_id"`
	DefaultUserActionResult types.String `tfsdk:"default_user_action_result"`
	LastActionCreatedAt     types.String `tfsdk:"last_action_created_at"`
	MessagingTemplates      types.String `tfsdk:"messaging_templates"`
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
				Description: "The default action behavior if no rules match. (i.e 'CHALLENGE')",
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
				Optional:    true,
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

	actionConfiguration, err := d.client.GetActionConfiguration(data.ActionCode.ValueString())

	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read Authsignal ActionConfiguration",
			err.Error(),
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

	actionConfigurationState := actionConfigurationDataSourceModel{
		ActionCode:              types.StringValue(actionConfiguration.ActionCode),
		TenantId:                types.StringValue(actionConfiguration.TenantId),
		DefaultUserActionResult: types.StringValue(actionConfiguration.DefaultUserActionResult),
		LastActionCreatedAt:     types.StringValue(actionConfiguration.LastActionCreatedAt),
	}

	if actionConfiguration.MessagingTemplates != nil {
		actionConfigurationState.MessagingTemplates = types.StringValue(string(messagingTemplatesJson))
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
