package provider

import (
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"reflect"
	"regexp"
	"sort"
	"strings"

	"github.com/authsignal/authsignal-management-go/v6"
)

const (
	actionTypeClassic = "CLASSIC"
	actionTypeFlow    = "FLOW"

	flowNodeTypeRule = "RULE"
	flowMaxRules     = 98

	// The API measures the compact JSON encoding of `actionNodes` in bytes.
	flowMaxActionNodesBytes = 300_000
)

var (
	flowRuleIdPattern   = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,64}$`)
	flowRuleNamePattern = regexp.MustCompile(`^[\w !?@#$%^&*(){}:;"'<>,.+=/\-\[\]]{0,256}$`)
)

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

type flowDocument struct {
	ActionNodes []authsignal.ActionNode
	Rules       []authsignal.ActionFlowRule
}

type flowNode struct {
	path    string
	nodeId  string
	payload map[string]any
}

type flowRule struct {
	path string
	rule authsignal.ActionFlowRule
}

type flowArm struct {
	path        string
	ruleId      string
	childNodeId string
}

func parseFlow(flowJson string) (flowDocument, []flowError) {
	decoder := json.NewDecoder(strings.NewReader(flowJson))
	decoder.UseNumber()

	var raw any
	if err := decoder.Decode(&raw); err != nil {
		return flowDocument{}, []flowError{{Message: "not valid JSON: " + err.Error()}}
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return flowDocument{}, []flowError{{Message: "not valid JSON: trailing content after the flow document"}}
	}

	document, ok := raw.(map[string]any)
	if !ok {
		return flowDocument{}, []flowError{{Message: "a flow must be a JSON object with an `actionNodes` array and a `rules` array"}}
	}

	errs := checkFlowKeys(document)

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
		// The payload was decoded with UseNumber(), so re-marshalling it keeps every number literal.
		payload, err := json.Marshal(node.payload)
		if err != nil {
			return flowDocument{}, []flowError{{Path: node.path, Message: "could not be re-encoded: " + err.Error()}}
		}

		actionNodes[i] = payload
	}

	encodedNodes, err := json.Marshal(actionNodes)
	if err != nil {
		return flowDocument{}, []flowError{{Path: "actionNodes", Message: "could not be re-encoded: " + err.Error()}}
	}

	if len(encodedNodes) > flowMaxActionNodesBytes {
		return flowDocument{}, []flowError{{"actionNodes", fmt.Sprintf("a flow's nodes may encode to at most %d bytes, found %d", flowMaxActionNodesBytes, len(encodedNodes))}}
	}

	publishRules := make([]authsignal.ActionFlowRule, len(rules))
	for i, rule := range rules {
		publishRules[i] = rule.rule
	}

	return flowDocument{ActionNodes: actionNodes, Rules: publishRules}, nil
}

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

