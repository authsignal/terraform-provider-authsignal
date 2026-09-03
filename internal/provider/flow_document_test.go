package provider

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/authsignal/authsignal-management-go/v6"
)

// contractFlow is the self-contained graph from the plan's contract: one RULE node with two arms,
// a VERIFICATION node, a BLOCK node and a COMPLETE node.
const contractFlow = `[
  {
    "nodeId": "rule-abc",
    "nodeType": "RULE",
    "parentNodeIds": [],
    "ruleChildNodeIds": [["rule-nz", "verify-def"], ["rule-anon", "block-mno"]],
    "elseChildNodeId": "complete-ghi",
    "rules": [
      {"ruleId": "rule-nz",   "name": "From New Zealand", "conditions": {"and": [{"in": [{"var": "ip.location.country.countryCode"}, ["NZ"]]}]}},
      {"ruleId": "rule-anon", "name": "Anonymous IP",     "conditions": {"and": [{"==": [{"var": "ip.isAnonymous"}, true]}]}}
    ]
  },
  {"nodeId": "verify-def", "nodeType": "VERIFICATION", "parentNodeIds": ["rule-abc"], "name": "Sign in",
   "methodConfigurations": {"PASSKEY": {"isEnabled": true}, "EMAIL_OTP": {"isEnabled": true}}, "childNodeId": "complete-ghi"},
  {"nodeId": "block-mno",    "nodeType": "BLOCK",    "parentNodeIds": ["rule-abc"]},
  {"nodeId": "complete-ghi", "nodeType": "COMPLETE", "parentNodeIds": ["rule-abc", "verify-def"]}
]`

// serverNodes decodes a flow the way the SDK does (default json.Unmarshal, float64 numbers).
func serverNodes(t *testing.T, flowJson string) []authsignal.ActionNode {
	t.Helper()

	var nodes []authsignal.ActionNode
	if err := json.Unmarshal([]byte(flowJson), &nodes); err != nil {
		t.Fatalf("bad fixture: %v", err)
	}

	return nodes
}

func serverRule(ruleId string, name string, conditionsJson string) authsignal.RuleResponse {
	rule := authsignal.RuleResponse{
		RuleId:     ruleId,
		Name:       name,
		ActionCode: "sign-in",
		TenantId:   "tenant",
		Type:       "CHALLENGE",
		IsActive:   true,
		Priority:   0,
	}

	if conditionsJson != "" {
		var conditions any
		if err := json.Unmarshal([]byte(conditionsJson), &conditions); err != nil {
			panic(err)
		}
		rule.Conditions = conditions
	}

	return rule
}

func TestLiftFlowSeparatesRulesFromNodes(t *testing.T) {
	doc, errs := liftFlow(contractFlow)
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}

	if len(doc.Nodes) != 4 {
		t.Fatalf("expected 4 nodes, got %d", len(doc.Nodes))
	}

	for i, node := range doc.Nodes {
		if _, has := node["rules"]; has {
			t.Fatalf("node [%d] still carries rules after lifting", i)
		}
	}

	if arms := doc.Nodes[0]["ruleChildNodeIds"].([]any); len(arms) != 2 {
		t.Fatalf("lifting must keep the rest of the node; arms: %v", arms)
	}

	if len(doc.Rules) != 2 {
		t.Fatalf("expected 2 lifted rules, got %d", len(doc.Rules))
	}

	if doc.Rules[0].RuleId != "rule-nz" || doc.Rules[0].Name != "From New Zealand" || doc.Rules[0].Conditions == nil {
		t.Fatalf("bad first rule: %+v", doc.Rules[0])
	}

	if doc.Rules[1].RuleId != "rule-anon" {
		t.Fatalf("bad second rule: %+v", doc.Rules[1])
	}

	// The publish body is wire-shaped: nodes without rules, rules alongside.
	body, err := json.Marshal(authsignal.ActionFlow{ActionNodes: doc.actionNodes(), Rules: doc.Rules})
	if err != nil {
		t.Fatal(err)
	}

	if strings.Contains(string(body), `"rules":[{`) && strings.Count(string(body), `"rules":`) != 1 {
		t.Fatalf("publish body must carry rules once, at the top level: %s", body)
	}
}

