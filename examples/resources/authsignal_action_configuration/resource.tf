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

# create a flow-based action from the file the admin portal's flow builder exports
resource "authsignal_action_configuration" "sign_in" {
  action_code                = "sign-in"
  action_type                = "MULTI_STEP_AUTHENTICATION"
  default_user_action_result = "CHALLENGE"
  flow                       = file("${path.module}/flow-sign-in.json")
}

# or write the flow inline: each RULE node defines the rules its arms reference
resource "authsignal_action_configuration" "high-risk-payment" {
  action_code                = "high-risk-payment"
  action_type                = "MULTI_STEP_AUTHENTICATION"
  default_user_action_result = "CHALLENGE"
  flow = jsonencode([
    {
      nodeId           = "rule-anonymous"
      nodeType         = "RULE"
      parentNodeIds    = []
      ruleChildNodeIds = [["anonymous-ip", "block"]]
      elseChildNodeId  = "verify"
      rules = [
        {
          ruleId     = "anonymous-ip"
          name       = "Anonymous IP"
          conditions = { and = [{ "==" = [{ var = "ip.isAnonymous" }, true] }] }
        }
      ]
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
  ])
}
