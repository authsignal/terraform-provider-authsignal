package provider

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/authsignal/authsignal-management-go/v6"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// An axis the tenant never set comes back absent, and `0` is a padding they chose. Reading one as
// the other would either lose a zero or invent an override, so the response carries pointers.
func TestAxisPaddingReadsZeroAsAValueRatherThanAsAbsence(t *testing.T) {
	none := int64(0)
	narrow := int64(24)
	tall := int64(48)
	slim := int64(8)

	testCases := []struct {
		name               string
		response           authsignal.ContainerResponse
		expectedNull       bool
		horizontalIsNull   bool
		verticalIsNull     bool
		expectedHorizontal int64
		expectedVertical   int64
	}{
		{
			name:             "never set",
			response:         authsignal.ContainerResponse{},
			expectedNull:     true,
			horizontalIsNull: true,
			verticalIsNull:   true,
		},
		{
			name:               "zero on both axes",
			response:           authsignal.ContainerResponse{PaddingHorizontal: &none, PaddingVertical: &none},
			expectedHorizontal: 0,
			expectedVertical:   0,
		},
		{
			name:               "a value on both axes",
			response:           authsignal.ContainerResponse{PaddingHorizontal: &narrow, PaddingVertical: &tall},
			expectedHorizontal: 24,
			expectedVertical:   48,
		},
		{
			name:               "one axis only",
			response:           authsignal.ContainerResponse{PaddingHorizontal: &narrow},
			expectedHorizontal: 24,
			verticalIsNull:     true,
		},
		{
			name:             "an axis alongside a universal padding",
			response:         authsignal.ContainerResponse{Padding: 32, PaddingVertical: &slim},
			horizontalIsNull: true,
			expectedVertical: 8,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			var container containerModel
			object := container.CreateObject(testCase.response)

			// An axis padding on its own is a container, not an empty one.
			if object.IsNull() != testCase.expectedNull {
				t.Fatalf("bad container object. expected null: %v. got null : %v", testCase.expectedNull, object.IsNull())
			}

			if container.PaddingHorizontal.IsNull() != testCase.horizontalIsNull {
				t.Fatalf("bad padding_horizontal. expected null: %v. got null : %v", testCase.horizontalIsNull, container.PaddingHorizontal.IsNull())
			}

			if container.PaddingVertical.IsNull() != testCase.verticalIsNull {
				t.Fatalf("bad padding_vertical. expected null: %v. got null : %v", testCase.verticalIsNull, container.PaddingVertical.IsNull())
			}

			if !testCase.horizontalIsNull && container.PaddingHorizontal.ValueInt64() != testCase.expectedHorizontal {
				t.Fatalf("bad padding_horizontal. expected: %v. got : %v", testCase.expectedHorizontal, container.PaddingHorizontal.ValueInt64())
			}

			if !testCase.verticalIsNull && container.PaddingVertical.ValueInt64() != testCase.expectedVertical {
				t.Fatalf("bad padding_vertical. expected: %v. got : %v", testCase.expectedVertical, container.PaddingVertical.ValueInt64())
			}
		})
	}
}