// The API rejects unknown rule keys, so the check below mirrors it.
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
		if !ok {
			errs = append(errs, flowError{path + ".name", "must be a string"})
			valid = false
		} else if !flowRuleNamePattern.MatchString(name) {
			errs = append(errs, flowError{path + ".name", "must be 0-256 characters of letters, digits, spaces and common punctuation"})
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
		errs = append(errs, checkNodeTargetPairs(node, "verificationMethodChildNodeIds", nodeIds)...)
		errs = append(errs, checkNodeTargetPairs(node, "buttonChildNodeIds", nodeIds)...)

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

func checkNodeTargetPairs(node flowNode, key string, nodeIds map[string]bool) []flowError {
	raw, present := node.payload[key]
	if !present {
		return nil
	}

	path := node.path + "." + key
	pairs, ok := raw.([]any)
	if !ok {
		return []flowError{{path, "must be an array of two-string pairs"}}
	}

	var errs []flowError
	for i, rawPair := range pairs {
		pairPath := fmt.Sprintf("%s[%d]", path, i)
		pair, ok := rawPair.([]any)
		if !ok || len(pair) != 2 {
			errs = append(errs, flowError{pairPath, "must be a pair of two non-empty strings"})
			continue
		}

		first, firstOk := pair[0].(string)
		childNodeId, childOk := pair[1].(string)
		if !firstOk || first == "" || !childOk || childNodeId == "" {
			errs = append(errs, flowError{pairPath, "must be a pair of two non-empty strings"})
			continue
		}

		if !nodeIds[childNodeId] {
			errs = append(errs, flowError{pairPath + "[1]", fmt.Sprintf("no node has nodeId %q", childNodeId)})
		}
	}

	return errs
}

func checkNodeTarget(node flowNode, key string, nodeIds map[string]bool) []flowError {
	raw, present := node.payload[key]
	if !present {
		return nil
	}

	path := node.path + "." + key

	target, ok := raw.(string)
	if !ok || target == "" {
		return []flowError{{path, "must be a non-empty node id string"}}
	}

	if !nodeIds[target] {
		return []flowError{{path, fmt.Sprintf("no node has nodeId %q", target)}}
	}

	return nil
}

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

func composeFlow(nodes []authsignal.ActionNode, rules []authsignal.RuleResponse) (string, error) {
	return composeFlowWithRuleOrder(nodes, rules, nil)
}

func composeFlowWithRuleOrder(nodes []authsignal.ActionNode, rules []authsignal.RuleResponse, preferredRuleIds []string) (string, error) {
	actionNodes := nodes
	if actionNodes == nil {
		actionNodes = []authsignal.ActionNode{}
	}

	preferredIndex := make(map[string]int, len(preferredRuleIds))
	for i, ruleId := range preferredRuleIds {
		preferredIndex[ruleId] = i
	}

	sorted := make([]authsignal.RuleResponse, len(rules))
	copy(sorted, rules)
	sort.SliceStable(sorted, func(i, j int) bool {
		iIndex, iPreferred := preferredIndex[sorted[i].RuleId]
		jIndex, jPreferred := preferredIndex[sorted[j].RuleId]
		if iPreferred && jPreferred {
			return iIndex < jIndex
		}
		if iPreferred != jPreferred {
			return iPreferred
		}
		return sorted[i].RuleId < sorted[j].RuleId
	})

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

func flowRuleIds(flowJson string) []string {
	var document struct {
		Rules []struct {
			RuleId string `json:"ruleId"`
		} `json:"rules"`
	}
	if err := json.Unmarshal([]byte(flowJson), &document); err != nil {
		return nil
	}

	ruleIds := make([]string, len(document.Rules))
	for i, rule := range document.Rules {
		ruleIds[i] = rule.RuleId
	}
	return ruleIds
}

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

func flowsEqual(a, b string) bool {
	canonicalA, okA := canonicalFlow(a)
	canonicalB, okB := canonicalFlow(b)

	if !okA || !okB {
		return a == b
	}

	return reflect.DeepEqual(canonicalA, canonicalB)
}

func canonicalFlow(flowJson string) (any, bool) {
	decoder := json.NewDecoder(strings.NewReader(flowJson))
	decoder.UseNumber()

	var raw any
	if err := decoder.Decode(&raw); err != nil {
		return nil, false
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return nil, false
	}

	canonical, ok := canonicalValue(raw).(map[string]any)
	if !ok {
		return nil, false
	}

	if rules, ok := canonical["rules"].([]any); ok {
		sorted := make([]any, len(rules))
		copy(sorted, rules)
		for _, rawRule := range sorted {
			if rule, ok := rawRule.(map[string]any); ok && rule["conditions"] == nil {
				delete(rule, "conditions")
			}
		}
		sort.SliceStable(sorted, func(i, j int) bool { return canonicalRuleId(sorted[i]) < canonicalRuleId(sorted[j]) })
		canonical["rules"] = sorted
	}

	return canonical, true
}

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
		if number, ok := new(big.Rat).SetString(typed.String()); ok {
			return canonicalNumber(number.RatString())
		}
		return canonicalNumber(typed.String())
	case int:
		return canonicalNumber(big.NewRat(int64(typed), 1).RatString())
	case int64:
		return canonicalNumber(big.NewRat(typed, 1).RatString())
	default:
		return value
	}
}

type canonicalNumber string

func sortedKeys(object map[string]any) []string {
	keys := make([]string, 0, len(object))
	for key := range object {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	return keys
}
