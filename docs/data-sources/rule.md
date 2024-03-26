---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "authsignal_rule Data Source - terraform-provider-authsignal"
subcategory: ""
description: |-
  
---

# authsignal_rule (Data Source)



## Example Usage

```terraform
# Retrieve rule by action code and rule id.
data "authsignal_rule" "test" {
  tenant_id   = "680137cd-1bb7-40b3-b3c3-6f0a665e9d94"
  action_code = "test-rules"
  rule_id     = "a2d9670f-4028-424c-9f0f-1493ed9efc45"
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `action_code` (String) The name of the action that users perform which you will track. (e.g 'login')
- `rule_id` (String) The ID of the rule. This can be obtained from the Authsignal portal.
- `tenant_id` (String) The ID of your tenant.

### Read-Only

- `conditions` (String) The logical conditions to match tracked actions against. If the conditions are met then the rule's type will be returned in the track action response.
- `default_verification_method` (String) The default verification method that users should be prompted with.
- `description` (String) A description of the rule.
- `is_active` (Boolean) Toggles whether or not the rule is actively applied.
- `name` (String) A string used to name the rule.
- `priority` (Number) Determines the order which the rules are applied in, where 0 is applied first, 1 is applied second...
- `prompt_to_enroll_verification_methods` (List of String) If this is set then users will be prompted to add a passkey after a challenge is completed.
- `type` (String) The result that the rule should return when the conditions are met. (e.g. ALLOW, CHALLENGE)
- `verification_methods` (List of String) A list of permitted authenticators that can be used if the type of the rule is 'CHALLENGE'