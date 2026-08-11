package provider

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/authsignal/authsignal-management-go/v6"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// Where the exit control sits is theme-wide, so the API rejects it under dark mode.
func TestTheDarkModeContainerHasNoExitPosition(t *testing.T) {
	response := &resource.SchemaResponse{}
	NewThemeResource().(*themeResource).Schema(context.Background(), resource.SchemaRequest{}, response)

	container := response.Schema.Attributes["container"].(schema.SingleNestedAttribute)
	if _, found := container.Attributes["exit_position"]; !found {
		t.Fatalf("expected the theme container to take an exit position")
	}

	darkMode := response.Schema.Attributes["dark_mode"].(schema.SingleNestedAttribute)
	darkModeContainer := darkMode.Attributes["container"].(schema.SingleNestedAttribute)
	if _, found := darkModeContainer.Attributes["exit_position"]; found {
		t.Fatalf("expected the dark mode container to have no exit position")
	}
}

// A null clears the stored value, and the theme editor is where an exit position gets set, so an
// update Terraform was not given one has to leave the key out.
func TestExitPositionIsOmittedFromAnUpdateUntilItIsSet(t *testing.T) {
	testCases := []struct {
		name         string
		exitPosition types.String
		expectedJson string
	}{
		{
			name:         "unconfigured",
			exitPosition: types.StringNull(),
			expectedJson: "{\"container\":{\"contentAlignment\":null,\"padding\":null,\"logoAlignment\":null,\"logoPosition\":null,\"logoHeight\":null}}",
		},
		{
			name:         "configured",
			exitPosition: types.StringValue("bottom"),
			expectedJson: "{\"container\":{\"contentAlignment\":null,\"padding\":null,\"logoAlignment\":null,\"logoPosition\":null,\"logoHeight\":null,\"exitPosition\":\"bottom\"}}",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			theme := authsignal.Theme{
				Container: authsignal.SetValue(buildAuthsignalContainerUpdateObject(containerModel{ExitPosition: testCase.exitPosition})),
			}

			jsonBody, err := json.Marshal(theme)
			if err != nil {
				t.Fatalf("failed to marshal json: %v", err)
			}

			if string(jsonBody) != testCase.expectedJson {
				t.Fatalf("bad json. expected: %v. got : %v", testCase.expectedJson, string(jsonBody))
			}
		})
	}
}

// Without a container block the plan has to carry the stored exit position, or the apply comes back
// with a value the plan did not have. The rest of the container still clears.
func TestAContainerLeavingTheConfigurationKeepsOnlyItsExitPosition(t *testing.T) {
	attributeTypes := containerModel{}.AttributeTypes()

	testCases := []struct {
		name                 string
		state                containerModel
		stateIsNull          bool
		expectedNull         bool
		expectedExitPosition string
	}{
		{name: "nothing in state", stateIsNull: true, expectedNull: true},
		{
			name:         "state without an exit position",
			state:        containerModel{Padding: types.Int64Value(61)},
			expectedNull: true,
		},
		{
			name:                 "state with an exit position",
			state:                containerModel{Padding: types.Int64Value(61), ExitPosition: types.StringValue("bottom")},
			expectedExitPosition: "bottom",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			ctx := context.Background()

			state := types.ObjectNull(attributeTypes)
			if !testCase.stateIsNull {
				object, diags := types.ObjectValue(attributeTypes, testCase.state.AttributeValues())
				if diags.HasError() {
					t.Fatalf("failed to build the state object: %v", diags)
				}
				state = object
			}

			request := planmodifier.ObjectRequest{
				ConfigValue: types.ObjectNull(attributeTypes),
				StateValue:  state,
				PlanValue:   state,
			}
			response := &planmodifier.ObjectResponse{PlanValue: request.PlanValue}

			containerKeepsExitPosition{}.PlanModifyObject(ctx, request, response)

			if response.PlanValue.IsNull() != testCase.expectedNull {
				t.Fatalf("bad container plan. expected null: %v. got null : %v", testCase.expectedNull, response.PlanValue.IsNull())
			}

			if testCase.expectedNull {
				return
			}

			var planned containerModel
			if diags := response.PlanValue.As(ctx, &planned, basetypes.ObjectAsOptions{}); diags.HasError() {
				t.Fatalf("failed to read the planned container: %v", diags)
			}

			if planned.ExitPosition.ValueString() != testCase.expectedExitPosition {
				t.Fatalf("bad exit position. expected: %v. got : %v", testCase.expectedExitPosition, planned.ExitPosition.ValueString())
			}

			if !planned.Padding.IsNull() {
				t.Fatalf("bad padding. expected: unset. got : %v", planned.Padding.ValueInt64())
			}
		})
	}
}

// A container block the configuration does set is Terraform's, untouched by the modifier.
func TestAConfiguredContainerIsPlannedAsWritten(t *testing.T) {
	attributeTypes := containerModel{}.AttributeTypes()

	configured, diags := types.ObjectValue(attributeTypes, containerModel{Padding: types.Int64Value(61)}.AttributeValues())
	if diags.HasError() {
		t.Fatalf("failed to build the config object: %v", diags)
	}

	request := planmodifier.ObjectRequest{
		ConfigValue: configured,
		StateValue:  types.ObjectNull(attributeTypes),
		PlanValue:   configured,
	}
	response := &planmodifier.ObjectResponse{PlanValue: request.PlanValue}

	containerKeepsExitPosition{}.PlanModifyObject(context.Background(), request, response)

	if !response.PlanValue.Equal(configured) {
		t.Fatalf("bad container plan. expected: %v. got : %v", configured, response.PlanValue)
	}
}
