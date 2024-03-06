package provider

import (
	"context"
	"fmt"

	"authsignal.com/authsignal-management-sdk-go"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &actionConfigurationDataSource{}
	_ datasource.DataSourceWithConfigure = &actionConfigurationDataSource{}
)

// NewActionConfigurationDataSource is a helper function to simplify the provider implementation.
func NewActionConfigurationDataSource() datasource.DataSource {
	return &actionConfigurationDataSource{}
}

// actionConfigurationDataSource is the data source implementation.
type actionConfigurationDataSource struct {
	client *authsignal.Client
}

type actionConfigurationDataSourceModel struct {
	ActionCode              types.String `tfsdk:"action_code"`
	TenantId                types.String `tfsdk:"tenant_id"`
	DefaultUserActionResult types.String `tfsdk:"default_user_action_result"`
	LastActionCreatedAt     types.String `tfsdk:"last_action_created_at"`
}

// Metadata returns the data source type name.
func (d *actionConfigurationDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_action_configuration"
}

// Schema defines the schema for the data source.
func (d *actionConfigurationDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"action_code": schema.StringAttribute{
				Required: true,
			},
			"tenant_id": schema.StringAttribute{
				Computed: true,
			},
			"default_user_action_result": schema.StringAttribute{
				Computed: true,
			},
			"last_action_created_at": schema.StringAttribute{
				Computed: true,
			},
		},
	}
}

// Read refreshes the Terraform state with the latest data.
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

	actionConfigurationState := actionConfigurationDataSourceModel{
		ActionCode:              types.StringValue(actionConfiguration.ActionCode),
		TenantId:                types.StringValue(actionConfiguration.TenantId),
		DefaultUserActionResult: types.StringValue(actionConfiguration.DefaultUserActionResult),
		LastActionCreatedAt:     types.StringValue(actionConfiguration.LastActionCreatedAt),
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
