# Retrieve rule by action code and rule id.
data "authsignal_rule" "test" {
  tenant_id   = "680137cd-1bb7-40b3-b3c3-6f0a665e9d94"
  action_code = "test-rules"
  rule_id     = "a2d9670f-4028-424c-9f0f-1493ed9efc45"
}
