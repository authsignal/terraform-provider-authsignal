package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func configureWithHost(t *testing.T, host string) *provider.ConfigureResponse {
	t.Helper()

	ctx := context.Background()
	p := &authsignalProvider{}

	schemaResp := &provider.SchemaResponse{}
	p.Schema(ctx, provider.SchemaRequest{}, schemaResp)

	config := tfsdk.Config{
		Schema: schemaResp.Schema,
		Raw: tftypes.NewValue(schemaResp.Schema.Type().TerraformType(ctx), map[string]tftypes.Value{
			"host":       tftypes.NewValue(tftypes.String, host),
			"tenant_id":  tftypes.NewValue(tftypes.String, "tenant"),
			"api_secret": tftypes.NewValue(tftypes.String, "secret"),
		}),
	}

	resp := &provider.ConfigureResponse{}
	p.Configure(ctx, provider.ConfigureRequest{Config: config}, resp)

	return resp
}

// The API secret goes out with the first request, so a host that would carry it in clear
// text has to be refused before a client exists.
func TestConfigureRefusesAHostThatWouldSendTheSecretInClearText(t *testing.T) {
	testCases := []struct {
		host      string
		wantError bool
	}{
		{host: "https://api.authsignal.com/v1/management", wantError: false},
		{host: "HTTPS://api.authsignal.com/v1/management", wantError: false},
		{host: "http://localhost:8080", wantError: false},
		{host: "http://127.0.0.1:52345", wantError: false},
		{host: "http://[::1]:8080", wantError: false},
		{host: "http://api.authsignal.com/v1/management", wantError: true},
		{host: "api.authsignal.com/v1/management", wantError: true},
		{host: "https://", wantError: true},
		{host: "http://localhost.example.com", wantError: true},
		{host: "http://127.0.0.1.example.com", wantError: true},
	}

	for _, testCase := range testCases {
		t.Run(testCase.host, func(t *testing.T) {
			resp := configureWithHost(t, testCase.host)

			if resp.Diagnostics.HasError() != testCase.wantError {
				t.Fatalf("expected an error=%v, got %s", testCase.wantError, detailsOf(resp.Diagnostics))
			}

			if testCase.wantError && resp.ResourceData != nil {
				t.Error("no client may be configured for a refused host")
			}
		})
	}
}
