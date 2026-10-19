package provider

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

const deletedRuleJson = `{"ruleId":"rule-1","tenantId":"tenant","actionCode":"sign-in","name":"Anonymous IP",` +
	`"isActive":true,"priority":0,"type":"CHALLENGE","conditions":{"and":[]},"deletedAt":"2026-10-13T00:00:00.000Z"}`

// The API soft-deletes rules, so a rule deleted outside Terraform still reads back. Dropping it
// from state is what lets the next plan recreate it.
func TestRuleReadDropsARuleDeletedOutsideTerraform(t *testing.T) {
	client, _ := newStubAPI(t, map[string][]stubResponse{
		actionGetRoute: okResponse(actionConfigurationJson(actionTypeClassic, "null", "null")),
		ruleGetRoute:   okResponse(deletedRuleJson),
	})

	state := ruleState(t)
	resp := &resource.ReadResponse{State: state}

	(&ruleResource{client: client}).Read(context.Background(), resource.ReadRequest{State: state}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatal(detailsOf(resp.Diagnostics))
	}

	if !resp.State.Raw.IsNull() {
		t.Error("a deleted rule must be removed from state")
	}
}

func TestRuleDataSourceRefusesADeletedRule(t *testing.T) {
	client, _ := newStubAPI(t, map[string][]stubResponse{
		ruleGetRoute: okResponse(deletedRuleJson),
	})

	dataSource := &ruleDataSource{client: client}

	schemaResp := &datasource.SchemaResponse{}
	dataSource.Schema(context.Background(), datasource.SchemaRequest{}, schemaResp)

	config := tfsdk.Config{
		Schema: schemaResp.Schema,
		Raw: resourceObject(t, schemaResp.Schema, map[string]tftypes.Value{
			"action_code": tftypes.NewValue(tftypes.String, "sign-in"),
			"rule_id":     tftypes.NewValue(tftypes.String, "rule-1"),
		}),
	}

	resp := &datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	dataSource.Read(context.Background(), datasource.ReadRequest{Config: config}, resp)

	if !resp.Diagnostics.HasError() || !strings.Contains(detailsOf(resp.Diagnostics), "was deleted") {
		t.Fatalf("expected reading a deleted rule to fail, got %s", detailsOf(resp.Diagnostics))
	}
}
