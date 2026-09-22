package provider

import (
	"context"
	"encoding/json"
	"os"
	"regexp"
	"testing"

	"github.com/authsignal/authsignal-management-go/v6"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestSizeAdjustReadsAbsenceRatherThanZero(t *testing.T) {
	small := int64(125)

	testCases := []struct {
		name         string
		response     authsignal.TypefaceResponse
		expectedNull bool
		valueIsNull  bool
		expected     int64
	}{
		{
			name:         "never set",
			response:     authsignal.TypefaceResponse{},
			expectedNull: true,
			valueIsNull:  true,
		},
		{
			name:     "a size adjust on its own",
			response: authsignal.TypefaceResponse{SizeAdjust: &small},
			expected: 125,
		},
		{
			name:     "a size adjust alongside the faces",
			response: authsignal.TypefaceResponse{Faces: []authsignal.FontFaceResponse{{Url: "https://example.com/regular.woff2", Weight: "400"}}, SizeAdjust: &small},
			expected: 125,
		},
		{
			name:        "faces without a size adjust",
			response:    authsignal.TypefaceResponse{Faces: []authsignal.FontFaceResponse{{Url: "https://example.com/regular.woff2"}}},
			valueIsNull: true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			var typeface typefaceModel
			object := typeface.CreateObject(testCase.response)

			// A size adjust on its own is a typeface, not an empty one.
			if object.IsNull() != testCase.expectedNull {
				t.Fatalf("bad typeface object. expected null: %v. got null : %v", testCase.expectedNull, object.IsNull())
			}

			if typeface.SizeAdjust.IsNull() != testCase.valueIsNull {
				t.Fatalf("bad size_adjust. expected null: %v. got null : %v", testCase.valueIsNull, typeface.SizeAdjust.IsNull())
			}

			if !testCase.valueIsNull && typeface.SizeAdjust.ValueInt64() != testCase.expected {
				t.Fatalf("bad size_adjust. expected: %v. got : %v", testCase.expected, typeface.SizeAdjust.ValueInt64())
			}
		})
	}
}

func TestSizeAdjustIsOmittedFromACreateUntilItIsSet(t *testing.T) {
	testCases := []struct {
		name         string
		sizeAdjust   types.Int64
		expectedJson string
	}{
		{name: "unconfigured", sizeAdjust: types.Int64Null(), expectedJson: "{}"},
		{name: "set", sizeAdjust: types.Int64Value(125), expectedJson: "{\"sizeAdjust\":125}"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			typeface := buildAuthsignalTypefaceCreateObject(context.Background(), &fwresource.CreateResponse{}, typefaceModel{SizeAdjust: testCase.sizeAdjust})

			jsonBody, err := json.Marshal(typeface)
			if err != nil {
				t.Fatalf("failed to marshal json: %v", err)
			}

			if string(jsonBody) != testCase.expectedJson {
				t.Fatalf("bad json. expected: %v. got : %v", testCase.expectedJson, string(jsonBody))
			}
		})
	}
}

func TestSizeAdjustIsClearedWhenItLeavesTheConfiguration(t *testing.T) {
	testCases := []struct {
		name         string
		sizeAdjust   types.Int64
		expectedJson string
	}{
		{
			name:         "dropped from the configuration",
			sizeAdjust:   types.Int64Null(),
			expectedJson: "{\"faces\":null,\"fontUrl\":null,\"sizeAdjust\":null}",
		},
		{
			name:         "set",
			sizeAdjust:   types.Int64Value(125),
			expectedJson: "{\"faces\":null,\"fontUrl\":null,\"sizeAdjust\":125}",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			typeface := buildAuthsignalTypefaceUpdateObject(context.Background(), &fwresource.UpdateResponse{}, typefaceModel{SizeAdjust: testCase.sizeAdjust})

			jsonBody, err := json.Marshal(typeface)
			if err != nil {
				t.Fatalf("failed to marshal json: %v", err)
			}

			if string(jsonBody) != testCase.expectedJson {
				t.Fatalf("bad json. expected: %v. got : %v", testCase.expectedJson, string(jsonBody))
			}
		})
	}
}

func TestADeleteClearsTheSizeAdjustOnEveryTypeface(t *testing.T) {
	theme := authsignal.Theme{Typography: authsignal.SetValue(buildAuthsignalTypographyDeleteObject())}

	jsonBody, err := json.Marshal(theme)
	if err != nil {
		t.Fatalf("failed to marshal json: %v", err)
	}

	expectedJson := "{\"typography\":{" +
		"\"text\":{\"faces\":null,\"fontUrl\":null,\"sizeAdjust\":null}," +
		"\"display\":{\"faces\":null,\"fontUrl\":null,\"sizeAdjust\":null}," +
		"\"button\":{\"faces\":null,\"fontUrl\":null,\"sizeAdjust\":null}}}"

	if string(jsonBody) != expectedJson {
		t.Fatalf("bad json. expected: %v. got : %v", expectedJson, string(jsonBody))
	}
}

