package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// Import seeds an empty state that the follow-up Read fills in. The empty state has to carry the
// schema's attribute types, otherwise Terraform cannot convert the nested webhook objects back.
func TestTenantSettingsImportSeedsTypedEmptyState(t *testing.T) {
	res := &tenantSettingsResource{}
	resourceSchema := resourceSchemaOf(t, res)
	schemaType := resourceSchema.Type().TerraformType(context.Background())

	resp := &resource.ImportStateResponse{
		State: tfsdk.State{Schema: resourceSchema, Raw: tftypes.NewValue(schemaType, nil)},
	}
	res.ImportState(context.Background(), resource.ImportStateRequest{ID: ""}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatal(detailsOf(resp.Diagnostics))
	}

	if !resp.State.Raw.Type().Equal(schemaType) {
		t.Fatalf("imported state has type %s, want the schema type %s", resp.State.Raw.Type(), schemaType)
	}

	var model tenantSettingsResourceModel
	if diags := resp.State.Get(context.Background(), &model); diags.HasError() {
		t.Fatalf("imported state cannot be read back: %s", detailsOf(diags))
	}

	if !model.AuthenticatorEventsWebhookConfig.IsNull() || !model.LogEventsWebhookConfig.IsNull() ||
		!model.TokenDurationInMinutes.IsNull() || !model.IpWhitelist.IsNull() {
		t.Errorf("expected every attribute to be null after import, got %+v", model)
	}
}
