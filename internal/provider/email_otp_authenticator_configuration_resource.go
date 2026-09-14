package provider

import (
	"context"
	"fmt"

	"github.com/authsignal/authsignal-management-go/v6"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                   = &emailOtpAuthenticatorConfigurationResource{}
	_ resource.ResourceWithConfigure      = &emailOtpAuthenticatorConfigurationResource{}
	_ resource.ResourceWithImportState    = &emailOtpAuthenticatorConfigurationResource{}
	_ resource.ResourceWithValidateConfig = &emailOtpAuthenticatorConfigurationResource{}
)

func NewEmailOtpAuthenticatorConfigurationResource() resource.Resource {
	return &emailOtpAuthenticatorConfigurationResource{}
}

type emailOtpAuthenticatorConfigurationResource struct {
	client *authsignal.Client
}

type emailOtpAuthenticatorConfigurationResourceModel struct {
	AuthenticatorId    types.String `tfsdk:"authenticator_id"`
	VerificationMethod types.String `tfsdk:"verification_method"`
	IsActive           types.Bool   `tfsdk:"is_active"`
	EmailProvider      types.String `tfsdk:"email_provider"`

	SubmissionRateLimitConfiguration types.Object `tfsdk:"submission_rate_limit_configuration"`
	SendingRateLimitConfigurations   types.List   `tfsdk:"sending_rate_limit_configurations"`
	AllowedCustomEmailVariables      types.Set    `tfsdk:"allowed_custom_email_variables"`

	BirdEmailCredentials            types.Object `tfsdk:"bird_email_credentials"`
	BirdEmailCredentialsVersion     types.String `tfsdk:"bird_email_credentials_version"`
	MailjetEmailCredentials         types.Object `tfsdk:"mailjet_email_credentials"`
	MailjetEmailCredentialsVersion  types.String `tfsdk:"mailjet_email_credentials_version"`
	MailgunEmailCredentials         types.Object `tfsdk:"mailgun_email_credentials"`
	MailgunEmailCredentialsVersion  types.String `tfsdk:"mailgun_email_credentials_version"`
	MandrillEmailCredentials        types.Object `tfsdk:"mandrill_email_credentials"`
	MandrillEmailCredentialsVersion types.String `tfsdk:"mandrill_email_credentials_version"`
	SendgridEmailCredentials        types.Object `tfsdk:"sendgrid_email_credentials"`
	SendgridEmailCredentialsVersion types.String `tfsdk:"sendgrid_email_credentials_version"`
	SmtpEmailCredentials            types.Object `tfsdk:"smtp_email_credentials"`
	SmtpEmailCredentialsVersion     types.String `tfsdk:"smtp_email_credentials_version"`
}

const (
	birdEmailCredentialsBlock     = "bird_email_credentials"
	mailjetEmailCredentialsBlock  = "mailjet_email_credentials"
	mailgunEmailCredentialsBlock  = "mailgun_email_credentials"
	mandrillEmailCredentialsBlock = "mandrill_email_credentials"
	sendgridEmailCredentialsBlock = "sendgrid_email_credentials"
	smtpEmailCredentialsBlock     = "smtp_email_credentials"
)

var birdEmailBlock = credentialBlock{
	Name:        birdEmailCredentialsBlock,
	Providers:   []string{"BIRD"},
	Description: "Bird email credentials. Required when `email_provider` is `BIRD` and no Bird credentials are stored yet.",
	Members: []credentialMember{
		{Name: "access_key", Kind: credentialString, Secret: true, Required: true, Description: "Bird access key."},
		{Name: "workspace_id", Kind: credentialString, Required: true, Description: "Bird workspace ID."},
		{Name: "channel_id", Kind: credentialString, Required: true, Description: "Bird channel ID."},
		{Name: "project_id", Kind: credentialString, Required: true, Description: "Bird project ID."},
		{Name: "version_id", Kind: credentialString, Description: "Bird template version ID. It is unrelated to `bird_email_credentials_version`, the credential rotation marker."},
		{Name: "locale", Kind: credentialString, Description: "The locale Bird renders message templates in."},
		{Name: "sender_email", Kind: credentialString, Description: "Address the message is sent from."},
		{Name: "sender_name", Kind: credentialString, Description: "Display name the message is sent from."},
	},
}

