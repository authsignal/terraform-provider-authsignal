package provider

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type credentialMemberKind int

const (
	credentialString credentialMemberKind = iota
	credentialInt64
	credentialBool
)

type credentialMember struct {
	Name        string
	Kind        credentialMemberKind
	Secret      bool
	Required    bool
	Description string
}

type credentialBlock struct {
	Name        string
	Providers   []string
	Description string
	Members     []credentialMember
}

func (b credentialBlock) allSecret() bool {
	for _, member := range b.Members {
		if !member.Secret {
			return false
		}
	}

	return true
}

// A fully write-only block has no readable metadata, so its state value is null however many times it
// has been applied. The version marker is the only durable record that it was ever configured, which is
// why the marker is required whenever such a block is set.
func (b credentialBlock) tracked(stateObject types.Object, priorVersion types.String) bool {
	if b.allSecret() {
		return isKnownAndSet(priorVersion)
	}

	return isKnownAndSet(stateObject)
}

func (b credentialBlock) appliesTo(provider string) bool {
	for _, candidate := range b.Providers {
		if candidate == provider {
			return true
		}
	}

	return false
}

func (b credentialBlock) versionName() string {
	return b.Name + "_version"
}

func credentialMemberType(kind credentialMemberKind) attr.Type {
	switch kind {
	case credentialInt64:
		return types.Int64Type
	case credentialBool:
		return types.BoolType
	default:
		return types.StringType
	}
}

func (b credentialBlock) attributeTypes() map[string]attr.Type {
	attributeTypes := map[string]attr.Type{}
	for _, member := range b.Members {
		attributeTypes[member.Name] = credentialMemberType(member.Kind)
	}

	return attributeTypes
}

func (b credentialBlock) objectType() types.ObjectType {
	return types.ObjectType{AttrTypes: b.attributeTypes()}
}

func (b credentialBlock) schemaAttributes() map[string]schema.Attribute {
	blockIsWriteOnly := b.allSecret()
	members := map[string]schema.Attribute{}

	for _, member := range b.Members {
		writeOnly := member.Secret || blockIsWriteOnly
		description := member.Description
		if member.Secret {
			if description != "" {
				description += " "
			}
			description += "Requires Terraform 1.11 or later; supply it through an ephemeral, sensitive variable."
		}

		// Terraform cannot require a write-only attribute on a later plan, because it never stored one.
		// Completeness is checked in ValidateConfig instead.
		required := member.Required && !writeOnly

		switch member.Kind {
		case credentialInt64:
			members[member.Name] = schema.Int64Attribute{
				Description: description,
				Required:    required,
				Optional:    !required,
				WriteOnly:   writeOnly,
			}
		case credentialBool:
			members[member.Name] = schema.BoolAttribute{
				Description: description,
				Required:    required,
				Optional:    !required,
				WriteOnly:   writeOnly,
			}
		default:
			members[member.Name] = schema.StringAttribute{
				Description: description,
				Required:    required,
				Optional:    !required,
				Sensitive:   member.Secret,
				WriteOnly:   writeOnly,
			}
		}
	}

	return map[string]schema.Attribute{
		b.Name: schema.SingleNestedAttribute{
			Description: b.Description + versionRequirementDescription(b),
			Optional:    true,
			WriteOnly:   blockIsWriteOnly,
			Attributes:  members,
		},
		b.versionName(): schema.StringAttribute{
			Description: fmt.Sprintf("Credential rotation marker for `%s`. Change it to resend the credentials. Use an opaque counter or date, never a secret.", b.Name),
			Optional:    true,
		},
	}
}

func versionRequirementDescription(b credentialBlock) string {
	if !b.allSecret() {
		return ""
	}

	return fmt.Sprintf(" This block is entirely write-only, so `%s` is required when it is set.", b.versionName())
}

type credentialValues struct {
	present bool
	members map[string]attr.Value
}

func (v credentialValues) has(name string) bool {
	_, ok := v.members[name]

	return ok
}

func (v credentialValues) stringValue(name string) string {
	value, ok := v.members[name].(types.String)
	if !ok {
		return ""
	}

	return value.ValueString()
}

func (v credentialValues) stringPointer(name string) *string {
	if !v.has(name) {
		return nil
	}

	value := v.stringValue(name)

	return &value
}

func (v credentialValues) int64Value(name string) int64 {
	value, ok := v.members[name].(types.Int64)
	if !ok {
		return 0
	}

	return value.ValueInt64()
}

func (v credentialValues) int64Pointer(name string) *int64 {
	if !v.has(name) {
		return nil
	}

	value := v.int64Value(name)

	return &value
}

func (v credentialValues) boolValue(name string) bool {
	value, ok := v.members[name].(types.Bool)
	if !ok {
		return false
	}

	return value.ValueBool()
}

func (v credentialValues) boolPointer(name string) *bool {
	if !v.has(name) {
		return nil
	}

	value := v.boolValue(name)

	return &value
}

