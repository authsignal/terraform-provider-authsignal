package provider

import (
	"context"
	"fmt"

	"github.com/authsignal/authsignal-management-go/v6"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                   = &valueListResource{}
	_ resource.ResourceWithConfigure      = &valueListResource{}
	_ resource.ResourceWithImportState    = &valueListResource{}
	_ resource.ResourceWithValidateConfig = &valueListResource{}
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
				Description: "The type of the items in the value list: `string` or `number`. The type can't change after the list is created, so changing it replaces the list.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("string", "number"),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"is_active": schema.BoolAttribute{
				Description: "Whether or not the list is active. This currently has no effect, please set the value to `true`.",
				Required:    true,
			},
			"value_list_items_strings": schema.ListAttribute{
				ElementType: types.StringType,
				Description: "The items of a `string` value list. Leave it out for an empty list.",
				Optional:    true,
			},
			"value_list_items_numbers": schema.ListAttribute{
				ElementType: types.Float64Type,
				Description: "The items of a `number` value list. Leave it out for an empty list.",
				Optional:    true,
			},
		},
	}
}

func (r *valueListResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config valueListResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	rejectEmptyItems(resp, "value_list_items_strings", config.ValueListItemsStrings)
	rejectEmptyItems(resp, "value_list_items_numbers", config.ValueListItemsNumbers)

	switch config.ItemType.ValueString() {
	case "string":
		rejectItemsOfTheOtherType(resp, "value_list_items_numbers", config.ValueListItemsNumbers, "string", "value_list_items_strings")
	case "number":
		rejectItemsOfTheOtherType(resp, "value_list_items_strings", config.ValueListItemsStrings, "number", "value_list_items_numbers")
	}
}

// Read stores a list with no items as null, so `[]` in the configuration would plan a change on every run.
func rejectEmptyItems(resp *resource.ValidateConfigResponse, attribute string, items types.List) {
	if !isKnownAndSet(items) || len(items.Elements()) > 0 {
		return
	}

	resp.Diagnostics.AddAttributeError(
		path.Root(attribute),
		"Empty Value List Items",
		fmt.Sprintf("`%s` is set to `[]`. Leave it out of the configuration for an empty value list.", attribute),
	)
}

func rejectItemsOfTheOtherType(resp *resource.ValidateConfigResponse, attribute string, items types.List, itemType string, itemsAttribute string) {
	if !isKnownAndSet(items) {
		return
	}

	resp.Diagnostics.AddAttributeError(
		path.Root(attribute),
		"Value List Items Do Not Match the Item Type",
		fmt.Sprintf("`%s` is set but `item_type` is %q. The items of a %s value list go in `%s`.", attribute, itemType, itemType, itemsAttribute),
	)
}

// The API replaces the items on every update and rejects null for them, so an empty list is sent as [].
func prepareValueListItems(ctx context.Context, plan valueListResourceModel) (authsignal.NullableJsonInput[[]authsignal.ValueListItem], diag.Diagnostics) {
	itemList := []authsignal.ValueListItem{}

	if plan.ItemType.ValueString() == "number" {
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

	return authsignal.SetValue(itemList), nil
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
		ItemType: authsignal.SetValue(plan.ItemType.ValueString()),
	}

	valueListItems, diags := prepareValueListItems(ctx, plan)
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

	var valueListToUpdate = authsignal.ValueList{
		IsActive: authsignal.SetValue(plan.IsActive.ValueBool()),
	}

	valueListItems, diags := prepareValueListItems(ctx, plan)
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