var mailjetEmailBlock = credentialBlock{
	Name:        mailjetEmailCredentialsBlock,
	Providers:   []string{"MAILJET"},
	Description: "Mailjet credentials. Required when `email_provider` is `MAILJET` and no Mailjet credentials are stored yet.",
	Members: []credentialMember{
		{Name: "private_key", Kind: credentialString, Secret: true, Required: true, Description: "Mailjet API secret key."},
		{Name: "public_key", Kind: credentialString, Required: true, Description: "Mailjet API key."},
		{Name: "template_id", Kind: credentialInt64, Required: true, Description: "Mailjet template ID."},
	},
}

var mailgunEmailBlock = credentialBlock{
	Name:        mailgunEmailCredentialsBlock,
	Providers:   []string{"MAILGUN"},
	Description: "Mailgun credentials. Required when `email_provider` is `MAILGUN` and no Mailgun credentials are stored yet.",
	Members: []credentialMember{
		{Name: "api_key", Kind: credentialString, Secret: true, Required: true, Description: "Mailgun API key."},
		{Name: "url", Kind: credentialString, Required: true, Description: "The Mailgun API base URL for the region the domain is in."},
		{Name: "template_name", Kind: credentialString, Required: true, Description: "Mailgun template name."},
		{Name: "domain", Kind: credentialString, Required: true, Description: "Mailgun sending domain."},
		{Name: "from", Kind: credentialString, Description: "Address the message is sent from."},
	},
}

var mandrillEmailBlock = credentialBlock{
	Name:        mandrillEmailCredentialsBlock,
	Providers:   []string{"MANDRILL"},
	Description: "Mandrill credentials. Required when `email_provider` is `MANDRILL` and no Mandrill credentials are stored yet.",
	Members: []credentialMember{
		{Name: "api_key", Kind: credentialString, Secret: true, Required: true, Description: "Mandrill API key."},
		{Name: "template_name", Kind: credentialString, Required: true, Description: "Mandrill template name."},
	},
}

var sendgridEmailBlock = credentialBlock{
	Name:        sendgridEmailCredentialsBlock,
	Providers:   []string{"SENDGRID"},
	Description: "SendGrid credentials. Required when `email_provider` is `SENDGRID` and no SendGrid credentials are stored yet.",
	Members: []credentialMember{
		{Name: "api_key", Kind: credentialString, Secret: true, Required: true, Description: "SendGrid API key."},
		{Name: "template_id", Kind: credentialString, Required: true, Description: "SendGrid dynamic template ID."},
		{Name: "from_email", Kind: credentialString, Required: true, Description: "Address the message is sent from."},
		{Name: "from_name", Kind: credentialString, Description: "Display name the message is sent from."},
	},
}

var smtpEmailBlock = credentialBlock{
	Name:        smtpEmailCredentialsBlock,
	Providers:   []string{"SMTP"},
	Description: "SMTP credentials. Required when `email_provider` is `SMTP` and no SMTP credentials are stored yet.",
	Members: []credentialMember{
		{Name: "host", Kind: credentialString, Required: true, Description: "SMTP server hostname."},
		{Name: "port", Kind: credentialInt64, Required: true, Description: "SMTP server port."},
		{Name: "secure", Kind: credentialBool, Required: true, Description: "Whether the connection uses TLS."},
		{Name: "user", Kind: credentialString, Required: true, Description: "SMTP username."},
		{Name: "password", Kind: credentialString, Secret: true, Required: true, Description: "SMTP password."},
		{Name: "from", Kind: credentialString, Required: true, Description: "Address the message is sent from."},
		{Name: "from_name", Kind: credentialString, Description: "Display name the message is sent from."},
		{Name: "reply_to", Kind: credentialString, Description: "Address replies are directed to."},
		{Name: "reply_to_name", Kind: credentialString, Description: "Display name replies are directed to."},
	},
}

var emailOtpCredentialBlocks = []credentialBlock{birdEmailBlock, mailjetEmailBlock, mailgunEmailBlock, mandrillEmailBlock, sendgridEmailBlock, smtpEmailBlock}

func (r *emailOtpAuthenticatorConfigurationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_email_otp_authenticator_configuration"
}

