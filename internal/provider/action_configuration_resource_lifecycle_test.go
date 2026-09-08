package provider

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/authsignal/authsignal-management-go/v6"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func createActionConfiguration(t *testing.T, client *authsignal.Client) *resource.CreateResponse {
	t.Helper()

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

	return resp
}

// A create states the action type, so it can convert an action it did not expect to find.
// When the pre-flight cannot establish the type, nothing may be written.
func TestCreateFailsClosedWhenTheActionTypeCannotBeRead(t *testing.T) {
	unreadable := map[string][]stubResponse{
		"forbidden":            {{status: http.StatusForbidden, body: `{"error":"forbidden"}`}},
		"unauthorised":         {{status: http.StatusUnauthorized, body: `{"error":"unauthorized"}`}},
		"a server error":       {{status: http.StatusInternalServerError, body: `{"error":"boom"}`}},
		"a malformed response": {{status: http.StatusOK, body: `not json`}},
	}

	for name, responses := range unreadable {
		t.Run("authsignal_flow with "+name, func(t *testing.T) {
			client, stub := newStubAPI(t, map[string][]stubResponse{actionGetRoute: responses})

			resp := createFlow(t, client, testFlow)

			assertPreflightAbandonedTheCreate(t, stub, resp.Diagnostics, resp.State)
		})

		t.Run("authsignal_action_configuration with "+name, func(t *testing.T) {
			client, stub := newStubAPI(t, map[string][]stubResponse{actionGetRoute: responses})

			resp := createActionConfiguration(t, client)

			assertPreflightAbandonedTheCreate(t, stub, resp.Diagnostics, resp.State)
		})
	}
}

func assertPreflightAbandonedTheCreate(t *testing.T, stub *stubAPI, diagnostics diag.Diagnostics, state tfsdk.State) {
	t.Helper()

	if !diagnostics.HasError() {
		t.Fatal("expected a create to fail when the action type cannot be established")
	}

	stub.assertNotCalled(t, actionCreateRoute)
	stub.assertNotCalled(t, flowPublishRoute)
	stub.assertRoutes(t, actionGetRoute)

	if !state.Raw.IsNull() {
		t.Error("nothing was created, so nothing may be recorded in state")
	}
}

func TestCreateAbandonedByThePreflightSaysNothingWasCreated(t *testing.T) {
	client, _ := newStubAPI(t, map[string][]stubResponse{
		actionGetRoute: {{status: http.StatusInternalServerError, body: `{"error":"boom"}`}},
	})

	details := detailsOf(createFlow(t, client, testFlow).Diagnostics)

	if !strings.Contains(details, "Nothing has been created") {
		t.Errorf("the error must say that nothing was written, got %s", details)
	}
}

func TestActionConfigurationCreateKeepsTheActionInStateWhenTheApiReturnsAFlowAction(t *testing.T) {
	client, stub := newStubAPI(t, map[string][]stubResponse{
		actionGetRoute:    {notFoundResponse()},
		actionCreateRoute: okResponse(actionConfigurationJson(actionTypeFlow, "null", "null")),
	})

	resp := createActionConfiguration(t, client)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected a create that produced a FLOW action to fail")
	}

	stub.assertRoutes(t, actionGetRoute, actionCreateRoute)

	if resp.State.Raw.IsNull() {
		t.Fatal("expected the created action to be recorded in state, got no state at all")
	}

	var actionCode string
	resp.State.GetAttribute(context.Background(), path.Root("action_code"), &actionCode)
	if actionCode != "sign-in" {
		t.Errorf("expected the created action code in state, got %q", actionCode)
	}

	assertWrongTypeRecoveryIsAccurate(t, resp.Diagnostics, "authsignal_flow")
}

func TestFlowCreateWrongTypeRecoveryIsAccurate(t *testing.T) {
	client, _ := newStubAPI(t, map[string][]stubResponse{
		actionGetRoute:    {notFoundResponse()},
		actionCreateRoute: okResponse(actionConfigurationJson(actionTypeClassic, "null", "null")),
	})

	assertWrongTypeRecoveryIsAccurate(t, createFlow(t, client, testFlow).Diagnostics, "authsignal_action_configuration")
}

// The resource's Read rejects the other action type, so the recovery this diagnostic
// offers has to work without a refresh. Promising an ordinary destroy would be wrong.
func assertWrongTypeRecoveryIsAccurate(t *testing.T, diagnostics diag.Diagnostics, otherResource string) {
	t.Helper()

	details := detailsOf(diagnostics)

	for _, required := range []string{
		"will now all fail while refreshing",
		"terraform destroy -refresh=false",
		"terraform state rm",
		otherResource,
	} {
		if !strings.Contains(details, required) {
			t.Errorf("the recovery must mention %q, got %s", required, details)
		}
	}

	// Ordinary refresh-based recovery does not work, so it must not be offered.
	if strings.Contains(details, "the next refresh") || strings.Contains(details, "destroy this resource to remove it") {
		t.Errorf("the recovery must not promise an ordinary destroy or refresh, got %s", details)
	}
}
