package provider

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

const (
	smsGetRoute    = "GET /authenticator-configurations/sms"
	smsCreateRoute = "POST /authenticator-configurations/sms"
	smsPatchRoute  = "PATCH /authenticator-configurations/sms"
	smsDeleteRoute = "DELETE /authenticator-configurations/sms"

	pushGetRoute   = "GET /authenticator-configurations/push"
	pushPatchRoute = "PATCH /authenticator-configurations/push"
)

const twilioSmsJson = `{"authenticatorId":"auth_1","isActive":true,"verificationMethod":"SMS",` +
	`"smsProvider":"TWILIO","twilioCredentials":{"messagingServiceSid":"MG123","accountSid":"AC123"}}`

func smsConfigValue(t *testing.T, attributes map[string]tftypes.Value) tftypes.Value {
	t.Helper()

	return resourceObject(t, resourceSchemaOf(t, &smsAuthenticatorConfigurationResource{}), attributes)
}

func twilioTftypesBlock(t *testing.T, authToken string) tftypes.Value {
	t.Helper()

	return credentialTftypesObject(t, resourceSchemaOf(t, &smsAuthenticatorConfigurationResource{}), twilioCredentialsBlock, map[string]tftypes.Value{
		"auth_token":            optionalString(authToken),
		"messaging_service_sid": tftypes.NewValue(tftypes.String, "MG123"),
		"account_sid":           tftypes.NewValue(tftypes.String, "AC123"),
	})
}

func TestSmsCreateStoresTheServerOwnedAttributes(t *testing.T) {
	client, stub := newStubAPI(t, map[string][]stubResponse{
		smsCreateRoute: okResponse(twilioSmsJson),
	})

	resourceSchema := resourceSchemaOf(t, &smsAuthenticatorConfigurationResource{})

	planValue := smsConfigValue(t, map[string]tftypes.Value{
		"sms_provider":       tftypes.NewValue(tftypes.String, "TWILIO"),
		"twilio_credentials": twilioTftypesBlock(t, ""),
	})

	configValue := smsConfigValue(t, map[string]tftypes.Value{
		"sms_provider":       tftypes.NewValue(tftypes.String, "TWILIO"),
		"twilio_credentials": twilioTftypesBlock(t, "secret-token"),
	})

	resp := &resource.CreateResponse{State: tfsdk.State{Schema: resourceSchema}}
	(&smsAuthenticatorConfigurationResource{client: client}).Create(
		context.Background(),
		resource.CreateRequest{
			Plan:   tfsdk.Plan{Schema: resourceSchema, Raw: planValue},
			Config: tfsdk.Config{Schema: resourceSchema, Raw: configValue},
		},
		resp,
	)

	if resp.Diagnostics.HasError() {
		t.Fatal(detailsOf(resp.Diagnostics))
	}

	stub.assertRoutes(t, smsCreateRoute)

	if !strings.Contains(stub.bodyOf(t, smsCreateRoute), `"authToken":"secret-token"`) {
		t.Errorf("the create body should carry the secret from the configuration, got %s", stub.bodyOf(t, smsCreateRoute))
	}

	var state smsAuthenticatorConfigurationResourceModel
	resp.State.Get(context.Background(), &state)

	if state.AuthenticatorId.ValueString() != "auth_1" {
		t.Errorf("authenticator_id is %q, want auth_1", state.AuthenticatorId.ValueString())
	}

	if state.VerificationMethod.ValueString() != "SMS" {
		t.Errorf("verification_method is %q, want SMS", state.VerificationMethod.ValueString())
	}

	credentials := state.TwilioCredentials.Attributes()
	if sid, ok := credentials["messaging_service_sid"].(types.String); !ok || sid.ValueString() != "MG123" {
		t.Errorf("messaging_service_sid is %v, want MG123", credentials["messaging_service_sid"])
	}
}

func TestSmsCreateKeepsTheSecretOutOfState(t *testing.T) {
	client, _ := newStubAPI(t, map[string][]stubResponse{
		smsCreateRoute: okResponse(twilioSmsJson),
	})

	resourceSchema := resourceSchemaOf(t, &smsAuthenticatorConfigurationResource{})
	configValue := smsConfigValue(t, map[string]tftypes.Value{
		"sms_provider":       tftypes.NewValue(tftypes.String, "TWILIO"),
		"twilio_credentials": twilioTftypesBlock(t, "secret-token"),
	})

	resp := &resource.CreateResponse{State: tfsdk.State{Schema: resourceSchema}}
	(&smsAuthenticatorConfigurationResource{client: client}).Create(
		context.Background(),
		resource.CreateRequest{
			Plan:   tfsdk.Plan{Schema: resourceSchema, Raw: configValue},
			Config: tfsdk.Config{Schema: resourceSchema, Raw: configValue},
		},
		resp,
	)

	if resp.Diagnostics.HasError() {
		t.Fatal(detailsOf(resp.Diagnostics))
	}

	if strings.Contains(resp.State.Raw.String(), "secret-token") {
		t.Error("the auth token reached state; a write-only value must never be stored")
	}
}

func TestSmsCreateConflictTellsThePractitionerToImport(t *testing.T) {
	client, _ := newStubAPI(t, map[string][]stubResponse{
		smsCreateRoute: {{status: http.StatusConflict, body: `{"error":"already_exists"}`}},
	})

	resourceSchema := resourceSchemaOf(t, &smsAuthenticatorConfigurationResource{})
	configValue := smsConfigValue(t, map[string]tftypes.Value{
		"sms_provider":       tftypes.NewValue(tftypes.String, "TWILIO"),
		"twilio_credentials": twilioTftypesBlock(t, "secret-token"),
	})

	resp := &resource.CreateResponse{State: tfsdk.State{Schema: resourceSchema}}
	(&smsAuthenticatorConfigurationResource{client: client}).Create(
		context.Background(),
		resource.CreateRequest{
			Plan:   tfsdk.Plan{Schema: resourceSchema, Raw: configValue},
			Config: tfsdk.Config{Schema: resourceSchema, Raw: configValue},
		},
		resp,
	)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected a conflict diagnostic")
	}

	if !strings.Contains(detailsOf(resp.Diagnostics), "terraform import authsignal_sms_authenticator_configuration.sms sms") {
		t.Errorf("the conflict should name the import command, got %s", detailsOf(resp.Diagnostics))
	}
}

func TestSmsReadRemovesTheResourceWhenTheApiHasNone(t *testing.T) {
	client, _ := newStubAPI(t, map[string][]stubResponse{
		smsGetRoute: {notFoundResponse()},
	})

	resourceSchema := resourceSchemaOf(t, &smsAuthenticatorConfigurationResource{})
	state := tfsdk.State{Schema: resourceSchema, Raw: smsConfigValue(t, map[string]tftypes.Value{
		"sms_provider": tftypes.NewValue(tftypes.String, "TWILIO"),
	})}

	resp := &resource.ReadResponse{State: state}
	(&smsAuthenticatorConfigurationResource{client: client}).Read(
		context.Background(),
		resource.ReadRequest{State: state},
		resp,
	)

	if resp.Diagnostics.HasError() {
		t.Fatal(detailsOf(resp.Diagnostics))
	}

	if !resp.State.Raw.IsNull() {
		t.Error("a 404 should remove the resource from state")
	}
}

func TestSmsDeleteCallsTheApi(t *testing.T) {
	client, stub := newStubAPI(t, map[string][]stubResponse{
		smsDeleteRoute: okResponse(`{"success":true}`),
	})

	resp := &resource.DeleteResponse{}
	(&smsAuthenticatorConfigurationResource{client: client}).Delete(
		context.Background(),
		resource.DeleteRequest{},
		resp,
	)

	if resp.Diagnostics.HasError() {
		t.Fatal(detailsOf(resp.Diagnostics))
	}

	stub.assertRoutes(t, smsDeleteRoute)
}

func TestSmsImportStateRequiresTheSlug(t *testing.T) {
	resourceSchema := resourceSchemaOf(t, &smsAuthenticatorConfigurationResource{})

	accepted := &resource.ImportStateResponse{State: tfsdk.State{Schema: resourceSchema}}
	(&smsAuthenticatorConfigurationResource{}).ImportState(
		context.Background(),
		resource.ImportStateRequest{ID: "sms"},
		accepted,
	)

	if accepted.Diagnostics.HasError() {
		t.Fatalf("importing by the slug should be accepted, got %s", detailsOf(accepted.Diagnostics))
	}

	rejected := &resource.ImportStateResponse{State: tfsdk.State{Schema: resourceSchema}}
	(&smsAuthenticatorConfigurationResource{}).ImportState(
		context.Background(),
		resource.ImportStateRequest{ID: "auth_1"},
		rejected,
	)

	if !rejected.Diagnostics.HasError() {
		t.Fatal("importing by anything other than the slug should be refused")
	}

	if !strings.Contains(detailsOf(rejected.Diagnostics), `"sms"`) {
		t.Errorf("the error should name the expected slug, got %s", detailsOf(rejected.Diagnostics))
	}
}

