package provider

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	testresource "github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

const (
	valueListAddress     = "authsignal_value_list.blocked"
	valueListGetRoute    = "GET /value-lists/blocked"
	valueListCreateRoute = "POST /value-lists"
	valueListPatchRoute  = "PATCH /value-lists/blocked"
	valueListDeleteRoute = "DELETE /value-lists/blocked"
)

func valueListJson(itemType string, items string) string {
	return `{"name":"Blocked","alias":"blocked","itemType":"` + itemType + `","isActive":true,"valueListItems":` + items + `}`
}

func valueListConfig(itemType string, items string) string {
	return `
resource "authsignal_value_list" "blocked" {
  name      = "Blocked"
  is_active = true
  item_type = "` + itemType + `"
  ` + items + `
}
`
}

func numberValueListValue(t *testing.T, resourceSchema schema.Schema, numbers tftypes.Value) tftypes.Value {
	t.Helper()

	return resourceObject(t, resourceSchema, map[string]tftypes.Value{
		"name":                     tftypes.NewValue(tftypes.String, "Blocked"),
		"alias":                    tftypes.NewValue(tftypes.String, "blocked"),
		"is_active":                tftypes.NewValue(tftypes.Bool, true),
		"item_type":                tftypes.NewValue(tftypes.String, "number"),
		"value_list_items_numbers": numbers,
	})
}

func TestAnEmptyNumberListIsCreatedAsANumberList(t *testing.T) {
	client, stub := newStubAPI(t, map[string][]stubResponse{
		valueListCreateRoute: okResponse(valueListJson("number", "[]")),
	})

	resourceSchema := resourceSchemaOf(t, &valueListResource{})
	plan := numberValueListValue(t, resourceSchema, tftypes.NewValue(tftypes.List{ElementType: tftypes.Number}, nil))

	resp := &resource.CreateResponse{State: tfsdk.State{Schema: resourceSchema}}
	(&valueListResource{client: client}).Create(
		context.Background(),
		resource.CreateRequest{Plan: tfsdk.Plan{Schema: resourceSchema, Raw: plan}},
		resp,
	)

	if resp.Diagnostics.HasError() {
		t.Fatal(detailsOf(resp.Diagnostics))
	}

	body := stub.bodyOf(t, valueListCreateRoute)

	if !strings.Contains(body, `"itemType":"number"`) || !strings.Contains(body, `"valueListItems":[]`) {
		t.Errorf("expected an empty number list, got %s", body)
	}
}

func TestAnEmptyValueListPlansClean(t *testing.T) {
	skipUnlessAcceptance(t)

	_, stub := newStubAPI(t, map[string][]stubResponse{
		valueListGetRoute:    okResponse(valueListJson("number", "[]")),
		valueListDeleteRoute: okResponse(`{"success":true}`),
	})
	t.Setenv("AUTHSIGNAL_HOST", stub.url)

	config := valueListConfig("number", "")

	testresource.UnitTest(t, testresource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []testresource.TestStep{
			{
				Config:             config,
				ResourceName:       valueListAddress,
				ImportState:        true,
				ImportStateId:      "blocked",
				ImportStatePersist: true,
			},
			{
				Config:   config,
				PlanOnly: true,
			},
		},
	})
}

func TestEmptyingAValueListUpdatesItInPlace(t *testing.T) {
	skipUnlessAcceptance(t)

	_, stub := newStubAPI(t, map[string][]stubResponse{
		valueListGetRoute:    okResponse(valueListJson("number", "[1,2]")),
		valueListDeleteRoute: okResponse(`{"success":true}`),
	})
	t.Setenv("AUTHSIGNAL_HOST", stub.url)

	testresource.UnitTest(t, testresource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []testresource.TestStep{
			{
				Config:             valueListConfig("number", "value_list_items_numbers = [1, 2]"),
				ResourceName:       valueListAddress,
				ImportState:        true,
				ImportStateId:      "blocked",
				ImportStatePersist: true,
			},
			{
				Config:             valueListConfig("number", ""),
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
				ConfigPlanChecks: testresource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(valueListAddress, plancheck.ResourceActionUpdate),
					},
				},
			},
		},
	})
}

func TestEmptyingAValueListSendsNoItems(t *testing.T) {
	client, stub := newStubAPI(t, map[string][]stubResponse{
		valueListPatchRoute: okResponse(valueListJson("number", "[]")),
	})

	resourceSchema := resourceSchemaOf(t, &valueListResource{})
	numbersType := tftypes.List{ElementType: tftypes.Number}

	state := numberValueListValue(t, resourceSchema, tftypes.NewValue(numbersType, []tftypes.Value{
		tftypes.NewValue(tftypes.Number, 1),
		tftypes.NewValue(tftypes.Number, 2),
	}))
	plan := numberValueListValue(t, resourceSchema, tftypes.NewValue(numbersType, nil))

	resp := &resource.UpdateResponse{State: tfsdk.State{Schema: resourceSchema}}
	(&valueListResource{client: client}).Update(
		context.Background(),
		resource.UpdateRequest{
			Plan:   tfsdk.Plan{Schema: resourceSchema, Raw: plan},
			Config: tfsdk.Config{Schema: resourceSchema, Raw: plan},
			State:  tfsdk.State{Schema: resourceSchema, Raw: state},
		},
		resp,
	)

	if resp.Diagnostics.HasError() {
		t.Fatal(detailsOf(resp.Diagnostics))
	}

	body := stub.bodyOf(t, valueListPatchRoute)

	if !strings.Contains(body, `"valueListItems":[]`) {
		t.Errorf("expected the update to send [], which the API takes as no items, got %s", body)
	}
}

func TestValueListConfigRejectsEmptyBrackets(t *testing.T) {
	res := &valueListResource{}

	summaries := validateResourceConfig(t, res, resourceSchemaOf(t, res), map[string]tftypes.Value{
		"item_type":                tftypes.NewValue(tftypes.String, "string"),
		"value_list_items_strings": tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, []tftypes.Value{}),
	})

	if len(summaries) != 1 || summaries[0] != "Empty Value List Items" {
		t.Errorf("expected an empty-items error, got %v", summaries)
	}
}

func TestValueListConfigRejectsItemsOfTheOtherType(t *testing.T) {
	res := &valueListResource{}

	summaries := validateResourceConfig(t, res, resourceSchemaOf(t, res), map[string]tftypes.Value{
		"item_type": tftypes.NewValue(tftypes.String, "string"),
		"value_list_items_numbers": tftypes.NewValue(tftypes.List{ElementType: tftypes.Number}, []tftypes.Value{
			tftypes.NewValue(tftypes.Number, 1),
		}),
	})

	if len(summaries) != 1 || summaries[0] != "Value List Items Do Not Match the Item Type" {
		t.Errorf("expected an items-do-not-match error, got %v", summaries)
	}
}
