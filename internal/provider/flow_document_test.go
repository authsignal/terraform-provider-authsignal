package provider

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/authsignal/authsignal-management-go/v6"
)

// contractFlow is the document from the contract: an actionNodes graph with one RULE node branching
// two ways, and the flat rules array its arms reference.
const contractFlow = `{
  "actionNodes": [
    {
      "nodeId": "rule-abc",
      "nodeType": "RULE",
      "parentNodeIds": [],
      "ruleChildNodeIds": [["rule-nz", "verify-def"], ["rule-anon", "block-mno"]],
      "elseChildNodeId": "complete-ghi"
    },
    {"nodeId": "verify-def", "nodeType": "VERIFICATION", "parentNodeIds": ["rule-abc"], "name": "Sign in",
     "methodConfigurations": {"PASSKEY": {"isEnabled": true}, "EMAIL_OTP": {"isEnabled": true}}, "childNodeId": "complete-ghi"},
    {"nodeId": "block-mno",    "nodeType": "BLOCK",    "parentNodeIds": ["rule-abc"]},
    {"nodeId": "complete-ghi", "nodeType": "COMPLETE", "parentNodeIds": ["rule-abc", "verify-def"]}
  ],
  "rules": [
    {"ruleId": "rule-nz",   "name": "From New Zealand", "conditions": {"and": [{"in": [{"var": "ip.location.country.countryCode"}, ["NZ"]]}]}},
    {"ruleId": "rule-anon", "name": "Anonymous IP",     "conditions": {"and": [{"==": [{"var": "ip.isAnonymous"}, true]}]}}
  ]
}`

// serverNodes decodes action nodes the way the SDK does (default json.Unmarshal, float64 numbers).
func serverNodes(t *testing.T, nodesJson string) []authsignal.ActionNode {
	t.Helper()

	var nodes []authsignal.ActionNode
	if err := json.Unmarshal([]byte(nodesJson), &nodes); err != nil {
		t.Fatalf("bad fixture: %v", err)
	}

	return nodes
}

// serverRule is a rule as the API lists it, with the fields it derives that a flow never publishes.
func serverRule(ruleId string, name string, conditionsJson string) authsignal.RuleResponse {
	rule := authsignal.RuleResponse{
		RuleId:     ruleId,
		Name:       name,
		ActionCode: "sign-in",
		TenantId:   "tenant",
		Type:       "ALLOW",
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

func TestParseFlowPassesTheDocumentThroughUnchanged(t *testing.T) {
	doc, errs := parseFlow(contractFlow)
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}

	if len(doc.ActionNodes) != 4 {
		t.Fatalf("expected 4 nodes, got %d", len(doc.ActionNodes))
	}

	if len(doc.Rules) != 2 {
		t.Fatalf("expected 2 rules, got %d", len(doc.Rules))
	}

	// Node order is the graph, and rules keep the order they were written in.
	first, ok := doc.ActionNodes[0].(map[string]any)
	if !ok || first["nodeId"] != "rule-abc" {
		t.Fatalf("expected the first node first, got %v", doc.ActionNodes[0])
	}

	if doc.Rules[0].RuleId != "rule-nz" || doc.Rules[0].Name != "From New Zealand" || doc.Rules[0].Conditions == nil {
		t.Fatalf("bad first rule: %+v", doc.Rules[0])
	}

	// Nothing is lifted, embedded or renamed: the publish body is the document plus the version.
	body, err := json.Marshal(authsignal.ActionFlow{ActionNodes: doc.ActionNodes, Rules: doc.Rules})
	if err != nil {
		t.Fatal(err)
	}

	if !flowsEqual(contractFlow, string(body)) {
		t.Fatalf("the publish body must mean the same as the document\ndocument: %s\nbody:     %s", contractFlow, body)
	}

	if strings.Contains(string(body), "expectedFlowVersion") {
		t.Fatalf("the document must not carry a flow version: %s", body)
	}
}

func TestParseFlowKeepsNumberLiteralsAndOmitsAbsentConditions(t *testing.T) {
	flow := `{"actionNodes":[{"nodeId":"r","nodeType":"RULE","ruleChildNodeIds":[["a","c"]],"elseChildNodeId":"c","weight":1.50},
	  {"nodeId":"c","nodeType":"COMPLETE"}],
	  "rules":[{"ruleId":"a","name":"A","conditions":null}]}`

	doc, errs := parseFlow(flow)
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}

	if doc.Rules[0].Conditions != nil {
		t.Fatalf("null conditions must parse as absent, got %v", doc.Rules[0].Conditions)
	}

	body, _ := json.Marshal(authsignal.ActionFlow{ActionNodes: doc.ActionNodes, Rules: doc.Rules})
	if !strings.Contains(string(body), `"weight":1.50`) {
		t.Fatalf("number literals must survive the parse: %s", body)
	}

	if strings.Contains(string(body), `"conditions"`) {
		t.Fatalf("absent conditions must be omitted from the publish body: %s", body)
	}
}