func (r *emailOtpAuthenticatorConfigurationResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	attributes := map[string]schema.Attribute{
		"authenticator_id": schema.StringAttribute{
			Computed: true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"verification_method": schema.StringAttribute{
			Computed: true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"is_active": schema.BoolAttribute{
			Optional: true,
			Computed: true,
		},
		"email_provider": schema.StringAttribute{
			Description: "Email delivery provider. Supply its credential block when first selecting it." + allowedValues(allowedEmailProviders),
			Required:    true,
			Validators: []validator.String{
				stringvalidator.OneOf(allowedEmailProviders...),
			},
		},
		"submission_rate_limit_configuration": schema.SingleNestedAttribute{
			Description: "Code submission limit.",
			Optional:    true,
			Computed:    true,
			Attributes:  rateLimitAttributes(),
		},
		"sending_rate_limit_configurations": schema.ListNestedAttribute{
			Description: "Email delivery limits. All configured limits apply.",
			Optional:    true,
			Computed:    true,
			NestedObject: schema.NestedAttributeObject{
				Attributes: rateLimitAttributes(),
			},
			Validators: []validator.List{
				listvalidator.SizeAtLeast(1),
				listvalidator.SizeAtMost(6),
			},
		},
		"allowed_custom_email_variables": schema.SetAttribute{
			Description: "Custom email template variables. Set to `[]` to clear them.",
			Optional:    true,
			Computed:    true,
			ElementType: types.StringType,
		},
	}

	for _, block := range emailOtpCredentialBlocks {
		for name, attribute := range block.schemaAttributes() {
			attributes[name] = attribute
		}
	}

	resp.Schema = schema.Schema{
		Description: "Manages the tenant's email OTP authenticator configuration. Destroying the resource deletes the configuration.",
		Attributes:  attributes,
	}
}

func (r *emailOtpAuthenticatorConfigurationResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config emailOtpAuthenticatorConfigurationResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(writeOnlyBlockDiagnostics(emailOtpCredentialBlocks, config.credentialObject, config.credentialVersion)...)

	if !isKnownAndSet(config.EmailProvider) {
		return
	}

	provider := config.EmailProvider.ValueString()

	for _, block := range emailOtpCredentialBlocks {
		if block.appliesTo(provider) {
			continue
		}

		if isKnownAndSet(config.credentialObject(block.Name)) {
			resp.Diagnostics.Append(mismatchedCredentialDiagnostic(block.Name, "email_provider", provider))
		}
	}
}

func (m emailOtpAuthenticatorConfigurationResourceModel) credentialObject(name string) types.Object {
	switch name {
	case birdEmailCredentialsBlock:
		return m.BirdEmailCredentials
	case mailjetEmailCredentialsBlock:
		return m.MailjetEmailCredentials
	case mailgunEmailCredentialsBlock:
		return m.MailgunEmailCredentials
	case mandrillEmailCredentialsBlock:
		return m.MandrillEmailCredentials
	case sendgridEmailCredentialsBlock:
		return m.SendgridEmailCredentials
	case smtpEmailCredentialsBlock:
		return m.SmtpEmailCredentials
	default:
		return types.ObjectNull(map[string]attr.Type{})
	}
}

func (m emailOtpAuthenticatorConfigurationResourceModel) credentialVersion(name string) types.String {
	switch name {
	case birdEmailCredentialsBlock:
		return m.BirdEmailCredentialsVersion
	case mailjetEmailCredentialsBlock:
		return m.MailjetEmailCredentialsVersion
	case mailgunEmailCredentialsBlock:
		return m.MailgunEmailCredentialsVersion
	case mandrillEmailCredentialsBlock:
		return m.MandrillEmailCredentialsVersion
	case sendgridEmailCredentialsBlock:
		return m.SendgridEmailCredentialsVersion
	case smtpEmailCredentialsBlock:
		return m.SmtpEmailCredentialsVersion
	default:
		return types.StringNull()
	}
}

func (r *emailOtpAuthenticatorConfigurationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan, config emailOtpAuthenticatorConfigurationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body, diags := emailOtpCreateBody(ctx, plan, config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	response, statusCode, err := r.client.CreateEmailOtpAuthenticatorConfiguration(body)

	if statusCode == 409 {
		resp.Diagnostics.Append(createConflictDiagnostic("authsignal_email_otp_authenticator_configuration", "email_otp", emailOtpAuthenticatorSlug))
		return
	}

	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating email OTP authenticator configuration",
			"Could not create the email OTP authenticator configuration, unexpected error: "+err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(emailOtpResponseDiagnostics(response)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, emailOtpStateAfterWrite(plan, response))...)
}

func (r *emailOtpAuthenticatorConfigurationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state emailOtpAuthenticatorConfigurationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	response, statusCode, err := r.client.GetEmailOtpAuthenticatorConfiguration()

	if statusCode == 404 {
		resp.State.RemoveResource(ctx)
		return
	}

	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read Authsignal Email OTP Authenticator Configuration",
			err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(emailOtpResponseDiagnostics(response)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, emailOtpStateFromResponse(response, state))...)
}

func (r *emailOtpAuthenticatorConfigurationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, config, state emailOtpAuthenticatorConfigurationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body, diags := emailOtpUpdateBody(ctx, plan, config, state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	response, _, err := r.client.UpdateEmailOtpAuthenticatorConfiguration(body)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating email OTP authenticator configuration",
			"Could not update the email OTP authenticator configuration, unexpected error: "+err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(emailOtpResponseDiagnostics(response)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, emailOtpStateAfterWrite(plan, response))...)
}

func (r *emailOtpAuthenticatorConfigurationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	_, statusCode, err := r.client.DeleteEmailOtpAuthenticatorConfiguration()

	if statusCode == 404 {
		return
	}

	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting email OTP authenticator configuration",
			"Could not delete the email OTP authenticator configuration, unexpected error: "+err.Error(),
		)
	}
}

func (r *emailOtpAuthenticatorConfigurationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(importStateSlug(req.ID, emailOtpAuthenticatorSlug)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, emptyEmailOtpModel())...)
}

func emptyEmailOtpModel() emailOtpAuthenticatorConfigurationResourceModel {
	model := emailOtpAuthenticatorConfigurationResourceModel{
		SubmissionRateLimitConfiguration: types.ObjectNull(rateLimitAttributeTypes),
		SendingRateLimitConfigurations:   types.ListNull(rateLimitObjectType()),
		AllowedCustomEmailVariables:      types.SetNull(types.StringType),
	}

	model.BirdEmailCredentials = types.ObjectNull(birdEmailBlock.attributeTypes())
	model.MailjetEmailCredentials = types.ObjectNull(mailjetEmailBlock.attributeTypes())
	model.MailgunEmailCredentials = types.ObjectNull(mailgunEmailBlock.attributeTypes())
	model.MandrillEmailCredentials = types.ObjectNull(mandrillEmailBlock.attributeTypes())
	model.SendgridEmailCredentials = types.ObjectNull(sendgridEmailBlock.attributeTypes())
	model.SmtpEmailCredentials = types.ObjectNull(smtpEmailBlock.attributeTypes())

	return model
}

func (r *emailOtpAuthenticatorConfigurationResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*authsignal.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *authsignal.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = client
}

func emailOtpCreateBody(ctx context.Context, plan emailOtpAuthenticatorConfigurationResourceModel, config emailOtpAuthenticatorConfigurationResourceModel) (authsignal.CreateEmailOtpAuthenticatorConfigurationBody, diag.Diagnostics) {
	var diags diag.Diagnostics

	body := authsignal.CreateEmailOtpAuthenticatorConfigurationBody{
		IsActive:      boolPointer(config.IsActive),
		EmailProvider: plan.EmailProvider.ValueString(),
	}

	if isKnownAndSet(config.AllowedCustomEmailVariables) {
		variables, setDiags := sortedStringsFrom(ctx, config.AllowedCustomEmailVariables)
		diags.Append(setDiags...)
		body.AllowedCustomEmailVariables = authsignal.SetList(variables)
	}

	submission, submissionDiags := rateLimitFrom(ctx, config.SubmissionRateLimitConfiguration)
	diags.Append(submissionDiags...)
	body.SubmissionRateLimitConfiguration = submission

	sending, sendingDiags := rateLimitsFrom(ctx, config.SendingRateLimitConfigurations)
	diags.Append(sendingDiags...)
	if sending != nil {
		body.SendingRateLimitConfigurations = authsignal.SetList(*sending)
	}

	provider := plan.EmailProvider.ValueString()

	for _, block := range emailOtpCredentialBlocks {
		if !block.appliesTo(provider) {
			continue
		}

		values := readCredentialValues(config.credentialObject(block.Name), block)
		if !values.present {
			diags.Append(missingCredentialDiagnostic(block.Name, "email_provider", provider))
			continue
		}

		if missing := block.missingRequired(values); len(missing) > 0 {
			diags.Append(incompleteCredentialDiagnostic(block.Name, missing))
			continue
		}

		applyEmailOtpCreateCredentials(&body, block.Name, values)
	}

	return body, diags
}