func TestLiftFlowKeepsNumberLiteralsAndDropsNullConditions(t *testing.T) {
	flow := `[{"nodeId":"r","nodeType":"RULE","ruleChildNodeIds":[["a","c"]],"elseChildNodeId":"c",
	  "rules":[{"ruleId":"a","name":"A","conditions":null}], "weight": 1.50},
	  {"nodeId":"c","nodeType":"COMPLETE"}]`

	doc, errs := liftFlow(flow)
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}

	if doc.Rules[0].Conditions != nil {
		t.Fatalf("null conditions must lift as absent, got %v", doc.Rules[0].Conditions)
	}

	body, _ := json.Marshal(authsignal.ActionFlow{ActionNodes: doc.actionNodes(), Rules: doc.Rules})
	if !strings.Contains(string(body), `"weight":1.50`) {
		t.Fatalf("number literals must survive the lift: %s", body)
	}

	if strings.Contains(string(body), `"conditions"`) {
		t.Fatalf("absent conditions must be omitted from the publish body: %s", body)
	}
}

func TestLiftFlowReportsEachInvariantWithItsPath(t *testing.T) {
	ruleNode := func(rules string, arms string) string {
		return fmt.Sprintf(`{"nodeId":"r","nodeType":"RULE","ruleChildNodeIds":%s,"elseChildNodeId":"c","rules":%s}`, arms, rules)
	}
	complete := `{"nodeId":"c","nodeType":"COMPLETE"}`
	wrap := func(nodes ...string) string { return "[" + strings.Join(nodes, ",") + "]" }

	tooManyRules := func() string {
		var arms, rules []string
		for i := 0; i < flowMaxRules+1; i++ {
			arms = append(arms, fmt.Sprintf(`["rule-%d","c"]`, i))
			rules = append(rules, fmt.Sprintf(`{"ruleId":"rule-%d","name":"Rule %d"}`, i, i))
		}
		return wrap(ruleNode("["+strings.Join(rules, ",")+"]", "["+strings.Join(arms, ",")+"]"), complete)
	}()

	testCases := []struct {
		name    string
		flow    string
		path    string
		message string
	}{
		{"not json", `{`, "", "not valid JSON"},
		{"not an array", `{"nodeId":"x"}`, "", "must be a JSON array"},
		{"empty array", `[]`, "", "at least one node"},
		{"node not an object", `["x"]`, "[0]", "must be a JSON object"},
		{"missing nodeId", `[{"nodeType":"COMPLETE"}]`, "[0].nodeId", "non-empty string"},
		{"missing nodeType", `[{"nodeId":"c"}]`, "[0].nodeType", "non-empty string"},
		{"duplicate nodeId", wrap(complete, complete), "[1].nodeId", "duplicates [0].nodeId"},
		{"rules on a non-RULE node", `[{"nodeId":"c","nodeType":"COMPLETE","rules":[]}]`, "[0].rules", "only RULE nodes"},
		{"arms missing", `[{"nodeId":"r","nodeType":"RULE","childNodeIdIfTrue":"c"}]`, "[0].ruleChildNodeIds", "array of [ruleId, childNodeId] pairs"},
		{"arm not a pair", wrap(ruleNode(`[]`, `[["a"]]`)), "[0].ruleChildNodeIds[0]", "pair"},
		{"rules not an array", wrap(ruleNode(`{}`, `[["a","c"]]`)), "[0].rules", "must be an array"},
		{"rule not an object", wrap(ruleNode(`["a"]`, `[["a","c"]]`)), "[0].rules[0]", "object"},
		{"bad ruleId characters", wrap(ruleNode(`[{"ruleId":"has space","name":"A"}]`, `[["has space","c"]]`)), "[0].rules[0].ruleId", "1-64 characters"},
		{"ruleId too long", wrap(ruleNode(`[{"ruleId":"`+strings.Repeat("a", 65)+`","name":"A"}]`, `[["a","c"]]`)), "[0].rules[0].ruleId", "1-64 characters"},
		{"empty name", wrap(ruleNode(`[{"ruleId":"a","name":""}]`, `[["a","c"]]`)), "[0].rules[0].name", "non-empty string"},
		{"name with a character the API rejects", wrap(ruleNode(`[{"ruleId":"a","name":"Rule ~ tilde"}]`, `[["a","c"]]`)), "[0].rules[0].name", "1-256 characters"},
		{"name too long", wrap(ruleNode(`[{"ruleId":"a","name":"`+strings.Repeat("n", 257)+`"}]`, `[["a","c"]]`)), "[0].rules[0].name", "1-256 characters"},
		{"conditions not an object", wrap(ruleNode(`[{"ruleId":"a","name":"A","conditions":[1]}]`, `[["a","c"]]`)), "[0].rules[0].conditions", "JSON object or absent"},
		{"unknown rule key", wrap(ruleNode(`[{"ruleId":"a","name":"A","description":"x"}]`, `[["a","c"]]`)), "[0].rules[0].description", "unknown key"},
		{"arm references undefined rule", wrap(ruleNode(`[{"ruleId":"a","name":"A"}]`, `[["a","c"],["b","c"]]`)), "[0].ruleChildNodeIds[1]", `references rule "b"`},
		{"rule not referenced by any arm", wrap(ruleNode(`[{"ruleId":"a","name":"A"},{"ruleId":"b","name":"B"}]`, `[["a","c"]]`)), "[0].rules[1].ruleId", "not referenced by any arm"},
		{"ruleId defined twice in one node", wrap(ruleNode(`[{"ruleId":"a","name":"A"},{"ruleId":"a","name":"A2"}]`, `[["a","c"]]`)), "[0].rules[1].ruleId", "already defined at [0].rules[0]"},
		{"ruleId defined on two nodes", wrap(
			ruleNode(`[{"ruleId":"a","name":"A"}]`, `[["a","c"]]`),
			`{"nodeId":"r2","nodeType":"RULE","ruleChildNodeIds":[["a","c"]],"rules":[{"ruleId":"a","name":"A"}]}`,
			complete), "[1].rules[0].ruleId", "already defined at [0].rules[0]"},
		{"too many rules", tooManyRules, "", "at most 98 rules"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			_, errs := liftFlow(testCase.flow)
			if len(errs) == 0 {
				t.Fatalf("expected an error at %q", testCase.path)
			}

			for _, err := range errs {
				if err.Path == testCase.path && strings.Contains(err.Message, testCase.message) {
					return
				}
			}

			t.Fatalf("expected an error at %q containing %q, got %v", testCase.path, testCase.message, errs)
		})
	}
}

