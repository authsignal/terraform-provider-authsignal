package provider

import (
	"context"
	"fmt"

	"github.com/authsignal/authsignal-management-go/v2"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ datasource.DataSource              = &themeDataSource{}
	_ datasource.DataSourceWithConfigure = &themeDataSource{}
)

func NewThemeDataSource() datasource.DataSource {
	return &themeDataSource{}
}

type themeDataSource struct {
	client *authsignal.Client
}

func (d *themeDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_theme"
}

func (d *themeDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Description: "The name of the tenant which is visible to users.",
				Computed:    true,
			},
			"logo_url": schema.StringAttribute{
				Description: "The URL of an image to be used as a logo for the tenant.",
				Computed:    true,
			},
			"watermark_url": schema.StringAttribute{
				Description: "placeholder",
				Computed:    true,
			},
			"favicon_url": schema.StringAttribute{
				Description: "The URL of an image to be used as a favicon for the tenant",
				Computed:    true,
			},
			"primary_color": schema.StringAttribute{
				Description: "The primary color for the tenant.",
				Computed:    true,
			},
			"borders": schema.SingleNestedAttribute{
				Attributes: map[string]schema.Attribute{
					"button_border_radius": schema.Int64Attribute{
						Computed: true,
					},
					"button_border_width": schema.Int64Attribute{
						Computed: true,
					},
					"card_border_radius": schema.Int64Attribute{
						Computed: true,
					},
					"card_border_width": schema.Int64Attribute{
						Computed: true,
					},
					"input_border_radius": schema.Int64Attribute{
						Computed: true,
					},
					"input_border_width": schema.Int64Attribute{
						Computed: true,
					},
					"container_border_radius": schema.Int64Attribute{
						Computed: true,
					},
				},
				Computed: true,
			},
			"container": schema.SingleNestedAttribute{
				Attributes: map[string]schema.Attribute{
					"content_alignment": schema.StringAttribute{
						Computed: true,
					},
					"position": schema.StringAttribute{
						Computed: true,
					},
					"padding": schema.Int64Attribute{
						Computed: true,
					},
					"logo_alignment": schema.StringAttribute{
						Computed: true,
					},
					"logo_position": schema.StringAttribute{
						Computed: true,
					},
					"logo_height": schema.Int64Attribute{
						Computed: true,
					},
				},
				Computed: true,
			},
			"typography": schema.SingleNestedAttribute{
				Attributes: map[string]schema.Attribute{
					"display": schema.SingleNestedAttribute{
						Attributes: map[string]schema.Attribute{
							"font_url": schema.StringAttribute{
								Description: "The URL of a font file to be used for the tenant.",
								Computed:    true,
							},
						},
						Computed: true,
					},
				},
				Computed: true,
			},
			"page_background": schema.SingleNestedAttribute{
				Attributes: map[string]schema.Attribute{
					"background_color": schema.StringAttribute{
						Description: "The color to be used for the background in the pre-built UI.",
						Computed:    true,
					},
					"background_image_url": schema.StringAttribute{
						Description: "The URL of an image which will be used as the background in the pre-built UI.",
						Computed:    true,
					},
				},
				Computed: true,
			},
			"colors": schema.SingleNestedAttribute{
				Attributes: map[string]schema.Attribute{
					"button_primary_text": schema.StringAttribute{
						Computed: true,
					},
					"button_primary_border": schema.StringAttribute{
						Computed: true,
					},
					"button_secondary_text": schema.StringAttribute{
						Computed: true,
					},
					"button_secondary_background": schema.StringAttribute{
						Computed: true,
					},
					"button_secondary_border": schema.StringAttribute{
						Computed: true,
					},
					"card_background": schema.StringAttribute{
						Computed: true,
					},
					"card_border": schema.StringAttribute{
						Computed: true,
					},
					"input_background": schema.StringAttribute{
						Computed: true,
					},
					"input_border": schema.StringAttribute{
						Computed: true,
					},
					"link": schema.StringAttribute{
						Computed: true,
					},
					"heading_text": schema.StringAttribute{
						Computed: true,
					},
					"body_text": schema.StringAttribute{
						Computed: true,
					},
					"container_background": schema.StringAttribute{
						Computed: true,
					},
					"container_border": schema.StringAttribute{
						Computed: true,
					},
					"divider": schema.StringAttribute{
						Computed: true,
					},
					"icon": schema.StringAttribute{
						Computed: true,
					},
					"loader": schema.StringAttribute{
						Computed: true,
					},
					"positive": schema.StringAttribute{
						Computed: true,
					},
					"critical": schema.StringAttribute{
						Computed: true,
					},
					"information": schema.StringAttribute{
						Computed: true,
					},
					"hover": schema.StringAttribute{
						Computed: true,
					},
					"focus": schema.StringAttribute{
						Computed: true,
					},
				},
				Computed: true,
			},
			"dark_mode": schema.SingleNestedAttribute{
				Attributes: map[string]schema.Attribute{
					"logo_url": schema.StringAttribute{
						Description: "The URL of an image to be used as a logo for the tenant.",
						Computed:    true,
					},
					"watermark_url": schema.StringAttribute{
						Description: "placeholder",
						Computed:    true,
					},
					"favicon_url": schema.StringAttribute{
						Description: "The URL of an image to be used as a favicon for the tenant",
						Computed:    true,
					},
					"primary_color": schema.StringAttribute{
						Description: "The primary color for the tenant.",
						Computed:    true,
					},
					"borders": schema.SingleNestedAttribute{
						Attributes: map[string]schema.Attribute{
							"button_border_radius": schema.Int64Attribute{
								Computed: true,
							},
							"button_border_width": schema.Int64Attribute{
								Computed: true,
							},
							"card_border_radius": schema.Int64Attribute{
								Computed: true,
							},
							"card_border_width": schema.Int64Attribute{
								Computed: true,
							},
							"input_border_radius": schema.Int64Attribute{
								Computed: true,
							},
							"input_border_width": schema.Int64Attribute{
								Computed: true,
							},
							"container_border_radius": schema.Int64Attribute{
								Computed: true,
							},
						},
						Computed: true,
					},
					"container": schema.SingleNestedAttribute{
						Attributes: map[string]schema.Attribute{
							"content_alignment": schema.StringAttribute{
								Computed: true,
							},
							"position": schema.StringAttribute{
								Computed: true,
							},
							"padding": schema.Int64Attribute{
								Computed: true,
							},
							"logo_alignment": schema.StringAttribute{
								Computed: true,
							},
							"logo_position": schema.StringAttribute{
								Computed: true,
							},
							"logo_height": schema.Int64Attribute{
								Computed: true,
							},
						},
						Computed: true,
					},
					"typography": schema.SingleNestedAttribute{
						Attributes: map[string]schema.Attribute{
							"display": schema.SingleNestedAttribute{
								Attributes: map[string]schema.Attribute{
									"font_url": schema.StringAttribute{
										Description: "The URL of a font file to be used for the tenant.",
										Computed:    true,
									},
								},
								Computed: true,
							},
						},
						Computed: true,
					},
					"page_background": schema.SingleNestedAttribute{
						Attributes: map[string]schema.Attribute{
							"background_color": schema.StringAttribute{
								Description: "The color to be used for the background in the pre-built UI.",
								Computed:    true,
							},
							"background_image_url": schema.StringAttribute{
								Description: "The URL of an image which will be used as the background in the pre-built UI.",
								Computed:    true,
							},
						},
						Computed: true,
					},
					"colors": schema.SingleNestedAttribute{
						Attributes: map[string]schema.Attribute{
							"button_primary_text": schema.StringAttribute{
								Computed: true,
							},
							"button_primary_border": schema.StringAttribute{
								Computed: true,
							},
							"button_secondary_text": schema.StringAttribute{
								Computed: true,
							},
							"button_secondary_background": schema.StringAttribute{
								Computed: true,
							},
							"button_secondary_border": schema.StringAttribute{
								Computed: true,
							},
							"card_background": schema.StringAttribute{
								Computed: true,
							},
							"card_border": schema.StringAttribute{
								Computed: true,
							},
							"input_background": schema.StringAttribute{
								Computed: true,
							},
							"input_border": schema.StringAttribute{
								Computed: true,
							},
							"link": schema.StringAttribute{
								Computed: true,
							},
							"heading_text": schema.StringAttribute{
								Computed: true,
							},
							"body_text": schema.StringAttribute{
								Computed: true,
							},
							"container_background": schema.StringAttribute{
								Computed: true,
							},
							"container_border": schema.StringAttribute{
								Computed: true,
							},
							"divider": schema.StringAttribute{
								Computed: true,
							},
							"icon": schema.StringAttribute{
								Computed: true,
							},
							"loader": schema.StringAttribute{
								Computed: true,
							},
							"positive": schema.StringAttribute{
								Computed: true,
							},
							"critical": schema.StringAttribute{
								Computed: true,
							},
							"information": schema.StringAttribute{
								Computed: true,
							},
							"hover": schema.StringAttribute{
								Computed: true,
							},
							"focus": schema.StringAttribute{
								Computed: true,
							},
						},
						Computed: true,
					},
				},
				Computed: true,
			},
		},
	}
}

func (d *themeDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	theme, err := d.client.GetTheme()

	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read Authsignal Theme",
			err.Error(),
		)
		return
	}

	var themeState themeModel
	themeState.CreateObject(*theme)

	diags := resp.State.Set(ctx, &themeState)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}
}

func (d *themeDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
