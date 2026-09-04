package provider

import (
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"sort"
	"strings"

	"github.com/authsignal/authsignal-management-go/v6"
)

// The flow attribute holds the body the API's publish endpoint accepts, verbatim: an `actionNodes`
// array and the flat `rules` array those nodes reference. Nothing here reshapes a flow. Publishing
// hands the parsed document straight to the API, and a read composes the same two arrays back from
// the action configuration and the action's rules, so the portal export, this provider and the API
// all describe a flow the same way. `expectedFlowVersion` is the one part of the publish body that
// is not in the document: the provider adds it from the version it last read.

const (
	actionTypeLegacy    = "LEGACY"
	actionTypeMultiStep = "MULTI_STEP_AUTHENTICATION"

	flowNodeTypeRule = "RULE"
	flowMaxRules     = 98
)

var (
	flowRuleIdPattern = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,64}$`)
	// The API's pattern for rule names, minus the empty string it would otherwise allow.
	flowRuleNamePattern = regexp.MustCompile(`^[\w !?@#$%^&*(){}:;"'<>,.+=/\-\[\]]{1,256}$`)
)

// flowError is one broken invariant, located by the JSON path of the offending value within the
// flow document (for example `actionNodes[0].nodeId`). An empty path means the document as a whole.
type flowError struct {
	Path    string
	Message string
}

func (e flowError) Error() string {
	if e.Path == "" {
		return e.Message
	}

	return e.Path + ": " + e.Message
}

// flowDocument is a parsed flow, ready to publish.
type flowDocument struct {
	ActionNodes []authsignal.ActionNode
	Rules       []authsignal.ActionFlowRule
}

// flowNode is one entry of `actionNodes`. Only nodeId and nodeType have a meaning here; the rest of
// the payload is opaque and travels to the API untouched.
type flowNode struct {
	path    string
	nodeId  string
	payload map[string]any
}

// flowRule is one entry of `rules`, with the path it came from so an error can point back at it.
type flowRule struct {
	path string
	rule authsignal.ActionFlowRule
}

// flowArm is one `[ruleId, childNodeId]` pair of a node's `ruleChildNodeIds`.
type flowArm struct {
	path        string
	ruleId      string
	childNodeId string
}

// parseFlow decodes a flow document and checks every invariant the API enforces, reporting each one
// with the JSON path of the offending value. Numbers are kept as json.Number so the publish body
// carries the literals the user wrote. A document with any error is not returned.
func parseFlow(flowJson string) (flowDocument, []flowError) {
	decoder := json.NewDecoder(strings.NewReader(flowJson))
	decoder.UseNumber()

	var raw any
	if err := decoder.Decode(&raw); err != nil {
		return flowDocument{}, []flowError{{Message: "not valid JSON: " + err.Error()}}
	}

	document, ok := raw.(map[string]any)
	if !ok {
		return flowDocument{}, []flowError{{Message: "a flow must be a JSON object with an `actionNodes` array and a `rules` array"}}
	}

	errs := checkFlowKeys(document)

	// Each array is parsed only when the key is there. A key that is present but null falls to the
	// parsers, which reject it, rather than passing as an absent array.
	var nodes []flowNode
	var nodeErrs []flowError
	if raw, has := document["actionNodes"]; has {
		nodes, nodeErrs = parseActionNodes(raw)
		errs = append(errs, nodeErrs...)
	}

	var rules []flowRule
	var ruleErrs []flowError
	if raw, has := document["rules"]; has {
		rules, ruleErrs = parseRules(raw)
		errs = append(errs, ruleErrs...)
	}

	if len(nodeErrs) == 0 && len(ruleErrs) == 0 {
		errs = append(errs, checkFlowReferences(nodes, rules)...)
	}

	if len(errs) > 0 {
		return flowDocument{}, errs
	}

	actionNodes := make([]authsignal.ActionNode, len(nodes))
	for i, node := range nodes {
		actionNodes[i] = node.payload
	}

	publishRules := make([]authsignal.ActionFlowRule, len(rules))
	for i, rule := range rules {
		publishRules[i] = rule.rule
	}

	return flowDocument{ActionNodes: actionNodes, Rules: publishRules}, nil
}

