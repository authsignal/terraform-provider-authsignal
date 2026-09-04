package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

var _ validator.String = flowValidator{}

// flowValidator runs the same parse the publish uses and reports every broken invariant with the
// JSON path of the offending value, so a bad document is caught at plan time rather than by the API.
type flowValidator struct{}

func (v flowValidator) Description(_ context.Context) string {
	return "must be a JSON object holding an `actionNodes` array and the flat `rules` array its nodes reference"
}

func (v flowValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v flowValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	_, errs := parseFlow(req.ConfigValue.ValueString())

	for _, err := range errs {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid action flow",
			err.Error(),
		)
	}
}
