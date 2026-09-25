# A tenant has one set of settings, so the provider ignores the import ID and "" works here.
# In an import block, use a non-empty placeholder instead, such as id = "tenant_settings", because Terraform rejects an empty id.
terraform import authsignal_tenant_settings.tenant_settings ""