func TestLiftFlowAcceptsRuleNamesWithTheApisPunctuation(t *testing.T) {
	flow := `[{"nodeId":"r","nodeType":"RULE","ruleChildNodeIds":[["a","c"]],"elseChildNodeId":"c",
	  "rules":[{"ruleId":"a","name":"Rule #1: is (\"risky\") & <new>? [$100+ / -50%] {ok}; yes!"}]},{"nodeId":"c","nodeType":"COMPLETE"}]`

	if _, errs := liftFlow(flow); len(errs) > 0 {
		t.Fatalf("the API accepts this name, so must the validator: %v", errs)
	}
}

func TestLiftFlowAcceptsRuleNodeWithoutRulesKey(t *testing.T) {
	flow := `[{"nodeId":"r","nodeType":"RULE","ruleChildNodeIds":[],"elseChildNodeId":"c"},{"nodeId":"c","nodeType":"COMPLETE"}]`

	doc, errs := liftFlow(flow)
	if len(errs) > 0 {
		t.Fatalf("a RULE node with no arms needs no rules: %v", errs)
	}

	if len(doc.Rules) != 0 {
		t.Fatalf("expected no rules, got %v", doc.Rules)
	}
}

func TestEmbedFlowRoundTripsTheContractFlow(t *testing.T) {
	doc, errs := liftFlow(contractFlow)
	if len(errs) > 0 {
		t.Fatal(errs)
	}

	// What the server hands back: the lifted nodes (through a JSON round trip, as the SDK decodes
	// them) and the rules with their server-side fields, in an arbitrary order.
	liftedJson, _ := json.Marshal(doc.actionNodes())
	nodes := serverNodes(t, string(liftedJson))
	rules := []authsignal.RuleResponse{
		serverRule("rule-anon", "Anonymous IP", `{"and":[{"==":[{"var":"ip.isAnonymous"},true]}]}`),
		serverRule("rule-nz", "From New Zealand", `{"and":[{"in":[{"var":"ip.location.country.countryCode"},["NZ"]]}]}`),
	}

	embedded, unreferenced, err := embedFlow(nodes, rules)
	if err != nil {
		t.Fatal(err)
	}

	if len(unreferenced) != 0 {
		t.Fatalf("every rule is referenced, got unreferenced %v", unreferenced)
	}

	if !flowsEqual(contractFlow, embedded) {
		t.Fatalf("embedding the server's rules must reproduce the original flow.\noriginal: %s\nembedded: %s", contractFlow, embedded)
	}

	// Embedding follows arm order, not the server's order, and projects server-only fields away.
	var result []map[string]any
	if err := json.Unmarshal([]byte(embedded), &result); err != nil {
		t.Fatal(err)
	}

	embeddedRules := result[0]["rules"].([]any)
	first := embeddedRules[0].(map[string]any)
	if first["ruleId"] != "rule-nz" {
		t.Fatalf("expected the first arm's rule first, got %v", first["ruleId"])
	}

	for _, key := range []string{"type", "priority", "isActive", "actionCode", "tenantId"} {
		if _, has := first[key]; has {
			t.Fatalf("server-only field %q must not be embedded: %v", key, first)
		}
	}

	if _, has := result[1]["rules"]; has {
		t.Fatalf("non-RULE nodes must not gain a rules array: %v", result[1])
	}
}

