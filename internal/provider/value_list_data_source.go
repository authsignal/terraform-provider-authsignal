package provider

import (
	"context"
	"fmt"

	"github.com/authsignal/authsignal-management-go/v2"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ datasource.DataSource              = &valueListDataSource{}
	_ datasource.DataSourceWithConfigure = &valueListDataSource{}
)

func NewValueListDataSource() datasource.DataSource {
	return &valueListDataSource{}
}

type valueListDataSource struct {
	client *authsignal.Client
}

type valueListDataSourceModel struct {
	Name                  types.String `tfsdk:"name"`
	Alias                 types.String `tfsdk:"alias"`
	ItemType              types.String `tfsdk:"item_type"`
	IsActive              types.Bool   `tfsdk:"is_active"`
	ValueListItemsStrings types.List   `tfsdk:"value_list_items_strings"`
	ValueListItemsNumbers types.List   `tfsdk:"value_list_items_numbers"`
}

func (d *valueListDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_value_list"
}

func (d *valueListDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Description: "The name of the list.",
				Computed:    true,
			},
			"alias": schema.StringAttribute{
				Description: "The hypenated name of the list.",
				Required:    true,
			},
			"item_type": schema.StringAttribute{
				Description: "The type of the items in the list. Allowed values: `string`, `number`.",
				Computed:    true,
			},
			"value_list_items_strings": schema.ListAttribute{
				ElementType: types.StringType,
				Description: "The list of items.",
				Computed:    true,
			},
			"value_list_items_numbers": schema.ListAttribute{
				ElementType: types.Float64Type,
				Description: "The list of items.",
				Computed:    true,
			},
			"is_active": schema.BoolAttribute{
				Description: "Whether or not the list is active. This currently has no effect.",
				Computed:    true,
			},
		},
	}
}

func RestructureValueList(ctx context.Context, itemType string, valueListFromApi []authsignal.ValueListItem) (RestructuredValueListStrings basetypes.ListValue, RestructuredValueListNumbers basetypes.ListValue, diags diag.Diagnostics) {
	if len(valueListFromApi) == 0 {
		return types.ListNull(types.StringType), types.ListNull(types.NumberType), nil
	}

	if itemType == "number" {
		valueListItemsNumbers, diags := types.ListValueFrom(ctx, types.Float64Type, valueListFromApi)
		if diags.HasError() {
			return RestructuredValueListStrings, RestructuredValueListNumbers, diags
		}
		return types.ListNull(types.StringType), valueListItemsNumbers, nil
	}

	valueListItemsStrings, diags := types.ListValueFrom(ctx, types.StringType, valueListFromApi)
	if diags.HasError() {
		return RestructuredValueListStrings, RestructuredValueListNumbers, diags
	}
	return valueListItemsStrings, types.ListNull(types.NumberType), nil
}

func (d *valueListDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data valueListDataSourceModel
	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	valueList, err := d.client.GetValueList(data.Alias.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to Read Authsignal ValueList", err.Error())
		return
	}

	valueListState := valueListDataSourceModel{
		Name:     types.StringValue(valueList.Name),
		Alias:    types.StringValue(valueList.Alias),
		ItemType: types.StringValue(valueList.ItemType),
		IsActive: types.BoolValue(valueList.IsActive),
	}

	valueListItemsStrings, valueListItemsNumbers, diags := RestructureValueList(ctx, valueList.ItemType, valueList.ValueListItems)

	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	valueListState.ValueListItemsStrings = valueListItemsStrings
	valueListState.ValueListItemsNumbers = valueListItemsNumbers

	diags = resp.State.Set(ctx, &valueListState)
	resp.Diagnostics.Append(diags...)
}

func (d *valueListDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