func applyEmailOtpCreateCredentials(body *authsignal.CreateEmailOtpAuthenticatorConfigurationBody, name string, values credentialValues) {
	switch name {
	case birdEmailCredentialsBlock:
		body.BirdEmailCredentials = &authsignal.BirdEmailCredentialsCreate{
			AccessKey:   values.stringValue("access_key"),
			WorkspaceId: values.stringValue("workspace_id"),
			ChannelId:   values.stringValue("channel_id"),
			ProjectId:   values.stringValue("project_id"),
			VersionId:   values.stringPointer("version_id"),
			Locale:      values.stringPointer("locale"),
			SenderEmail: values.stringPointer("sender_email"),
			SenderName:  values.stringPointer("sender_name"),
		}
	case mailjetEmailCredentialsBlock:
		body.MailjetEmailCredentials = &authsignal.MailjetEmailCredentialsCreate{
			PrivateKey: values.stringValue("private_key"),
			PublicKey:  values.stringValue("public_key"),
			TemplateId: values.int64Value("template_id"),
		}
	case mailgunEmailCredentialsBlock:
		body.MailgunEmailCredentials = &authsignal.MailgunEmailCredentialsCreate{
			ApiKey:       values.stringValue("api_key"),
			Url:          values.stringValue("url"),
			TemplateName: values.stringValue("template_name"),
			Domain:       values.stringValue("domain"),
			From:         values.stringPointer("from"),
		}
	case mandrillEmailCredentialsBlock:
		body.MandrillEmailCredentials = &authsignal.MandrillEmailCredentialsCreate{
			ApiKey:       values.stringValue("api_key"),
			TemplateName: values.stringValue("template_name"),
		}
	case sendgridEmailCredentialsBlock:
		body.SendgridEmailCredentials = &authsignal.SendgridEmailCredentialsCreate{
			ApiKey:     values.stringValue("api_key"),
			TemplateId: values.stringValue("template_id"),
			FromEmail:  values.stringValue("from_email"),
			FromName:   values.stringPointer("from_name"),
		}
	case smtpEmailCredentialsBlock:
		body.SmtpEmailCredentials = &authsignal.SmtpEmailCredentialsCreate{
			Host:        values.stringValue("host"),
			Port:        values.int64Value("port"),
			Secure:      values.boolValue("secure"),
			User:        values.stringValue("user"),
			Password:    values.stringValue("password"),
			From:        values.stringValue("from"),
			FromName:    values.stringPointer("from_name"),
			ReplyTo:     values.stringPointer("reply_to"),
			ReplyToName: values.stringPointer("reply_to_name"),
		}
	}
}