func TestParseFlowReportsEachInvariantWithItsPath(t *testing.T) {
	document := func(nodes string, rules string) string {
		return fmt.Sprintf(`{"actionNodes":%s,"rules":%s}`, nodes, rules)
	}
	ruleNode := func(arms string) string {
		return fmt.Sprintf(`{"nodeId":"r","nodeType":"RULE","ruleChildNodeIds":%s,"elseChildNodeId":"c"}`, arms)
	}
	complete := `{"nodeId":"c","nodeType":"COMPLETE"}`
	nodes := func(list ...string) string { return "[" + strings.Join(list, ",") + "]" }

	tooManyRules := func() string {
		var arms, rules []string
		for i := 0; i < flowMaxRules+1; i++ {
			arms = append(arms, fmt.Sprintf(`["rule-%d","c"]`, i))
			rules = append(rules, fmt.Sprintf(`{"ruleId":"rule-%d","name":"Rule %d"}`, i, i))
		}
		return document(nodes(ruleNode("["+strings.Join(arms, ",")+"]"), complete), "["+strings.Join(rules, ",")+"]")
	}()

	testCases := []struct {
		name    string
		flow    string
		path    string
		message string
	}{
		{"not json", `{`, "", "not valid JSON"},
		{"an array, as the old shape was", `[{"nodeId":"c","nodeType":"COMPLETE"}]`, "", "must be a JSON object"},
		{"actionNodes missing", `{"rules":[]}`, "actionNodes", "is required"},
		{"rules missing", `{"actionNodes":[` + complete + `]}`, "rules", "is required"},
		{"unknown top-level key", document(nodes(complete), `[]`)[:len(document(nodes(complete), `[]`))-1] + `,"actionType":"MULTI_STEP_AUTHENTICATION"}`, "actionType", "unknown key"},
		{"expectedFlowVersion is not part of the document", `{"actionNodes":[` + complete + `],"rules":[],"expectedFlowVersion":3}`, "expectedFlowVersion", "not part of a flow"},
		{"actionNodes not an array", `{"actionNodes":{},"rules":[]}`, "actionNodes", "must be an array"},
		{"actionNodes null", `{"actionNodes":null,"rules":[]}`, "actionNodes", "must be an array"},
		{"rules null", `{"actionNodes":[` + complete + `],"rules":null}`, "rules", "must be an array"},
		{"empty actionNodes", `{"actionNodes":[],"rules":[]}`, "actionNodes", "at least one node"},
		{"node not an object", document(`["x"]`, `[]`), "actionNodes[0]", "must be a JSON object"},
		{"missing nodeId", document(`[{"nodeType":"COMPLETE"}]`, `[]`), "actionNodes[0].nodeId", "non-empty string"},
		{"missing nodeType", document(`[{"nodeId":"c"}]`, `[]`), "actionNodes[0].nodeType", "non-empty string"},
		{"duplicate nodeId", document(nodes(complete, complete), `[]`), "actionNodes[1].nodeId", "duplicates actionNodes[0].nodeId"},
		{"rules not an array", document(nodes(complete), `{}`), "rules", "must be an array"},
		{"rule not an object", document(nodes(complete), `["a"]`), "rules[0]", "must be a {ruleId, name, conditions} object"},
		{"bad ruleId characters", document(nodes(ruleNode(`[["has space","c"]]`), complete), `[{"ruleId":"has space","name":"A"}]`), "rules[0].ruleId", "1-64 characters"},
		{"ruleId too long", document(nodes(ruleNode(`[["a","c"]]`), complete), `[{"ruleId":"`+strings.Repeat("a", 65)+`","name":"A"}]`), "rules[0].ruleId", "1-64 characters"},
		{"empty name", document(nodes(ruleNode(`[["a","c"]]`), complete), `[{"ruleId":"a","name":""}]`), "rules[0].name", "non-empty string"},
		{"name with a character the API rejects", document(nodes(ruleNode(`[["a","c"]]`), complete), `[{"ruleId":"a","name":"Rule ~ tilde"}]`), "rules[0].name", "1-256 characters"},
		{"name too long", document(nodes(ruleNode(`[["a","c"]]`), complete), `[{"ruleId":"a","name":"`+strings.Repeat("n", 257)+`"}]`), "rules[0].name", "1-256 characters"},
		{"conditions not an object", document(nodes(ruleNode(`[["a","c"]]`), complete), `[{"ruleId":"a","name":"A","conditions":[1]}]`), "rules[0].conditions", "JSON object or absent"},
		{"unknown rule key", document(nodes(ruleNode(`[["a","c"]]`), complete), `[{"ruleId":"a","name":"A","description":"x"}]`), "rules[0].description", "unknown key"},
		{"ruleId defined twice", document(nodes(ruleNode(`[["a","c"]]`), complete), `[{"ruleId":"a","name":"A"},{"ruleId":"a","name":"A2"}]`), "rules[1].ruleId", "already defined at rules[0]"},
		{"too many rules", tooManyRules, "rules", "at most 98 rules"},

		{"RULE node without arms", document(`[{"nodeId":"r","nodeType":"RULE","elseChildNodeId":"c"},`+complete+`]`, `[]`), "actionNodes[0].ruleChildNodeIds", "a RULE node must have"},
		{"arms not an array", document(nodes(ruleNode(`{}`), complete), `[]`), "actionNodes[0].ruleChildNodeIds", "must be an array"},
		{"arm not a pair", document(nodes(ruleNode(`[["a"]]`), complete), `[]`), "actionNodes[0].ruleChildNodeIds[0]", "must be a [ruleId, childNodeId] pair"},

		{"arm references a rule rules does not define", document(nodes(ruleNode(`[["a","c"],["b","c"]]`), complete), `[{"ruleId":"a","name":"A"}]`), "actionNodes[0].ruleChildNodeIds[1][0]", `references rule "b", which ` + "`rules`" + ` does not define`},
		{"rule not referenced by any node", document(nodes(ruleNode(`[["a","c"]]`), complete), `[{"ruleId":"a","name":"A"},{"ruleId":"b","name":"B"}]`), "rules[1].ruleId", "not referenced by any node"},
		{"one rule referenced by two nodes", document(
			`[`+ruleNode(`[["a","c"]]`)+`,{"nodeId":"r2","nodeType":"RULE","ruleChildNodeIds":[["a","c"]],"elseChildNodeId":"c"},`+complete+`]`,
			`[{"ruleId":"a","name":"A"}]`), "actionNodes[1].ruleChildNodeIds[0][0]", "already references"},

		{"childNodeId names no node", document(`[{"nodeId":"v","nodeType":"VERIFICATION","childNodeId":"gone"},`+complete+`]`, `[]`), "actionNodes[0].childNodeId", `no node has nodeId "gone"`},
		{"elseChildNodeId names no node", document(nodes(`{"nodeId":"r","nodeType":"RULE","ruleChildNodeIds":[["a","c"]],"elseChildNodeId":"gone"}`, complete), `[{"ruleId":"a","name":"A"}]`), "actionNodes[0].elseChildNodeId", `no node has nodeId "gone"`},
		{"arm target names no node", document(nodes(ruleNode(`[["a","gone"]]`), complete), `[{"ruleId":"a","name":"A"}]`), "actionNodes[0].ruleChildNodeIds[0][1]", `no node has nodeId "gone"`},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			_, errs := parseFlow(testCase.flow)
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

// The publish schema puts rule names on the API's description pattern, which allows a broad sweep of
// punctuation. A name the API would take must not fail the plan.
func TestParseFlowAcceptsRuleNamesWithTheApisPunctuation(t *testing.T) {
	flow := `{"actionNodes":[{"nodeId":"r","nodeType":"RULE","ruleChildNodeIds":[["a","c"]],"elseChildNodeId":"c"},{"nodeId":"c","nodeType":"COMPLETE"}],
	  "rules":[{"ruleId":"a","name":"Rule #1: is (\"risky\") & <new>? [$100+ / -50%] {ok}; yes!"}]}`

	if _, errs := parseFlow(flow); len(errs) > 0 {
		t.Fatalf("the API accepts this name, so must the validator: %v", errs)
	}
}

// A flow with no rules at all is legitimate: the API's schema puts no minimum on the array.
func TestParseFlowAcceptsAFlowWithNoRules(t *testing.T) {
	flow := `{"actionNodes":[{"nodeId":"r","nodeType":"RULE","ruleChildNodeIds":[],"elseChildNodeId":"c"},{"nodeId":"c","nodeType":"COMPLETE"}],"rules":[]}`

	doc, errs := parseFlow(flow)
	if len(errs) > 0 {
		t.Fatalf("a RULE node with no arms needs no rules: %v", errs)
	}

	if len(doc.Rules) != 0 {
		t.Fatalf("expected no rules, got %v", doc.Rules)
	}
}

// Everything on a node other than nodeId and nodeType belongs to the API, and reaches it untouched.
func TestParseFlowLeavesNodePayloadsAlone(t *testing.T) {
	flow := `{"actionNodes":[{"nodeId":"s","nodeType":"CUSTOM_SCREEN","content":{"title":{"en":"Hi"},"blocks":[]},"buttonChildNodeIds":[["ok","s"]],"unknownToTheProvider":{"deeply":["nested",1,null,true]}}],"rules":[]}`

	doc, errs := parseFlow(flow)
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}

	body, err := json.Marshal(doc.ActionNodes)
	if err != nil {
		t.Fatal(err)
	}

	for _, fragment := range []string{`"unknownToTheProvider"`, `"deeply":["nested",1,null,true]`, `"buttonChildNodeIds":[["ok","s"]]`} {
		if !strings.Contains(string(body), fragment) {
			t.Fatalf("expected %s to survive untouched: %s", fragment, body)
		}
	}
}

func TestComposeFlowRoundTripsTheContractFlow(t *testing.T) {
	doc, errs := parseFlow(contractFlow)
	if len(errs) > 0 {
		t.Fatal(errs)
	}

	// What the server hands back: the published nodes (through a JSON round trip, as the SDK decodes
	// them) and the rules with their derived fields.
	nodesJson, _ := json.Marshal(doc.ActionNodes)
	nodes := serverNodes(t, string(nodesJson))
	rules := []authsignal.RuleResponse{
		serverRule("rule-anon", "Anonymous IP", `{"and":[{"==":[{"var":"ip.isAnonymous"},true]}]}`),
		serverRule("rule-nz", "From New Zealand", `{"and":[{"in":[{"var":"ip.location.country.countryCode"},["NZ"]]}]}`),
	}

	composed, err := composeFlow(nodes, rules)
	if err != nil {
		t.Fatal(err)
	}

	if !flowsEqual(contractFlow, composed) {
		t.Fatalf("composing what the server returns must reproduce the document.\ndocument: %s\ncomposed: %s", contractFlow, composed)
	}

	var result map[string]any
	if err := json.Unmarshal([]byte(composed), &result); err != nil {
		t.Fatal(err)
	}

	composedRules, ok := result["rules"].([]any)
	if !ok || len(composedRules) != 2 {
		t.Fatalf("expected two rules alongside the nodes, got %v", result["rules"])
	}

	for _, key := range []string{"type", "priority", "isActive", "actionCode", "tenantId"} {
		if _, has := composedRules[0].(map[string]any)[key]; has {
			t.Fatalf("the API's derived field %q must not be composed into the document: %v", key, composedRules[0])
		}
	}
}

// Flow rules all carry priority 0, so the order the API lists them in is arbitrary and can differ
// between two reads. Composing sorts by ruleId, so a read produces the same string either way.
func TestComposeFlowIsIndependentOfTheOrderTheApiListsRulesIn(t *testing.T) {
	nodes := serverNodes(t, `[{"nodeId":"r","nodeType":"RULE","ruleChildNodeIds":[["b","c"],["a","c"]],"elseChildNodeId":"c"},{"nodeId":"c","nodeType":"COMPLETE"}]`)

	oneOrder, err := composeFlow(nodes, []authsignal.RuleResponse{
		serverRule("b", "B", `{"and":[]}`),
		serverRule("a", "A", ""),
	})
	if err != nil {
		t.Fatal(err)
	}

	otherOrder, err := composeFlow(nodes, []authsignal.RuleResponse{
		serverRule("a", "A", ""),
		serverRule("b", "B", `{"and":[]}`),
	})
	if err != nil {
		t.Fatal(err)
	}

	if oneOrder != otherOrder {
		t.Fatalf("two listings of the same rules must compose to the same document\none:   %s\nother: %s", oneOrder, otherOrder)
	}

	expected := `{"actionNodes":[{"elseChildNodeId":"c","nodeId":"r","nodeType":"RULE","ruleChildNodeIds":[["b","c"],["a","c"]]},{"nodeId":"c","nodeType":"COMPLETE"}],"rules":[{"name":"A","ruleId":"a"},{"conditions":{"and":[]},"name":"B","ruleId":"b"}]}`
	if oneOrder != expected {
		t.Fatalf("expected sorted-key JSON with the rules sorted by ruleId and no conditions key on A\nexpected: %s\ngot:      %s", expected, oneOrder)
	}
}

// A rule created outside Terraform on a flow action is listed until the next publish prunes it, so
// it composes into the document and shows as an ordinary difference.
func TestComposeFlowIncludesARuleTheConfigurationHasDropped(t *testing.T) {
	nodes := serverNodes(t, `[{"nodeId":"r","nodeType":"RULE","ruleChildNodeIds":[["a","c"]],"elseChildNodeId":"c"},{"nodeId":"c","nodeType":"COMPLETE"}]`)
	configured := `{"actionNodes":[{"nodeId":"r","nodeType":"RULE","ruleChildNodeIds":[["a","c"]],"elseChildNodeId":"c"},{"nodeId":"c","nodeType":"COMPLETE"}],"rules":[{"ruleId":"a","name":"A"}]}`

	composed, err := composeFlow(nodes, []authsignal.RuleResponse{
		serverRule("a", "A", ""),
		serverRule("stray", "Made in the portal", `{"and":[]}`),
	})
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(composed, `"ruleId":"stray"`) {
		t.Fatalf("a rule the server still holds belongs in the document it reads back: %s", composed)
	}

	// The document read back does not satisfy every invariant, because no node references the stray
	// rule. Comparison must not depend on that, or the difference could not be shown.
	if _, errs := parseFlow(composed); len(errs) == 0 {
		t.Fatal("expected the stray rule to breach the reference invariant")
	}

	if flowsEqual(configured, composed) {
		t.Fatal("a rule the configuration has dropped must show as a difference")
	}
}

func TestComposeFlowOfAnUnpublishedActionIsAnEmptyDocument(t *testing.T) {
	composed, err := composeFlow(nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	if composed != `{"actionNodes":[],"rules":[]}` {
		t.Fatalf("expected an empty document, got %q", composed)
	}
}

func TestComposeFlowDoesNotMutateTheServerNodes(t *testing.T) {
	nodes := serverNodes(t, `[{"nodeId":"r","nodeType":"RULE","ruleChildNodeIds":[["a","c"]]}]`)

	if _, err := composeFlow(nodes, []authsignal.RuleResponse{serverRule("a", "A", "")}); err != nil {
		t.Fatal(err)
	}

	node := nodes[0].(map[string]any)
	if len(node) != 3 {
		t.Fatalf("composeFlow must leave the server's nodes alone, got %v", node)
	}
}

// The flow the documentation tells people to copy has to be one the provider accepts.
func TestTheShippedExampleFlowValidates(t *testing.T) {
	example, err := os.ReadFile("../../examples/resources/authsignal_action_configuration/flow-sign-in.json")
	if err != nil {
		t.Fatal(err)
	}

	if _, errs := parseFlow(string(example)); len(errs) > 0 {
		t.Fatalf("the example flow must validate: %v", errs)
	}
}
