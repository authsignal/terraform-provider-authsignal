package provider

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/authsignal/authsignal-management-go/v6"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func flowState(t *testing.T, flow string, flowVersion tftypes.Value) tfsdk.State {
	t.Helper()

	resourceSchema := flowSchema(t)

	return tfsdk.State{
		Schema: resourceSchema,
		Raw: resourceObject(t, resourceSchema, map[string]tftypes.Value{
			"action_code":            tftypes.NewValue(tftypes.String, "sign-in"),
			"flow":                   tftypes.NewValue(tftypes.String, flow),
			"flow_version":           flowVersion,
			"tenant_id":              tftypes.NewValue(tftypes.String, "tenant"),
			"last_action_created_at": tftypes.NewValue(tftypes.String, "2026-09-07T00:00:00.000Z"),
		}),
	}
}

func flowPlan(t *testing.T, flow string) tfsdk.Plan {
	t.Helper()

	state := flowState(t, flow, tftypes.NewValue(tftypes.Number, nil))

	return tfsdk.Plan{Schema: state.Schema, Raw: state.Raw}
}

func createFlow(t *testing.T, client *authsignal.Client, flow string) *resource.CreateResponse {
	t.Helper()

	resourceSchema := flowSchema(t)
	resp := &resource.CreateResponse{State: tfsdk.State{Schema: resourceSchema}}

	(&flowResource{client: client}).Create(
		context.Background(),
		resource.CreateRequest{Plan: flowPlan(t, flow)},
		resp,
	)

	return resp
}

func TestFlowCreateCreatesTheActionThenPublishesTheFlow(t *testing.T) {
	client, stub := newStubAPI(t, map[string][]stubResponse{
		actionGetRoute: {
			notFoundResponse(),
			{status: http.StatusOK, body: actionConfigurationJson(actionTypeFlow, testNodes, "1")},
		},
		actionCreateRoute: okResponse(actionConfigurationJson(actionTypeFlow, "null", "null")),
		flowPublishRoute:  okResponse(`{"tenantId":"tenant","actionCode":"sign-in","actionType":"FLOW","actionNodes":` + testNodes + `,"flowVersion":1}`),
		rulesListRoute:    okResponse(`[]`),
	})

	resp := createFlow(t, client, testFlow)

	if resp.Diagnostics.HasError() {
		t.Fatal(detailsOf(resp.Diagnostics))
	}

	stub.assertRoutes(t,
		actionGetRoute, // the pre-flight type check
		actionCreateRoute,
		flowPublishRoute,
		actionGetRoute,
		rulesListRoute,
	)

	created := stub.bodyOf(t, actionCreateRoute)
	if !strings.Contains(created, `"actionType":"FLOW"`) {
		t.Errorf("the create request must state the FLOW action type, got %s", created)
	}
	if strings.Contains(created, "actionNodes") {
		t.Errorf("the graph is only ever published through the flow endpoint, got %s", created)
	}

	published := stub.bodyOf(t, flowPublishRoute)
	if !strings.Contains(published, `"actionNodes"`) || !strings.Contains(published, `"rules"`) {
		t.Errorf("the publish request must carry the nodes and the rules, got %s", published)
	}
	if strings.Contains(published, "expectedFlowVersion") {
		t.Errorf("a first publish has no version to expect, got %s", published)
	}

	var flowVersion int64
	resp.State.GetAttribute(context.Background(), path.Root("flow_version"), &flowVersion)
	if flowVersion != 1 {
		t.Errorf("expected the published flow_version 1 in state, got %d", flowVersion)
	}
}

func TestFlowCreateKeepsTheActionInStateWhenPublishingFails(t *testing.T) {
	client, stub := newStubAPI(t, map[string][]stubResponse{
		actionGetRoute:    {notFoundResponse()},
		actionCreateRoute: okResponse(actionConfigurationJson(actionTypeFlow, "null", "null")),
		flowPublishRoute:  {{status: http.StatusInternalServerError, body: `{"error":"boom"}`}},
	})

	resp := createFlow(t, client, testFlow)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected a failed publish to fail the create")
	}

	stub.assertRoutes(t, actionGetRoute, actionCreateRoute, flowPublishRoute)

	assertFlowActionRecorded(t, resp.State, "a created action whose flow failed to publish")
}

