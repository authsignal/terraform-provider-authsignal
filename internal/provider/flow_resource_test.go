package provider

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// The flow is pretty-printed with a key order the API does not return, so the
// configured text never matches the document composed from a read.
func testAccFlowConfig(countryCode string) string {
	return fmt.Sprintf(`
		resource "authsignal_flow" "flow" {
			action_code = "terraform-acceptance-test-flow"
			flow        = <<-EOT
				{
				  "rules": [
				    {
				      "ruleId": "from-country",
				      "name": "From %[1]s",
				      "conditions": {
				        "and": [
				          { "in": [{ "var": "ip.location.country.countryCode" }, ["%[1]s"]] }
				        ]
				      }
				    },
				    {
				      "ruleId": "anonymous-ip",
				      "name": "Anonymous IP",
				      "conditions": {
				        "and": [
				          { "==": [{ "var": "ip.isAnonymous" }, true] }
				        ]
				      }
				    }
				  ],
				  "actionNodes": [
				    {
				      "nodeType": "RULE",
				      "nodeId": "rule-country",
				      "parentNodeIds": [],
				      "ruleChildNodeIds": [["from-country", "rule-anonymous"]],
				      "elseChildNodeId": "complete"
				    },
				    {
				      "nodeType": "RULE",
				      "nodeId": "rule-anonymous",
				      "parentNodeIds": ["rule-country"],
				      "ruleChildNodeIds": [["anonymous-ip", "complete"]],
				      "elseChildNodeId": "complete"
				    },
				    {
				      "nodeType": "COMPLETE",
				      "nodeId": "complete",
				      "parentNodeIds": ["rule-country", "rule-anonymous"]
				    }
				  ]
				}
			EOT
		}
	`, countryCode)
}

// terraform import needs the address free, and the harness has no state rm.
func testAccFlowForgetConfig() string {
	return `
		removed {
			from = authsignal_flow.flow

			lifecycle {
				destroy = false
			}
		}
	`
}

func TestAccFlowResource(t *testing.T) {
	var firstVersion string

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccFlowConfig("NZ"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("authsignal_flow.flow", "action_code", "terraform-acceptance-test-flow"),
					resource.TestCheckResourceAttrSet("authsignal_flow.flow", "flow_version"),
					resource.TestCheckResourceAttrWith("authsignal_flow.flow", "flow_version", func(value string) error {
						firstVersion = value
						return nil
					}),
					resource.TestCheckResourceAttrWith("authsignal_flow.flow", "flow", func(value string) error {
						if !strings.Contains(value, `"ruleId": "from-country"`) || !strings.Contains(value, `["NZ"]`) {
							return fmt.Errorf("expected the rule in the flow's rules array, got %s", value)
						}
						if !strings.Contains(value, `"actionNodes"`) || !strings.Contains(value, `"rules"`) {
							return fmt.Errorf("expected a document with both arrays, got %s", value)
						}
						return nil
					}),
				),
			},
			{
				Config: testAccFlowConfig("AU"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrWith("authsignal_flow.flow", "flow_version", func(value string) error {
						first, err := strconv.Atoi(firstVersion)
						if err != nil {
							return err
						}
						second, err := strconv.Atoi(value)
						if err != nil {
							return err
						}
						if second != first+1 {
							return fmt.Errorf("expected flow_version %d after changing a condition, got %d", first+1, second)
						}
						return nil
					}),
					resource.TestCheckResourceAttrWith("authsignal_flow.flow", "flow", func(value string) error {
						if !strings.Contains(value, `["AU"]`) {
							return fmt.Errorf("expected the changed condition in the flow, got %s", value)
						}
						return nil
					}),
				),
			},
			{
				ResourceName:                         "authsignal_flow.flow",
				ImportState:                          true,
				ImportStateId:                        "terraform-acceptance-test-flow",
				ImportStateVerify:                    true,
				ImportStateVerifyIgnore:              []string{"flow"},
				ImportStateVerifyIdentifierAttribute: "action_code",
			},
			{
				Config: testAccFlowForgetConfig(),
			},
			{
				Config:             testAccFlowConfig("AU"),
				ResourceName:       "authsignal_flow.flow",
				ImportState:        true,
				ImportStateId:      "terraform-acceptance-test-flow",
				ImportStatePersist: true,
			},
			// The imported flow is the composed document, not the configured text.
			{
				Config:   testAccFlowConfig("AU"),
				PlanOnly: true,
			},
		},
	})
}
