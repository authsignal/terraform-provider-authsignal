package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func resourceSchemaOf(t *testing.T, res resource.Resource) schema.Schema {
	t.Helper()

	resp := &resource.SchemaResponse{}
	res.Schema(context.Background(), resource.SchemaRequest{}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}

	return resp.Schema
}

func flowSchema(t *testing.T) schema.Schema {
	t.Helper()

	return resourceSchemaOf(t, &flowResource{})
}

// Attributes the caller leaves out are null, as Terraform sends them for an unset optional.
func resourceObject(t *testing.T, resourceSchema schema.Schema, attributes map[string]tftypes.Value) tftypes.Value {
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

func TestActionConfigurationSchemaIsClassicOnly(t *testing.T) {
	resourceSchema := resourceSchemaOf(t, &actionConfigurationResource{})

	for _, name := range []string{"action_type", "flow", "flow_version"} {
		if _, has := resourceSchema.Attributes[name]; has {
			t.Errorf("a CLASSIC-only resource must not expose %q; that belongs to authsignal_flow", name)
		}
	}

	defaultResult, has := resourceSchema.Attributes["default_user_action_result"]
	if !has {
		t.Fatal("expected a default_user_action_result attribute")
	}

	if !defaultResult.IsRequired() {
		t.Error("default_user_action_result must be required now that no flow action shares the schema")
	}

	for _, name := range []string{"messaging_templates", "verification_methods", "prompt_to_enroll_verification_methods", "default_verification_method"} {
		if _, has := resourceSchema.Attributes[name]; !has {
			t.Errorf("expected the CLASSIC setting %q to be kept", name)
		}
	}
}

func TestFlowSchema(t *testing.T) {
	resourceSchema := flowSchema(t)

	required := map[string]bool{"action_code": true, "flow": true}
	computed := map[string]bool{"flow_version": true, "tenant_id": true, "last_action_created_at": true}

	for name, attribute := range resourceSchema.Attributes {
		switch {
		case required[name]:
			if !attribute.IsRequired() {
				t.Errorf("expected %q to be required", name)
			}
		case computed[name]:
			if !attribute.IsComputed() {
				t.Errorf("expected %q to be computed", name)
			}
		default:
			t.Errorf("unexpected attribute %q on authsignal_flow", name)
		}
	}

	for name := range required {
		if _, has := resourceSchema.Attributes[name]; !has {
			t.Errorf("expected a %q attribute", name)
		}
	}

	// The CLASSIC settings are irrelevant to a graph-driven flow.
	for _, name := range []string{"action_type", "default_user_action_result", "messaging_templates", "verification_methods", "prompt_to_enroll_verification_methods", "default_verification_method"} {
		if _, has := resourceSchema.Attributes[name]; has {
			t.Errorf("authsignal_flow must not expose %q", name)
		}
	}

	if _, ok := resourceSchema.Attributes["flow"].GetType().(FlowType); !ok {
		t.Error("flow must keep the FlowType custom type so JSON stays semantically compared")
	}
}