func TestPushReadSurfacesFirebaseDerivedMetadata(t *testing.T) {
	client, _ := newStubAPI(t, map[string][]stubResponse{
		pushGetRoute: okResponse(`{"authenticatorId":"auth_2","isActive":true,"verificationMethod":"PUSH",` +
			`"pushProvider":"FIREBASE","fcmCredentials":{"projectId":"demo-project","clientEmail":"sa@demo.iam.gserviceaccount.com"}}`),
	})

	resourceSchema := resourceSchemaOf(t, &pushAuthenticatorConfigurationResource{})
	state := tfsdk.State{Schema: resourceSchema, Raw: resourceObject(t, resourceSchema, map[string]tftypes.Value{
		"push_provider": tftypes.NewValue(tftypes.String, "FIREBASE"),
	})}

	resp := &resource.ReadResponse{State: state}
	(&pushAuthenticatorConfigurationResource{client: client}).Read(
		context.Background(),
		resource.ReadRequest{State: state},
		resp,
	)

	if resp.Diagnostics.HasError() {
		t.Fatal(detailsOf(resp.Diagnostics))
	}

	var model pushAuthenticatorConfigurationResourceModel
	resp.State.Get(context.Background(), &model)

	if model.FcmProjectId.ValueString() != "demo-project" {
		t.Errorf("fcm_project_id is %q, want demo-project", model.FcmProjectId.ValueString())
	}

	if model.FcmClientEmail.ValueString() != "sa@demo.iam.gserviceaccount.com" {
		t.Errorf("fcm_client_email is %q, want the service account address", model.FcmClientEmail.ValueString())
	}

	if !model.FcmCredentials.IsNull() {
		t.Errorf("fcm_credentials is %v, want null — the block is write-only", model.FcmCredentials)
	}
}

func TestSmsProviderSwitchSendsNoSupersededBlockOverTheWire(t *testing.T) {
	client, stub := newStubAPI(t, map[string][]stubResponse{
		smsPatchRoute: okResponse(`{"authenticatorId":"auth_1","isActive":true,"verificationMethod":"SMS",` +
			`"smsProvider":"MODICA_GROUP","modicaGroupCredentials":{"username":"modica-user"}}`),
	})

	resourceSchema := resourceSchemaOf(t, &smsAuthenticatorConfigurationResource{})

	state := tfsdk.State{Schema: resourceSchema, Raw: smsConfigValue(t, map[string]tftypes.Value{
		"sms_provider":       tftypes.NewValue(tftypes.String, "TWILIO"),
		"twilio_credentials": twilioTftypesBlock(t, ""),
	})}

	modicaBlock := credentialTftypesObject(t, resourceSchema, modicaGroupCredentialsBlock, map[string]tftypes.Value{
		"username": tftypes.NewValue(tftypes.String, "modica-user"),
		"password": tftypes.NewValue(tftypes.String, "modica-password"),
	})

	configValue := smsConfigValue(t, map[string]tftypes.Value{
		"sms_provider":             tftypes.NewValue(tftypes.String, "MODICA_GROUP"),
		"modica_group_credentials": modicaBlock,
	})

	resp := &resource.UpdateResponse{State: tfsdk.State{Schema: resourceSchema}}
	(&smsAuthenticatorConfigurationResource{client: client}).Update(
		context.Background(),
		resource.UpdateRequest{
			Plan:   tfsdk.Plan{Schema: resourceSchema, Raw: configValue},
			Config: tfsdk.Config{Schema: resourceSchema, Raw: configValue},
			State:  state,
		},
		resp,
	)

	if resp.Diagnostics.HasError() {
		t.Fatal(detailsOf(resp.Diagnostics))
	}

	body := stub.bodyOf(t, smsPatchRoute)

	if strings.Contains(body, "twilioCredentials") {
		t.Errorf("the switch patch names twilioCredentials; the API rejects a request naming a provider while carrying another's block. Body: %s", body)
	}

	if !strings.Contains(body, `"smsProvider":"MODICA_GROUP"`) {
		t.Errorf("the switch patch should name the new provider, got %s", body)
	}

	if !strings.Contains(body, `"password":"modica-password"`) {
		t.Errorf("the switch patch should carry the new provider's secret, got %s", body)
	}
}

const (
	emailOtpGetRoute    = "GET /authenticator-configurations/email-otp"
	emailOtpCreateRoute = "POST /authenticator-configurations/email-otp"
	emailOtpPatchRoute  = "PATCH /authenticator-configurations/email-otp"
	emailOtpDeleteRoute = "DELETE /authenticator-configurations/email-otp"

	passkeyGetRoute    = "GET /authenticator-configurations/passkey"
	passkeyCreateRoute = "POST /authenticator-configurations/passkey"
	passkeyPatchRoute  = "PATCH /authenticator-configurations/passkey"
	passkeyDeleteRoute = "DELETE /authenticator-configurations/passkey"
)

func emailOtpSchema(t *testing.T) schema.Schema {
	t.Helper()

	return resourceSchemaOf(t, &emailOtpAuthenticatorConfigurationResource{})
}

func emailOtpConfigValue(t *testing.T, attributes map[string]tftypes.Value) tftypes.Value {
	t.Helper()

	return resourceObject(t, emailOtpSchema(t), attributes)
}

func sendgridTftypesBlock(t *testing.T, apiKey string, templateId string) tftypes.Value {
	t.Helper()

	return credentialTftypesObject(t, emailOtpSchema(t), sendgridEmailCredentialsBlock, map[string]tftypes.Value{
		"api_key":     optionalString(apiKey),
		"template_id": tftypes.NewValue(tftypes.String, templateId),
		"from_email":  tftypes.NewValue(tftypes.String, "noreply@example.com"),
	})
}

func sendgridEmailOtpJson(templateId string) string {
	return `{"authenticatorId":"auth_email","isActive":true,"verificationMethod":"EMAIL_OTP",` +
		`"emailProvider":"SENDGRID","sendgridEmailCredentials":{"templateId":"` + templateId +
		`","fromEmail":"noreply@example.com"}}`
}

func TestEmailOtpCreateSendsTheConfiguredSecret(t *testing.T) {
	apiKey := randomId("SG")
	templateId := randomId("template")

	client, stub := newStubAPI(t, map[string][]stubResponse{
		emailOtpCreateRoute: okResponse(sendgridEmailOtpJson(templateId)),
	})

	resourceSchema := emailOtpSchema(t)
	configValue := emailOtpConfigValue(t, map[string]tftypes.Value{
		"email_provider":             tftypes.NewValue(tftypes.String, "SENDGRID"),
		"sendgrid_email_credentials": sendgridTftypesBlock(t, apiKey, templateId),
	})
	planValue := emailOtpConfigValue(t, map[string]tftypes.Value{
		"email_provider":             tftypes.NewValue(tftypes.String, "SENDGRID"),
		"sendgrid_email_credentials": sendgridTftypesBlock(t, "", templateId),
	})

	resp := &resource.CreateResponse{State: tfsdk.State{Schema: resourceSchema}}
	(&emailOtpAuthenticatorConfigurationResource{client: client}).Create(
		context.Background(),
		resource.CreateRequest{
			Plan:   tfsdk.Plan{Schema: resourceSchema, Raw: planValue},
			Config: tfsdk.Config{Schema: resourceSchema, Raw: configValue},
		},
		resp,
	)

	if resp.Diagnostics.HasError() {
		t.Fatal(detailsOf(resp.Diagnostics))
	}

	stub.assertRoutes(t, emailOtpCreateRoute)

	if !strings.Contains(stub.bodyOf(t, emailOtpCreateRoute), apiKey) {
		t.Errorf("the create body should carry the secret from the configuration, got %s", stub.bodyOf(t, emailOtpCreateRoute))
	}
}

func TestEmailOtpCreateKeepsTheSecretOutOfState(t *testing.T) {
	apiKey := randomId("SG")
	templateId := randomId("template")

	client, _ := newStubAPI(t, map[string][]stubResponse{
		emailOtpCreateRoute: okResponse(sendgridEmailOtpJson(templateId)),
	})

	resourceSchema := emailOtpSchema(t)
	configValue := emailOtpConfigValue(t, map[string]tftypes.Value{
		"email_provider":             tftypes.NewValue(tftypes.String, "SENDGRID"),
		"sendgrid_email_credentials": sendgridTftypesBlock(t, apiKey, templateId),
	})

	resp := &resource.CreateResponse{State: tfsdk.State{Schema: resourceSchema}}
	(&emailOtpAuthenticatorConfigurationResource{client: client}).Create(
		context.Background(),
		resource.CreateRequest{
			Plan:   tfsdk.Plan{Schema: resourceSchema, Raw: configValue},
			Config: tfsdk.Config{Schema: resourceSchema, Raw: configValue},
		},
		resp,
	)

	if resp.Diagnostics.HasError() {
		t.Fatal(detailsOf(resp.Diagnostics))
	}

	if strings.Contains(resp.State.Raw.String(), apiKey) {
		t.Error("the API key reached state; a write-only value must never be stored")
	}
}

func TestEmailOtpCreateFailureDoesNotEchoTheSecret(t *testing.T) {
	apiKey := randomId("SG")

	client, _ := newStubAPI(t, map[string][]stubResponse{
		emailOtpCreateRoute: {{status: http.StatusBadRequest, body: `{"error":"invalid_request","errorDescription":"bad template"}`}},
	})

	resourceSchema := emailOtpSchema(t)
	configValue := emailOtpConfigValue(t, map[string]tftypes.Value{
		"email_provider":             tftypes.NewValue(tftypes.String, "SENDGRID"),
		"sendgrid_email_credentials": sendgridTftypesBlock(t, apiKey, randomId("template")),
	})

	resp := &resource.CreateResponse{State: tfsdk.State{Schema: resourceSchema}}
	(&emailOtpAuthenticatorConfigurationResource{client: client}).Create(
		context.Background(),
		resource.CreateRequest{
			Plan:   tfsdk.Plan{Schema: resourceSchema, Raw: configValue},
			Config: tfsdk.Config{Schema: resourceSchema, Raw: configValue},
		},
		resp,
	)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected the 400 to surface as a diagnostic")
	}

	if strings.Contains(detailsOf(resp.Diagnostics), apiKey) {
		t.Errorf("the API key appears in the diagnostic: %s", detailsOf(resp.Diagnostics))
	}
}

