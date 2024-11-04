package provider

import (
	"context"
	"fmt"

	"github.com/authsignal/authsignal-management-go/v3"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &customDataPointResource{}
	_ resource.ResourceWithConfigure   = &customDataPointResource{}
	_ resource.ResourceWithImportState = &customDataPointResource{}
)

var (
	allowedDataTypes  = []string{"text", "number", "boolean", "multiselect"}
	allowedModelTypes = []string{"action", "user"}
)

func NewCustomDataPointResource() resource.Resource {
	return &customDataPointResource{}
}

type customDataPointResource struct {
	client *authsignal.Client
}

type customDataPointResourceModel struct {
	Id          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	DataType    types.String `tfsdk:"data_type"`
	ModelType   types.String `tfsdk:"model_type"`
	Description types.String `tfsdk:"description"`
}

func (r *customDataPointResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_custom_data_point"
}

func (r *customDataPointResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The id of the custom data point.",
				Computed:    true,
			},
			"name": schema.StringAttribute{
				Description: "The name of the custom data point.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"data_type": schema.StringAttribute{
				Description: "The data type of the custom data point. Allowed values: `text`, `number`, `boolean`, 'multiselect'.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf(allowedDataTypes...),
				},
			},
			"model_type": schema.StringAttribute{
				Description: "The model type of the custom data point. Allowed values: `action`, `user`.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf(allowedModelTypes...),
				},
			},
			"description": schema.StringAttribute{
				Description: "The description of the custom data point.",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *customDataPointResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan customDataPointResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var customDataPointToCreate = authsignal.CustomDataPoint{
		Name:        authsignal.SetValue(plan.Name.ValueString()),
		DataType:    authsignal.SetValue(plan.DataType.ValueString()),
		ModelType:   authsignal.SetValue(plan.ModelType.ValueString()),
		Description: authsignal.SetValue(plan.Description.ValueString()),
	}

	customDataPoint, _, err := r.client.CreateCustomDataPoint(customDataPointToCreate)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating custom data point",
			"Could not create custom data point, unexpected error: "+err.Error(),
		)
		return
	}

	plan.Id = types.StringValue(customDataPoint.Id)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *customDataPointResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state customDataPointResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	customDataPoint, statusCode, err := r.client.GetCustomDataPoint(state.Id.ValueString())

	if err != nil {
		if statusCode == 404 {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Unable to Read Authsignal CustomDataPoint",
			fmt.Sprintf("Error reading custom data point ID %s: %s", state.Id.ValueString(), err.Error()),
		)
		return
	}

	if customDataPoint == nil {
		resp.Diagnostics.AddError(
			"Unexpected Empty Response",
			fmt.Sprintf("Received an empty response for custom data point ID %s.", state.Id.ValueString()),
		)
		return
	}

	state.Name = types.StringValue(customDataPoint.Name)
	state.DataType = types.StringValue(customDataPoint.DataType)
	state.ModelType = types.StringValue(customDataPoint.ModelType)
	state.Description = types.StringValue(customDataPoint.Description)

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *customDataPointResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError(
		"Authsignal Custom Data Points cannot be updated and this function should never be called.",
		"Please raise this as an issue on our GitHub repository.",
	)
}

func (r *customDataPointResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state customDataPointResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, _, err := r.client.DeleteCustomDataPoint(state.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting Authsignal custom data point",
			"Could not delete custom data point, unexpected error: "+err.Error(),
		)
		return
	}
}

func (r *customDataPointResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *customDataPointResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
