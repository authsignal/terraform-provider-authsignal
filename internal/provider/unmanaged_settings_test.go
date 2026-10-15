package provider

import (
	"context"
	"encoding/json"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/authsignal/authsignal-management-go/v6"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// A setting that keeps the tenant's value when the configuration leaves it out has to fail the plan
// when that value is unexpected, wherever it sits in the schema and whichever resource it is on.
func TestEverySettingTheConfigurationCanLeaveOutIsChecked(t *testing.T) {
	ctx := context.Background()

	for _, newResource := range New("test")().Resources(ctx) {
		res := newResource()
		metadata := &resource.MetadataResponse{}
		res.Metadata(ctx, resource.MetadataRequest{ProviderTypeName: "authsignal"}, metadata)

		t.Run(metadata.TypeName, func(t *testing.T) {
			resourceSchema := resourceSchemaOf(t, res)
			paths := keptWhenLeftOut("", resourceSchema.Attributes)
			if len(paths) == 0 {
				return
			}

			modifier, ok := res.(resource.ResourceWithModifyPlan)
			if !ok {
				t.Fatalf("%v keep the tenant's value when left out, but the resource has no ModifyPlan", paths)
			}

			sample := sampleValue(t, resourceSchema.Type().TerraformType(ctx))
			req := resource.ModifyPlanRequest{
				Config: tfsdk.Config{Schema: resourceSchema, Raw: resourceObject(t, resourceSchema, nil)},
				State:  tfsdk.State{Schema: resourceSchema, Raw: sample},
				Plan:   tfsdk.Plan{Schema: resourceSchema, Raw: sample},
			}
			resp := &resource.ModifyPlanResponse{Plan: req.Plan}
			modifier.ModifyPlan(ctx, req, resp)

			detail := unmanagedSettingsDetail(t, resp.Diagnostics)
			listed := regexp.MustCompile(`(?m)^ +\S+ += .*  # `).FindAllString(detail, -1)
			if len(listed) != len(paths) {
				t.Errorf("expected %d settings listed, got %d:\n%s", len(paths), len(listed), detail)
			}

			for _, path := range paths {
				block, name := splitSettingPath(path)
				if block != "" && !regexp.MustCompile(`(?m)^  `+regexp.QuoteMeta(block)+` = \{$`).MatchString(detail) {
					t.Errorf("%s is not listed inside its block:\n%s", path, detail)
				}

				if !regexp.MustCompile(`(?m)^ +` + regexp.QuoteMeta(name) + ` += `).MatchString(detail) {
					t.Errorf("%s is not listed:\n%s", path, detail)
				}
			}
		})
	}
}

func TestUnmanagedSettingsAreListedWithTheirDefaults(t *testing.T) {
	resourceSchema := passkeySchema(t)
	state := resourceObject(t, resourceSchema, map[string]tftypes.Value{
		"is_active":                     tftypes.NewValue(tftypes.Bool, false),
		"passkey_registration_hints":    tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, []tftypes.Value{tftypes.NewValue(tftypes.String, "security-key")}),
		"user_verification_requirement": tftypes.NewValue(tftypes.String, "required"),
		"authenticator_attachment":      tftypes.NewValue(tftypes.String, "all-supported"),
		"prevent_removing_last_passkey": tftypes.NewValue(tftypes.Bool, false),
	})

	diags := unmanagedSettingsDiagnostics(context.Background(), resourceSchema, resourceObject(t, resourceSchema, nil), state, state, passkeyDefaults())

	expected := "The tenant has values for these settings, but the configuration doesn't set them:\n\n" +
		`  is_active                     = false             # default: true` + "\n" +
		`  passkey_registration_hints    = ["security-key"]  # no default` + "\n" +
		`  user_verification_requirement = "required"        # default: "preferred"` + "\n\n" +
		"Changes made to them outside Terraform won't show in a plan, and a tenant built from this configuration " +
		"won't get these values. Add them to the configuration."

	if detail := unmanagedSettingsDetail(t, diags); detail != expected {
		t.Errorf("expected\n%s\ngot\n%s", expected, detail)
	}
}

func TestUnmanagedSettingsInsideABlockAreListedInsideIt(t *testing.T) {
	resourceSchema := resourceSchemaOf(t, &tenantSettingsResource{})
	config := resourceObject(t, resourceSchema, map[string]tftypes.Value{
		"authenticator_events_webhook_config": authenticatorEventsWebhookValue(t, types.BoolNull()),
	})
	state := resourceObject(t, resourceSchema, map[string]tftypes.Value{
		"authenticator_events_webhook_config": authenticatorEventsWebhookValue(t, types.BoolValue(true)),
	})

	diags := unmanagedSettingsDiagnostics(context.Background(), resourceSchema, config, state, state, authsignal.Defaults{})

	expected := "  authenticator_events_webhook_config = {\n" +
		"    include_credential_public_key = true  # no default\n" +
		"  }"

	if detail := unmanagedSettingsDetail(t, diags); !strings.Contains(detail, expected) {
		t.Errorf("expected the detail to contain\n%s\ngot\n%s", expected, detail)
	}
}