func TestEmailOtpCreateConflictDirectsThePractitionerToImport(t *testing.T) {
	client, _ := newStubAPI(t, map[string][]stubResponse{
		emailOtpCreateRoute: {{status: http.StatusConflict, body: `{"error":"already_exists"}`}},
	})

	resourceSchema := emailOtpSchema(t)
	configValue := emailOtpConfigValue(t, map[string]tftypes.Value{
		"email_provider":             tftypes.NewValue(tftypes.String, "SENDGRID"),
		"sendgrid_email_credentials": sendgridTftypesBlock(t, randomId("SG"), randomId("template")),
	})

	resp := &resource.CreateResponse{State: tfsdk.State{Schema: resourceSchema}}
	(&emailOtpAuthenticatorConfigurationResource{client: client}).Create(
		context.Background(),
		resource.CreateRequest{
			Plan:   tfsdk.Plan{Schema: resourceSchema, Raw: configValue},
			Config: tfsdk.Config{Schema: resourceSchema, Raw: configValue},
		},
		resp,
	)

	if !strings.Contains(detailsOf(resp.Diagnostics), "terraform import authsignal_email_otp_authenticator_configuration.email_otp email-otp") {
		t.Errorf("the conflict should name the import command, got %s", detailsOf(resp.Diagnostics))
	}
}

func TestEmailOtpReadRefreshesStateFromTheApi(t *testing.T) {
	templateId := randomId("template")

	client, _ := newStubAPI(t, map[string][]stubResponse{
		emailOtpGetRoute: okResponse(sendgridEmailOtpJson(templateId)),
	})

	resourceSchema := emailOtpSchema(t)

	state := tfsdk.State{Schema: resourceSchema, Raw: emailOtpConfigValue(t, map[string]tftypes.Value{
		"email_provider":             tftypes.NewValue(tftypes.String, "SENDGRID"),
		"sendgrid_email_credentials": sendgridTftypesBlock(t, "", randomId("stale-template")),
	})}

	resp := &resource.ReadResponse{State: state}
	(&emailOtpAuthenticatorConfigurationResource{client: client}).Read(
		context.Background(),
		resource.ReadRequest{State: state},
		resp,
	)

	if resp.Diagnostics.HasError() {
		t.Fatal(detailsOf(resp.Diagnostics))
	}

	var model emailOtpAuthenticatorConfigurationResourceModel
	resp.State.Get(context.Background(), &model)

	if model.AuthenticatorId.ValueString() != "auth_email" {
		t.Errorf("authenticator_id is %q, want auth_email", model.AuthenticatorId.ValueString())
	}

	credentials := model.SendgridEmailCredentials.Attributes()
	if stored, ok := credentials["template_id"].(types.String); !ok || stored.ValueString() != templateId {
		t.Errorf("template_id is %v, want the value the API returned", credentials["template_id"])
	}

	if !credentials["api_key"].IsNull() {
		t.Errorf("api_key is %v, want null — the API never returns a secret", credentials["api_key"])
	}
}

func TestEmailOtpReadRemovesTheResourceWhenTheApiHasNone(t *testing.T) {
	client, _ := newStubAPI(t, map[string][]stubResponse{
		emailOtpGetRoute: {notFoundResponse()},
	})

	resourceSchema := emailOtpSchema(t)
	state := tfsdk.State{Schema: resourceSchema, Raw: emailOtpConfigValue(t, map[string]tftypes.Value{
		"email_provider": tftypes.NewValue(tftypes.String, "SENDGRID"),
	})}

	resp := &resource.ReadResponse{State: state}
	(&emailOtpAuthenticatorConfigurationResource{client: client}).Read(
		context.Background(),
		resource.ReadRequest{State: state},
		resp,
	)

	if !resp.State.Raw.IsNull() {
		t.Error("a 404 should remove the resource from state")
	}
}

func TestEmailOtpUpdatePatchesTheChangedAttributes(t *testing.T) {
	templateId := randomId("template")
	nextTemplateId := randomId("template")

	client, stub := newStubAPI(t, map[string][]stubResponse{
		emailOtpPatchRoute: okResponse(sendgridEmailOtpJson(nextTemplateId)),
	})

	resourceSchema := emailOtpSchema(t)

	state := tfsdk.State{Schema: resourceSchema, Raw: emailOtpConfigValue(t, map[string]tftypes.Value{
		"email_provider":             tftypes.NewValue(tftypes.String, "SENDGRID"),
		"sendgrid_email_credentials": sendgridTftypesBlock(t, "", templateId),
	})}

	configValue := emailOtpConfigValue(t, map[string]tftypes.Value{
		"email_provider":             tftypes.NewValue(tftypes.String, "SENDGRID"),
		"sendgrid_email_credentials": sendgridTftypesBlock(t, "", nextTemplateId),
	})

	resp := &resource.UpdateResponse{State: tfsdk.State{Schema: resourceSchema}}
	(&emailOtpAuthenticatorConfigurationResource{client: client}).Update(
		context.Background(),
		resource.UpdateRequest{
			Plan:   tfsdk.Plan{Schema: resourceSchema, Raw: configValue},
			Config: tfsdk.Config{Schema: resourceSchema, Raw: configValue},
			State:  state,
		},
		resp,
	)

	if resp.Diagnostics.HasError() {
		t.Fatal(detailsOf(resp.Diagnostics))
	}

	stub.assertRoutes(t, emailOtpPatchRoute)

	if !strings.Contains(stub.bodyOf(t, emailOtpPatchRoute), nextTemplateId) {
		t.Errorf("the patch should carry the changed template id, got %s", stub.bodyOf(t, emailOtpPatchRoute))
	}
}

func TestEmailOtpDeleteRemovesTheConfiguration(t *testing.T) {
	client, stub := newStubAPI(t, map[string][]stubResponse{
		emailOtpDeleteRoute: okResponse(`{"success":true}`),
	})

	resp := &resource.DeleteResponse{}
	(&emailOtpAuthenticatorConfigurationResource{client: client}).Delete(
		context.Background(),
		resource.DeleteRequest{},
		resp,
	)

	if resp.Diagnostics.HasError() {
		t.Fatal(detailsOf(resp.Diagnostics))
	}

	stub.assertRoutes(t, emailOtpDeleteRoute)
}

func TestEmailOtpImportAcceptsOnlyItsOwnSlug(t *testing.T) {
	resourceSchema := emailOtpSchema(t)

	accepted := &resource.ImportStateResponse{State: tfsdk.State{Schema: resourceSchema}}
	(&emailOtpAuthenticatorConfigurationResource{}).ImportState(
		context.Background(),
		resource.ImportStateRequest{ID: "email-otp"},
		accepted,
	)

	if accepted.Diagnostics.HasError() {
		t.Fatalf("importing by the slug should be accepted, got %s", detailsOf(accepted.Diagnostics))
	}

	rejected := &resource.ImportStateResponse{State: tfsdk.State{Schema: resourceSchema}}
	(&emailOtpAuthenticatorConfigurationResource{}).ImportState(
		context.Background(),
		resource.ImportStateRequest{ID: "email_otp"},
		rejected,
	)

	if !rejected.Diagnostics.HasError() {
		t.Error("the resource name is not the slug, so importing by it should be refused")
	}
}

func passkeySchema(t *testing.T) schema.Schema {
	t.Helper()

	return resourceSchemaOf(t, &passkeyAuthenticatorConfigurationResource{})
}

func originSet(origins ...string) tftypes.Value {
	values := make([]tftypes.Value, 0, len(origins))
	for _, origin := range origins {
		values = append(values, tftypes.NewValue(tftypes.String, origin))
	}

	return tftypes.NewValue(tftypes.Set{ElementType: tftypes.String}, values)
}

func passkeyJson(relyingParty string, attachment string) string {
	return `{"authenticatorId":"auth_passkey","isActive":true,"verificationMethod":"PASSKEY",` +
		`"relyingParty":"` + relyingParty + `","expectedOrigins":["https://` + relyingParty + `"],` +
		`"userVerificationRequirement":"preferred","authenticatorAttachment":` + attachment + `}`
}

func TestPasskeyCreateStoresTheServerOwnedAttributes(t *testing.T) {
	client, stub := newStubAPI(t, map[string][]stubResponse{
		passkeyCreateRoute: okResponse(passkeyJson("example.com", `"platform"`)),
	})

	resourceSchema := passkeySchema(t)
	configValue := resourceObject(t, resourceSchema, map[string]tftypes.Value{
		"relying_party":    tftypes.NewValue(tftypes.String, "example.com"),
		"expected_origins": originSet("https://example.com"),
	})

	resp := &resource.CreateResponse{State: tfsdk.State{Schema: resourceSchema}}
	(&passkeyAuthenticatorConfigurationResource{client: client}).Create(
		context.Background(),
		resource.CreateRequest{
			Plan:   tfsdk.Plan{Schema: resourceSchema, Raw: configValue},
			Config: tfsdk.Config{Schema: resourceSchema, Raw: configValue},
		},
		resp,
	)

	if resp.Diagnostics.HasError() {
		t.Fatal(detailsOf(resp.Diagnostics))
	}

	stub.assertRoutes(t, passkeyCreateRoute)

	var model passkeyAuthenticatorConfigurationResourceModel
	resp.State.Get(context.Background(), &model)

	if model.AuthenticatorId.ValueString() != "auth_passkey" {
		t.Errorf("authenticator_id is %q, want auth_passkey", model.AuthenticatorId.ValueString())
	}

	if model.VerificationMethod.ValueString() != "PASSKEY" {
		t.Errorf("verification_method is %q, want PASSKEY", model.VerificationMethod.ValueString())
	}
}

