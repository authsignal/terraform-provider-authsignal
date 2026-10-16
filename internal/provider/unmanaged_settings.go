package provider

import (
	"context"
	"encoding/json"
	"reflect"

	"github.com/authsignal/authsignal-management-go/v6"
	"github.com/hashicorp/terraform-plugin-framework/diag"
)

const defaultsPrivateStateKey = "defaults"

func storeDefaults(ctx context.Context, private interface {
	SetKey(context.Context, string, []byte) diag.Diagnostics
}, defaults authsignal.Defaults) diag.Diagnostics {
	// The framework always sets Private, but a test that calls a resource directly leaves it nil.
	if reflect.ValueOf(private).IsNil() {
		return nil
	}

	encoded, err := json.Marshal(defaults)
	if err != nil {
		var diags diag.Diagnostics
		diags.AddError("Unable to Store the Management API's Defaults", err.Error())
		return diags
	}

	return private.SetKey(ctx, defaultsPrivateStateKey, encoded)
}
