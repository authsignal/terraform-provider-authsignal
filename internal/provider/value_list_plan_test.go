package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

const valueListAddress = "authsignal_value_list.blocked"

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

func TestAnEmptyNumberListIsCreatedAsANumberList(t *testing.T) {
	skipUnlessAcceptance(t)
	fake := newFakeValueListAPI(t)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: valueListConfig("number", ""),
				Check: func(*terraform.State) error {
					return fake.expectList("blocked", "number", 0)
				},
			},
		},
	})
}

func TestEmptyingAValueListUpdatesItInPlace(t *testing.T) {
	skipUnlessAcceptance(t)
	fake := newFakeValueListAPI(t)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: valueListConfig("number", "value_list_items_numbers = [1, 2]"),
			},
			{
				Config: valueListConfig("number", ""),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(valueListAddress, plancheck.ResourceActionUpdate),
					},
				},
				Check: func(*terraform.State) error {
					return fake.expectList("blocked", "number", 0)
				},
			},
		},
	})
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