func emailOtpUpdateBody(ctx context.Context, plan emailOtpAuthenticatorConfigurationResourceModel, config emailOtpAuthenticatorConfigurationResourceModel, state emailOtpAuthenticatorConfigurationResourceModel) (authsignal.UpdateEmailOtpAuthenticatorConfigurationBody, diag.Diagnostics) {
	var diags diag.Diagnostics

	var body authsignal.UpdateEmailOtpAuthenticatorConfigurationBody

	providerChanging := !plan.EmailProvider.Equal(state.EmailProvider)
	provider := plan.EmailProvider.ValueString()

	if providerChanging {
		body.EmailProvider = authsignal.SetValue(provider)
	}

	if isKnownAndSet(config.IsActive) {
		body.IsActive = authsignal.SetValue(config.IsActive.ValueBool())
	}

	if isKnownAndSet(config.AllowedCustomEmailVariables) {
		variables, setDiags := sortedStringsFrom(ctx, config.AllowedCustomEmailVariables)
		diags.Append(setDiags...)
		body.AllowedCustomEmailVariables = authsignal.SetList(variables)
	}

	if isKnownAndSet(config.SubmissionRateLimitConfiguration) {
		submission, submissionDiags := rateLimitFrom(ctx, config.SubmissionRateLimitConfiguration)
		diags.Append(submissionDiags...)
		if submission != nil {
			body.SubmissionRateLimitConfiguration = authsignal.SetValue(*submission)
		}
	}

	if isKnownAndSet(config.SendingRateLimitConfigurations) {
		sending, sendingDiags := rateLimitsFrom(ctx, config.SendingRateLimitConfigurations)
		diags.Append(sendingDiags...)
		if sending != nil {
			body.SendingRateLimitConfigurations = authsignal.SetValue(*sending)
		}
	}

	for _, block := range emailOtpCredentialBlocks {
		values := readCredentialValues(config.credentialObject(block.Name), block)

		decision := credentialPlanFor(block, credentialContext{
			ProviderChanging: providerChanging,
			IsEffective:      block.appliesTo(provider),
			ConfigValues:     values,
			StateObject:      state.credentialObject(block.Name),
			PriorVersion:     state.credentialVersion(block.Name),
			VersionChanged:   !plan.credentialVersion(block.Name).Equal(state.credentialVersion(block.Name)),
		})

		if decision.Send == credentialWrite {
			applyEmailOtpUpdateCredentials(&body, block, values, decision.IncludeSecrets)
		}
	}

	return body, diags
}

func applyEmailOtpUpdateCredentials(body *authsignal.UpdateEmailOtpAuthenticatorConfigurationBody, block credentialBlock, values credentialValues, includeSecrets bool) {
	secret := func(name string) *string {
		if !includeSecrets {
			return nil
		}

		return values.stringPointer(name)
	}

	switch block.Name {
	case birdEmailCredentialsBlock:
		body.BirdEmailCredentials = authsignal.SetValue(authsignal.BirdEmailCredentialsUpdate{
			AccessKey:   secret("access_key"),
			WorkspaceId: values.stringPointer("workspace_id"),
			ChannelId:   values.stringPointer("channel_id"),
			ProjectId:   values.stringPointer("project_id"),
			VersionId:   values.stringPointer("version_id"),
			Locale:      values.stringPointer("locale"),
			SenderEmail: values.stringPointer("sender_email"),
			SenderName:  values.stringPointer("sender_name"),
		})
	case mailjetEmailCredentialsBlock:
		body.MailjetEmailCredentials = authsignal.SetValue(authsignal.MailjetEmailCredentialsUpdate{
			PrivateKey: secret("private_key"),
			PublicKey:  values.stringPointer("public_key"),
			TemplateId: values.int64Pointer("template_id"),
		})
	case mailgunEmailCredentialsBlock:
		body.MailgunEmailCredentials = authsignal.SetValue(authsignal.MailgunEmailCredentialsUpdate{
			ApiKey:       secret("api_key"),
			Url:          values.stringPointer("url"),
			TemplateName: values.stringPointer("template_name"),
			Domain:       values.stringPointer("domain"),
			From:         values.stringPointer("from"),
		})
	case mandrillEmailCredentialsBlock:
		body.MandrillEmailCredentials = authsignal.SetValue(authsignal.MandrillEmailCredentialsUpdate{
			ApiKey:       secret("api_key"),
			TemplateName: values.stringPointer("template_name"),
		})
	case sendgridEmailCredentialsBlock:
		body.SendgridEmailCredentials = authsignal.SetValue(authsignal.SendgridEmailCredentialsUpdate{
			ApiKey:     secret("api_key"),
			TemplateId: values.stringPointer("template_id"),
			FromEmail:  values.stringPointer("from_email"),
			FromName:   values.stringPointer("from_name"),
		})
	case smtpEmailCredentialsBlock:
		body.SmtpEmailCredentials = authsignal.SetValue(authsignal.SmtpEmailCredentialsUpdate{
			Host:        values.stringPointer("host"),
			Port:        values.int64Pointer("port"),
			Secure:      values.boolPointer("secure"),
			User:        values.stringPointer("user"),
			Password:    secret("password"),
			From:        values.stringPointer("from"),
			FromName:    values.stringPointer("from_name"),
			ReplyTo:     values.stringPointer("reply_to"),
			ReplyToName: values.stringPointer("reply_to_name"),
		})
	}
}