func TestPasskeyReadRefreshesStateFromTheApi(t *testing.T) {
	client, _ := newStubAPI(t, map[string][]stubResponse{
		passkeyGetRoute: okResponse(passkeyJson("example.com", `"platform"`)),
	})

	resourceSchema := passkeySchema(t)
	state := tfsdk.State{Schema: resourceSchema, Raw: resourceObject(t, resourceSchema, map[string]tftypes.Value{
		"relying_party":    tftypes.NewValue(tftypes.String, "example.com"),
		"expected_origins": originSet("https://example.com"),
	})}

	resp := &resource.ReadResponse{State: state}
	(&passkeyAuthenticatorConfigurationResource{client: client}).Read(
		context.Background(),
		resource.ReadRequest{State: state},
		resp,
	)

	if resp.Diagnostics.HasError() {
		t.Fatal(detailsOf(resp.Diagnostics))
	}

	var model passkeyAuthenticatorConfigurationResourceModel
	resp.State.Get(context.Background(), &model)

	if model.AuthenticatorAttachment.ValueString() != "platform" {
		t.Errorf("authenticator_attachment is %q, want platform", model.AuthenticatorAttachment.ValueString())
	}

	if len(model.ExpectedOrigins.Elements()) != 1 {
		t.Errorf("expected_origins is %v, want the single origin the API returned", model.ExpectedOrigins)
	}
}

func TestPasskeyReadRemovesTheResourceWhenTheApiHasNone(t *testing.T) {
	client, _ := newStubAPI(t, map[string][]stubResponse{
		passkeyGetRoute: {notFoundResponse()},
	})

	resourceSchema := passkeySchema(t)
	state := tfsdk.State{Schema: resourceSchema, Raw: resourceObject(t, resourceSchema, map[string]tftypes.Value{
		"relying_party":    tftypes.NewValue(tftypes.String, "example.com"),
		"expected_origins": originSet("https://example.com"),
	})}

	resp := &resource.ReadResponse{State: state}
	(&passkeyAuthenticatorConfigurationResource{client: client}).Read(
		context.Background(),
		resource.ReadRequest{State: state},
		resp,
	)

	if !resp.State.Raw.IsNull() {
		t.Error("a 404 should remove the resource from state")
	}
}

func TestPasskeyUpdateOmitsTheAttachmentWhenItLeavesTheConfiguration(t *testing.T) {
	client, stub := newStubAPI(t, map[string][]stubResponse{
		passkeyPatchRoute: okResponse(passkeyJson("example.com", "null")),
	})

	resourceSchema := passkeySchema(t)

	state := tfsdk.State{Schema: resourceSchema, Raw: resourceObject(t, resourceSchema, map[string]tftypes.Value{
		"relying_party":            tftypes.NewValue(tftypes.String, "example.com"),
		"expected_origins":         originSet("https://example.com"),
		"authenticator_attachment": tftypes.NewValue(tftypes.String, "platform"),
	})}

	configValue := resourceObject(t, resourceSchema, map[string]tftypes.Value{
		"relying_party":    tftypes.NewValue(tftypes.String, "example.com"),
		"expected_origins": originSet("https://example.com"),
	})

	resp := &resource.UpdateResponse{State: tfsdk.State{Schema: resourceSchema}}
	(&passkeyAuthenticatorConfigurationResource{client: client}).Update(
		context.Background(),
		resource.UpdateRequest{
			Plan:   tfsdk.Plan{Schema: resourceSchema, Raw: configValue},
			Config: tfsdk.Config{Schema: resourceSchema, Raw: configValue},
			State:  state,
		},
		resp,
	)

	if resp.Diagnostics.HasError() {
		t.Fatal(detailsOf(resp.Diagnostics))
	}

	if strings.Contains(stub.bodyOf(t, passkeyPatchRoute), "authenticatorAttachment") {
		t.Errorf("dropping a nullable attribute must leave the key out of the patch entirely, got %s", stub.bodyOf(t, passkeyPatchRoute))
	}
}

func TestPasskeyUpdateNeverSendsAnEmptyOriginList(t *testing.T) {
	client, stub := newStubAPI(t, map[string][]stubResponse{
		passkeyPatchRoute: okResponse(passkeyJson("example.com", "null")),
	})

	resourceSchema := passkeySchema(t)

	state := tfsdk.State{Schema: resourceSchema, Raw: resourceObject(t, resourceSchema, map[string]tftypes.Value{
		"relying_party":    tftypes.NewValue(tftypes.String, "old.example.com"),
		"expected_origins": originSet("https://old.example.com"),
	})}

	configValue := resourceObject(t, resourceSchema, map[string]tftypes.Value{
		"relying_party":    tftypes.NewValue(tftypes.String, "example.com"),
		"expected_origins": originSet("https://example.com"),
	})

	resp := &resource.UpdateResponse{State: tfsdk.State{Schema: resourceSchema}}
	(&passkeyAuthenticatorConfigurationResource{client: client}).Update(
		context.Background(),
		resource.UpdateRequest{
			Plan:   tfsdk.Plan{Schema: resourceSchema, Raw: configValue},
			Config: tfsdk.Config{Schema: resourceSchema, Raw: configValue},
			State:  state,
		},
		resp,
	)

	if resp.Diagnostics.HasError() {
		t.Fatal(detailsOf(resp.Diagnostics))
	}

	body := stub.bodyOf(t, passkeyPatchRoute)

	if strings.Contains(body, `"expectedOrigins":[]`) {
		t.Errorf("the patch empties the origin list, which leaves passkeys unusable: %s", body)
	}

	if !strings.Contains(body, `"expectedOrigins":["https://example.com"]`) {
		t.Errorf("the patch should carry the new origin, got %s", body)
	}
}

func TestPasskeyUpdateCarriesTheRelyingPartyAndItsOriginsTogether(t *testing.T) {
	client, stub := newStubAPI(t, map[string][]stubResponse{
		passkeyPatchRoute: okResponse(passkeyJson("example.com", "null")),
	})

	resourceSchema := passkeySchema(t)

	state := tfsdk.State{Schema: resourceSchema, Raw: resourceObject(t, resourceSchema, map[string]tftypes.Value{
		"relying_party":    tftypes.NewValue(tftypes.String, "old.example.com"),
		"expected_origins": originSet("https://old.example.com"),
	})}

	configValue := resourceObject(t, resourceSchema, map[string]tftypes.Value{
		"relying_party":    tftypes.NewValue(tftypes.String, "example.com"),
		"expected_origins": originSet("https://example.com"),
	})

	resp := &resource.UpdateResponse{State: tfsdk.State{Schema: resourceSchema}}
	(&passkeyAuthenticatorConfigurationResource{client: client}).Update(
		context.Background(),
		resource.UpdateRequest{
			Plan:   tfsdk.Plan{Schema: resourceSchema, Raw: configValue},
			Config: tfsdk.Config{Schema: resourceSchema, Raw: configValue},
			State:  state,
		},
		resp,
	)

	if resp.Diagnostics.HasError() {
		t.Fatal(detailsOf(resp.Diagnostics))
	}

	body := stub.bodyOf(t, passkeyPatchRoute)

	if !strings.Contains(body, `"relyingParty":"example.com"`) {
		t.Errorf("the patch should carry the changed relying party, got %s", body)
	}

	if !strings.Contains(body, "expectedOrigins") {
		t.Errorf("a relying party change has to carry its origins too, got %s", body)
	}
}

func TestPasskeyDeleteRemovesTheConfiguration(t *testing.T) {
	client, stub := newStubAPI(t, map[string][]stubResponse{
		passkeyDeleteRoute: okResponse(`{"success":true}`),
	})

	resp := &resource.DeleteResponse{}
	(&passkeyAuthenticatorConfigurationResource{client: client}).Delete(
		context.Background(),
		resource.DeleteRequest{},
		resp,
	)

	if resp.Diagnostics.HasError() {
		t.Fatal(detailsOf(resp.Diagnostics))
	}

	stub.assertRoutes(t, passkeyDeleteRoute)
}

func TestPasskeyImportAcceptsOnlyItsOwnSlug(t *testing.T) {
	resourceSchema := passkeySchema(t)

	accepted := &resource.ImportStateResponse{State: tfsdk.State{Schema: resourceSchema}}
	(&passkeyAuthenticatorConfigurationResource{}).ImportState(
		context.Background(),
		resource.ImportStateRequest{ID: "passkey"},
		accepted,
	)

	if accepted.Diagnostics.HasError() {
		t.Fatalf("importing by the slug should be accepted, got %s", detailsOf(accepted.Diagnostics))
	}

	rejected := &resource.ImportStateResponse{State: tfsdk.State{Schema: resourceSchema}}
	(&passkeyAuthenticatorConfigurationResource{}).ImportState(
		context.Background(),
		resource.ImportStateRequest{ID: "sms"},
		rejected,
	)

	if !rejected.Diagnostics.HasError() {
		t.Error("importing another resource's slug should be refused")
	}
}

