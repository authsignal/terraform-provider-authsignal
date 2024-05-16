# Configuration-based authentication
provider "authsignal" {
  // For production systems please configure these as environment variables in your CI/CD process.
  host       = "https://api.authsignal.com/v1/management" // AUTHSIGNAL_HOST
  tenant_id  = "123"                                      // AUTHSIGNAL_TENANT_ID
  api_secret = "helloworld"                               // AUTHSIGNAL_API_SECRET
}

# These values can be found under the `Settings -> API keys` section of Authsignal's admin portal.