func TestSettingsThatAreNotUnmanagedPassThePlan(t *testing.T) {
	resourceSchema := resourceSchemaOf(t, &tenantSettingsResource{})
	webhook := authenticatorEventsWebhookValue(t, types.BoolValue(true))
	emptyWhitelist := tftypes.NewValue(tftypes.Set{ElementType: tftypes.String}, []tftypes.Value{})

	testCases := map[string]struct {
		config map[string]tftypes.Value
		state  map[string]tftypes.Value
		plan   map[string]tftypes.Value
	}{
		"set in the configuration": {
			config: map[string]tftypes.Value{"token_duration_in_minutes": tftypes.NewValue(tftypes.Number, 30)},
			state:  map[string]tftypes.Value{"token_duration_in_minutes": tftypes.NewValue(tftypes.Number, 15)},
			plan:   map[string]tftypes.Value{"token_duration_in_minutes": tftypes.NewValue(tftypes.Number, 30)},
		},
		"blank on the tenant": {
			state: map[string]tftypes.Value{"ip_whitelist": emptyWhitelist},
			plan:  map[string]tftypes.Value{"ip_whitelist": emptyWhitelist},
		},
		"at the default": {
			state: map[string]tftypes.Value{"token_duration_in_minutes": tftypes.NewValue(tftypes.Number, 10)},
			plan:  map[string]tftypes.Value{"token_duration_in_minutes": tftypes.NewValue(tftypes.Number, 10)},
		},
		"inside a block the plan removes": {
			state: map[string]tftypes.Value{"authenticator_events_webhook_config": webhook},
		},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			diags := unmanagedSettingsDiagnostics(
				context.Background(),
				resourceSchema,
				resourceObject(t, resourceSchema, testCase.config),
				resourceObject(t, resourceSchema, testCase.state),
				resourceObject(t, resourceSchema, testCase.plan),
				authsignal.Defaults{"tokenDurationInMinutes": json.RawMessage(`10`)},
			)

			if diags.HasError() {
				t.Error(detailsOf(diags))
			}
		})
	}
}

func passkeyDefaults() authsignal.Defaults {
	return authsignal.Defaults{
		"isActive":                    json.RawMessage(`true`),
		"userVerificationRequirement": json.RawMessage(`"preferred"`),
		"authenticatorAttachment":     json.RawMessage(`"all-supported"`),
		"preventRemovingLastPasskey":  json.RawMessage(`false`),
	}
}

func authenticatorEventsWebhookValue(t *testing.T, includeCredentialPublicKey attr.Value) tftypes.Value {
	t.Helper()

	value, err := authenticatorEventsWebhook(types.StringValue("https://example.com/hook"), includeCredentialPublicKey).ToTerraformValue(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	return value
}

func unmanagedSettingsDetail(t *testing.T, diags diag.Diagnostics) string {
	t.Helper()

	if len(diags) != 1 || diags[0].Summary() != "Settings Missing From the Configuration" {
		t.Fatalf("expected one Settings Missing From the Configuration error, got %d:\n%s", len(diags), detailsOf(diags))
	}

	return diags[0].Detail()
}

// keptWhenLeftOut is written independently of the check, so a setting the check fails to reach shows up
// as a difference rather than being skipped by both.
func keptWhenLeftOut(prefix string, attributes map[string]schema.Attribute) []string {
	var paths []string

	for name, attribute := range attributes {
		path := prefix + name

		if single, ok := attribute.(schema.SingleNestedAttribute); ok {
			paths = append(paths, keptWhenLeftOut(path+".", single.Attributes)...)
			continue
		}

		if attribute.IsOptional() && attribute.IsComputed() && !hasSchemaDefault(attribute) {
			paths = append(paths, path)
		}

		if nested, ok := collectionNestedAttributes(attribute); ok {
			paths = append(paths, keptWhenLeftOut(path+".", nested)...)
		}
	}

	sort.Strings(paths)

	return paths
}

func collectionNestedAttributes(attribute schema.Attribute) (map[string]schema.Attribute, bool) {
	switch typed := attribute.(type) {
	case schema.ListNestedAttribute:
		return typed.NestedObject.Attributes, true
	case schema.SetNestedAttribute:
		return typed.NestedObject.Attributes, true
	case schema.MapNestedAttribute:
		return typed.NestedObject.Attributes, true
	default:
		return nil, false
	}
}

func hasSchemaDefault(attribute schema.Attribute) bool {
	field := reflect.ValueOf(attribute).FieldByName("Default")

	return field.IsValid() && !field.IsNil()
}

func splitSettingPath(path string) (string, string) {
	index := strings.LastIndex(path, ".")
	if index < 0 {
		return "", path
	}

	return path[:index], path[index+1:]
}

// sampleValue fills every attribute with a value that is neither blank nor any API default.
func sampleValue(t *testing.T, valueType tftypes.Type) tftypes.Value {
	t.Helper()

	switch typed := valueType.(type) {
	case tftypes.Object:
		attributes := make(map[string]tftypes.Value, len(typed.AttributeTypes))
		for name, attributeType := range typed.AttributeTypes {
			attributes[name] = sampleValue(t, attributeType)
		}

		return tftypes.NewValue(typed, attributes)
	case tftypes.List:
		return tftypes.NewValue(typed, []tftypes.Value{sampleValue(t, typed.ElementType)})
	case tftypes.Set:
		return tftypes.NewValue(typed, []tftypes.Value{sampleValue(t, typed.ElementType)})
	case tftypes.Map:
		return tftypes.NewValue(typed, map[string]tftypes.Value{"sample": sampleValue(t, typed.ElementType)})
	}

	switch {
	case valueType.Is(tftypes.String):
		return tftypes.NewValue(valueType, "sample")
	case valueType.Is(tftypes.Number):
		return tftypes.NewValue(valueType, 7)
	case valueType.Is(tftypes.Bool):
		return tftypes.NewValue(valueType, true)
	}

	t.Fatalf("no sample value for %s", valueType)

	return tftypes.Value{}
}