func TestSmsUpdateLeavesAnOmittedCredentialBlockOutOfState(t *testing.T) {
	client, stub := newStubAPI(t, map[string][]stubResponse{
		smsPatchRoute: okResponse(twilioSmsJson),
	})

	resourceSchema := resourceSchemaOf(t, &smsAuthenticatorConfigurationResource{})

	state := tfsdk.State{Schema: resourceSchema, Raw: smsConfigValue(t, map[string]tftypes.Value{
		"sms_provider":       tftypes.NewValue(tftypes.String, "TWILIO"),
		"twilio_credentials": twilioTftypesBlock(t, ""),
	})}

	planValue := smsConfigValue(t, map[string]tftypes.Value{
		"sms_provider": tftypes.NewValue(tftypes.String, "TWILIO"),
	})

	resp := &resource.UpdateResponse{State: tfsdk.State{Schema: resourceSchema}}
	(&smsAuthenticatorConfigurationResource{client: client}).Update(
		context.Background(),
		resource.UpdateRequest{
			Plan:   tfsdk.Plan{Schema: resourceSchema, Raw: planValue},
			Config: tfsdk.Config{Schema: resourceSchema, Raw: planValue},
			State:  state,
		},
		resp,
	)

	if resp.Diagnostics.HasError() {
		t.Fatal(detailsOf(resp.Diagnostics))
	}

	var applied smsAuthenticatorConfigurationResourceModel
	resp.State.Get(context.Background(), &applied)

	if !applied.TwilioCredentials.IsNull() {
		t.Errorf("twilio_credentials is %v but the plan said null; Terraform reports that as an inconsistent result", applied.TwilioCredentials)
	}

	if strings.Contains(stub.bodyOf(t, smsPatchRoute), "twilioCredentials") {
		t.Errorf("omitting the block from the configuration should leave the stored credentials alone, got %s", stub.bodyOf(t, smsPatchRoute))
	}
}

func TestSmsReadLeavesAnUntrackedCredentialBlockAlone(t *testing.T) {
	client, _ := newStubAPI(t, map[string][]stubResponse{
		smsGetRoute: okResponse(twilioSmsJson),
	})

	resourceSchema := resourceSchemaOf(t, &smsAuthenticatorConfigurationResource{})
	state := tfsdk.State{Schema: resourceSchema, Raw: smsConfigValue(t, map[string]tftypes.Value{
		"sms_provider": tftypes.NewValue(tftypes.String, "TWILIO"),
	})}

	resp := &resource.ReadResponse{State: state}
	(&smsAuthenticatorConfigurationResource{client: client}).Read(
		context.Background(),
		resource.ReadRequest{State: state},
		resp,
	)

	if resp.Diagnostics.HasError() {
		t.Fatal(detailsOf(resp.Diagnostics))
	}

	var refreshed smsAuthenticatorConfigurationResourceModel
	resp.State.Get(context.Background(), &refreshed)

	if !refreshed.TwilioCredentials.IsNull() {
		t.Errorf("twilio_credentials is %v; a read that volunteers a block nobody configured creates a diff that never settles", refreshed.TwilioCredentials)
	}
}

func TestSmsReadRefreshesATrackedCredentialBlock(t *testing.T) {
	client, _ := newStubAPI(t, map[string][]stubResponse{
		smsGetRoute: okResponse(twilioSmsJson),
	})

	resourceSchema := resourceSchemaOf(t, &smsAuthenticatorConfigurationResource{})
	stale := credentialTftypesObject(t, resourceSchema, twilioCredentialsBlock, map[string]tftypes.Value{
		"messaging_service_sid": tftypes.NewValue(tftypes.String, "MG-stale"),
		"account_sid":           tftypes.NewValue(tftypes.String, "AC-stale"),
	})

	state := tfsdk.State{Schema: resourceSchema, Raw: smsConfigValue(t, map[string]tftypes.Value{
		"sms_provider":       tftypes.NewValue(tftypes.String, "TWILIO"),
		"twilio_credentials": stale,
	})}

	resp := &resource.ReadResponse{State: state}
	(&smsAuthenticatorConfigurationResource{client: client}).Read(
		context.Background(),
		resource.ReadRequest{State: state},
		resp,
	)

	var refreshed smsAuthenticatorConfigurationResourceModel
	resp.State.Get(context.Background(), &refreshed)

	sid, ok := refreshed.TwilioCredentials.Attributes()["messaging_service_sid"].(types.String)
	if !ok || sid.ValueString() != "MG123" {
		t.Errorf("messaging_service_sid is %v, want the value the API reports so drift is visible", refreshed.TwilioCredentials)
	}
}

func TestSmsImportThenReadTracksNoCredentialBlock(t *testing.T) {
	client, _ := newStubAPI(t, map[string][]stubResponse{
		smsGetRoute: okResponse(twilioSmsJson),
	})

	resourceSchema := resourceSchemaOf(t, &smsAuthenticatorConfigurationResource{})

	imported := &resource.ImportStateResponse{State: tfsdk.State{Schema: resourceSchema}}
	(&smsAuthenticatorConfigurationResource{}).ImportState(
		context.Background(),
		resource.ImportStateRequest{ID: "sms"},
		imported,
	)

	if imported.Diagnostics.HasError() {
		t.Fatal(detailsOf(imported.Diagnostics))
	}

	read := &resource.ReadResponse{State: imported.State}
	(&smsAuthenticatorConfigurationResource{client: client}).Read(
		context.Background(),
		resource.ReadRequest{State: imported.State},
		read,
	)

	if read.Diagnostics.HasError() {
		t.Fatal(detailsOf(read.Diagnostics))
	}

	var refreshed smsAuthenticatorConfigurationResourceModel
	read.State.Get(context.Background(), &refreshed)

	if !refreshed.TwilioCredentials.IsNull() {
		t.Errorf("twilio_credentials is %v after import; an imported resource must plan clean against a configuration that omits it", refreshed.TwilioCredentials)
	}

	if refreshed.SmsProvider.ValueString() != "TWILIO" {
		t.Errorf("sms_provider is %q, want the imported value", refreshed.SmsProvider.ValueString())
	}
}

func TestPushSwitchClearsTheSupersededBlockFromState(t *testing.T) {
	client, _ := newStubAPI(t, map[string][]stubResponse{
		pushPatchRoute: okResponse(`{"authenticatorId":"auth_2","isActive":true,"verificationMethod":"PUSH",` +
			`"pushProvider":"FIREBASE","fcmCredentials":{"projectId":"demo-project","clientEmail":"sa@demo.iam.gserviceaccount.com"}}`),
	})

	resourceSchema := resourceSchemaOf(t, &pushAuthenticatorConfigurationResource{})

	apnsStored := credentialTftypesObject(t, resourceSchema, apnsCredentialsBlock, map[string]tftypes.Value{
		"team_id":   tftypes.NewValue(tftypes.String, "TEAM"),
		"key_id":    tftypes.NewValue(tftypes.String, "KEY"),
		"bundle_id": tftypes.NewValue(tftypes.String, "com.example.app"),
	})

	state := tfsdk.State{Schema: resourceSchema, Raw: resourceObject(t, resourceSchema, map[string]tftypes.Value{
		"push_provider":    tftypes.NewValue(tftypes.String, "DEFAULT"),
		"apns_credentials": apnsStored,
	})}

	planValue := resourceObject(t, resourceSchema, map[string]tftypes.Value{
		"push_provider": tftypes.NewValue(tftypes.String, "FIREBASE"),
		"fcm_credentials": credentialTftypesObject(t, resourceSchema, fcmCredentialsBlock, map[string]tftypes.Value{
			"service_account_key": tftypes.NewValue(tftypes.String, "{}"),
		}),
		"fcm_credentials_version": tftypes.NewValue(tftypes.String, "1"),
	})

	resp := &resource.UpdateResponse{State: tfsdk.State{Schema: resourceSchema}}
	(&pushAuthenticatorConfigurationResource{client: client}).Update(
		context.Background(),
		resource.UpdateRequest{
			Plan:   tfsdk.Plan{Schema: resourceSchema, Raw: planValue},
			Config: tfsdk.Config{Schema: resourceSchema, Raw: planValue},
			State:  state,
		},
		resp,
	)

	if resp.Diagnostics.HasError() {
		t.Fatal(detailsOf(resp.Diagnostics))
	}

	var applied pushAuthenticatorConfigurationResourceModel
	resp.State.Get(context.Background(), &applied)

	if !applied.ApnsCredentials.IsNull() {
		t.Errorf("apns_credentials is %v after switching to Firebase; the superseded block has to leave state", applied.ApnsCredentials)
	}

	if applied.FcmProjectId.ValueString() != "demo-project" {
		t.Errorf("fcm_project_id is %q; the derived attributes still come from the response", applied.FcmProjectId.ValueString())
	}
}

