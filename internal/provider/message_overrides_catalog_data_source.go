package provider

import (
	"context"
	"fmt"

	"github.com/authsignal/authsignal-management-go/v6"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ datasource.DataSource              = &messageOverridesCatalogDataSource{}
	_ datasource.DataSourceWithConfigure = &messageOverridesCatalogDataSource{}
)

func NewMessageOverridesCatalogDataSource() datasource.DataSource {
	return &messageOverridesCatalogDataSource{}
}

type messageOverridesCatalogDataSource struct {
	client *authsignal.Client
}

type messageOverridesCatalogDataSourceModel struct {
	CatalogVersion types.Int64                          `tfsdk:"catalog_version"`
	Screens        []messageOverridesCatalogScreenModel `tfsdk:"screens"`
	Points         []messageOverridesCatalogPointModel  `tfsdk:"points"`
}

type messageOverridesCatalogScreenModel struct {
	Id     types.String `tfsdk:"id"`
	Label  types.String `tfsdk:"label"`
	Family types.String `tfsdk:"family"`
}

type messageOverridesCatalogPointModel struct {
	PublicId            types.String `tfsdk:"public_id"`
	Screen              types.String `tfsdk:"screen"`
	Role                types.String `tfsdk:"role"`
	Item                types.String `tfsdk:"item"`
	Label               types.String `tfsdk:"label"`
	Products            types.List   `tfsdk:"products"`
	MaxLength           types.Int64  `tfsdk:"max_length"`
	AllowedPlaceholders types.List   `tfsdk:"allowed_placeholders"`
	AllowedTags         types.List   `tfsdk:"allowed_tags"`
	DefaultCopy         types.Map    `tfsdk:"default_copy"`
}

func (d *messageOverridesCatalogDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_message_overrides_catalog"
}

func (d *messageOverridesCatalogDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieves the catalog of overridable pre-built UI message points, including their default copy, allowed placeholders and tags, and maximum length. Use it to discover valid `authsignal_message_overrides` IDs and locales.",
		Attributes: map[string]schema.Attribute{
			"catalog_version": schema.Int64Attribute{
				Description: "The version of the message override catalog.",
				Computed:    true,
			},
			"screens": schema.ListNestedAttribute{
				Description: "The pre-built UI screens that contain overridable message points.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":     schema.StringAttribute{Computed: true},
						"label":  schema.StringAttribute{Computed: true},
						"family": schema.StringAttribute{Computed: true},
					},
				},
			},
			"points": schema.ListNestedAttribute{
				Description: "The overridable message points.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"public_id": schema.StringAttribute{
							Description: "The message override ID, e.g. `sms-code-entry.heading`.",
							Computed:    true,
						},
						"screen": schema.StringAttribute{
							Description: "The ID of the screen this message point belongs to.",
							Computed:    true,
						},
						"role": schema.StringAttribute{
							Description: "The role of the message point within the screen, e.g. `heading`, `description`, `primaryCta`.",
							Computed:    true,
						},
						"item": schema.StringAttribute{
							Description: "An optional sub-item identifier for list-style message points.",
							Computed:    true,
						},
						"label": schema.StringAttribute{
							Description: "A human-friendly label for the message point.",
							Computed:    true,
						},
						"products": schema.ListAttribute{
							Description: "The pre-built UI product versions this message point applies to.",
							ElementType: types.StringType,
							Computed:    true,
						},
						"max_length": schema.Int64Attribute{
							Description: "The maximum allowed length for an override value.",
							Computed:    true,
						},
						"allowed_placeholders": schema.ListAttribute{
							Description: "Placeholders (e.g. `{tenantName}`) that may appear in an override value.",
							ElementType: types.StringType,
							Computed:    true,
						},
						"allowed_tags": schema.ListAttribute{
							Description: "Rich-text tags (e.g. `identifier`, `link`) that may appear in an override value.",
							ElementType: types.StringType,
							Computed:    true,
						},
						"default_copy": schema.MapAttribute{
							Description: "The default copy for each locale.",
							ElementType: types.StringType,
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (d *messageOverridesCatalogDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	catalog, _, err := d.client.GetMessageOverridesCatalog()
	if err != nil {
		resp.Diagnostics.AddError("Unable to Read Authsignal Message Overrides Catalog", err.Error())
		return
	}

	state := messageOverridesCatalogDataSourceModel{
		CatalogVersion: types.Int64Value(catalog.CatalogVersion),
	}

	for _, screen := range catalog.Screens {
		state.Screens = append(state.Screens, messageOverridesCatalogScreenModel{
			Id:     types.StringValue(screen.Id),
			Label:  types.StringValue(screen.Label),
			Family: types.StringValue(screen.Family),
		})
	}

	for _, point := range catalog.Points {
		products, diags := types.ListValueFrom(ctx, types.StringType, point.Products)
		resp.Diagnostics.Append(diags...)
		allowedPlaceholders, diags := types.ListValueFrom(ctx, types.StringType, point.AllowedPlaceholders)
		resp.Diagnostics.Append(diags...)
		allowedTags, diags := types.ListValueFrom(ctx, types.StringType, point.AllowedTags)
		resp.Diagnostics.Append(diags...)
		defaultCopy, diags := types.MapValueFrom(ctx, types.StringType, point.DefaultCopy)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		item := types.StringNull()
		if len(point.Item) > 0 {
			item = types.StringValue(point.Item)
		}

		state.Points = append(state.Points, messageOverridesCatalogPointModel{
			PublicId:            types.StringValue(point.PublicId),
			Screen:              types.StringValue(point.Screen),
			Role:                types.StringValue(point.Role),
			Item:                item,
			Label:               types.StringValue(point.Label),
			Products:            products,
			MaxLength:           types.Int64Value(point.MaxLength),
			AllowedPlaceholders: allowedPlaceholders,
			AllowedTags:         allowedTags,
			DefaultCopy:         defaultCopy,
		})
	}

	diags := resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (d *messageOverridesCatalogDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