func TestFlowCreateRefusesAnExistingClassicAction(t *testing.T) {
	client, stub := newStubAPI(t, map[string][]stubResponse{
		actionGetRoute: okResponse(actionConfigurationJson(actionTypeClassic, "null", "null")),
	})

	resp := createFlow(t, client, testFlow)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected creating a flow over a CLASSIC action to fail")
	}

	if !strings.Contains(detailsOf(resp.Diagnostics), "authsignal_action_configuration") {
		t.Errorf("the error must name authsignal_action_configuration, got %s", detailsOf(resp.Diagnostics))
	}

	// Nothing may be written: the create request carries the action type and would convert it.
	stub.assertRoutes(t, actionGetRoute)
	stub.assertNotCalled(t, actionCreateRoute)
	stub.assertNotCalled(t, flowPublishRoute)
}

func TestFlowCreateKeepsTheActionInStateWhenTheApiIgnoresTheFlowType(t *testing.T) {
	client, stub := newStubAPI(t, map[string][]stubResponse{
		actionGetRoute:    {notFoundResponse()},
		actionCreateRoute: okResponse(actionConfigurationJson(actionTypeClassic, "null", "null")),
	})

	resp := createFlow(t, client, testFlow)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected a create that did not produce a FLOW action to fail")
	}

	// The flow was never published, because the action is not a flow action.
	stub.assertNotCalled(t, flowPublishRoute)

	assertFlowActionRecorded(t, resp.State, "an action the API created with the wrong type")
}

func TestFlowUpdatePublishesTheChangeWithTheExpectedVersion(t *testing.T) {
	const changed = `{"actionNodes":[{"nodeId":"c","nodeType":"BLOCK"}],"rules":[]}`

	client, stub := newStubAPI(t, map[string][]stubResponse{
		flowPublishRoute: okResponse(`{"tenantId":"tenant","actionCode":"sign-in","actionType":"FLOW","flowVersion":5}`),
		actionGetRoute:   okResponse(actionConfigurationJson(actionTypeFlow, `[{"nodeId":"c","nodeType":"BLOCK"}]`, "5")),
		rulesListRoute:   okResponse(`[]`),
	})

	resourceSchema := flowSchema(t)
	resp := &resource.UpdateResponse{State: tfsdk.State{Schema: resourceSchema}}

	(&flowResource{client: client}).Update(
		context.Background(),
		resource.UpdateRequest{
			Plan:  flowPlan(t, changed),
			State: flowState(t, testFlow, tftypes.NewValue(tftypes.Number, 4)),
		},
		resp,
	)

	if resp.Diagnostics.HasError() {
		t.Fatal(detailsOf(resp.Diagnostics))
	}

	// The action configuration itself is never written by this resource.
	stub.assertNotCalled(t, actionCreateRoute)
	stub.assertNotCalled(t, actionPatchRoute)
	stub.assertRoutes(t, flowPublishRoute, actionGetRoute, rulesListRoute)

	published := stub.bodyOf(t, flowPublishRoute)
	if !strings.Contains(published, `"expectedFlowVersion":4`) {
		t.Errorf("the publish must be based on the version in state, got %s", published)
	}
}

func TestFlowUpdateDoesNotPublishAnUnchangedFlow(t *testing.T) {
	const reformatted = `{"rules":[],"actionNodes":[{"nodeType":"COMPLETE","nodeId":"c"}]}`

	client, stub := newStubAPI(t, map[string][]stubResponse{
		actionGetRoute: okResponse(actionConfigurationJson(actionTypeFlow, testNodes, "4")),
		rulesListRoute: okResponse(`[]`),
	})

	resourceSchema := flowSchema(t)
	resp := &resource.UpdateResponse{State: tfsdk.State{Schema: resourceSchema}}

	(&flowResource{client: client}).Update(
		context.Background(),
		resource.UpdateRequest{
			Plan:  flowPlan(t, reformatted),
			State: flowState(t, testFlow, tftypes.NewValue(tftypes.Number, 4)),
		},
		resp,
	)

	if resp.Diagnostics.HasError() {
		t.Fatal(detailsOf(resp.Diagnostics))
	}

	stub.assertNotCalled(t, flowPublishRoute)
}