// checkFlowKeys enforces the two keys a flow document has. Keys are visited in order so a document
// with several stray keys reports them the same way every time.
func checkFlowKeys(document map[string]any) []flowError {
	var errs []flowError

	keys := make([]string, 0, len(document))
	for key := range document {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		switch key {
		case "actionNodes", "rules":
		case "expectedFlowVersion":
			errs = append(errs, flowError{key, "is not part of a flow; the provider publishes with the version it last read"})
		default:
			errs = append(errs, flowError{key, "unknown key; a flow has `actionNodes` and `rules` only"})
		}
	}

	if _, has := document["actionNodes"]; !has {
		errs = append(errs, flowError{"actionNodes", "is required"})
	}

	if _, has := document["rules"]; !has {
		errs = append(errs, flowError{"rules", "is required"})
	}

	return errs
}

// parseActionNodes checks the shape every node shares.
func parseActionNodes(raw any) ([]flowNode, []flowError) {
	rawNodes, ok := raw.([]any)
	if !ok {
		return nil, []flowError{{"actionNodes", "must be an array of action nodes"}}
	}

	if len(rawNodes) == 0 {
		return nil, []flowError{{"actionNodes", "a flow needs at least one node"}}
	}

	var errs []flowError
	nodes := make([]flowNode, 0, len(rawNodes))
	nodeIdPaths := map[string]string{}

	for i, rawNode := range rawNodes {
		path := fmt.Sprintf("actionNodes[%d]", i)

		payload, ok := rawNode.(map[string]any)
		if !ok {
			errs = append(errs, flowError{path, "a node must be a JSON object"})
			continue
		}

		nodeId, ok := payload["nodeId"].(string)
		if !ok || nodeId == "" {
			errs = append(errs, flowError{path + ".nodeId", "must be a non-empty string"})
		} else if previous, seen := nodeIdPaths[nodeId]; seen {
			errs = append(errs, flowError{path + ".nodeId", fmt.Sprintf("duplicates %s.nodeId (%q)", previous, nodeId)})
		} else {
			nodeIdPaths[nodeId] = path
		}

		if nodeType, ok := payload["nodeType"].(string); !ok || nodeType == "" {
			errs = append(errs, flowError{path + ".nodeType", "must be a non-empty string"})
		}

		nodes = append(nodes, flowNode{path: path, nodeId: nodeId, payload: payload})
	}

	return nodes, errs
}

// parseRules checks the flat rule list. The API accepts these three keys and silently drops any
// other, so an unknown key is an error rather than something to pass through.
func parseRules(raw any) ([]flowRule, []flowError) {
	rawRules, ok := raw.([]any)
	if !ok {
		return nil, []flowError{{"rules", "must be an array of {ruleId, name, conditions} objects"}}
	}

	var errs []flowError
	rules := make([]flowRule, 0, len(rawRules))
	ruleIdPaths := map[string]string{}

	for i, rawRule := range rawRules {
		path := fmt.Sprintf("rules[%d]", i)

		rule, ok := rawRule.(map[string]any)
		if !ok {
			errs = append(errs, flowError{path, "must be a {ruleId, name, conditions} object"})
			continue
		}

		valid := true

		ruleId, ok := rule["ruleId"].(string)
		if !ok || !flowRuleIdPattern.MatchString(ruleId) {
			errs = append(errs, flowError{path + ".ruleId", "must be 1-64 characters of letters, digits, `_` or `-`"})
			valid = false
		} else if previous, seen := ruleIdPaths[ruleId]; seen {
			errs = append(errs, flowError{path + ".ruleId", fmt.Sprintf("rule %q is already defined at %s", ruleId, previous)})
			valid = false
		} else {
			ruleIdPaths[ruleId] = path
		}

		name, ok := rule["name"].(string)
		if !ok || name == "" {
			errs = append(errs, flowError{path + ".name", "must be a non-empty string"})
			valid = false
		} else if !flowRuleNamePattern.MatchString(name) {
			errs = append(errs, flowError{path + ".name", "must be 1-256 characters of letters, digits, spaces and common punctuation"})
			valid = false
		}

		conditions, hasConditions := rule["conditions"]
		if hasConditions && conditions != nil {
			if _, ok := conditions.(map[string]any); !ok {
				errs = append(errs, flowError{path + ".conditions", "must be a JSON object or absent"})
				valid = false
			}
		}

		for _, key := range sortedKeys(rule) {
			if key != "ruleId" && key != "name" && key != "conditions" {
				errs = append(errs, flowError{path + "." + key, "unknown key; a flow rule has ruleId, name and conditions only"})
				valid = false
			}
		}

		if !valid {
			continue
		}

		rules = append(rules, flowRule{
			path: path,
			rule: authsignal.ActionFlowRule{RuleId: ruleId, Name: name, Conditions: conditions},
		})
	}

	if len(rawRules) > flowMaxRules {
		errs = append(errs, flowError{"rules", fmt.Sprintf("a flow may define at most %d rules, found %d", flowMaxRules, len(rawRules))})
	}

	return rules, errs
}