func (b credentialBlock) missingRequired(values credentialValues) []string {
	var missing []string

	for _, member := range b.Members {
		if member.Required && !values.has(member.Name) {
			missing = append(missing, member.Name)
		}
	}

	return missing
}

func readCredentialValues(configObject types.Object, block credentialBlock) credentialValues {
	if !isKnownAndSet(configObject) {
		return credentialValues{}
	}

	members := map[string]attr.Value{}
	attributes := configObject.Attributes()

	for _, member := range block.Members {
		value, ok := attributes[member.Name]
		if !ok || !isKnownAndSet(value) {
			continue
		}

		members[member.Name] = value
	}

	return credentialValues{present: true, members: members}
}

// Terraform already nullifies write-only values before a plan reaches the provider. Nulling them again
// here makes "a secret never reaches state" the provider's own guarantee rather than an inherited one.
func credentialPlannedState(block credentialBlock, planned types.Object) types.Object {
	if block.allSecret() {
		return types.ObjectNull(block.attributeTypes())
	}

	if !isKnownAndSet(planned) {
		return types.ObjectNull(block.attributeTypes())
	}

	attributes := planned.Attributes()
	values := map[string]attr.Value{}

	for _, member := range block.Members {
		if member.Secret {
			values[member.Name] = nullValueFor(credentialMemberType(member.Kind))
			continue
		}

		if value, ok := attributes[member.Name]; ok {
			values[member.Name] = value
			continue
		}

		values[member.Name] = nullValueFor(credentialMemberType(member.Kind))
	}

	return types.ObjectValueMust(block.attributeTypes(), values)
}

// Terraform forbids a computed nested attribute from holding a write-only child, so a credential block
// is optional and not computed. An optional attribute's stored value must match the plan, and the plan
// for an unmentioned block is null, so a read must not volunteer a block, or a member of one, that the
// configuration never mentioned: that produces a diff apply can never settle.
func credentialReadState(block credentialBlock, metadata map[string]attr.Value, present bool, prior types.Object) types.Object {
	if !isKnownAndSet(prior) || !present || block.allSecret() {
		return types.ObjectNull(block.attributeTypes())
	}

	priorMembers := prior.Attributes()
	values := map[string]attr.Value{}

	for _, member := range block.Members {
		memberType := credentialMemberType(member.Kind)

		if member.Secret {
			values[member.Name] = nullValueFor(memberType)
			continue
		}

		if tracked, ok := priorMembers[member.Name]; !ok || !isKnownAndSet(tracked) {
			values[member.Name] = nullValueFor(memberType)
			continue
		}

		if value, ok := metadata[member.Name]; ok && value != nil {
			values[member.Name] = value
			continue
		}

		values[member.Name] = nullValueFor(memberType)
	}

	return types.ObjectValueMust(block.attributeTypes(), values)
}

func nullValueFor(attributeType attr.Type) attr.Value {
	switch attributeType {
	case types.Int64Type:
		return types.Int64Null()
	case types.BoolType:
		return types.BoolNull()
	default:
		return types.StringNull()
	}
}

type credentialSend int

const (
	credentialOmit credentialSend = iota
	credentialWrite
)

type credentialDecision struct {
	Send           credentialSend
	IncludeSecrets bool
}

type credentialContext struct {
	ProviderChanging bool
	IsEffective      bool
	ConfigValues     credentialValues
	StateObject      types.Object
	PriorVersion     types.String
	VersionChanged   bool
}

func credentialPlanFor(block credentialBlock, credentialCtx credentialContext) credentialDecision {
	stateHasBlock := block.tracked(credentialCtx.StateObject, credentialCtx.PriorVersion)

	if !credentialCtx.ConfigValues.present {
		return credentialDecision{Send: credentialOmit}
	}

	if !credentialCtx.IsEffective {
		return credentialDecision{Send: credentialOmit}
	}

	isNew := !stateHasBlock
	metadataChanged := credentialMetadataChanged(block, credentialCtx.ConfigValues, credentialCtx.StateObject)

	if credentialCtx.ProviderChanging || isNew || credentialCtx.VersionChanged {
		return credentialDecision{Send: credentialWrite, IncludeSecrets: true}
	}

	if metadataChanged {
		return credentialDecision{Send: credentialWrite}
	}

	return credentialDecision{Send: credentialOmit}
}

func credentialMetadataChanged(block credentialBlock, values credentialValues, stateObject types.Object) bool {
	if block.allSecret() {
		return false
	}

	if !isKnownAndSet(stateObject) {
		return true
	}

	attributes := stateObject.Attributes()

	for _, member := range block.Members {
		if member.Secret {
			continue
		}

		configured, isConfigured := values.members[member.Name]
		stored, isStored := attributes[member.Name]

		if !isConfigured {
			if isStored && isKnownAndSet(stored) {
				return true
			}
			continue
		}

		if !isStored || !configured.Equal(stored) {
			return true
		}
	}

	return false
}