func TestFlowUpdateReportsAFlowPublishedElsewhere(t *testing.T) {
	const changed = `{"actionNodes":[{"nodeId":"c","nodeType":"BLOCK"}],"rules":[]}`

	client, _ := newStubAPI(t, map[string][]stubResponse{
		flowPublishRoute: {{status: http.StatusConflict, body: `{"error":"conflict"}`}},
	})

	resourceSchema := flowSchema(t)
	resp := &resource.UpdateResponse{State: tfsdk.State{Schema: resourceSchema}}

	(&flowResource{client: client}).Update(
		context.Background(),
		resource.UpdateRequest{
			Plan:  flowPlan(t, changed),
			State: flowState(t, testFlow, tftypes.NewValue(tftypes.Number, 4)),
		},
		resp,
	)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected a conflicting publish to fail the update")
	}

	details := detailsOf(resp.Diagnostics)
	if !strings.Contains(details, "Flow changed outside Terraform") || !strings.Contains(details, "flow version 4") {
		t.Errorf("the error must report the version Terraform expected, got %s", details)
	}
}

func TestFlowDeleteDeletesTheWholeActionConfiguration(t *testing.T) {
	client, stub := newStubAPI(t, map[string][]stubResponse{
		actionDeleteRoute: okResponse(`{"success":true}`),
	})

	resp := &resource.DeleteResponse{State: tfsdk.State{Schema: flowSchema(t)}}

	(&flowResource{client: client}).Delete(
		context.Background(),
		resource.DeleteRequest{State: flowState(t, testFlow, tftypes.NewValue(tftypes.Number, 4))},
		resp,
	)

	if resp.Diagnostics.HasError() {
		t.Fatal(detailsOf(resp.Diagnostics))
	}

	// A flow cannot be unpublished on its own, so the whole action goes.
	stub.assertRoutes(t, actionDeleteRoute)
}

func TestFlowReadComposesTheFlowFromTheNodesAndTheRules(t *testing.T) {
	client, stub := newStubAPI(t, map[string][]stubResponse{
		actionGetRoute: okResponse(actionConfigurationJson(actionTypeFlow, `[{"nodeId":"r","nodeType":"RULE","ruleChildNodeIds":[["only","c"]]},{"nodeId":"c","nodeType":"COMPLETE"}]`, "7")),
		rulesListRoute: okResponse(`[{"ruleId":"only","name":"Only","conditions":{"and":[]}}]`),
	})

	resourceSchema := flowSchema(t)
	resp := &resource.ReadResponse{State: tfsdk.State{Schema: resourceSchema}}

	(&flowResource{client: client}).Read(
		context.Background(),
		resource.ReadRequest{State: flowState(t, testFlow, tftypes.NewValue(tftypes.Number, 7))},
		resp,
	)

	if resp.Diagnostics.HasError() {
		t.Fatal(detailsOf(resp.Diagnostics))
	}

	stub.assertRoutes(t, actionGetRoute, rulesListRoute)

	var flow string
	resp.State.GetAttribute(context.Background(), path.Root("flow"), &flow)

	if !strings.Contains(flow, `"nodeId":"r"`) || !strings.Contains(flow, `"ruleId":"only"`) {
		t.Errorf("the flow must compose the server's nodes and rules, got %s", flow)
	}
}

func TestFlowReadForgetsAnActionThatIsGone(t *testing.T) {
	client, _ := newStubAPI(t, map[string][]stubResponse{})

	resp := &resource.ReadResponse{State: flowState(t, testFlow, tftypes.NewValue(tftypes.Number, 4))}

	(&flowResource{client: client}).Read(
		context.Background(),
		resource.ReadRequest{State: flowState(t, testFlow, tftypes.NewValue(tftypes.Number, 4))},
		resp,
	)

	if resp.Diagnostics.HasError() {
		t.Fatal(detailsOf(resp.Diagnostics))
	}

	if !resp.State.Raw.IsNull() {
		t.Error("a 404 must remove the resource from state so it can be recreated")
	}
}

// A create that fails after the action exists must still leave Terraform holding the
// action, so that a refresh can reconcile it or a destroy can remove it.
func assertFlowActionRecorded(t *testing.T, state tfsdk.State, what string) {
	t.Helper()

	if state.Raw.IsNull() {
		t.Fatalf("expected %s to be recorded in state, got no state at all", what)
	}

	var actionCode string
	state.GetAttribute(context.Background(), path.Root("action_code"), &actionCode)
	if actionCode != "sign-in" {
		t.Errorf("expected the action code of %s in state, got %q", what, actionCode)
	}

	var flowVersion *int64
	state.GetAttribute(context.Background(), path.Root("flow_version"), &flowVersion)
	if flowVersion != nil {
		t.Errorf("expected no flow_version for %s, got %d", what, *flowVersion)
	}
}
