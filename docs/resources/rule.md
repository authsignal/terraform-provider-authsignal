---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "authsignal_rule Resource - terraform-provider-authsignal"
subcategory: ""
description: |-
  
---

# authsignal_rule (Resource)



## Example Usage

```terraform
# create a rule
resource "authsignal_rule" "test" {
  action_code = "test-rules"
  name        = "made-with-terraform"
  description = "hello world"
  priority    = 2
  type        = "CHALLENGE"
  is_active   = false
  verification_methods = [
    "AUTHENTICATOR_APP",
    "EMAIL_MAGIC_LINK",
    "EMAIL_OTP"
  ]
  prompt_to_enroll_verification_methods = [
    "PASSKEY"
  ]
  default_verification_method = "EMAIL_OTP"
  conditions = jsonencode({
    "and" : [
      {
        "==" : [
          {
            "var" : "ip.isAnonymous"
          },
          false
        ]
      }
    ]
  })
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `action_code` (String) The name of the action that users perform which you will track. (e.g 'login')
- `conditions` (String) The logical conditions to match tracked actions against. If the conditions are met then the rule's type will be returned in the track action response.
- `is_active` (Boolean) Toggles whether or not the rule is actively applied.
- `name` (String) A string used to name the rule.
- `priority` (Number) Determines the order which the rules are applied in, where 0 is applied first, 1 is applied second...
- `type` (String) The result that the rule should return when the conditions are met. (e.g. ALLOW, CHALLENGE)

### Optional

- `default_verification_method` (String) <tbd>
- `description` (String) A description of the rule.
- `prompt_to_enroll_verification_methods` (List of String) If this is set then users will be prompted to add a passkey after a challenge is completed.
- `verification_methods` (List of String) A list of permitted authenticators that can be used if the type of the rule is 'CHALLENGE'

### Read-Only

- `rule_id` (String) The ID of the rule.
- `tenant_id` (String) The ID of your tenant.

## Import

Import is supported using the following syntax:

```shell
# rules can be imported by specifying the action code.
terraform import authsignal_rule.test "test-action-code/a2d9670f-4028-424c-9f0f-1493ed9efc45"
```