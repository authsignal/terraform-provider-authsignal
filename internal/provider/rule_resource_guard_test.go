package provider

import (
	"context"
	"strings"
	"testing"

	"github.com/authsignal/authsignal-management-go/v6"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

const (
	ruleCreateRoute = "POST /action-configurations/sign-in/rules"
	ruleUpdateRoute = "PATCH /action-configurations/sign-in/rules/rule-1"
	ruleGetRoute    = "GET /action-configurations/sign-in/rules/rule-1"

	ruleJson = `{"ruleId":"rule-1","tenantId":"tenant","actionCode":"sign-in","name":"Anonymous IP",` +
		`"isActive":true,"priority":0,"type":"CHALLENGE","conditions":{"and":[]}}`
)

func ruleState(t *testing.T) tfsdk.State {
	t.Helper()

	resourceSchema := resourceSchemaOf(t, &ruleResource{})

	return tfsdk.State{
		Schema: resourceSchema,
		Raw: resourceObject(t, resourceSchema, map[string]tftypes.Value{
			"action_code": tftypes.NewValue(tftypes.String, "sign-in"),
			"name":        tftypes.NewValue(tftypes.String, "Anonymous IP"),
			"is_active":   tftypes.NewValue(tftypes.Bool, true),
			"priority":    tftypes.NewValue(tftypes.Number, 0),
			"type":        tftypes.NewValue(tftypes.String, "CHALLENGE"),
			"conditions":  tftypes.NewValue(tftypes.String, `{"and":[]}`),
			"rule_id":     tftypes.NewValue(tftypes.String, "rule-1"),
			"tenant_id":   tftypes.NewValue(tftypes.String, "tenant"),
		}),
	}
}

// The guard exists to stop a rule being written to a FLOW action, so the proof is that
// the rules endpoints are never reached, not merely that a diagnostic came back.
func TestRuleCreateWritesNothingToAFlowAction(t *testing.T) {
	client, stub := newStubAPI(t, map[string][]stubResponse{
		actionGetRoute:  okResponse(actionConfigurationJson(actionTypeFlow, "null", "null")),
		ruleCreateRoute: okResponse(`{"ruleId":"rule-1","tenantId":"tenant"}`),
	})

	state := ruleState(t)
	resp := &resource.CreateResponse{State: tfsdk.State{Schema: state.Schema}}

	(&ruleResource{client: client}).Create(
		context.Background(),
		resource.CreateRequest{Plan: tfsdk.Plan{Schema: state.Schema, Raw: state.Raw}},
		resp,
	)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected creating a rule on a FLOW action to fail")
	}

	if !strings.Contains(detailsOf(resp.Diagnostics), "authsignal_flow") {
		t.Errorf("the error must name authsignal_flow, got %s", detailsOf(resp.Diagnostics))
	}

	stub.assertNotCalled(t, ruleCreateRoute)
	stub.assertRoutes(t, actionGetRoute)
}

func TestRuleUpdateWritesNothingToAFlowAction(t *testing.T) {
	client, stub := newStubAPI(t, map[string][]stubResponse{
		actionGetRoute:  okResponse(actionConfigurationJson(actionTypeFlow, "null", "null")),
		ruleUpdateRoute: okResponse(`{"ruleId":"rule-1","tenantId":"tenant"}`),
	})

	state := ruleState(t)
	resp := &resource.UpdateResponse{State: tfsdk.State{Schema: state.Schema}}

	(&ruleResource{client: client}).Update(
		context.Background(),
		resource.UpdateRequest{
			Plan:  tfsdk.Plan{Schema: state.Schema, Raw: state.Raw},
			State: state,
		},
		resp,
	)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected updating a rule on a FLOW action to fail")
	}

	stub.assertNotCalled(t, ruleUpdateRoute)
	stub.assertRoutes(t, actionGetRoute)
}

func TestRuleCreateReachesTheRulesEndpointForAClassicAction(t *testing.T) {
	client, stub := newStubAPI(t, map[string][]stubResponse{
		actionGetRoute:  okResponse(actionConfigurationJson(actionTypeClassic, "null", "null")),
		ruleCreateRoute: okResponse(`{"ruleId":"rule-1","tenantId":"tenant"}`),
	})

	state := ruleState(t)
	resp := &resource.CreateResponse{State: tfsdk.State{Schema: state.Schema}}

	(&ruleResource{client: client}).Create(
		context.Background(),
		resource.CreateRequest{Plan: tfsdk.Plan{Schema: state.Schema, Raw: state.Raw}},
		resp,
	)

	if resp.Diagnostics.HasError() {
		t.Fatal(detailsOf(resp.Diagnostics))
	}

	// The guard must not stand between a CLASSIC action and its rules.
	stub.assertRoutes(t, actionGetRoute, ruleCreateRoute)
}