// checkFlowReferences ties the two arrays together: every rule belongs to exactly one node, and
// every node a flow points at is a node the flow defines.
func checkFlowReferences(nodes []flowNode, rules []flowRule) []flowError {
	var errs []flowError

	nodeIds := make(map[string]bool, len(nodes))
	for _, node := range nodes {
		nodeIds[node.nodeId] = true
	}

	definedRules := make(map[string]bool, len(rules))
	for _, rule := range rules {
		definedRules[rule.rule.RuleId] = true
	}

	referencedBy := map[string]string{}

	for _, node := range nodes {
		errs = append(errs, checkNodeTarget(node, "childNodeId", nodeIds)...)
		errs = append(errs, checkNodeTarget(node, "elseChildNodeId", nodeIds)...)

		arms, armErrs := parseArms(node)
		errs = append(errs, armErrs...)

		for _, arm := range arms {
			if !nodeIds[arm.childNodeId] {
				errs = append(errs, flowError{arm.path + "[1]", fmt.Sprintf("no node has nodeId %q", arm.childNodeId)})
			}

			if !definedRules[arm.ruleId] {
				errs = append(errs, flowError{arm.path + "[0]", fmt.Sprintf("references rule %q, which `rules` does not define", arm.ruleId)})
			}

			if previous, seen := referencedBy[arm.ruleId]; seen {
				errs = append(errs, flowError{arm.path + "[0]", fmt.Sprintf("references rule %q, which %s already references; a rule belongs to one node", arm.ruleId, previous)})
			} else {
				referencedBy[arm.ruleId] = arm.path
			}
		}
	}

	for _, rule := range rules {
		if _, referenced := referencedBy[rule.rule.RuleId]; !referenced {
			errs = append(errs, flowError{rule.path + ".ruleId", fmt.Sprintf("rule %q is not referenced by any node's ruleChildNodeIds", rule.rule.RuleId)})
		}
	}

	return errs
}

// checkNodeTarget resolves one of the node pointers that names a single child. A value that is not
// a string is left alone: node payloads are opaque, and only the API knows which keys each node
// type must carry and what else they may hold.
func checkNodeTarget(node flowNode, key string, nodeIds map[string]bool) []flowError {
	target, ok := node.payload[key].(string)
	if !ok {
		return nil
	}

	if !nodeIds[target] {
		return []flowError{{node.path + "." + key, fmt.Sprintf("no node has nodeId %q", target)}}
	}

	return nil
}

// parseArms reads the `[ruleId, childNodeId]` pairs a RULE node branches on. Only a RULE node needs
// them, but any node carrying the key is read the same way.
func parseArms(node flowNode) ([]flowArm, []flowError) {
	armsPath := node.path + ".ruleChildNodeIds"

	raw, present := node.payload["ruleChildNodeIds"]
	if !present {
		if node.payload["nodeType"] == flowNodeTypeRule {
			return nil, []flowError{{armsPath, "a RULE node must have an array of [ruleId, childNodeId] pairs"}}
		}

		return nil, nil
	}

	rawArms, ok := raw.([]any)
	if !ok {
		return nil, []flowError{{armsPath, "must be an array of [ruleId, childNodeId] pairs"}}
	}

	var errs []flowError
	arms := make([]flowArm, 0, len(rawArms))

	for i, rawArm := range rawArms {
		armPath := fmt.Sprintf("%s[%d]", armsPath, i)

		arm, ok := rawArm.([]any)
		if !ok || len(arm) != 2 {
			errs = append(errs, flowError{armPath, "must be a [ruleId, childNodeId] pair"})
			continue
		}

		ruleId, ruleIdOk := arm[0].(string)
		childNodeId, childOk := arm[1].(string)
		if !ruleIdOk || ruleId == "" || !childOk || childNodeId == "" {
			errs = append(errs, flowError{armPath, "must be a [ruleId, childNodeId] pair of non-empty strings"})
			continue
		}

		arms = append(arms, flowArm{path: armPath, ruleId: ruleId, childNodeId: childNodeId})
	}

	return arms, errs
}

