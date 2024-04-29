# Configuration-based authentication
provider "authsignal" {
  // For production systems please configure these as environment variables in your CI/CD process.
  host       = "https://authsignal.com/not-a-real-endpoint"
  tenant_id  = "123"
  api_secret = "helloworld"
}
