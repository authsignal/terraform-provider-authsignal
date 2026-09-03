package provider

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/authsignal/authsignal-management-go/v6"
)

// The flow attribute holds a self-contained graph: the action node array the portal exports, where
// every RULE node also carries a `rules` array defining the rules its arms reference. The API is
// wire-shaped and wants `{actionNodes, rules}` instead, so the provider lifts the embedded rules out
// before publishing and embeds the server's rules back into the graph when it reads.

const (
	actionTypeLegacy    = "LEGACY"
	actionTypeMultiStep = "MULTI_STEP_AUTHENTICATION"

	flowNodeTypeRule = "RULE"
	flowMaxRules     = 98
)

var flowRuleIdPattern = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,64}$`)

// flowError is one broken invariant, located by the JSON path of the offending value within the
// flow array (for example `[0].rules[1].ruleId`). An empty path means the document as a whole.
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

// flowDocument is the lifted form of a flow: the nodes without their embedded rules, and the rules
// collected in node and arm order.
type flowDocument struct {
	Nodes []map[string]any
	Rules []authsignal.ActionFlowRule
}

// liftFlow parses the self-contained graph, strips the `rules` arrays off the RULE nodes and checks
// every invariant. Numbers are kept as json.Number so the publish body carries the user's literals.
func liftFlow(flowJson string) (flowDocument, []flowError) {
	var errs []flowError

	decoder := json.NewDecoder(strings.NewReader(flowJson))
	decoder.UseNumber()

	var raw any
	if err := decoder.Decode(&raw); err != nil {
		return flowDocument{}, []flowError{{Message: "not valid JSON: " + err.Error()}}
	}

	rawNodes, ok := raw.([]any)
	if !ok {
		return flowDocument{}, []flowError{{Message: "a flow must be a JSON array of action nodes"}}
	}

	doc := flowDocument{
		Nodes: make([]map[string]any, 0, len(rawNodes)),
		Rules: []authsignal.ActionFlowRule{},
	}

	nodeIdPaths := map[string]string{}
	ruleIdPaths := map[string]string{}

	for i, rawNode := range rawNodes {
		nodePath := fmt.Sprintf("[%d]", i)

		node, ok := rawNode.(map[string]any)
		if !ok {
			errs = append(errs, flowError{nodePath, "a node must be a JSON object"})
			continue
		}

		nodeId, ok := node["nodeId"].(string)
		if !ok || nodeId == "" {
			errs = append(errs, flowError{nodePath + ".nodeId", "must be a non-empty string"})
		} else if previous, seen := nodeIdPaths[nodeId]; seen {
			errs = append(errs, flowError{nodePath + ".nodeId", fmt.Sprintf("duplicates %s.nodeId (%q)", previous, nodeId)})
		} else {
			nodeIdPaths[nodeId] = nodePath
		}

		nodeType, ok := node["nodeType"].(string)
		if !ok || nodeType == "" {
			errs = append(errs, flowError{nodePath + ".nodeType", "must be a non-empty string"})
		}

		lifted := make(map[string]any, len(node))
		for key, value := range node {
			if key != "rules" {
				lifted[key] = value
			}
		}
		doc.Nodes = append(doc.Nodes, lifted)

		rawRules, hasRules := node["rules"]

		if nodeType != flowNodeTypeRule {
			if hasRules {
				errs = append(errs, flowError{nodePath + ".rules", "only RULE nodes carry rules"})
			}
			continue
		}

		armRuleIds, armErrs := liftArmRuleIds(node["ruleChildNodeIds"], nodePath)
		errs = append(errs, armErrs...)

		rules, ruleErrs := liftNodeRules(rawRules, hasRules, nodePath, ruleIdPaths)
		errs = append(errs, ruleErrs...)

		if len(armErrs) == 0 && len(ruleErrs) == 0 {
			errs = append(errs, checkArmsMatchRules(armRuleIds, rules, nodePath)...)
		}

		for _, rule := range rules {
			doc.Rules = append(doc.Rules, rule.rule)
		}
	}

	if len(doc.Rules) > flowMaxRules {
		errs = append(errs, flowError{Message: fmt.Sprintf("a flow may define at most %d rules, found %d", flowMaxRules, len(doc.Rules))})
	}

	if len(errs) > 0 {
		return flowDocument{}, errs
	}

	return doc, nil
}

// liftArmRuleIds reads the ruleId out of every `[ruleId, childNodeId]` arm of a RULE node.
func liftArmRuleIds(rawArms any, nodePath string) ([]string, []flowError) {
	armsPath := nodePath + ".ruleChildNodeIds"

	arms, ok := rawArms.([]any)
	if !ok {
		return nil, []flowError{{armsPath, "must be an array of [ruleId, childNodeId] pairs"}}
	}

	var errs []flowError
	ruleIds := make([]string, 0, len(arms))

	for k, rawArm := range arms {
		armPath := fmt.Sprintf("%s[%d]", armsPath, k)

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

		ruleIds = append(ruleIds, ruleId)
	}

	return ruleIds, errs
}

type liftedRule struct {
	path string
	rule authsignal.ActionFlowRule
}

// liftNodeRules validates the `rules` array of one RULE node. A missing array is the same as an
// empty one. ruleIdPaths is shared across nodes so a ruleId can only be defined once per flow.
func liftNodeRules(rawRules any, hasRules bool, nodePath string, ruleIdPaths map[string]string) ([]liftedRule, []flowError) {
	rulesPath := nodePath + ".rules"

	if !hasRules || rawRules == nil {
		return []liftedRule{}, nil
	}

	rawList, ok := rawRules.([]any)
	if !ok {
		return nil, []flowError{{rulesPath, "must be an array of {ruleId, name, conditions} objects"}}
	}

	var errs []flowError
	rules := make([]liftedRule, 0, len(rawList))

	for k, rawRule := range rawList {
		rulePath := fmt.Sprintf("%s[%d]", rulesPath, k)

		rule, ok := rawRule.(map[string]any)
		if !ok {
			errs = append(errs, flowError{rulePath, "must be a {ruleId, name, conditions} object"})
			continue
		}

		valid := true

		ruleId, ok := rule["ruleId"].(string)
		if !ok || !flowRuleIdPattern.MatchString(ruleId) {
			errs = append(errs, flowError{rulePath + ".ruleId", "must be 1-64 characters of letters, digits, `_` or `-`"})
			valid = false
		} else if previous, seen := ruleIdPaths[ruleId]; seen {
			errs = append(errs, flowError{rulePath + ".ruleId", fmt.Sprintf("rule %q is already defined at %s", ruleId, previous)})
			valid = false
		} else {
			ruleIdPaths[ruleId] = rulePath
		}

		name, ok := rule["name"].(string)
		if !ok || name == "" {
			errs = append(errs, flowError{rulePath + ".name", "must be a non-empty string"})
			valid = false
		}

		conditions, hasConditions := rule["conditions"]
		if hasConditions && conditions != nil {
			if _, ok := conditions.(map[string]any); !ok {
				errs = append(errs, flowError{rulePath + ".conditions", "must be a JSON object or absent"})
				valid = false
			}
		}

		for key := range rule {
			if key != "ruleId" && key != "name" && key != "conditions" {
				errs = append(errs, flowError{rulePath + "." + key, "unknown key; a flow rule has ruleId, name and conditions only"})
				valid = false
			}
		}

		if !valid {
			continue
		}

		rules = append(rules, liftedRule{
			path: rulePath,
			rule: authsignal.ActionFlowRule{RuleId: ruleId, Name: name, Conditions: conditions},
		})
	}

	return rules, errs
}

// checkArmsMatchRules enforces that the rules a RULE node defines are exactly the rules its arms use.
func checkArmsMatchRules(armRuleIds []string, rules []liftedRule, nodePath string) []flowError {
	var errs []flowError

	defined := map[string]bool{}
	for _, rule := range rules {
		defined[rule.rule.RuleId] = true
	}

	referenced := map[string]bool{}
	for k, ruleId := range armRuleIds {
		referenced[ruleId] = true
		if !defined[ruleId] {
			errs = append(errs, flowError{fmt.Sprintf("%s.ruleChildNodeIds[%d]", nodePath, k), fmt.Sprintf("references rule %q, which is not defined in this node's rules", ruleId)})
		}
	}

	for _, rule := range rules {
		if !referenced[rule.rule.RuleId] {
			errs = append(errs, flowError{rule.path + ".ruleId", fmt.Sprintf("rule %q is not referenced by any arm of this node's ruleChildNodeIds", rule.rule.RuleId)})
		}
	}

	return errs
}

// actionNodes returns the lifted nodes in the SDK's shape.
func (doc flowDocument) actionNodes() []authsignal.ActionNode {
	nodes := make([]authsignal.ActionNode, len(doc.Nodes))
	for i, node := range doc.Nodes {
		nodes[i] = node
	}

	return nodes
}

// embedFlow attaches each server rule to the RULE node whose arms reference it, in arm order, and
// marshals the result with Go's sorted keys. The API enforces one rule to one node, so this is
// unambiguous. Rules no arm references cannot live in a self-contained graph; their ids and names
// are returned so the caller can warn about them. Rule fields other than ruleId, name and
// conditions are projected away.
func embedFlow(nodes []authsignal.ActionNode, rules []authsignal.RuleResponse) (string, []string, error) {
	byId := make(map[string]authsignal.RuleResponse, len(rules))
	for _, rule := range rules {
		byId[rule.RuleId] = rule
	}

	referenced := map[string]bool{}
	embedded := make([]any, 0, len(nodes))

	for _, rawNode := range nodes {
		node, ok := rawNode.(map[string]any)
		if !ok || node["nodeType"] != flowNodeTypeRule {
			embedded = append(embedded, rawNode)
			continue
		}

		copied := make(map[string]any, len(node)+1)
		for key, value := range node {
			copied[key] = value
		}

		nodeRules := []any{}
		arms, _ := node["ruleChildNodeIds"].([]any)
		for _, rawArm := range arms {
			arm, ok := rawArm.([]any)
			if !ok || len(arm) == 0 {
				continue
			}

			ruleId, ok := arm[0].(string)
			if !ok {
				continue
			}

			rule, found := byId[ruleId]
			if !found {
				continue
			}

			referenced[ruleId] = true
			nodeRules = append(nodeRules, projectFlowRule(rule))
		}

		copied["rules"] = nodeRules
		embedded = append(embedded, copied)
	}

	var unreferenced []string
	for _, rule := range rules {
		if !referenced[rule.RuleId] {
			unreferenced = append(unreferenced, fmt.Sprintf("%s (%s)", rule.Name, rule.RuleId))
		}
	}

	flowJson, err := json.Marshal(embedded)
	if err != nil {
		return "", nil, err
	}

	return string(flowJson), unreferenced, nil
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

// canonical reduces a lifted flow to a shape reflect.DeepEqual can compare: rules sorted by ruleId,
// null values dropped (so absent conditions equal null conditions), every number a float64 (so
// `1.0` equals `1`). Node order is kept because the array is the graph.
func canonical(doc flowDocument) any {
	nodes := make([]any, len(doc.Nodes))
	for i, node := range doc.Nodes {
		nodes[i] = canonicalValue(node)
	}

	sorted := make([]authsignal.ActionFlowRule, len(doc.Rules))
	copy(sorted, doc.Rules)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].RuleId < sorted[j].RuleId })

	rules := make([]any, len(sorted))
	for i, rule := range sorted {
		entry := map[string]any{"ruleId": rule.RuleId, "name": rule.Name}
		if rule.Conditions != nil {
			entry["conditions"] = rule.Conditions
		}
		rules[i] = canonicalValue(entry)
	}

	return map[string]any{"actionNodes": nodes, "rules": rules}
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
