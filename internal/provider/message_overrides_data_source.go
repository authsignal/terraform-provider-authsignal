package provider

import (
	"context"
	"fmt"

	"github.com/authsignal/authsignal-management-go/v5"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ datasource.DataSource              = &messageOverridesDataSource{}
	_ datasource.DataSourceWithConfigure = &messageOverridesDataSource{}
)

func NewMessageOverridesDataSource() datasource.DataSource {
	return &messageOverridesDataSource{}
}

type messageOverridesDataSource struct {
	client *authsignal.Client
}

type messageOverridesDataSourceModel struct {
	Overrides types.Map `tfsdk:"overrides"`
}

func (d *messageOverridesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_message_overrides"
}

func (d *messageOverridesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieves a tenant's currently configured pre-built UI message overrides.",
		Attributes: map[string]schema.Attribute{
			"overrides": schema.MapAttribute{
				Description: "Override copy keyed by locale (e.g. `en`, `pt-br`), then by message override ID (e.g. `sms-code-entry.heading`).",
				ElementType: messageOverridesElemType,
				Computed:    true,
			},
		},
	}
}

func (d *messageOverridesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	messageOverrides, _, err := d.client.GetMessageOverrides()
	if err != nil {
		resp.Diagnostics.AddError("Unable to Read Authsignal Message Overrides", err.Error())
		return
	}

	overridesValue, diags := messageOverridesToMapValue(ctx, messageOverrides.MessageOverrides)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, messageOverridesDataSourceModel{Overrides: overridesValue})
	resp.Diagnostics.Append(diags...)
}

func (d *messageOverridesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