// composeFlow builds the flow document a read stores: the action configuration's nodes as the API
// returned them, and the action's rules projected down to the three keys a flow publishes. Rules
// are sorted by ruleId so an unchanged flow reads back as the same string every time.
func composeFlow(nodes []authsignal.ActionNode, rules []authsignal.RuleResponse) (string, error) {
	actionNodes := nodes
	if actionNodes == nil {
		actionNodes = []authsignal.ActionNode{}
	}

	sorted := make([]authsignal.RuleResponse, len(rules))
	copy(sorted, rules)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].RuleId < sorted[j].RuleId })

	projected := make([]map[string]any, 0, len(sorted))
	for _, rule := range sorted {
		projected = append(projected, projectFlowRule(rule))
	}

	flowJson, err := json.Marshal(map[string]any{"actionNodes": actionNodes, "rules": projected})
	if err != nil {
		return "", err
	}

	return string(flowJson), nil
}

// projectFlowRule drops the fields the API derives for a rule, keeping the three a flow publishes.
func projectFlowRule(rule authsignal.RuleResponse) map[string]any {
	projected := map[string]any{
		"ruleId": rule.RuleId,
		"name":   rule.Name,
	}

	if rule.Conditions != nil {
		projected["conditions"] = rule.Conditions
	}

	return projected
}

// flowsEqual reports whether two flow documents mean the same thing. Comparison is structural and
// does not depend on the invariants parseFlow checks, so a flow read back from a server that holds
// a rule the configuration has dropped still compares as the ordinary difference it is.
func flowsEqual(a, b string) bool {
	canonicalA, okA := canonicalFlow(a)
	canonicalB, okB := canonicalFlow(b)

	if !okA || !okB {
		return a == b
	}

	return reflect.DeepEqual(canonicalA, canonicalB)
}

// canonicalFlow reduces a flow to a value reflect.DeepEqual can compare: rules sorted by ruleId,
// null values dropped (so absent conditions equal null conditions), every number a float64 (so
// `1.0` equals `1`). The order of `actionNodes` is kept because the array is the graph.
func canonicalFlow(flowJson string) (any, bool) {
	decoder := json.NewDecoder(strings.NewReader(flowJson))
	decoder.UseNumber()

	var raw any
	if err := decoder.Decode(&raw); err != nil {
		return nil, false
	}

	canonical, ok := canonicalValue(raw).(map[string]any)
	if !ok {
		return nil, false
	}

	if rules, ok := canonical["rules"].([]any); ok {
		sorted := make([]any, len(rules))
		copy(sorted, rules)
		sort.SliceStable(sorted, func(i, j int) bool { return canonicalRuleId(sorted[i]) < canonicalRuleId(sorted[j]) })
		canonical["rules"] = sorted
	}

	return canonical, true
}

// canonicalRuleId is the sort key of a rule. Anything that is not a rule sorts first and keeps its
// relative order, so a malformed list still compares consistently.
func canonicalRuleId(rule any) string {
	entry, ok := rule.(map[string]any)
	if !ok {
		return ""
	}

	ruleId, _ := entry["ruleId"].(string)

	return ruleId
}

func canonicalValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(typed))
		for key, entry := range typed {
			if entry == nil {
				continue
			}
			out[key] = canonicalValue(entry)
		}
		return out
	case []any:
		out := make([]any, len(typed))
		for i, entry := range typed {
			out[i] = canonicalValue(entry)
		}
		return out
	case json.Number:
		if number, err := typed.Float64(); err == nil {
			return number
		}
		return typed.String()
	case int:
		return float64(typed)
	case int64:
		return float64(typed)
	default:
		return value
	}
}

func sortedKeys(object map[string]any) []string {
	keys := make([]string, 0, len(object))
	for key := range object {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	return keys
}