func TestPushSwitchAwayFromWebhookAdoptsTheApiNullEndpoint(t *testing.T) {
	client, stub := newStubAPI(t, map[string][]stubResponse{
		pushPatchRoute: okResponse(`{"authenticatorId":"auth_2","isActive":true,"verificationMethod":"PUSH",` +
			`"pushProvider":"FIREBASE","fcmCredentials":{"projectId":"demo-project","clientEmail":"sa@demo.iam.gserviceaccount.com"}}`),
	})

	resourceSchema := resourceSchemaOf(t, &pushAuthenticatorConfigurationResource{})

	state := tfsdk.State{Schema: resourceSchema, Raw: resourceObject(t, resourceSchema, map[string]tftypes.Value{
		"push_provider": tftypes.NewValue(tftypes.String, "WEBHOOK"),
		"webhook_url":   tftypes.NewValue(tftypes.String, "https://example.com/push"),
	})}

	config := resourceObject(t, resourceSchema, map[string]tftypes.Value{
		"push_provider": tftypes.NewValue(tftypes.String, "FIREBASE"),
		"fcm_credentials": credentialTftypesObject(t, resourceSchema, fcmCredentialsBlock, map[string]tftypes.Value{
			"service_account_key": tftypes.NewValue(tftypes.String, "{}"),
		}),
		"fcm_credentials_version": tftypes.NewValue(tftypes.String, "1"),
	})

	planValue := resourceObject(t, resourceSchema, map[string]tftypes.Value{
		"push_provider": tftypes.NewValue(tftypes.String, "FIREBASE"),
		"webhook_url":   tftypes.NewValue(tftypes.String, tftypes.UnknownValue),
		"fcm_credentials": credentialTftypesObject(t, resourceSchema, fcmCredentialsBlock, map[string]tftypes.Value{
			"service_account_key": tftypes.NewValue(tftypes.String, "{}"),
		}),
		"fcm_credentials_version": tftypes.NewValue(tftypes.String, "1"),
	})

	resp := &resource.UpdateResponse{State: tfsdk.State{Schema: resourceSchema}}
	(&pushAuthenticatorConfigurationResource{client: client}).Update(
		context.Background(),
		resource.UpdateRequest{
			Plan:   tfsdk.Plan{Schema: resourceSchema, Raw: planValue},
			Config: tfsdk.Config{Schema: resourceSchema, Raw: config},
			State:  state,
		},
		resp,
	)

	if resp.Diagnostics.HasError() {
		t.Fatal(detailsOf(resp.Diagnostics))
	}

	var applied pushAuthenticatorConfigurationResourceModel
	resp.State.Get(context.Background(), &applied)

	if !applied.WebhookUrl.IsNull() {
		t.Errorf("webhook_url is %v after switching away from WEBHOOK; the API's null has to reach state in the same apply", applied.WebhookUrl)
	}

	if strings.Contains(stub.bodyOf(t, pushPatchRoute), "webhookUrl") {
		t.Errorf("the switch patch carries webhookUrl; it has to be absent, not null, got %s", stub.bodyOf(t, pushPatchRoute))
	}
}

const smsWithRemoteExtras = `{"authenticatorId":"auth_1","isActive":true,"verificationMethod":"SMS",` +
	`"smsProvider":"TWILIO","smsCountryCodes":["AU","NZ"],"defaultCountryCode":"NZ",` +
	`"allowedCustomSmsVariables":["firstName"],` +
	`"twilioCredentials":{"messagingServiceSid":"MG123","accountSid":"AC123"}}`

func TestSmsCreateTakesTheApiValueForAnAttributeTheConfigurationOmits(t *testing.T) {
	client, _ := newStubAPI(t, map[string][]stubResponse{
		smsCreateRoute: okResponse(smsWithRemoteExtras),
	})

	resourceSchema := resourceSchemaOf(t, &smsAuthenticatorConfigurationResource{})

	configValue := smsConfigValue(t, map[string]tftypes.Value{
		"sms_provider":       tftypes.NewValue(tftypes.String, "TWILIO"),
		"twilio_credentials": twilioTftypesBlock(t, "secret-token"),
	})

	planValue := smsConfigValue(t, map[string]tftypes.Value{
		"sms_provider":         tftypes.NewValue(tftypes.String, "TWILIO"),
		"twilio_credentials":   twilioTftypesBlock(t, ""),
		"sms_country_codes":    tftypes.NewValue(tftypes.Set{ElementType: tftypes.String}, tftypes.UnknownValue),
		"default_country_code": tftypes.NewValue(tftypes.String, tftypes.UnknownValue),
	})

	resp := &resource.CreateResponse{State: tfsdk.State{Schema: resourceSchema}}
	(&smsAuthenticatorConfigurationResource{client: client}).Create(
		context.Background(),
		resource.CreateRequest{
			Plan:   tfsdk.Plan{Schema: resourceSchema, Raw: planValue},
			Config: tfsdk.Config{Schema: resourceSchema, Raw: configValue},
		},
		resp,
	)

	if resp.Diagnostics.HasError() {
		t.Fatal(detailsOf(resp.Diagnostics))
	}

	var created smsAuthenticatorConfigurationResourceModel
	resp.State.Get(context.Background(), &created)

	if created.DefaultCountryCode.IsUnknown() || created.DefaultCountryCode.ValueString() != "NZ" {
		t.Errorf("default_country_code is %v, want the API's own value; an unknown left in state fails the apply", created.DefaultCountryCode)
	}

	if created.SmsCountryCodes.IsUnknown() || len(created.SmsCountryCodes.Elements()) != 2 {
		t.Errorf("sms_country_codes is %v, want the two codes the API returned", created.SmsCountryCodes)
	}
}

func TestSmsCreateSendsNoKeyForAnAttributeTheConfigurationOmits(t *testing.T) {
	client, stub := newStubAPI(t, map[string][]stubResponse{
		smsCreateRoute: okResponse(smsWithRemoteExtras),
	})

	resourceSchema := resourceSchemaOf(t, &smsAuthenticatorConfigurationResource{})

	configValue := smsConfigValue(t, map[string]tftypes.Value{
		"sms_provider":       tftypes.NewValue(tftypes.String, "TWILIO"),
		"twilio_credentials": twilioTftypesBlock(t, "secret-token"),
	})

	resp := &resource.CreateResponse{State: tfsdk.State{Schema: resourceSchema}}
	(&smsAuthenticatorConfigurationResource{client: client}).Create(
		context.Background(),
		resource.CreateRequest{
			Plan:   tfsdk.Plan{Schema: resourceSchema, Raw: configValue},
			Config: tfsdk.Config{Schema: resourceSchema, Raw: configValue},
		},
		resp,
	)

	if resp.Diagnostics.HasError() {
		t.Fatal(detailsOf(resp.Diagnostics))
	}

	body := stub.bodyOf(t, smsCreateRoute)
	for _, absent := range []string{"smsCountryCodes", "defaultCountryCode", "allowedCustomSmsVariables"} {
		if strings.Contains(body, absent) {
			t.Errorf("the create body carries %q; only what the configuration sets may be sent, got %s", absent, body)
		}
	}
}

func TestSmsImportThenReadKeepsTheRemoteOptionalValues(t *testing.T) {
	client, _ := newStubAPI(t, map[string][]stubResponse{
		smsGetRoute: okResponse(smsWithRemoteExtras),
	})

	resourceSchema := resourceSchemaOf(t, &smsAuthenticatorConfigurationResource{})

	imported := &resource.ImportStateResponse{State: tfsdk.State{Schema: resourceSchema}}
	(&smsAuthenticatorConfigurationResource{}).ImportState(
		context.Background(),
		resource.ImportStateRequest{ID: "sms"},
		imported,
	)

	if imported.Diagnostics.HasError() {
		t.Fatal(detailsOf(imported.Diagnostics))
	}

	read := &resource.ReadResponse{State: imported.State}
	(&smsAuthenticatorConfigurationResource{client: client}).Read(
		context.Background(),
		resource.ReadRequest{State: imported.State},
		read,
	)

	if read.Diagnostics.HasError() {
		t.Fatal(detailsOf(read.Diagnostics))
	}

	var refreshed smsAuthenticatorConfigurationResourceModel
	read.State.Get(context.Background(), &refreshed)

	if refreshed.DefaultCountryCode.ValueString() != "NZ" {
		t.Errorf("default_country_code is %v; an imported resource has to carry what the tenant holds", refreshed.DefaultCountryCode)
	}

	if len(refreshed.SmsCountryCodes.Elements()) != 2 {
		t.Errorf("sms_country_codes is %v, want the two codes the API returned", refreshed.SmsCountryCodes)
	}

	if len(refreshed.AllowedCustomSmsVariables.Elements()) != 1 {
		t.Errorf("allowed_custom_sms_variables is %v, want the variable the API returned", refreshed.AllowedCustomSmsVariables)
	}
}

func TestSmsUpdateLeavesRemoteOnlyValuesAloneOnTheWire(t *testing.T) {
	client, stub := newStubAPI(t, map[string][]stubResponse{
		smsPatchRoute: okResponse(smsWithRemoteExtras),
	})

	resourceSchema := resourceSchemaOf(t, &smsAuthenticatorConfigurationResource{})

	state := tfsdk.State{Schema: resourceSchema, Raw: smsConfigValue(t, map[string]tftypes.Value{
		"sms_provider":         tftypes.NewValue(tftypes.String, "TWILIO"),
		"default_country_code": tftypes.NewValue(tftypes.String, "NZ"),
		"twilio_credentials":   twilioTftypesBlock(t, ""),
	})}

	configValue := smsConfigValue(t, map[string]tftypes.Value{
		"sms_provider":       tftypes.NewValue(tftypes.String, "TWILIO"),
		"twilio_credentials": twilioTftypesBlock(t, ""),
	})

	planValue := smsConfigValue(t, map[string]tftypes.Value{
		"sms_provider":         tftypes.NewValue(tftypes.String, "TWILIO"),
		"default_country_code": tftypes.NewValue(tftypes.String, "NZ"),
		"twilio_credentials":   twilioTftypesBlock(t, ""),
	})

	resp := &resource.UpdateResponse{State: tfsdk.State{Schema: resourceSchema}}
	(&smsAuthenticatorConfigurationResource{client: client}).Update(
		context.Background(),
		resource.UpdateRequest{
			Plan:   tfsdk.Plan{Schema: resourceSchema, Raw: planValue},
			Config: tfsdk.Config{Schema: resourceSchema, Raw: configValue},
			State:  state,
		},
		resp,
	)

	if resp.Diagnostics.HasError() {
		t.Fatal(detailsOf(resp.Diagnostics))
	}

	if body := stub.bodyOf(t, smsPatchRoute); strings.Contains(body, "defaultCountryCode") {
		t.Errorf("the patch carries defaultCountryCode (%s); the plan repeating a prior value is not the practitioner configuring it", body)
	}

	var applied smsAuthenticatorConfigurationResourceModel
	resp.State.Get(context.Background(), &applied)

	if applied.DefaultCountryCode.ValueString() != "NZ" {
		t.Errorf("default_country_code is %v; the applied state has to match the plan Terraform proposed", applied.DefaultCountryCode)
	}
}

