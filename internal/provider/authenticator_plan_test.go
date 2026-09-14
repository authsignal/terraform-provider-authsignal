package provider

import (
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/go-version"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
)

func skipUnlessAcceptance(t *testing.T) {
	t.Helper()

	if os.Getenv("TF_ACC") == "" {
		t.Skip("set TF_ACC=1 to run the framework plan tests; they need a Terraform binary but no live tenant")
	}

	t.Setenv("AUTHSIGNAL_HOST", "http://127.0.0.1:1/v1/management")
	t.Setenv("AUTHSIGNAL_TENANT_ID", "tenant")
	t.Setenv("AUTHSIGNAL_API_SECRET", "secret")
}

// writeOnlyTerraform is the floor for write-only attributes. Below it Terraform rejects the schema
// outright. terraform-plugin-testing has no constant for 1.11 yet, so the version is spelled out.
var writeOnlyTerraform = tfversion.SkipBelow(version.Must(version.NewVersion("1.11.0")))

func TestPlanSmsConfigurationWithWriteOnlyCredentialsIsAccepted(t *testing.T) {
	skipUnlessAcceptance(t)

	resource.UnitTest(t, resource.TestCase{
		TerraformVersionChecks:   []tfversion.TerraformVersionCheck{writeOnlyTerraform},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "authsignal_sms_authenticator_configuration" "sms" {
  sms_provider = "TWILIO"

  twilio_credentials = {
    auth_token            = "secret-token"
    messaging_service_sid = "MG123"
    account_sid           = "AC123"
  }

  twilio_credentials_version = "v1"
}
`,
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestPlanSmsRejectsCredentialsForAnotherProvider(t *testing.T) {
	skipUnlessAcceptance(t)

	resource.UnitTest(t, resource.TestCase{
		TerraformVersionChecks:   []tfversion.TerraformVersionCheck{writeOnlyTerraform},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "authsignal_sms_authenticator_configuration" "sms" {
  sms_provider = "TWILIO"

  modica_group_credentials = {
    username = "modica-user"
    password = "modica-password"
  }
}
`,
				PlanOnly:    true,
				ExpectError: regexp.MustCompile(`Credentials Do Not Match the Configured Provider`),
			},
		},
	})
}

func TestPlanSmsRejectsAProviderTheContractDropped(t *testing.T) {
	skipUnlessAcceptance(t)

	resource.UnitTest(t, resource.TestCase{
		TerraformVersionChecks:   []tfversion.TerraformVersionCheck{writeOnlyTerraform},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "authsignal_sms_authenticator_configuration" "sms" {
  sms_provider = "WEBHOOK"
}
`,
				PlanOnly:    true,
				ExpectError: regexp.MustCompile(`Attribute sms_provider value must be one of`),
			},
		},
	})
}

func TestPlanFullyWriteOnlyBlocksAreAcceptedBySchemaValidation(t *testing.T) {
	skipUnlessAcceptance(t)

	resource.UnitTest(t, resource.TestCase{
		TerraformVersionChecks:   []tfversion.TerraformVersionCheck{writeOnlyTerraform},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "authsignal_sms_authenticator_configuration" "tnz" {
  sms_provider = "TNZ"

  tnz_credentials = {
    api_key = "tnz-key"
  }

  tnz_credentials_version = "v1"
}

resource "authsignal_push_authenticator_configuration" "push" {
  push_provider = "FIREBASE"

  fcm_credentials = {
    service_account_key = "{}"
  }

  fcm_credentials_version = "v1"
}
`,
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestPlanPasskeyRequiresAnOriginToBeUsable(t *testing.T) {
	skipUnlessAcceptance(t)

	resource.UnitTest(t, resource.TestCase{
		TerraformVersionChecks:   []tfversion.TerraformVersionCheck{writeOnlyTerraform},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "authsignal_passkey_authenticator_configuration" "passkey" {
  relying_party    = "example.com"
  expected_origins = []
}
`,
				PlanOnly:    true,
				ExpectError: regexp.MustCompile(`set must contain at least 1 elements`),
			},
		},
	})
}

func TestPlanPushWebhookProviderRequiresAnEndpoint(t *testing.T) {
	skipUnlessAcceptance(t)

	resource.UnitTest(t, resource.TestCase{
		TerraformVersionChecks:   []tfversion.TerraformVersionCheck{writeOnlyTerraform},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "authsignal_push_authenticator_configuration" "push" {
  push_provider = "WEBHOOK"
}
`,
				PlanOnly:    true,
				ExpectError: regexp.MustCompile(`Missing Webhook URL`),
			},
		},
	})
}

func TestPlanCredentialBlocksAreAcceptedAlongsideWriteOnlySecrets(t *testing.T) {
	skipUnlessAcceptance(t)

	resource.UnitTest(t, resource.TestCase{
		TerraformVersionChecks:   []tfversion.TerraformVersionCheck{writeOnlyTerraform},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "authsignal_sms_authenticator_configuration" "sms" {
  sms_provider = "TWILIO"

  twilio_credentials = {
    auth_token            = "secret-token"
    messaging_service_sid = "MG123"
    account_sid           = "AC123"
  }
}

resource "authsignal_push_authenticator_configuration" "push" {
  push_provider                  = "FIREBASE"
  credential_lifetime_in_minutes = 525600

  fcm_credentials = {
    service_account_key = "{}"
  }

  fcm_credentials_version = "1"
}
`,
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestPlanPushRejectsACredentialLifetimeBeyondAYear(t *testing.T) {
	skipUnlessAcceptance(t)

	resource.UnitTest(t, resource.TestCase{
		TerraformVersionChecks:   []tfversion.TerraformVersionCheck{writeOnlyTerraform},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "authsignal_push_authenticator_configuration" "push" {
  push_provider                  = "WEBHOOK"
  webhook_url                    = "https://example.com/hook"
  credential_lifetime_in_minutes = 525601
}
`,
				PlanOnly:    true,
				ExpectError: regexp.MustCompile(`Attribute credential_lifetime_in_minutes value must be between 0 and\s+525600`),
			},
		},
	})
}

