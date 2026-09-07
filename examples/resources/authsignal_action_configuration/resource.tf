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

# Load the JSON the Authsignal portal exports for a flow.
resource "authsignal_action_configuration" "sign_in" {
  action_code = "sign-in"
  action_type = "FLOW"
  flow        = file("${path.module}/flow-sign-in.json")
}

# Or construct the flow with jsonencode().
resource "authsignal_action_configuration" "high_risk_payment" {
  action_code = "high-risk-payment"
  action_type = "FLOW"
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
        name                 = "Confirm payment"
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
