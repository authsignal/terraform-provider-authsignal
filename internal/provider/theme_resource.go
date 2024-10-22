package provider

import (
	"context"
	"fmt"

	"github.com/authsignal/authsignal-management-go/v3"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

var (
	_ resource.Resource                = &themeResource{}
	_ resource.ResourceWithConfigure   = &themeResource{}
	_ resource.ResourceWithImportState = &themeResource{}
)

func NewThemeResource() resource.Resource {
	return &themeResource{}
}

type themeResource struct {
	client *authsignal.Client
}

func (r *themeResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_theme"
}

func (r *themeResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Description: "The name of the tenant which is visible to users.",
				Required:    true,
			},
			"logo_url": schema.StringAttribute{
				Description: "The URL of an image to be used as a logo for the tenant.",
				Optional:    true,
			},
			"watermark_url": schema.StringAttribute{
				Description: "The URL of an image to be used as a watermark at the bottom of Authsignal's pre-built UI.",
				Optional:    true,
			},
			"favicon_url": schema.StringAttribute{
				Description: "The URL of an image to be used as a favicon for the tenant",
				Optional:    true,
			},
			"primary_color": schema.StringAttribute{
				Description: "The primary color for the tenant.",
				Optional:    true,
			},
			"borders": schema.SingleNestedAttribute{
				Attributes: map[string]schema.Attribute{
					"button_border_radius": schema.Int64Attribute{
						Optional: true,
					},
					"button_border_width": schema.Int64Attribute{
						Optional: true,
					},
					"card_border_radius": schema.Int64Attribute{
						Optional: true,
					},
					"card_border_width": schema.Int64Attribute{
						Optional: true,
					},
					"input_border_radius": schema.Int64Attribute{
						Optional: true,
					},
					"input_border_width": schema.Int64Attribute{
						Optional: true,
					},
					"container_border_radius": schema.Int64Attribute{
						Optional: true,
					},
				},
				Optional: true,
			},
			"container": schema.SingleNestedAttribute{
				Attributes: map[string]schema.Attribute{
					"content_alignment": schema.StringAttribute{
						Description: "Allowed values: `left`, `center`.",
						Optional:    true,
						Validators: []validator.String{
							stringvalidator.OneOf([]string{"left", "center"}...),
						},
					},
					"position": schema.StringAttribute{
						Description: "Allowed values: `inside`, `outside`.",
						Optional:    true,
						Validators: []validator.String{
							stringvalidator.OneOf([]string{"inside", "outside"}...),
						},
					},
					"padding": schema.Int64Attribute{
						Optional: true,
					},
					"logo_alignment": schema.StringAttribute{
						Description: "Allowed values: `left`, `center`.",
						Optional:    true,
						Validators: []validator.String{
							stringvalidator.OneOf([]string{"left", "center"}...),
						},
					},
					"logo_position": schema.StringAttribute{
						Description: "Allowed values: `inside`, `outside`.",
						Optional:    true,
						Validators: []validator.String{
							stringvalidator.OneOf([]string{"inside", "outside"}...),
						},
					},
					"logo_height": schema.Int64Attribute{
						Optional: true,
					},
				},
				Optional: true,
			},
			"typography": schema.SingleNestedAttribute{
				Attributes: map[string]schema.Attribute{
					"display": schema.SingleNestedAttribute{
						Attributes: map[string]schema.Attribute{
							"font_url": schema.StringAttribute{
								Description: "The URL of a font file to be used for the tenant.",
								Optional:    true,
							},
						},
						Optional: true,
					},
				},
				Optional: true,
			},
			"page_background": schema.SingleNestedAttribute{
				Attributes: map[string]schema.Attribute{
					"background_color": schema.StringAttribute{
						Description: "The color to be used for the background in the pre-built UI.",
						Optional:    true,
					},
					"background_image_url": schema.StringAttribute{
						Description: "The URL of an image which will be used as the background in the pre-built UI.",
						Optional:    true,
					},
				},
				Optional: true,
			},
			"colors": schema.SingleNestedAttribute{
				Attributes: map[string]schema.Attribute{
					"button_primary_text": schema.StringAttribute{
						Optional: true,
					},
					"button_primary_border": schema.StringAttribute{
						Optional: true,
					},
					"button_secondary_text": schema.StringAttribute{
						Optional: true,
					},
					"button_secondary_background": schema.StringAttribute{
						Optional: true,
					},
					"button_secondary_border": schema.StringAttribute{
						Optional: true,
					},
					"card_background": schema.StringAttribute{
						Optional: true,
					},
					"card_border": schema.StringAttribute{
						Optional: true,
					},
					"input_background": schema.StringAttribute{
						Optional: true,
					},
					"input_border": schema.StringAttribute{
						Optional: true,
					},
					"link": schema.StringAttribute{
						Optional: true,
					},
					"heading_text": schema.StringAttribute{
						Optional: true,
					},
					"body_text": schema.StringAttribute{
						Optional: true,
					},
					"container_background": schema.StringAttribute{
						Optional: true,
					},
					"container_border": schema.StringAttribute{
						Optional: true,
					},
					"divider": schema.StringAttribute{
						Optional: true,
					},
					"icon": schema.StringAttribute{
						Optional: true,
					},
					"loader": schema.StringAttribute{
						Optional: true,
					},
					"positive": schema.StringAttribute{
						Optional: true,
					},
					"critical": schema.StringAttribute{
						Optional: true,
					},
					"information": schema.StringAttribute{
						Optional: true,
					},
					"hover": schema.StringAttribute{
						Optional: true,
					},
					"focus": schema.StringAttribute{
						Optional: true,
					},
				},
				Optional: true,
			},
			"dark_mode": schema.SingleNestedAttribute{
				Attributes: map[string]schema.Attribute{
					"logo_url": schema.StringAttribute{
						Description: "The URL of an image to be used as a logo for the tenant.",
						Optional:    true,
					},
					"watermark_url": schema.StringAttribute{
						Description: "The URL of an image to be used as a watermark at the bottom of Authsignal's pre-built UI.",
						Optional:    true,
					},
					"favicon_url": schema.StringAttribute{
						Description: "The URL of an image to be used as a favicon for the tenant",
						Optional:    true,
					},
					"primary_color": schema.StringAttribute{
						Description: "The primary color for the tenant.",
						Optional:    true,
					},
					"borders": schema.SingleNestedAttribute{
						Attributes: map[string]schema.Attribute{
							"button_border_radius": schema.Int64Attribute{
								Optional: true,
							},
							"button_border_width": schema.Int64Attribute{
								Optional: true,
							},
							"card_border_radius": schema.Int64Attribute{
								Optional: true,
							},
							"card_border_width": schema.Int64Attribute{
								Optional: true,
							},
							"input_border_radius": schema.Int64Attribute{
								Optional: true,
							},
							"input_border_width": schema.Int64Attribute{
								Optional: true,
							},
							"container_border_radius": schema.Int64Attribute{
								Optional: true,
							},
						},
						Optional: true,
					},
					"container": schema.SingleNestedAttribute{
						Attributes: map[string]schema.Attribute{
							"content_alignment": schema.StringAttribute{
								Description: "Allowed values: `left`, `center`.",
								Optional:    true,
								Validators: []validator.String{
									stringvalidator.OneOf([]string{"left", "center"}...),
								},
							},
							"position": schema.StringAttribute{
								Description: "Allowed values: `inside`, `outside`.",
								Optional:    true,
								Validators: []validator.String{
									stringvalidator.OneOf([]string{"inside", "outside"}...),
								},
							},
							"padding": schema.Int64Attribute{
								Optional: true,
							},
							"logo_alignment": schema.StringAttribute{
								Description: "Allowed values: `left`, `center`.",
								Optional:    true,
								Validators: []validator.String{
									stringvalidator.OneOf([]string{"left", "center"}...),
								},
							},
							"logo_position": schema.StringAttribute{
								Description: "Allowed values: `inside`, `outside`.",
								Optional:    true,
								Validators: []validator.String{
									stringvalidator.OneOf([]string{"inside", "outside"}...),
								},
							},
							"logo_height": schema.Int64Attribute{
								Optional: true,
							},
						},
						Optional: true,
					},
					"typography": schema.SingleNestedAttribute{
						Attributes: map[string]schema.Attribute{
							"display": schema.SingleNestedAttribute{
								Attributes: map[string]schema.Attribute{
									"font_url": schema.StringAttribute{
										Description: "The URL of a font file to be used for the tenant.",
										Optional:    true,
									},
								},
								Optional: true,
							},
						},
						Optional: true,
					},
					"page_background": schema.SingleNestedAttribute{
						Attributes: map[string]schema.Attribute{
							"background_color": schema.StringAttribute{
								Description: "The color to be used for the background in the pre-built UI.",
								Optional:    true,
							},
							"background_image_url": schema.StringAttribute{
								Description: "The URL of an image which will be used as the background in the pre-built UI.",
								Optional:    true,
							},
						},
						Optional: true,
					},
					"colors": schema.SingleNestedAttribute{
						Attributes: map[string]schema.Attribute{
							"button_primary_text": schema.StringAttribute{
								Optional: true,
							},
							"button_primary_border": schema.StringAttribute{
								Optional: true,
							},
							"button_secondary_text": schema.StringAttribute{
								Optional: true,
							},
							"button_secondary_background": schema.StringAttribute{
								Optional: true,
							},
							"button_secondary_border": schema.StringAttribute{
								Optional: true,
							},
							"card_background": schema.StringAttribute{
								Optional: true,
							},
							"card_border": schema.StringAttribute{
								Optional: true,
							},
							"input_background": schema.StringAttribute{
								Optional: true,
							},
							"input_border": schema.StringAttribute{
								Optional: true,
							},
							"link": schema.StringAttribute{
								Optional: true,
							},
							"heading_text": schema.StringAttribute{
								Optional: true,
							},
							"body_text": schema.StringAttribute{
								Optional: true,
							},
							"container_background": schema.StringAttribute{
								Optional: true,
							},
							"container_border": schema.StringAttribute{
								Optional: true,
							},
							"divider": schema.StringAttribute{
								Optional: true,
							},
							"icon": schema.StringAttribute{
								Optional: true,
							},
							"loader": schema.StringAttribute{
								Optional: true,
							},
							"positive": schema.StringAttribute{
								Optional: true,
							},
							"critical": schema.StringAttribute{
								Optional: true,
							},
							"information": schema.StringAttribute{
								Optional: true,
							},
							"hover": schema.StringAttribute{
								Optional: true,
							},
							"focus": schema.StringAttribute{
								Optional: true,
							},
						},
						Optional: true,
					},
				},
				Optional: true,
			},
		},
	}
}

func (r *themeResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	resp.Diagnostics.AddError(
		"Please import the existing theme via `terraform import ...`",
		"Themes cannot be created via Terraform, only updated. Please import the existing theme through `terraform import ...` and then try applying again.",
	)
}

func (r *themeResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	theme, statusCode, err := r.client.GetTheme()

	if statusCode == 404 {
		resp.State.RemoveResource(ctx)
		return
	}

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

func (r *themeResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan themeModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var themeToCreate = buildAuthsignalThemeUpdateObject(ctx, resp, plan)

	theme, _, err := r.client.UpdateTheme(themeToCreate)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating theme",
			"Could not create theme, unexpected error: "+err.Error(),
		)
		return
	}

	var themeState themeModel
	themeState.CreateObject(*theme)

	diags = resp.State.Set(ctx, themeState)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *themeResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state themeModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var themeToCreate = buildAuthsignalThemeDeleteObject(state)

	_, _, err := r.client.UpdateTheme(themeToCreate)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating theme",
			"Could not create theme, unexpected error: "+err.Error(),
		)
		return
	}

	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *themeResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("name"), req, resp)
}

func (r *themeResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