func emailOtpStateAfterWrite(plan emailOtpAuthenticatorConfigurationResourceModel, response *authsignal.EmailOtpAuthenticatorConfiguration) emailOtpAuthenticatorConfigurationResourceModel {
	state := plan
	state.AuthenticatorId = types.StringValue(response.AuthenticatorId)
	state.VerificationMethod = types.StringValue(response.VerificationMethod)

	state.IsActive = resolvedValue(plan.IsActive, types.BoolValue(response.IsActive))
	state.SubmissionRateLimitConfiguration = resolvedValue(plan.SubmissionRateLimitConfiguration, rateLimitValue(response.SubmissionRateLimitConfiguration))
	state.SendingRateLimitConfigurations = resolvedValue(plan.SendingRateLimitConfigurations, rateLimitListValue(response.SendingRateLimitConfigurations))
	state.AllowedCustomEmailVariables = resolvedValue(plan.AllowedCustomEmailVariables, stringSetPointerValue(response.AllowedCustomEmailVariables))

	state.BirdEmailCredentials = credentialPlannedState(birdEmailBlock, plan.BirdEmailCredentials)
	state.MailjetEmailCredentials = credentialPlannedState(mailjetEmailBlock, plan.MailjetEmailCredentials)
	state.MailgunEmailCredentials = credentialPlannedState(mailgunEmailBlock, plan.MailgunEmailCredentials)
	state.MandrillEmailCredentials = credentialPlannedState(mandrillEmailBlock, plan.MandrillEmailCredentials)
	state.SendgridEmailCredentials = credentialPlannedState(sendgridEmailBlock, plan.SendgridEmailCredentials)
	state.SmtpEmailCredentials = credentialPlannedState(smtpEmailBlock, plan.SmtpEmailCredentials)

	return state
}

func emailOtpStateFromResponse(response *authsignal.EmailOtpAuthenticatorConfiguration, prior emailOtpAuthenticatorConfigurationResourceModel) emailOtpAuthenticatorConfigurationResourceModel {
	state := emailOtpAuthenticatorConfigurationResourceModel{
		AuthenticatorId:                  types.StringValue(response.AuthenticatorId),
		VerificationMethod:               types.StringValue(response.VerificationMethod),
		IsActive:                         types.BoolValue(response.IsActive),
		EmailProvider:                    types.StringPointerValue(response.EmailProvider),
		SubmissionRateLimitConfiguration: rateLimitValue(response.SubmissionRateLimitConfiguration),
		SendingRateLimitConfigurations:   rateLimitListValue(response.SendingRateLimitConfigurations),
		AllowedCustomEmailVariables:      stringSetPointerValue(response.AllowedCustomEmailVariables),

		BirdEmailCredentialsVersion:     prior.BirdEmailCredentialsVersion,
		MailjetEmailCredentialsVersion:  prior.MailjetEmailCredentialsVersion,
		MailgunEmailCredentialsVersion:  prior.MailgunEmailCredentialsVersion,
		MandrillEmailCredentialsVersion: prior.MandrillEmailCredentialsVersion,
		SendgridEmailCredentialsVersion: prior.SendgridEmailCredentialsVersion,
		SmtpEmailCredentialsVersion:     prior.SmtpEmailCredentialsVersion,
	}

	applyEmailOtpCredentialReadState(&state, response, prior)

	return state
}

