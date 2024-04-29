# Retrieve rule by action code and rule id.
data "authsignal_rule" "test" {
  action_code = "test-rules"
  rule_id     = "a2d9670f-4028-424c-9f0f-1493ed9efc45"
}