const smsClearedRestrictionJson = `{"authenticatorId":"auth_1","isActive":true,"verificationMethod":"SMS",` +
	`"smsProvider":"TWILIO","smsCountryCodes":[],` +
	`"twilioCredentials":{"messagingServiceSid":"MG123","accountSid":"AC123"}}`

func TestSmsClearingTheRestrictionDropsTheUnmanagedDefaultFromState(t *testing.T) {
	client, stub := newStubAPI(t, map[string][]stubResponse{
		smsPatchRoute: okResponse(smsClearedRestrictionJson),
	})

	resourceSchema := resourceSchemaOf(t, &smsAuthenticatorConfigurationResource{})

	state := tfsdk.State{Schema: resourceSchema, Raw: smsConfigValue(t, map[string]tftypes.Value{
		"sms_provider":         tftypes.NewValue(tftypes.String, "TWILIO"),
		"sms_country_codes":    countryCodeSet(t, "AU", "NZ"),
		"default_country_code": tftypes.NewValue(tftypes.String, "NZ"),
		"twilio_credentials":   twilioTftypesBlock(t, ""),
	})}

	configValue := smsConfigValue(t, map[string]tftypes.Value{
		"sms_provider":       tftypes.NewValue(tftypes.String, "TWILIO"),
		"sms_country_codes":  countryCodeSet(t),
		"twilio_credentials": twilioTftypesBlock(t, ""),
	})

	planValue := smsConfigValue(t, map[string]tftypes.Value{
		"sms_provider":         tftypes.NewValue(tftypes.String, "TWILIO"),
		"sms_country_codes":    countryCodeSet(t),
		"default_country_code": tftypes.NewValue(tftypes.String, tftypes.UnknownValue),
		"twilio_credentials":   twilioTftypesBlock(t, ""),
	})

	resp := &resource.UpdateResponse{State: tfsdk.State{Schema: resourceSchema}}
	(&smsAuthenticatorConfigurationResource{client: client}).Update(
		context.Background(),
		resource.UpdateRequest{
			Plan:   tfsdk.Plan{Schema: resourceSchema, Raw: planValue},
			Config: tfsdk.Config{Schema: resourceSchema, Raw: configValue},
			State:  state,
		},
		resp,
	)

	if resp.Diagnostics.HasError() {
		t.Fatal(detailsOf(resp.Diagnostics))
	}

	var applied smsAuthenticatorConfigurationResourceModel
	resp.State.Get(context.Background(), &applied)

	if !applied.DefaultCountryCode.IsNull() {
		t.Errorf("default_country_code is %v; clearing the restriction cascades the stored default away, so state must not keep claiming it",
			applied.DefaultCountryCode)
	}

	if body := stub.bodyOf(t, smsPatchRoute); strings.Contains(body, "defaultCountryCode") {
		t.Errorf("the patch carries defaultCountryCode (%s); the default is unmanaged and the cascade is the API's own", body)
	}
}

func TestSmsDefaultCountryCodeIsPlannedUnknownWhenTheRestrictionIsEmptied(t *testing.T) {
	resourceSchema := resourceSchemaOf(t, &smsAuthenticatorConfigurationResource{})

	planned := planModifiedDefaultCountryCode(t, resourceSchema, countryCodeSet(t), types.StringValue("NZ"))
	if !planned.IsUnknown() {
		t.Errorf("default_country_code planned as %v, want unknown so the write can take the API's answer", planned)
	}
}

func TestSmsDefaultCountryCodeKeepsItsPriorPlanWhenTheRestrictionIsNotEmptied(t *testing.T) {
	resourceSchema := resourceSchemaOf(t, &smsAuthenticatorConfigurationResource{})

	planned := planModifiedDefaultCountryCode(t, resourceSchema, countryCodeSet(t, "AU", "NZ"), types.StringValue("NZ"))
	if planned.IsUnknown() || planned.ValueString() != "NZ" {
		t.Errorf("default_country_code planned as %v, want the prior value left alone", planned)
	}
}

func TestSmsDefaultCountryCodeKeepsItsPriorPlanWhenTheRestrictionIsOmitted(t *testing.T) {
	resourceSchema := resourceSchemaOf(t, &smsAuthenticatorConfigurationResource{})

	planned := planModifiedDefaultCountryCode(t, resourceSchema,
		tftypes.NewValue(tftypes.Set{ElementType: tftypes.String}, nil), types.StringValue("NZ"))
	if planned.IsUnknown() || planned.ValueString() != "NZ" {
		t.Errorf("default_country_code planned as %v, want the prior value left alone", planned)
	}
}

func TestSmsDefaultCountryCodeIsPlannedUnknownWhenTheRestrictionNarrowsPastIt(t *testing.T) {
	resourceSchema := resourceSchemaOf(t, &smsAuthenticatorConfigurationResource{})

	planned := planModifiedDefaultCountryCode(t, resourceSchema, countryCodeSet(t, "AU", "GB"), types.StringValue("NZ"))
	if !planned.IsUnknown() {
		t.Errorf("default_country_code planned as %v, want unknown so the write can take the API's answer", planned)
	}
}

func TestSmsDefaultCountryCodeKeepsItsPriorPlanWhenTheRestrictionStillAllowsIt(t *testing.T) {
	resourceSchema := resourceSchemaOf(t, &smsAuthenticatorConfigurationResource{})

	planned := planModifiedDefaultCountryCode(t, resourceSchema, countryCodeSet(t, "AU", "NZ", "GB"), types.StringValue("NZ"))
	if planned.IsUnknown() || planned.ValueString() != "NZ" {
		t.Errorf("default_country_code planned as %v, want the prior value left alone", planned)
	}
}

func TestSmsDefaultCountryCodeKeepsItsPriorPlanWhenNoDefaultIsStored(t *testing.T) {
	resourceSchema := resourceSchemaOf(t, &smsAuthenticatorConfigurationResource{})

	planned := planModifiedDefaultCountryCode(t, resourceSchema, countryCodeSet(t), types.StringNull())
	if planned.IsUnknown() {
		t.Errorf("default_country_code planned as unknown, want the null prior left alone when nothing is stored")
	}
}

func TestSmsDefaultCountryCodeKeepsItsPriorPlanWhenTheRestrictionIsUnknown(t *testing.T) {
	resourceSchema := resourceSchemaOf(t, &smsAuthenticatorConfigurationResource{})

	unknownCodes := tftypes.NewValue(tftypes.Set{ElementType: tftypes.String}, tftypes.UnknownValue)
	planned := planModifiedDefaultCountryCode(t, resourceSchema, unknownCodes, types.StringValue("NZ"))
	if planned.IsUnknown() || planned.ValueString() != "NZ" {
		t.Errorf("default_country_code planned as %v, want the prior value left alone while the restriction is unresolved", planned)
	}
}

func TestSmsDefaultCountryCodeKeepsItsPriorPlanWhenARestrictionElementIsUnknown(t *testing.T) {
	resourceSchema := resourceSchemaOf(t, &smsAuthenticatorConfigurationResource{})

	partlyUnknown := tftypes.NewValue(tftypes.Set{ElementType: tftypes.String}, []tftypes.Value{
		tftypes.NewValue(tftypes.String, "AU"),
		tftypes.NewValue(tftypes.String, tftypes.UnknownValue),
	})
	planned := planModifiedDefaultCountryCode(t, resourceSchema, partlyUnknown, types.StringValue("NZ"))
	if planned.IsUnknown() || planned.ValueString() != "NZ" {
		t.Errorf("default_country_code planned as %v, want the prior value left alone while an element is unresolved", planned)
	}
}

func TestPushWebhookUrlIsPlannedUnknownWhenSwitchingAwayFromWebhook(t *testing.T) {
	resourceSchema := resourceSchemaOf(t, &pushAuthenticatorConfigurationResource{})

	planned := planModifiedWebhookUrl(t, resourceSchema, "WEBHOOK", "FIREBASE", types.StringValue("https://example.com/push"))
	if !planned.IsUnknown() {
		t.Errorf("webhook_url planned as %v, want unknown so the write can take the API's null", planned)
	}
}

func TestPushWebhookUrlKeepsItsPriorPlanWhenTheProviderStaysWebhook(t *testing.T) {
	resourceSchema := resourceSchemaOf(t, &pushAuthenticatorConfigurationResource{})

	planned := planModifiedWebhookUrl(t, resourceSchema, "WEBHOOK", "WEBHOOK", types.StringValue("https://example.com/push"))
	if planned.IsUnknown() || planned.ValueString() != "https://example.com/push" {
		t.Errorf("webhook_url planned as %v, want the prior endpoint left alone", planned)
	}
}