func TestRuleReadRefusesARuleOfAFlowAction(t *testing.T) {
	client, stub := newStubAPI(t, map[string][]stubResponse{
		actionGetRoute: okResponse(actionConfigurationJson(actionTypeFlow, "null", "null")),
		ruleGetRoute:   okResponse(ruleJson),
	})

	state := ruleState(t)
	resp := &resource.ReadResponse{State: state}

	(&ruleResource{client: client}).Read(context.Background(), resource.ReadRequest{State: state}, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected reading a rule of a FLOW action to fail")
	}

	if !strings.Contains(detailsOf(resp.Diagnostics), "terraform state rm") {
		t.Errorf("the error must say how to stop managing the rule, got %s", detailsOf(resp.Diagnostics))
	}

	// Dropping the rule from state would let a later plan create it again on the flow's action.
	if resp.State.Raw.IsNull() {
		t.Error("the rule must stay in state")
	}

	stub.assertRoutes(t, actionGetRoute)
}

func TestRuleReadReachesTheRuleOfAClassicAction(t *testing.T) {
	client, stub := newStubAPI(t, map[string][]stubResponse{
		actionGetRoute: okResponse(actionConfigurationJson(actionTypeClassic, "null", "null")),
		ruleGetRoute:   okResponse(ruleJson),
	})

	state := ruleState(t)
	resp := &resource.ReadResponse{State: state}

	(&ruleResource{client: client}).Read(context.Background(), resource.ReadRequest{State: state}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatal(detailsOf(resp.Diagnostics))
	}

	stub.assertRoutes(t, actionGetRoute, ruleGetRoute)
}

func importRule(t *testing.T, client *authsignal.Client) *resource.ImportStateResponse {
	t.Helper()

	resourceSchema := resourceSchemaOf(t, &ruleResource{})
	schemaType := resourceSchema.Type().TerraformType(context.Background())

	resp := &resource.ImportStateResponse{
		State: tfsdk.State{Schema: resourceSchema, Raw: tftypes.NewValue(schemaType, nil)},
	}

	(&ruleResource{client: client}).ImportState(context.Background(), resource.ImportStateRequest{ID: "sign-in/rule-1"}, resp)

	return resp
}

func TestRuleImportRefusesARuleOfAFlowAction(t *testing.T) {
	client, _ := newStubAPI(t, map[string][]stubResponse{
		actionGetRoute: okResponse(actionConfigurationJson(actionTypeFlow, "null", "null")),
	})

	resp := importRule(t, client)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected importing a rule of a FLOW action to fail")
	}

	if !strings.Contains(detailsOf(resp.Diagnostics), "authsignal_flow") {
		t.Errorf("the error must name authsignal_flow, got %s", detailsOf(resp.Diagnostics))
	}
}

func TestRuleImportOfAClassicActionKeepsItsIdentifiers(t *testing.T) {
	client, _ := newStubAPI(t, map[string][]stubResponse{
		actionGetRoute: okResponse(actionConfigurationJson(actionTypeClassic, "null", "null")),
	})

	resp := importRule(t, client)

	if resp.Diagnostics.HasError() {
		t.Fatal(detailsOf(resp.Diagnostics))
	}

	var model ruleResourceModel
	if diags := resp.State.Get(context.Background(), &model); diags.HasError() {
		t.Fatal(detailsOf(diags))
	}

	if model.ActionCode.ValueString() != "sign-in" || model.RuleId.ValueString() != "rule-1" {
		t.Errorf("expected sign-in/rule-1 in state, got %s/%s", model.ActionCode.ValueString(), model.RuleId.ValueString())
	}
}

func TestActionConfigurationCreateRefusesAnExistingFlowAction(t *testing.T) {
	client, stub := newStubAPI(t, map[string][]stubResponse{
		actionGetRoute:    okResponse(actionConfigurationJson(actionTypeFlow, "null", "null")),
		actionCreateRoute: okResponse(actionConfigurationJson(actionTypeClassic, "null", "null")),
	})

	resourceSchema := resourceSchemaOf(t, &actionConfigurationResource{})
	plan := tfsdk.Plan{
		Schema: resourceSchema,
		Raw: resourceObject(t, resourceSchema, map[string]tftypes.Value{
			"action_code":                tftypes.NewValue(tftypes.String, "sign-in"),
			"default_user_action_result": tftypes.NewValue(tftypes.String, "ALLOW"),
		}),
	}

	resp := &resource.CreateResponse{State: tfsdk.State{Schema: resourceSchema}}

	(&actionConfigurationResource{client: client}).Create(
		context.Background(),
		resource.CreateRequest{Plan: plan},
		resp,
	)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected creating a CLASSIC action over a FLOW action to fail")
	}

	if !strings.Contains(detailsOf(resp.Diagnostics), "authsignal_flow") {
		t.Errorf("the error must name authsignal_flow, got %s", detailsOf(resp.Diagnostics))
	}

	// The create request states the action type, so reaching it would discard the flow.
	stub.assertNotCalled(t, actionCreateRoute)
	stub.assertRoutes(t, actionGetRoute)
}
