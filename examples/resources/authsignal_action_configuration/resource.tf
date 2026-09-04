# create an action code
resource "authsignal_action_configuration" "terraform-provider-test" {
  action_code                = "terraform-provider-test"
  default_user_action_result = "BLOCK"
  messaging_templates = jsonencode({
    "en" : {
      "defaultTemplate" : "hello world!"
    }
  })
  verification_methods                  = ["EMAIL_OTP", "PASSKEY"]
  default_verification_method           = "EMAIL_OTP"
  prompt_to_enroll_verification_methods = ["PASSKEY"]
}

# a flow is one JSON document: the `actionNodes` graph and the flat `rules` array its RULE nodes
# reference. Keeping it in a file is the easiest way to move a flow between tenants, and it is the
# same document the admin portal's flow builder exports and the API publishes.
resource "authsignal_action_configuration" "sign_in" {
  action_code                = "sign-in"
  action_type                = "MULTI_STEP_AUTHENTICATION"
  default_user_action_result = "CHALLENGE"
  flow                       = file("${path.module}/flow-sign-in.json")
}

# or write the flow inline. Two things to watch for in HCL:
#   - a json-logic operator that is not a valid identifier has to be quoted: `"==" = [...]`
#   - a literal `${` has to be escaped as `$${`, or Terraform reads it as an interpolation
resource "authsignal_action_configuration" "high-risk-payment" {
  action_code                = "high-risk-payment"
  action_type                = "MULTI_STEP_AUTHENTICATION"
  default_user_action_result = "CHALLENGE"
  flow = jsonencode({
    actionNodes = [
      {
        nodeId           = "rule-anonymous"
        nodeType         = "RULE"
        parentNodeIds    = []
        ruleChildNodeIds = [["anonymous-ip", "block"]]
        elseChildNodeId  = "verify"
      },
      {
        nodeId               = "verify"
        nodeType             = "VERIFICATION"
        parentNodeIds        = ["rule-anonymous"]
        name                 = "Confirm payment of $${amount}"
        methodConfigurations = { PASSKEY = { isEnabled = true }, EMAIL_OTP = { isEnabled = true } }
        childNodeId          = "complete"
      },
      {
        nodeId        = "block"
        nodeType      = "BLOCK"
        parentNodeIds = ["rule-anonymous"]
      },
      {
        nodeId        = "complete"
        nodeType      = "COMPLETE"
        parentNodeIds = ["verify"]
      }
    ]
    rules = [
      {
        ruleId     = "anonymous-ip"
        name       = "Anonymous IP"
        conditions = { and = [{ "==" = [{ var = "ip.isAnonymous" }, true] }] }
      }
    ]
  })
}