func TestPushWebhookUrlKeepsItsPriorPlanWhenTheProviderWasNeverWebhook(t *testing.T) {
	resourceSchema := resourceSchemaOf(t, &pushAuthenticatorConfigurationResource{})

	planned := planModifiedWebhookUrl(t, resourceSchema, "DEFAULT", "FIREBASE", types.StringValue("https://example.com/push"))
	if planned.IsUnknown() || planned.ValueString() != "https://example.com/push" {
		t.Errorf("webhook_url planned as %v, want the prior value left alone when no WEBHOOK endpoint is being superseded", planned)
	}
}

func TestPushWebhookUrlKeepsItsPriorPlanWhenNoEndpointIsStored(t *testing.T) {
	resourceSchema := resourceSchemaOf(t, &pushAuthenticatorConfigurationResource{})

	planned := planModifiedWebhookUrl(t, resourceSchema, "WEBHOOK", "FIREBASE", types.StringNull())
	if planned.IsUnknown() {
		t.Errorf("webhook_url planned as unknown, want the null prior left alone when nothing is stored")
	}
}

func planModifiedWebhookUrl(t *testing.T, resourceSchema schema.Schema, storedProvider string, configuredProvider string, storedUrl types.String) types.String {
	t.Helper()

	endpoint := func() tftypes.Value {
		if storedUrl.IsNull() {
			return tftypes.NewValue(tftypes.String, nil)
		}

		return tftypes.NewValue(tftypes.String, storedUrl.ValueString())
	}

	configValue := resourceObject(t, resourceSchema, map[string]tftypes.Value{
		"push_provider": tftypes.NewValue(tftypes.String, configuredProvider),
	})

	planValue := resourceObject(t, resourceSchema, map[string]tftypes.Value{
		"push_provider": tftypes.NewValue(tftypes.String, configuredProvider),
		"webhook_url":   endpoint(),
	})

	stateValue := resourceObject(t, resourceSchema, map[string]tftypes.Value{
		"push_provider": tftypes.NewValue(tftypes.String, storedProvider),
		"webhook_url":   endpoint(),
	})

	resp := &planmodifier.StringResponse{PlanValue: storedUrl}
	supersededWebhookUrl{}.PlanModifyString(context.Background(), planmodifier.StringRequest{
		Config:      tfsdk.Config{Schema: resourceSchema, Raw: configValue},
		Plan:        tfsdk.Plan{Schema: resourceSchema, Raw: planValue},
		State:       tfsdk.State{Schema: resourceSchema, Raw: stateValue},
		ConfigValue: types.StringNull(),
		PlanValue:   storedUrl,
		StateValue:  storedUrl,
	}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatal(detailsOf(resp.Diagnostics))
	}

	return resp.PlanValue
}

func planModifiedDefaultCountryCode(t *testing.T, resourceSchema schema.Schema, countryCodes tftypes.Value, priorPlan types.String) types.String {
	t.Helper()

	configValue := resourceObject(t, resourceSchema, map[string]tftypes.Value{
		"sms_provider":      tftypes.NewValue(tftypes.String, "TWILIO"),
		"sms_country_codes": countryCodes,
	})

	planValue := resourceObject(t, resourceSchema, map[string]tftypes.Value{
		"sms_provider":         tftypes.NewValue(tftypes.String, "TWILIO"),
		"sms_country_codes":    countryCodes,
		"default_country_code": tftypes.NewValue(tftypes.String, priorPlan.ValueString()),
	})

	stateValue := resourceObject(t, resourceSchema, map[string]tftypes.Value{
		"sms_provider":         tftypes.NewValue(tftypes.String, "TWILIO"),
		"default_country_code": tftypes.NewValue(tftypes.String, priorPlan.ValueString()),
	})

	resp := &planmodifier.StringResponse{PlanValue: priorPlan}
	excludedDefaultCountryCode{}.PlanModifyString(context.Background(), planmodifier.StringRequest{
		Config:      tfsdk.Config{Schema: resourceSchema, Raw: configValue},
		Plan:        tfsdk.Plan{Schema: resourceSchema, Raw: planValue},
		State:       tfsdk.State{Schema: resourceSchema, Raw: stateValue},
		ConfigValue: types.StringNull(),
		PlanValue:   priorPlan,
		StateValue:  priorPlan,
	}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatal(detailsOf(resp.Diagnostics))
	}

	return resp.PlanValue
}

func TestSmsReadFailsClosedWhenTheResponseOmitsTheProvider(t *testing.T) {
	client, _ := newStubAPI(t, map[string][]stubResponse{
		smsGetRoute: okResponse(`{"authenticatorId":"auth_1","isActive":true,"verificationMethod":"SMS"}`),
	})

	resourceSchema := resourceSchemaOf(t, &smsAuthenticatorConfigurationResource{})
	state := tfsdk.State{Schema: resourceSchema, Raw: smsConfigValue(t, map[string]tftypes.Value{
		"sms_provider": tftypes.NewValue(tftypes.String, "TWILIO"),
	})}

	resp := &resource.ReadResponse{State: state}
	(&smsAuthenticatorConfigurationResource{client: client}).Read(
		context.Background(),
		resource.ReadRequest{State: state},
		resp,
	)

	if !resp.Diagnostics.HasError() {
		t.Fatal("a response with no smsProvider should be refused, not written to state as null")
	}

	if !strings.Contains(detailsOf(resp.Diagnostics), "smsProvider") {
		t.Errorf("the diagnostic should name the missing field, got %s", detailsOf(resp.Diagnostics))
	}

	var kept smsAuthenticatorConfigurationResourceModel
	resp.State.Get(context.Background(), &kept)

	if kept.SmsProvider.ValueString() != "TWILIO" {
		t.Errorf("sms_provider is %v; a refused read has to leave state as it was", kept.SmsProvider)
	}
}

func TestPasskeyReadFailsClosedWhenTheResponseOmitsTheRelyingParty(t *testing.T) {
	client, _ := newStubAPI(t, map[string][]stubResponse{
		"GET /authenticator-configurations/passkey": okResponse(
			`{"authenticatorId":"auth_passkey","isActive":true,"verificationMethod":"PASSKEY","expectedOrigins":["https://example.com"]}`),
	})

	resourceSchema := passkeySchema(t)
	state := tfsdk.State{Schema: resourceSchema, Raw: resourceObject(t, resourceSchema, map[string]tftypes.Value{
		"relying_party":    tftypes.NewValue(tftypes.String, "example.com"),
		"expected_origins": originSet("https://example.com"),
	})}

	resp := &resource.ReadResponse{State: state}
	(&passkeyAuthenticatorConfigurationResource{client: client}).Read(
		context.Background(),
		resource.ReadRequest{State: state},
		resp,
	)

	if !resp.Diagnostics.HasError() || !strings.Contains(detailsOf(resp.Diagnostics), "relyingParty") {
		t.Fatalf("a response with no relyingParty should be refused and say so, got %v", detailsOf(resp.Diagnostics))
	}
}

func TestPasskeyReadFailsClosedWhenTheResponseHasNoOrigins(t *testing.T) {
	client, _ := newStubAPI(t, map[string][]stubResponse{
		"GET /authenticator-configurations/passkey": okResponse(
			`{"authenticatorId":"auth_passkey","isActive":true,"verificationMethod":"PASSKEY","relyingParty":"example.com","expectedOrigins":[]}`),
	})

	resourceSchema := passkeySchema(t)
	state := tfsdk.State{Schema: resourceSchema, Raw: resourceObject(t, resourceSchema, map[string]tftypes.Value{
		"relying_party":    tftypes.NewValue(tftypes.String, "example.com"),
		"expected_origins": originSet("https://example.com"),
	})}

	resp := &resource.ReadResponse{State: state}
	(&passkeyAuthenticatorConfigurationResource{client: client}).Read(
		context.Background(),
		resource.ReadRequest{State: state},
		resp,
	)

	if !resp.Diagnostics.HasError() || !strings.Contains(detailsOf(resp.Diagnostics), "expectedOrigins") {
		t.Fatalf("a response with no origins should be refused and say so, got %v", detailsOf(resp.Diagnostics))
	}
}

func TestReadFailsClosedWhenTheResponseOmitsTheAuthenticatorId(t *testing.T) {
	client, _ := newStubAPI(t, map[string][]stubResponse{
		smsGetRoute: okResponse(`{"isActive":true,"verificationMethod":"SMS","smsProvider":"TWILIO"}`),
	})

	resourceSchema := resourceSchemaOf(t, &smsAuthenticatorConfigurationResource{})
	state := tfsdk.State{Schema: resourceSchema, Raw: smsConfigValue(t, map[string]tftypes.Value{
		"sms_provider": tftypes.NewValue(tftypes.String, "TWILIO"),
	})}

	resp := &resource.ReadResponse{State: state}
	(&smsAuthenticatorConfigurationResource{client: client}).Read(
		context.Background(),
		resource.ReadRequest{State: state},
		resp,
	)

	if !resp.Diagnostics.HasError() || !strings.Contains(detailsOf(resp.Diagnostics), "authenticatorId") {
		t.Fatalf("a response with no authenticatorId should be refused and say so, got %v", detailsOf(resp.Diagnostics))
	}
}

func TestIncompleteResponseDiagnosticEchoesNothingFromTheResponse(t *testing.T) {
	detail := missingResponseFieldDiagnostic("SMS", "smsProvider").Detail()

	for _, forbidden := range []string{"MG123", "AC123", "auth_1", "secret"} {
		if strings.Contains(detail, forbidden) {
			t.Errorf("the diagnostic contains %q; it may name the field and nothing else: %s", forbidden, detail)
		}
	}

	if !strings.Contains(detail, "smsProvider") {
		t.Errorf("the diagnostic should name the missing field, got %s", detail)
	}
}