func TestPlanConfigurationsMayOmitEveryOptionalAttribute(t *testing.T) {
	skipUnlessAcceptance(t)

	resource.UnitTest(t, resource.TestCase{
		TerraformVersionChecks:   []tfversion.TerraformVersionCheck{writeOnlyTerraform},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "authsignal_sms_authenticator_configuration" "sms" {
  sms_provider = "TWILIO"

  twilio_credentials = {
    auth_token            = "secret-token"
    messaging_service_sid = "MG123"
    account_sid           = "AC123"
  }
}

resource "authsignal_passkey_authenticator_configuration" "passkey" {
  relying_party    = "example.com"
  expected_origins = ["https://example.com"]
}
`,
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestPlanAnExplicitEmptyCollectionIsAcceptedWhereTheApiAllowsIt(t *testing.T) {
	skipUnlessAcceptance(t)

	resource.UnitTest(t, resource.TestCase{
		TerraformVersionChecks:   []tfversion.TerraformVersionCheck{writeOnlyTerraform},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "authsignal_sms_authenticator_configuration" "sms" {
  sms_provider                 = "TWILIO"
  sms_country_codes            = []
  allowed_custom_sms_variables = []

  twilio_credentials = {
    auth_token            = "secret-token"
    messaging_service_sid = "MG123"
    account_sid           = "AC123"
  }
}
`,
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestPlanSmsRejectsADefaultCountryCodeAlongsideAClearedRestriction(t *testing.T) {
	skipUnlessAcceptance(t)

	resource.UnitTest(t, resource.TestCase{
		TerraformVersionChecks:   []tfversion.TerraformVersionCheck{writeOnlyTerraform},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "authsignal_sms_authenticator_configuration" "sms" {
  sms_provider         = "TWILIO"
  sms_country_codes    = []
  default_country_code = "NZ"

  twilio_credentials = {
    auth_token            = "secret-token"
    messaging_service_sid = "MG123"
    account_sid           = "AC123"
  }
}
`,
				PlanOnly:    true,
				ExpectError: regexp.MustCompile(`Default Country Code Conflicts With an Empty Country\s+Restriction`),
			},
		},
	})
}

func TestPlanSmsRejectsADefaultCountryCodeOutsideTheRestriction(t *testing.T) {
	skipUnlessAcceptance(t)

	resource.UnitTest(t, resource.TestCase{
		TerraformVersionChecks:   []tfversion.TerraformVersionCheck{writeOnlyTerraform},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "authsignal_sms_authenticator_configuration" "sms" {
  sms_provider         = "TWILIO"
  sms_country_codes    = ["AU", "GB"]
  default_country_code = "NZ"

  twilio_credentials = {
    auth_token            = "secret-token"
    messaging_service_sid = "MG123"
    account_sid           = "AC123"
  }
}
`,
				PlanOnly:    true,
				ExpectError: regexp.MustCompile(`Default Country Code Is Not an Allowed Country`),
			},
		},
	})
}

func TestPlanWriteOnlyBlockWithoutItsVersionMarkerIsRejected(t *testing.T) {
	skipUnlessAcceptance(t)

	resource.UnitTest(t, resource.TestCase{
		TerraformVersionChecks:   []tfversion.TerraformVersionCheck{writeOnlyTerraform},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "authsignal_sms_authenticator_configuration" "sms" {
  sms_provider = "TNZ"

  tnz_credentials = {
    api_key = "tnz-secret"
  }
}
`,
				PlanOnly:    true,
				ExpectError: regexp.MustCompile(`Missing Version Marker for a Write-Only Credential\s+Block`),
			},
		},
	})
}

func TestPlanWriteOnlyBlockWithItsVersionMarkerIsAccepted(t *testing.T) {
	skipUnlessAcceptance(t)

	resource.UnitTest(t, resource.TestCase{
		TerraformVersionChecks:   []tfversion.TerraformVersionCheck{writeOnlyTerraform},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "authsignal_sms_authenticator_configuration" "sms" {
  sms_provider = "TNZ"

  tnz_credentials = {
    api_key = "tnz-secret"
  }

  tnz_credentials_version = "1"
}
`,
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestPlanSmsNarrowedRestrictionWithNoConfiguredDefaultIsAccepted(t *testing.T) {
	skipUnlessAcceptance(t)

	resource.UnitTest(t, resource.TestCase{
		TerraformVersionChecks:   []tfversion.TerraformVersionCheck{writeOnlyTerraform},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "authsignal_sms_authenticator_configuration" "sms" {
  sms_provider      = "TWILIO"
  sms_country_codes = ["AU", "GB"]

  twilio_credentials = {
    auth_token            = "secret-token"
    messaging_service_sid = "MG123"
    account_sid           = "AC123"
  }

  twilio_credentials_version = "1"
}
`,
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestPlanPushSwitchingAwayFromWebhookWithoutAnEndpointIsAccepted(t *testing.T) {
	skipUnlessAcceptance(t)

	resource.UnitTest(t, resource.TestCase{
		TerraformVersionChecks:   []tfversion.TerraformVersionCheck{writeOnlyTerraform},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "authsignal_push_authenticator_configuration" "push" {
  push_provider = "FIREBASE"

  fcm_credentials = {
    service_account_key = "{}"
  }

  fcm_credentials_version = "1"
}
`,
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}