func TestEveryTypefaceTakesARangeCheckedSizeAdjust(t *testing.T) {
	response := &fwresource.SchemaResponse{}
	NewThemeResource().(*themeResource).Schema(context.Background(), fwresource.SchemaRequest{}, response)

	typography := response.Schema.Attributes["typography"].(schema.SingleNestedAttribute)

	for _, slot := range []string{"text", "display", "button"} {
		typeface := typography.Attributes[slot].(schema.SingleNestedAttribute)

		sizeAdjust, ok := typeface.Attributes["size_adjust"].(schema.Int64Attribute)
		if !ok {
			t.Fatalf("expected the %v typeface to take a size_adjust number", slot)
		}

		if len(sizeAdjust.Description) == 0 {
			t.Fatalf("expected the %v size_adjust to be documented", slot)
		}

		for _, testCase := range []struct {
			value    int64
			rejected bool
		}{
			{value: 66, rejected: true},
			{value: 67},
			{value: 100},
			{value: 150},
			{value: 151, rejected: true},
		} {
			request := validator.Int64Request{ConfigValue: types.Int64Value(testCase.value)}
			validationResponse := &validator.Int64Response{}

			for _, rule := range sizeAdjust.Validators {
				rule.ValidateInt64(context.Background(), request, validationResponse)
			}

			if validationResponse.Diagnostics.HasError() != testCase.rejected {
				t.Fatalf("bad %v size_adjust of %v. expected rejected: %v. got rejected : %v", slot, testCase.value, testCase.rejected, validationResponse.Diagnostics.HasError())
			}
		}
	}
}

// The range check runs while planning, so the test needs no tenant.
func TestAccThemeRejectsASizeAdjustOutsideTheRange(t *testing.T) {
	for _, value := range []string{"66", "151"} {
		resource.Test(t, resource.TestCase{
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config: `resource "authsignal_theme" "theme" {
  name = "Management-API-Testing"
  typography = {
    button = {
      size_adjust = ` + value + `
    }
  }
}`,
					ExpectError: regexp.MustCompile("size_adjust"),
				},
			},
		})
	}
}

// Needs AUTHSIGNAL_DESTRUCTIVE_THEME_ACC=1. The test leaves the tenant with an empty theme, because
// Terraform destroys what it owns and a theme delete clears every field. Seed the theme again
// afterwards, or TestAccThemeDataSource fails.
func TestAccThemeSizeAdjust(t *testing.T) {
	if os.Getenv("AUTHSIGNAL_DESTRUCTIVE_THEME_ACC") == "" {
		t.Skip("set AUTHSIGNAL_DESTRUCTIVE_THEME_ACC=1 to run. The test clears the tenant's theme when it finishes.")
	}

	bothSlots := `
  typography = {
    text = {
      font_url    = "https://example.com/text.woff2"
      size_adjust = 110
    }
    button = {
      font_url    = "https://example.com/button.woff2"
      size_adjust = 125
    }
  }`

	buttonOnly := `
  typography = {
    text = {
      font_url = "https://example.com/text.woff2"
    }
    button = {
      font_url    = "https://example.com/button.woff2"
      size_adjust = 125
    }
  }`

	noCorrection := `
  typography = {
    text = {
      font_url = "https://example.com/text.woff2"
    }
    button = {
      font_url = "https://example.com/button.woff2"
    }
  }`

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Themes cannot be created, only updated, so the state has to be imported before the
			// applies below have anything to update.
			{
				Config:             testAccThemeFixtureConfig("padding = 61", noCorrection),
				ResourceName:       "authsignal_theme.theme",
				ImportState:        true,
				ImportStateIdFunc:  importTheEmptyThemeId,
				ImportStatePersist: true,
			},
			{
				Config: testAccThemeFixtureConfig("padding = 61", noCorrection),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr("authsignal_theme.theme", "typography.text.size_adjust"),
					resource.TestCheckNoResourceAttr("authsignal_theme.theme", "typography.button.size_adjust"),
				),
			},
			{
				Config:   testAccThemeFixtureConfig("padding = 61", noCorrection),
				PlanOnly: true,
			},
			{
				Config: testAccThemeFixtureConfig("padding = 61", bothSlots),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("authsignal_theme.theme", "typography.text.size_adjust", "110"),
					resource.TestCheckResourceAttr("authsignal_theme.theme", "typography.button.size_adjust", "125"),
				),
			},
			{
				Config:   testAccThemeFixtureConfig("padding = 61", bothSlots),
				PlanOnly: true,
			},
			{
				Config: testAccThemeFixtureConfig("padding = 61", buttonOnly),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr("authsignal_theme.theme", "typography.text.size_adjust"),
					resource.TestCheckResourceAttr("authsignal_theme.theme", "typography.button.size_adjust", "125"),
				),
			},
			{
				Config:   testAccThemeFixtureConfig("padding = 61", buttonOnly),
				PlanOnly: true,
			},
			{
				Config: testAccThemeFixtureConfig("padding = 61", noCorrection),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr("authsignal_theme.theme", "typography.text.size_adjust"),
					resource.TestCheckNoResourceAttr("authsignal_theme.theme", "typography.button.size_adjust"),
				),
			},
			{
				Config:   testAccThemeFixtureConfig("padding = 61", noCorrection),
				PlanOnly: true,
			},
			{
				ResourceName:      "authsignal_theme.theme",
				ImportState:       true,
				ImportStateIdFunc: importTheEmptyThemeId,
				ImportStateVerify: true,
				// A theme is identified by its name. It has no id.
				ImportStateVerifyIdentifierAttribute: "name",
			},
		},
	})
}