func applyEmailOtpCredentialReadState(state *emailOtpAuthenticatorConfigurationResourceModel, response *authsignal.EmailOtpAuthenticatorConfiguration, prior emailOtpAuthenticatorConfigurationResourceModel) {
	birdMetadata := map[string]attr.Value{}
	if response.BirdEmailCredentials != nil {
		birdMetadata = map[string]attr.Value{
			"workspace_id": types.StringPointerValue(response.BirdEmailCredentials.WorkspaceId),
			"channel_id":   types.StringPointerValue(response.BirdEmailCredentials.ChannelId),
			"project_id":   types.StringPointerValue(response.BirdEmailCredentials.ProjectId),
			"version_id":   types.StringPointerValue(response.BirdEmailCredentials.VersionId),
			"locale":       types.StringPointerValue(response.BirdEmailCredentials.Locale),
			"sender_email": types.StringPointerValue(response.BirdEmailCredentials.SenderEmail),
			"sender_name":  types.StringPointerValue(response.BirdEmailCredentials.SenderName),
		}
	}
	state.BirdEmailCredentials = credentialReadState(birdEmailBlock, birdMetadata, response.BirdEmailCredentials != nil, prior.BirdEmailCredentials)

	mailjetMetadata := map[string]attr.Value{}
	if response.MailjetEmailCredentials != nil {
		mailjetMetadata = map[string]attr.Value{
			"public_key":  types.StringPointerValue(response.MailjetEmailCredentials.PublicKey),
			"template_id": types.Int64PointerValue(response.MailjetEmailCredentials.TemplateId),
		}
	}
	state.MailjetEmailCredentials = credentialReadState(mailjetEmailBlock, mailjetMetadata, response.MailjetEmailCredentials != nil, prior.MailjetEmailCredentials)

	mailgunMetadata := map[string]attr.Value{}
	if response.MailgunEmailCredentials != nil {
		mailgunMetadata = map[string]attr.Value{
			"url":           types.StringPointerValue(response.MailgunEmailCredentials.Url),
			"template_name": types.StringPointerValue(response.MailgunEmailCredentials.TemplateName),
			"domain":        types.StringPointerValue(response.MailgunEmailCredentials.Domain),
			"from":          types.StringPointerValue(response.MailgunEmailCredentials.From),
		}
	}
	state.MailgunEmailCredentials = credentialReadState(mailgunEmailBlock, mailgunMetadata, response.MailgunEmailCredentials != nil, prior.MailgunEmailCredentials)

	mandrillMetadata := map[string]attr.Value{}
	if response.MandrillEmailCredentials != nil {
		mandrillMetadata = map[string]attr.Value{
			"template_name": types.StringPointerValue(response.MandrillEmailCredentials.TemplateName),
		}
	}
	state.MandrillEmailCredentials = credentialReadState(mandrillEmailBlock, mandrillMetadata, response.MandrillEmailCredentials != nil, prior.MandrillEmailCredentials)

	sendgridMetadata := map[string]attr.Value{}
	if response.SendgridEmailCredentials != nil {
		sendgridMetadata = map[string]attr.Value{
			"template_id": types.StringPointerValue(response.SendgridEmailCredentials.TemplateId),
			"from_email":  types.StringPointerValue(response.SendgridEmailCredentials.FromEmail),
			"from_name":   types.StringPointerValue(response.SendgridEmailCredentials.FromName),
		}
	}
	state.SendgridEmailCredentials = credentialReadState(sendgridEmailBlock, sendgridMetadata, response.SendgridEmailCredentials != nil, prior.SendgridEmailCredentials)

	smtpMetadata := map[string]attr.Value{}
	if response.SmtpEmailCredentials != nil {
		smtpMetadata = map[string]attr.Value{
			"host":          types.StringPointerValue(response.SmtpEmailCredentials.Host),
			"port":          types.Int64PointerValue(response.SmtpEmailCredentials.Port),
			"secure":        types.BoolPointerValue(response.SmtpEmailCredentials.Secure),
			"user":          types.StringPointerValue(response.SmtpEmailCredentials.User),
			"from":          types.StringPointerValue(response.SmtpEmailCredentials.From),
			"from_name":     types.StringPointerValue(response.SmtpEmailCredentials.FromName),
			"reply_to":      types.StringPointerValue(response.SmtpEmailCredentials.ReplyTo),
			"reply_to_name": types.StringPointerValue(response.SmtpEmailCredentials.ReplyToName),
		}
	}
	state.SmtpEmailCredentials = credentialReadState(smtpEmailBlock, smtpMetadata, response.SmtpEmailCredentials != nil, prior.SmtpEmailCredentials)
}
