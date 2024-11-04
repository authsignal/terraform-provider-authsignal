package provider

import (
	"context"
	"fmt"

	"github.com/authsignal/authsignal-management-go/v3"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ datasource.DataSource              = &customDataPointDataSource{}
	_ datasource.DataSourceWithConfigure = &customDataPointDataSource{}
)

func NewCustomDataPointDataSource() datasource.DataSource {
	return &customDataPointDataSource{}
}

type customDataPointDataSource struct {
	client *authsignal.Client
}

type customDataPointDataSourceModel struct {
	Id          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	DataType    types.String `tfsdk:"data_type"`
	ModelType   types.String `tfsdk:"model_type"`
	Description types.String `tfsdk:"description"`
}

func (d *customDataPointDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_custom_data_point"
}

func (d *customDataPointDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The id of the custom data point.",
				Required:    true,
			},
			"name": schema.StringAttribute{
				Description: "The name of the custom data point.",
				Computed:    true,
			},
			"data_type": schema.StringAttribute{
				Description: "The data type of the custom data point. Allowed values: `text`, `number`, `boolean`, 'multiselect'.",
				Computed:    true,
			},
			"model_type": schema.StringAttribute{
				Description: "The model type of the custom data point. Allowed values: `action`, `user`.",
				Computed:    true,
			},
			"description": schema.StringAttribute{
				Description: "The description of the custom data point.",
				Computed:    true,
			},
		},
	}
}

func (d *customDataPointDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data customDataPointDataSourceModel
	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	customDataPoint, _, err := d.client.GetCustomDataPoint(data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to Read Authsignal ValueList", err.Error())
		return
	}

	customDataPointState := customDataPointDataSourceModel{
		Id:          types.StringValue(customDataPoint.Id),
		Name:        types.StringValue(customDataPoint.Name),
		DataType:    types.StringValue(customDataPoint.DataType),
		ModelType:   types.StringValue(customDataPoint.ModelType),
		Description: types.StringValue(customDataPoint.Description),
	}

	diags = resp.State.Set(ctx, &customDataPointState)
	resp.Diagnostics.Append(diags...)
}

func (d *customDataPointDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
