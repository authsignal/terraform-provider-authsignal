package provider

import (
	"context"
	"fmt"

	"github.com/authsignal/authsignal-management-go/v3"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &valueListResource{}
	_ resource.ResourceWithConfigure   = &valueListResource{}
	_ resource.ResourceWithImportState = &valueListResource{}
)

func NewValueListResource() resource.Resource {
	return &valueListResource{}
}

type valueListResource struct {
	client *authsignal.Client
}

type valueListResourceModel struct {
	Name                  types.String `tfsdk:"name"`
	Alias                 types.String `tfsdk:"alias"`
	ItemType              types.String `tfsdk:"item_type"`
	IsActive              types.Bool   `tfsdk:"is_active"`
	ValueListItemsStrings types.List   `tfsdk:"value_list_items_strings"`
	ValueListItemsNumbers types.List   `tfsdk:"value_list_items_numbers"`
}

func (r *valueListResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_value_list"
}

func checkListTypeHasntChanged(ctx context.Context, req planmodifier.ListRequest, resp *listplanmodifier.RequiresReplaceIfFuncResponse) {
	var plan valueListResourceModel
	_ = req.Plan.Get(ctx, &plan)
	var state valueListResourceModel
	_ = req.State.Get(ctx, &state)
	if plan.ValueListItemsNumbers.IsNull() != state.ValueListItemsNumbers.IsNull() {
		resp.RequiresReplace = true
	}
	if plan.ValueListItemsStrings.IsNull() != state.ValueListItemsStrings.IsNull() {
		resp.RequiresReplace = true
	}
}

func (r *valueListResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Description: "The name of the value list.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"alias": schema.StringAttribute{
				Description: "The hyphenated alias of the value list, auto-generated upon creation.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"item_type": schema.StringAttribute{
				Description: "The type of items in the value list (string or number).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"is_active": schema.BoolAttribute{
				Description: "Whether or not the list is active. This currently has no effect, please set the value to `true`.",
				Required:    true,
			},
			"value_list_items_strings": schema.ListAttribute{
				ElementType: types.StringType,
				Description: "A list of string items in the value list.",
				Optional:    true,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.RequiresReplaceIf(checkListTypeHasntChanged, "Ensures the list type hasn't changed.", "Ensures the list type hasn't changed."),
				},
			},
			"value_list_items_numbers": schema.ListAttribute{
				ElementType: types.Float64Type,
				Description: "A list of number items in the value list.",
				Optional:    true,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.RequiresReplaceIf(checkListTypeHasntChanged, "Ensures the list type hasn't changed.", "Ensures the list type hasn't changed."),
				},
			},
		},
	}
}

func prepareValueListItems(ctx context.Context, plan valueListResourceModel, itemType string, setNullIfEmpty bool) (authsignal.NullableJsonInput[[]authsignal.ValueListItem], diag.Diagnostics) {
	var itemList []authsignal.ValueListItem

	if itemType == "number" {
		valueListItemsNumbersSlice := make([]float64, 0, len(plan.ValueListItemsNumbers.Elements()))
		diags := plan.ValueListItemsNumbers.ElementsAs(ctx, &valueListItemsNumbersSlice, false)
		if diags.HasError() {
			return nil, diags
		}

		for _, item := range valueListItemsNumbersSlice {
			itemList = append(itemList, item)
		}
	} else {
		valueListItemsStringsSlice := make([]string, 0, len(plan.ValueListItemsStrings.Elements()))
		diags := plan.ValueListItemsStrings.ElementsAs(ctx, &valueListItemsStringsSlice, false)
		if diags.HasError() {
			return nil, diags
		}

		for _, item := range valueListItemsStringsSlice {
			itemList = append(itemList, item)
		}
	}

	if len(itemList) > 0 {
		return authsignal.SetValue(itemList), nil
	} else if setNullIfEmpty {
		return authsignal.SetNull(itemList), nil
	}
	return nil, nil
}

func getListType(plan valueListResourceModel) string {
	if len(plan.ValueListItemsNumbers.Elements()) > 0 && len(plan.ValueListItemsStrings.Elements()) > 0 {
		return "error"
	}

	if len(plan.ValueListItemsNumbers.Elements()) > 0 {
		return "number"
	}

	return "string"
}

func (r *valueListResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan valueListResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var valueListToCreate = authsignal.ValueList{
		Name:     authsignal.SetValue(plan.Name.ValueString()),
		IsActive: authsignal.SetValue(plan.IsActive.ValueBool()),
	}

	itemType := getListType(plan)
	if itemType == "error" {
		resp.Diagnostics.AddError(
			"Invalid value list items",
			"Only one of \"value_list_items_strings\" or \"value_list_items_numbers\" can be set",
		)
		return
	}

	valueListToCreate.ItemType = authsignal.SetValue(itemType)

	valueListItems, diags := prepareValueListItems(ctx, plan, itemType, false)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	valueListToCreate.ValueListItems = valueListItems

	valueList, _, err := r.client.CreateValueList(valueListToCreate)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating value list",
			"Could not create value list, unexpected error: "+err.Error(),
		)
		return
	}

	plan.Alias = types.StringValue(valueList.Alias)
	plan.IsActive = types.BoolValue(valueList.IsActive)
	plan.ItemType = types.StringValue(valueList.ItemType)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *valueListResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state valueListResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	valueList, statusCode, err := r.client.GetValueList(state.Alias.ValueString())

	if statusCode == 404 {
		resp.State.RemoveResource(ctx)
		return
	}

	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read Authsignal ValueList",
			err.Error(),
		)
		return
	}

	valueListItemsStrings, valueListItemsNumbers, diags := RestructureValueList(ctx, valueList.ItemType, valueList.ValueListItems)

	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	state.ValueListItemsStrings = valueListItemsStrings
	state.ValueListItemsNumbers = valueListItemsNumbers

	state.Name = types.StringValue(valueList.Name)
	state.Alias = types.StringValue(valueList.Alias)
	state.ItemType = types.StringValue(valueList.ItemType)
	state.IsActive = types.BoolValue(valueList.IsActive)

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *valueListResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan valueListResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	itemType := getListType(plan)
	if itemType == "error" {
		resp.Diagnostics.AddError(
			"Invalid value list items",
			"Only one of \"value_list_items_strings\" or \"value_list_items_numbers\" can be set",
		)
		return
	}

	var valueListToUpdate = authsignal.ValueList{
		IsActive: authsignal.SetValue(plan.IsActive.ValueBool()),
	}

	valueListItems, diags := prepareValueListItems(ctx, plan, itemType, true)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	valueListToUpdate.ValueListItems = valueListItems

	valueList, _, err := r.client.UpdateValueList(plan.Alias.ValueString(), valueListToUpdate)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating value list",
			"Could not create value list, unexpected error: "+err.Error(),
		)
		return
	}

	plan.Alias = types.StringValue(valueList.Alias)
	plan.IsActive = types.BoolValue(valueList.IsActive)
	plan.ItemType = types.StringValue(valueList.ItemType)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *valueListResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state valueListResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, _, err := r.client.DeleteValueList(state.Alias.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting Authsignal value list",
			"Could not delete value list, unexpected error: "+err.Error(),
		)
		return
	}
}

func (r *valueListResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("alias"), req, resp)
}

func (r *valueListResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
