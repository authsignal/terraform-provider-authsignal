package provider

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

const emailOtpAddress = "authsignal_email_otp_authenticator_configuration.email_otp"

// The first plan after an import compares the configuration's credentials with the tenant's, so a
// mistyped value or a member the configuration leaves out shows as a change.
func TestPlanAfterImportComparesTheCredentialBlockWithTheTenant(t *testing.T) {
	skipUnlessAcceptance(t)

	_, stub := newStubAPI(t, map[string][]stubResponse{
		emailOtpGetRoute: okResponse(`{"authenticatorId":"auth_email","isActive":true,"verificationMethod":"EMAIL_OTP",` +
			`"emailProvider":"SENDGRID","sendgridEmailCredentials":{"templateId":"d-real","fromEmail":"noreply@example.com",` +
			`"fromName":"Support team"},"defaults":{"isActive":true}}`),
	})
	t.Setenv("AUTHSIGNAL_HOST", stub.url)

	config := func(templateId string, fromName string) string {
		return fmt.Sprintf(`
resource "authsignal_email_otp_authenticator_configuration" "email_otp" {
  email_provider = "SENDGRID"

  sendgrid_email_credentials = {
    api_key     = "SG.test"
    template_id = %q
    from_email  = "noreply@example.com"
%s  }
}
`, templateId, fromName)
	}

	matching := config("d-real", "    from_name   = \"Support team\"\n")

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:             matching,
				ResourceName:       emailOtpAddress,
				ImportState:        true,
				ImportStateId:      emailOtpAuthenticatorSlug,
				ImportStatePersist: true,
			},
			{
				Config:   matching,
				PlanOnly: true,
			},
			{
				Config:             config("d-typo", "    from_name   = \"Support team\"\n"),
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{
						credentialMemberChange{block: "sendgrid_email_credentials", member: "template_id", before: "d-real", after: "d-typo"},
					},
				},
			},
			{
				Config:             config("d-real", ""),
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{
						credentialMemberChange{block: "sendgrid_email_credentials", member: "from_name", before: "Support team", after: nil},
					},
				},
			},
		},
	})

	stub.assertNotCalled(t, emailOtpPatchRoute)
}

type credentialMemberChange struct {
	block  string
	member string
	before any
	after  any
}

func (c credentialMemberChange) CheckPlan(_ context.Context, req plancheck.CheckPlanRequest, resp *plancheck.CheckPlanResponse) {
	for _, change := range req.Plan.ResourceChanges {
		if change.Address != emailOtpAddress {
			continue
		}

		before := memberValue(change.Change.Before, c.block, c.member)
		after := memberValue(change.Change.After, c.block, c.member)

		if before != c.before || after != c.after {
			resp.Error = fmt.Errorf("%s.%s plans %v -> %v, want %v -> %v", c.block, c.member, before, after, c.before, c.after)
		}

		return
	}

	resp.Error = fmt.Errorf("%s isn't in the plan", emailOtpAddress)
}

func memberValue(object any, block string, member string) any {
	attributes, _ := object.(map[string]any)
	members, _ := attributes[block].(map[string]any)

	return members[member]
}
