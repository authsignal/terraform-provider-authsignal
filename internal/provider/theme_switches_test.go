package provider

import (
	"encoding/json"
	"testing"

	"github.com/authsignal/authsignal-management-go/v5"
	"github.com/hashicorp/terraform-plugin-framework/types"
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
