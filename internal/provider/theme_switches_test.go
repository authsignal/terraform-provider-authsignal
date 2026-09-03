package provider

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/authsignal/authsignal-management-go/v6"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

func TestSwitchesReadFalseAsAValueRatherThanAsAbsence(t *testing.T) {
	off := false
	on := true

	testCases := []struct {
		name          string
		response      authsignal.LinksResponse
		expectedNull  bool
		expectedValue bool
	}{
		{name: "never set", response: authsignal.LinksResponse{}, expectedNull: true},
		{name: "off", response: authsignal.LinksResponse{Underline: &off}, expectedValue: false},
		{name: "on", response: authsignal.LinksResponse{Underline: &on}, expectedValue: true},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			var links linksModel
			object := links.CreateObject(testCase.response)

			if object.IsNull() != testCase.expectedNull {
				t.Fatalf("bad links object. expected null: %v. got null : %v", testCase.expectedNull, object.IsNull())
			}

			if testCase.expectedNull {
				return
			}

			if links.Underline.ValueBool() != testCase.expectedValue {
				t.Fatalf("bad underline. expected: %v. got : %v", testCase.expectedValue, links.Underline.ValueBool())
			}
		})
	}
}

func TestSwitchesSendFalseRatherThanClearingIt(t *testing.T) {
	testCases := []struct {
		name         string
		underline    types.Bool
		enabled      types.Bool
		expectedJson string
	}{
		{
			name:         "unconfigured",
			underline:    types.BoolNull(),
			enabled:      types.BoolNull(),
			expectedJson: "{\"links\":{\"underline\":null},\"shadows\":{\"enabled\":null}}",
		},
		{
			name:         "off",
			underline:    types.BoolValue(false),
			enabled:      types.BoolValue(false),
			expectedJson: "{\"links\":{\"underline\":false},\"shadows\":{\"enabled\":false}}",
		},
		{
			name:         "on",
			underline:    types.BoolValue(true),
			enabled:      types.BoolValue(true),
			expectedJson: "{\"links\":{\"underline\":true},\"shadows\":{\"enabled\":true}}",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			theme := authsignal.Theme{
				Links:   authsignal.SetValue(buildAuthsignalLinksUpdateObject(linksModel{Underline: testCase.underline})),
				Shadows: authsignal.SetValue(buildAuthsignalShadowsUpdateObject(shadowsModel{Enabled: testCase.enabled})),
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

// The shadow switch takes a value per colour mode. The link underline does not.
func TestTheShadowSwitchIsTakenPerColourMode(t *testing.T) {
	response := &resource.SchemaResponse{}
	NewThemeResource().(*themeResource).Schema(context.Background(), resource.SchemaRequest{}, response)

	darkMode := response.Schema.Attributes["dark_mode"].(schema.SingleNestedAttribute)

	darkModeShadows, found := darkMode.Attributes["shadows"].(schema.SingleNestedAttribute)
	if !found {
		t.Fatalf("expected dark mode to take a shadow switch")
	}

	if _, found := darkModeShadows.Attributes["enabled"]; !found {
		t.Fatalf("expected the dark mode shadows to take an enabled switch")
	}

	if _, found := darkMode.Attributes["links"]; found {
		t.Fatalf("expected dark mode to have no links")
	}
}

// A dark mode carrying nothing but a shadow switch is still a dark mode, or the switch reads back as
// unset and leaves a permanent diff.
func TestADarkModeShadowSwitchIsReadBackOnItsOwn(t *testing.T) {
	off := false

	var darkMode darkModeModel
	object := darkMode.CreateObject(authsignal.DarkModeResponse{Shadows: authsignal.ShadowsResponse{Enabled: &off}})

	if object.IsNull() {
		t.Fatalf("expected a dark mode object carrying the shadow switch")
	}

	var shadows shadowsModel
	diags := darkMode.Shadows.As(context.Background(), &shadows, basetypes.ObjectAsOptions{})
	if diags.HasError() {
		t.Fatalf("failed to read the dark mode shadows: %v", diags)
	}

	if shadows.Enabled.IsNull() || shadows.Enabled.ValueBool() {
		t.Fatalf("bad dark mode shadows. expected: off. got : %v", shadows.Enabled)
	}
}