// Terraform owns the axis paddings, so an update sends a zero as a zero and a dropped axis as the
// null that clears the stored override.
func TestAxisPaddingSendsZeroRatherThanClearingIt(t *testing.T) {
	testCases := []struct {
		name         string
		horizontal   types.Int64
		vertical     types.Int64
		expectedJson string
	}{
		{
			name:         "unconfigured",
			horizontal:   types.Int64Null(),
			vertical:     types.Int64Null(),
			expectedJson: "{\"container\":{\"contentAlignment\":null,\"padding\":null,\"paddingHorizontal\":null,\"paddingVertical\":null,\"logoAlignment\":null,\"logoPosition\":null,\"logoHeight\":null}}",
		},
		{
			name:         "zero",
			horizontal:   types.Int64Value(0),
			vertical:     types.Int64Value(0),
			expectedJson: "{\"container\":{\"contentAlignment\":null,\"padding\":null,\"paddingHorizontal\":0,\"paddingVertical\":0,\"logoAlignment\":null,\"logoPosition\":null,\"logoHeight\":null}}",
		},
		{
			name:         "a value on both axes",
			horizontal:   types.Int64Value(24),
			vertical:     types.Int64Value(48),
			expectedJson: "{\"container\":{\"contentAlignment\":null,\"padding\":null,\"paddingHorizontal\":24,\"paddingVertical\":48,\"logoAlignment\":null,\"logoPosition\":null,\"logoHeight\":null}}",
		},
		{
			name:         "one axis dropped from the configuration",
			horizontal:   types.Int64Value(24),
			vertical:     types.Int64Null(),
			expectedJson: "{\"container\":{\"contentAlignment\":null,\"padding\":null,\"paddingHorizontal\":24,\"paddingVertical\":null,\"logoAlignment\":null,\"logoPosition\":null,\"logoHeight\":null}}",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			container := containerModel{PaddingHorizontal: testCase.horizontal, PaddingVertical: testCase.vertical}
			theme := authsignal.Theme{
				Container: authsignal.SetValue(buildAuthsignalContainerUpdateObject(container)),
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

// A create leaves out what it was not given rather than nulling it.
func TestAxisPaddingIsOmittedFromACreateUntilItIsSet(t *testing.T) {
	testCases := []struct {
		name         string
		horizontal   types.Int64
		vertical     types.Int64
		expectedJson string
	}{
		{
			name:         "unconfigured",
			horizontal:   types.Int64Null(),
			vertical:     types.Int64Null(),
			expectedJson: "{\"container\":{}}",
		},
		{
			name:         "zero",
			horizontal:   types.Int64Value(0),
			vertical:     types.Int64Value(0),
			expectedJson: "{\"container\":{\"paddingHorizontal\":0,\"paddingVertical\":0}}",
		},
		{
			name:         "one axis only",
			horizontal:   types.Int64Value(24),
			vertical:     types.Int64Null(),
			expectedJson: "{\"container\":{\"paddingHorizontal\":24}}",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			container := containerModel{PaddingHorizontal: testCase.horizontal, PaddingVertical: testCase.vertical}
			theme := authsignal.Theme{
				Container: authsignal.SetValue(buildAuthsignalContainerCreateObject(container)),
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

// A delete gives the theme back to the platform default, both axes included.
func TestADeleteClearsBothAxisPaddings(t *testing.T) {
	theme := authsignal.Theme{
		Container: authsignal.SetValue(buildAuthsignalContainerDeleteObject(containerModel{PaddingHorizontal: types.Int64Value(24), PaddingVertical: types.Int64Value(48)})),
	}

	jsonBody, err := json.Marshal(theme)
	if err != nil {
		t.Fatalf("failed to marshal json: %v", err)
	}

	expectedJson := "{\"container\":{\"contentAlignment\":null,\"padding\":null,\"paddingHorizontal\":null,\"paddingVertical\":null,\"logoAlignment\":null,\"logoPosition\":null,\"logoHeight\":null}}"
	if string(jsonBody) != expectedJson {
		t.Fatalf("bad json. expected: %v. got : %v", expectedJson, string(jsonBody))
	}
}

// A per-axis override is theme-wide, so the API keeps the axis paddings off dark mode.
func TestTheDarkModeContainerHasNoAxisPadding(t *testing.T) {
	response := &resource.SchemaResponse{}
	NewThemeResource().(*themeResource).Schema(context.Background(), resource.SchemaRequest{}, response)

	container := response.Schema.Attributes["container"].(schema.SingleNestedAttribute)
	for _, name := range []string{"padding", "padding_horizontal", "padding_vertical"} {
		if _, found := container.Attributes[name]; !found {
			t.Fatalf("expected the theme container to take a %v", name)
		}
	}

	darkMode := response.Schema.Attributes["dark_mode"].(schema.SingleNestedAttribute)
	darkModeContainer := darkMode.Attributes["container"].(schema.SingleNestedAttribute)
	if _, found := darkModeContainer.Attributes["padding"]; !found {
		t.Fatalf("expected the dark mode container to take a padding")
	}

	for _, name := range []string{"padding_horizontal", "padding_vertical"} {
		if _, found := darkModeContainer.Attributes[name]; found {
			t.Fatalf("expected the dark mode container to have no %v", name)
		}
	}
}

// The API rejects a padding outside its range, so the configuration is turned away first.
func TestEveryContainerPaddingIsRangeChecked(t *testing.T) {
	response := &resource.SchemaResponse{}
	NewThemeResource().(*themeResource).Schema(context.Background(), resource.SchemaRequest{}, response)

	container := response.Schema.Attributes["container"].(schema.SingleNestedAttribute)
	darkMode := response.Schema.Attributes["dark_mode"].(schema.SingleNestedAttribute)
	darkModeContainer := darkMode.Attributes["container"].(schema.SingleNestedAttribute)

	paddings := map[string]schema.Attribute{
		"padding":            container.Attributes["padding"],
		"padding_horizontal": container.Attributes["padding_horizontal"],
		"padding_vertical":   container.Attributes["padding_vertical"],
		"dark_mode padding":  darkModeContainer.Attributes["padding"],
	}

	for name, attribute := range paddings {
		padding, ok := attribute.(schema.Int64Attribute)
		if !ok {
			t.Fatalf("expected %v to be a number", name)
		}

		if len(padding.Validators) == 0 {
			t.Fatalf("expected %v to be range checked", name)
		}

		if len(padding.Description) == 0 {
			t.Fatalf("expected %v to be documented", name)
		}
	}
}