func TestEmbedFlowReportsUnreferencedRulesAndOmitsNilConditions(t *testing.T) {
	nodes := serverNodes(t, `[{"nodeId":"r","nodeType":"RULE","parentNodeIds":[],"ruleChildNodeIds":[["a","c"]],"elseChildNodeId":"c"},{"nodeId":"c","nodeType":"COMPLETE","parentNodeIds":["r"]}]`)
	rules := []authsignal.RuleResponse{
		serverRule("a", "A", ""),
		serverRule("stray", "Made in the portal", `{"and":[]}`),
	}

	embedded, unreferenced, err := embedFlow(nodes, rules)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(unreferenced, []string{"Made in the portal (stray)"}) {
		t.Fatalf("expected the stray rule to be reported, got %v", unreferenced)
	}

	expected := `[{"elseChildNodeId":"c","nodeId":"r","nodeType":"RULE","parentNodeIds":[],"ruleChildNodeIds":[["a","c"]],"rules":[{"name":"A","ruleId":"a"}]},{"nodeId":"c","nodeType":"COMPLETE","parentNodeIds":["r"]}]`
	if embedded != expected {
		t.Fatalf("expected sorted-key JSON without a conditions key\nexpected: %s\ngot:      %s", expected, embedded)
	}
}

func TestEmbedFlowOfUnpublishedActionIsEmptyArray(t *testing.T) {
	embedded, unreferenced, err := embedFlow(nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	if embedded != "[]" || len(unreferenced) != 0 {
		t.Fatalf("expected an empty graph, got %q and %v", embedded, unreferenced)
	}
}

func TestEmbedFlowDoesNotMutateTheServerNodes(t *testing.T) {
	nodes := serverNodes(t, `[{"nodeId":"r","nodeType":"RULE","ruleChildNodeIds":[["a","c"]]}]`)

	if _, _, err := embedFlow(nodes, []authsignal.RuleResponse{serverRule("a", "A", "")}); err != nil {
		t.Fatal(err)
	}

	if _, has := nodes[0].(map[string]any)["rules"]; has {
		t.Fatal("embedFlow must copy the nodes it decorates")
	}
}
