package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestDestroyingTenantSettingsFailsAndNamesWhatTheTenantKeeps(t *testing.T) {
	skipUnlessAcceptance(t)

	_, stub := newStubAPI(t, map[string][]stubResponse{
		"GET /tenant": okResponse(`{"tokenDurationInMinutes":30,"ipWhitelist":[],` +
			`"logEventsWebhookConfig":{"endpointUrl":"https://example.com/logs"},` +
			`"defaults":{"tokenDurationInMinutes":10}}`),
	})
	t.Setenv("AUTHSIGNAL_HOST", stub.url)

	config := `
resource "authsignal_tenant_settings" "tenant" {
  token_duration_in_minutes = 30
  ip_whitelist              = []

  log_events_webhook_config = {
    endpoint_url = "https://example.com/logs"
  }
}
`

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:             config,
				ResourceName:       "authsignal_tenant_settings.tenant",
				ImportState:        true,
				ImportStateId:      "tenant",
				ImportStatePersist: true,
			},
			{
				Config:  config,
				Destroy: true,
				ExpectError: regexp.MustCompile(`Can't Destroy authsignal_tenant_settings[\s\S]*` +
					`removed\s+\{\s+from\s+=\s+authsignal_tenant_settings\.<name>\s+lifecycle\s+\{\s+destroy\s+=\s+false\s+\}\s+\}[\s\S]*` +
					`token_duration_in_minutes\s+=\s+30\s+#\s+default:\s+10\s+` +
					`log_events_webhook_config\s+=\s+\{\s+endpoint_url\s+=\s+"https://example.com/logs"\s+#\s+no\s+default\s+\}`),
			},
			{
				Config: testAccRemovedConfig("authsignal_tenant_settings.tenant"),
			},
		},
	})

	stub.assertNotCalled(t, "PATCH /tenant")
}

// A theme destroy used to clear every field of the tenant's theme.
func TestDestroyingTheThemeFailsAndLeavesItUntouched(t *testing.T) {
	skipUnlessAcceptance(t)

	_, stub := newStubAPI(t, map[string][]stubResponse{
		"GET /theme": okResponse(`{"name":"Tenant","primaryColor":"#121282"}`),
	})
	t.Setenv("AUTHSIGNAL_HOST", stub.url)

	config := `
resource "authsignal_theme" "theme" {
  name          = "Tenant"
  primary_color = "#121282"
}
`

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:             config,
				ResourceName:       "authsignal_theme.theme",
				ImportState:        true,
				ImportStateIdFunc:  importTheEmptyThemeId,
				ImportStatePersist: true,
			},
			{
				Config:      config,
				Destroy:     true,
				ExpectError: regexp.MustCompile(`Can't Destroy authsignal_theme[\s\S]*destroy\s+=\s+false[\s\S]*The tenant keeps its current theme\.`),
			},
			{
				Config: testAccRemovedConfig("authsignal_theme.theme"),
			},
		},
	})

	stub.assertNotCalled(t, "PATCH /theme")
}

func testAccRemovedConfig(address string) string {
	return `
removed {
  from = ` + address + `

  lifecycle {
    destroy = false
  }
}
`
}
