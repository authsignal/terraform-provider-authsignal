package provider

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func detailsOf(diagnostics diag.Diagnostics) string {
	details := make([]string, 0, len(diagnostics))
	for _, diagnostic := range diagnostics {
		details = append(details, diagnostic.Summary()+": "+diagnostic.Detail())
	}

	return strings.Join(details, "\n")
}

func TestActionConfigurationReadRejectsAFlowAction(t *testing.T) {
	client, _ := newStubAPI(t, map[string][]stubResponse{
		actionGetRoute: okResponse(actionConfigurationJson(actionTypeFlow, "null", "null")),
	})

	resourceSchema := resourceSchemaOf(t, &actionConfigurationResource{})
	state := tfsdk.State{
		Schema: resourceSchema,
		Raw: resourceObject(t, resourceSchema, map[string]tftypes.Value{
			"action_code":                tftypes.NewValue(tftypes.String, "sign-in"),
			"default_user_action_result": tftypes.NewValue(tftypes.String, "ALLOW"),
		}),
	}

	resp := &resource.ReadResponse{State: state}
	(&actionConfigurationResource{client: client}).Read(
		context.Background(),
		resource.ReadRequest{State: state},
		resp,
	)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected reading a FLOW action through the CLASSIC resource to fail")
	}

	if !strings.Contains(detailsOf(resp.Diagnostics), "authsignal_flow") {
		t.Fatalf("the error must name authsignal_flow, got %s", detailsOf(resp.Diagnostics))
	}
}

func TestFlowReadRejectsAClassicAction(t *testing.T) {
	client, _ := newStubAPI(t, map[string][]stubResponse{
		actionGetRoute: okResponse(actionConfigurationJson(actionTypeClassic, "null", "null")),
	})

	state := flowState(t, testFlow, tftypes.NewValue(tftypes.Number, 4))

	resp := &resource.ReadResponse{State: state}
	(&flowResource{client: client}).Read(
		context.Background(),
		resource.ReadRequest{State: state},
		resp,
	)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected reading a CLASSIC action through the flow resource to fail")
	}

	if !strings.Contains(detailsOf(resp.Diagnostics), "authsignal_action_configuration") {
		t.Fatalf("the error must name authsignal_action_configuration, got %s", detailsOf(resp.Diagnostics))
	}
}

func TestRuleGuardOnActionType(t *testing.T) {
	testCases := []struct {
		name      string
		responses []stubResponse
		wantError bool
	}{
		{
			name:      "a FLOW action is refused",
			responses: okResponse(actionConfigurationJson(actionTypeFlow, "null", "null")),
			wantError: true,
		},
		{
			name:      "a CLASSIC action is allowed",
			responses: okResponse(actionConfigurationJson(actionTypeClassic, "null", "null")),
			wantError: false,
		},
		{
			// The guard must never turn an unreadable action configuration into a failed apply.
			name:      "an unreadable action configuration is allowed through",
			responses: []stubResponse{{status: http.StatusForbidden, body: `{"error":"forbidden"}`}},
			wantError: false,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			client, _ := newStubAPI(t, map[string][]stubResponse{
				actionGetRoute: testCase.responses,
			})

			diagnostics := (&ruleResource{client: client}).checkActionIsClassic("sign-in")

			if diagnostics.HasError() != testCase.wantError {
				t.Fatalf("expected an error=%v, got %s", testCase.wantError, detailsOf(diagnostics))
			}

			if testCase.wantError && !strings.Contains(detailsOf(diagnostics), "authsignal_flow") {
				t.Fatalf("the error must name authsignal_flow, got %s", detailsOf(diagnostics))
			}
		})
	}
}
