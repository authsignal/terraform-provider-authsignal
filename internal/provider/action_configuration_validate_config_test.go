package provider

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func actionConfigurationSchema(t *testing.T) schema.Schema {
	t.Helper()

	resp := &resource.SchemaResponse{}
	(&actionConfigurationResource{}).Schema(context.Background(), resource.SchemaRequest{}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}

	return resp.Schema
}

// Attributes the caller leaves out are null, as Terraform sends them for an unset optional.
func actionConfigurationObject(t *testing.T, resourceSchema schema.Schema, attributes map[string]tftypes.Value) tftypes.Value {
	t.Helper()

	objectType, ok := resourceSchema.Type().TerraformType(context.Background()).(tftypes.Object)
	if !ok {
		t.Fatalf("expected the resource schema to be an object")
	}

	values := make(map[string]tftypes.Value, len(objectType.AttributeTypes))
	for name, attributeType := range objectType.AttributeTypes {
		if value, has := attributes[name]; has {
			values[name] = value
			continue
		}

		values[name] = tftypes.NewValue(attributeType, nil)
	}

	for name := range attributes {
		if _, has := objectType.AttributeTypes[name]; !has {
			t.Fatalf("the schema has no attribute %q", name)
		}
	}

	return tftypes.NewValue(objectType, values)
}

func optionalString(value string) tftypes.Value {
	if value == "" {
		return tftypes.NewValue(tftypes.String, nil)
	}

	return tftypes.NewValue(tftypes.String, value)
}

func attributeErrors(diagnostics diag.Diagnostics) map[string]string {
	errors := map[string]string{}

	for _, diagnostic := range diagnostics.Errors() {
		withPath, ok := diagnostic.(diag.DiagnosticWithPath)
		if !ok {
			errors[""] = diagnostic.Summary() + " " + diagnostic.Detail()
			continue
		}

		errors[withPath.Path().String()] = diagnostic.Summary() + " " + diagnostic.Detail()
	}

	return errors
}

func TestActionConfigurationValidateConfig(t *testing.T) {
	const flow = `{"actionNodes":[{"nodeId":"c","nodeType":"COMPLETE"}],"rules":[]}`

	testCases := []struct {
		name          string
		actionType    string
		flow          string
		defaultResult string
		expected      []string
	}{
		{"FLOW with a flow", actionTypeFlow, flow, "", nil},
		{"FLOW with a flow and a default result", actionTypeFlow, flow, "ALLOW", []string{"default_user_action_result"}},
		{"FLOW without a flow", actionTypeFlow, "", "", []string{"flow"}},
		{"FLOW without a flow and with a default result", actionTypeFlow, "", "ALLOW", []string{"flow", "default_user_action_result"}},
		{"CLASSIC with a default result", actionTypeClassic, "", "ALLOW", nil},
		{"CLASSIC without a default result", actionTypeClassic, "", "", []string{"default_user_action_result"}},
		{"CLASSIC with a flow and a default result", actionTypeClassic, flow, "ALLOW", []string{"flow"}},
		{"CLASSIC with a flow and no default result", actionTypeClassic, flow, "", []string{"flow", "default_user_action_result"}},
		// action_type defaults to CLASSIC, and the default is not applied to the raw configuration.
		{"an unset action type with a default result", "", "", "ALLOW", nil},
		{"an unset action type without a default result", "", "", "", []string{"default_user_action_result"}},
		{"an unset action type with a flow", "", flow, "ALLOW", []string{"flow"}},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			resourceSchema := actionConfigurationSchema(t)

			config := actionConfigurationObject(t, resourceSchema, map[string]tftypes.Value{
				"action_code":                tftypes.NewValue(tftypes.String, "sign-in"),
				"action_type":                optionalString(testCase.actionType),
				"flow":                       optionalString(testCase.flow),
				"default_user_action_result": optionalString(testCase.defaultResult),
			})

			resp := &resource.ValidateConfigResponse{}
			(&actionConfigurationResource{}).ValidateConfig(
				context.Background(),
				resource.ValidateConfigRequest{Config: tfsdk.Config{Schema: resourceSchema, Raw: config}},
				resp,
			)

			errors := attributeErrors(resp.Diagnostics)

			if len(errors) != len(testCase.expected) {
				t.Fatalf("expected errors on %v, got %v", testCase.expected, errors)
			}

			for _, attribute := range testCase.expected {
				message, reported := errors[attribute]
				if !reported {
					t.Fatalf("expected an error on %q, got %v", attribute, errors)
				}

				if !strings.Contains(message, attribute) {
					t.Fatalf("the error on %q must name the attribute, got %q", attribute, message)
				}
			}
		})
	}
}
