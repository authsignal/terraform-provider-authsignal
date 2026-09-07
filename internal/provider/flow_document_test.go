package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/authsignal/authsignal-management-go/v6"
)

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

func serverNodes(t *testing.T, nodesJson string) []authsignal.ActionNode {
	t.Helper()

	var nodes []authsignal.ActionNode
	if err := json.Unmarshal([]byte(nodesJson), &nodes); err != nil {
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

// The SDK decodes rule conditions with UseNumber(), so a fixture standing in for ListRules must too.
func serverRuleFromApi(t *testing.T, ruleId string, name string, conditionsJson string) authsignal.RuleResponse {
	t.Helper()

	rule := serverRule(ruleId, name, "")

	decoder := json.NewDecoder(strings.NewReader(conditionsJson))
	decoder.UseNumber()

	var conditions any
	if err := decoder.Decode(&conditions); err != nil {
		t.Fatalf("bad fixture: %v", err)
	}

	rule.Conditions = conditions

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

	if !strings.Contains(string(doc.ActionNodes[0]), `"nodeId":"rule-abc"`) {
		t.Fatalf("expected the first node first, got %s", doc.ActionNodes[0])
	}

	if doc.Rules[0].RuleId != "rule-nz" || doc.Rules[0].Name != "From New Zealand" || doc.Rules[0].Conditions == nil {
		t.Fatalf("bad first rule: %+v", doc.Rules[0])
	}

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
		{"trailing token", document(nodes(complete), `[]`) + ` true`, "", "trailing content"},
		{"second document", document(nodes(complete), `[]`) + ` {}`, "", "trailing content"},
		{"an array, as the old shape was", `[{"nodeId":"c","nodeType":"COMPLETE"}]`, "", "must be a JSON object"},
		{"actionNodes missing", `{"rules":[]}`, "actionNodes", "is required"},
		{"rules missing", `{"actionNodes":[` + complete + `]}`, "rules", "is required"},
		{"unknown top-level key", document(nodes(complete), `[]`)[:len(document(nodes(complete), `[]`))-1] + `,"actionType":"FLOW"}`, "actionType", "unknown key"},
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
		{"name not a string", document(nodes(ruleNode(`[["a","c"]]`), complete), `[{"ruleId":"a","name":1}]`), "rules[0].name", "must be a string"},
		{"name with a character the API rejects", document(nodes(ruleNode(`[["a","c"]]`), complete), `[{"ruleId":"a","name":"Rule ~ tilde"}]`), "rules[0].name", "0-256 characters"},
		{"name too long", document(nodes(ruleNode(`[["a","c"]]`), complete), `[{"ruleId":"a","name":"`+strings.Repeat("n", 257)+`"}]`), "rules[0].name", "0-256 characters"},
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
		{"verification method target is not a pair", document(nodes(`{"nodeId":"v","nodeType":"VERIFICATION_METHOD_BRANCH","verificationMethodChildNodeIds":[["PASSKEY"]]}`, complete), `[]`), "actionNodes[0].verificationMethodChildNodeIds[0]", "pair of two non-empty strings"},
		{"verification method target names no node", document(nodes(`{"nodeId":"v","nodeType":"VERIFICATION_METHOD_BRANCH","verificationMethodChildNodeIds":[["PASSKEY","gone"]]}`, complete), `[]`), "actionNodes[0].verificationMethodChildNodeIds[0][1]", `no node has nodeId "gone"`},
		{"button target is not a pair", document(nodes(`{"nodeId":"s","nodeType":"CUSTOM_SCREEN","buttonChildNodeIds":[["continue"]]}`, complete), `[]`), "actionNodes[0].buttonChildNodeIds[0]", "pair of two non-empty strings"},
		{"button target names no node", document(nodes(`{"nodeId":"s","nodeType":"CUSTOM_SCREEN","buttonChildNodeIds":[["continue","gone"]]}`, complete), `[]`), "actionNodes[0].buttonChildNodeIds[0][1]", `no node has nodeId "gone"`},

		{"childNodeId is a number", document(`[{"nodeId":"v","nodeType":"VERIFICATION","childNodeId":7},`+complete+`]`, `[]`), "actionNodes[0].childNodeId", "must be a non-empty node id string"},
		{"childNodeId is null", document(`[{"nodeId":"v","nodeType":"VERIFICATION","childNodeId":null},`+complete+`]`, `[]`), "actionNodes[0].childNodeId", "must be a non-empty node id string"},
		{"childNodeId is an object", document(`[{"nodeId":"v","nodeType":"VERIFICATION","childNodeId":{"nodeId":"c"}},`+complete+`]`, `[]`), "actionNodes[0].childNodeId", "must be a non-empty node id string"},
		{"childNodeId is empty", document(`[{"nodeId":"v","nodeType":"VERIFICATION","childNodeId":""},`+complete+`]`, `[]`), "actionNodes[0].childNodeId", "must be a non-empty node id string"},
		{"elseChildNodeId is a number", document(nodes(`{"nodeId":"r","nodeType":"RULE","ruleChildNodeIds":[["a","c"]],"elseChildNodeId":7}`, complete), `[{"ruleId":"a","name":"A"}]`), "actionNodes[0].elseChildNodeId", "must be a non-empty node id string"},
		{"elseChildNodeId is null", document(nodes(`{"nodeId":"r","nodeType":"RULE","ruleChildNodeIds":[["a","c"]],"elseChildNodeId":null}`, complete), `[{"ruleId":"a","name":"A"}]`), "actionNodes[0].elseChildNodeId", "must be a non-empty node id string"},
		{"arm target is not a string", document(nodes(ruleNode(`[["a",7]]`), complete), `[{"ruleId":"a","name":"A"}]`), "actionNodes[0].ruleChildNodeIds[0]", "pair of non-empty strings"},
		{"arm rule id is not a string", document(nodes(ruleNode(`[[7,"c"]]`), complete), `[{"ruleId":"a","name":"A"}]`), "actionNodes[0].ruleChildNodeIds[0]", "pair of non-empty strings"},
		{"verification method target is not a string", document(nodes(`{"nodeId":"v","nodeType":"VERIFICATION_METHOD_BRANCH","verificationMethodChildNodeIds":[["PASSKEY",7]]}`, complete), `[]`), "actionNodes[0].verificationMethodChildNodeIds[0]", "pair of two non-empty strings"},
		{"button target is not a string", document(nodes(`{"nodeId":"s","nodeType":"CUSTOM_SCREEN","buttonChildNodeIds":[["continue",null]]}`, complete), `[]`), "actionNodes[0].buttonChildNodeIds[0]", "pair of two non-empty strings"},
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

func TestParseFlowAcceptsRuleNamesWithTheApisPunctuation(t *testing.T) {
	flow := `{"actionNodes":[{"nodeId":"r","nodeType":"RULE","ruleChildNodeIds":[["a","c"]],"elseChildNodeId":"c"},{"nodeId":"c","nodeType":"COMPLETE"}],
	  "rules":[{"ruleId":"a","name":"Rule #1: is (\"risky\") & <new>? [$100+ / -50%] {ok}; yes!"}]}`

	if _, errs := parseFlow(flow); len(errs) > 0 {
		t.Fatalf("the API accepts this name, so must the validator: %v", errs)
	}
}

func TestParseFlowAcceptsAnEmptyRuleNameLikeTheApi(t *testing.T) {
	flow := `{"actionNodes":[{"nodeId":"r","nodeType":"RULE","ruleChildNodeIds":[["a","c"]],"elseChildNodeId":"c"},{"nodeId":"c","nodeType":"COMPLETE"}],
	  "rules":[{"ruleId":"a","name":""}]}`

	if _, errs := parseFlow(flow); len(errs) > 0 {
		t.Fatalf("the API accepts an empty rule name, so must the validator: %v", errs)
	}
}

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

func TestComposeFlowKeepsTheNumberLiteralsTheApiSent(t *testing.T) {
	const nodeJson = `{"nodeId":"c","nodeType":"COMPLETE","attempts":9007199254740993,"score":1.10}`
	const conditionsJson = `{"and":[{"==":[{"var":"attempts"},9007199254740993]},{"==":[{"var":"score"},1.10]}]}`

	nodes := serverNodes(t, "["+nodeJson+"]")
	rules := []authsignal.RuleResponse{serverRuleFromApi(t, "a", "A", conditionsJson)}

	composed, err := composeFlowWithRuleOrder(nodes, rules, []string{"a"})
	if err != nil {
		t.Fatal(err)
	}

	expected := fmt.Sprintf(`{"actionNodes":[%s],"rules":[{"conditions":%s,"name":"A","ruleId":"a"}]}`, nodeJson, conditionsJson)
	if composed != expected {
		t.Fatalf("an integer above 2^53 and a trailing zero must read back untouched\nexpected: %s\ngot:      %s", expected, composed)
	}

	configured := fmt.Sprintf(`{"actionNodes":[%s],"rules":[{"ruleId":"a","name":"A","conditions":%s}]}`, nodeJson, conditionsJson)

	equal, diags := NewFlowValue(configured).StringSemanticEquals(context.Background(), NewFlowValue(composed))
	if diags.HasError() {
		t.Fatal(diags)
	}

	if !equal {
		t.Fatalf("the composed document must match the configuration\nconfigured: %s\ncomposed:   %s", configured, composed)
	}
}

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

	expected := `{"actionNodes":[{"nodeId":"r","nodeType":"RULE","ruleChildNodeIds":[["b","c"],["a","c"]],"elseChildNodeId":"c"},{"nodeId":"c","nodeType":"COMPLETE"}],"rules":[{"name":"A","ruleId":"a"},{"conditions":{"and":[]},"name":"B","ruleId":"b"}]}`
	if oneOrder != expected {
		t.Fatalf("expected the nodes verbatim with the rules sorted by ruleId and no conditions key on A\nexpected: %s\ngot:      %s", expected, oneOrder)
	}
}

func TestComposeFlowKeepsThePriorRuleOrderDuringDrift(t *testing.T) {
	nodes := serverNodes(t, `[{"nodeId":"c","nodeType":"COMPLETE","changed":true}]`)
	rules := []authsignal.RuleResponse{
		serverRule("a", "A", ""),
		serverRule("new", "New", ""),
		serverRule("b", "B", ""),
	}

	composed, err := composeFlowWithRuleOrder(nodes, rules, []string{"b", "a"})
	if err != nil {
		t.Fatal(err)
	}

	b := strings.Index(composed, `"ruleId":"b"`)
	a := strings.Index(composed, `"ruleId":"a"`)
	added := strings.Index(composed, `"ruleId":"new"`)
	if b == -1 || a == -1 || added == -1 || !(b < a && a < added) {
		t.Fatalf("expected prior rules in prior order and new rules appended by id: %s", composed)
	}
}

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
	const nodeJson = `{"nodeId":"r","nodeType":"RULE","ruleChildNodeIds":[["a","c"]]}`

	nodes := serverNodes(t, "["+nodeJson+"]")

	if _, err := composeFlow(nodes, []authsignal.RuleResponse{serverRule("a", "A", "")}); err != nil {
		t.Fatal(err)
	}

	if string(nodes[0]) != nodeJson {
		t.Fatalf("composeFlow must leave the server's nodes alone, got %s", nodes[0])
	}
}

func TestTheShippedExampleFlowValidates(t *testing.T) {
	example, err := os.ReadFile("../../examples/resources/authsignal_action_configuration/flow-sign-in.json")
	if err != nil {
		t.Fatal(err)
	}

	if _, errs := parseFlow(string(example)); len(errs) > 0 {
		t.Fatalf("the example flow must validate: %v", errs)
	}
}

// The API measures the compact encoding, so the fixture is padded to an exact byte count.
func flowWithActionNodesOfSize(t *testing.T, size int) string {
	t.Helper()

	node := map[string]any{"nodeId": "c", "nodeType": "COMPLETE", "pad": ""}

	unpadded, err := json.Marshal([]any{node})
	if err != nil {
		t.Fatal(err)
	}

	node["pad"] = strings.Repeat("x", size-len(unpadded))

	nodes, err := json.Marshal([]any{node})
	if err != nil {
		t.Fatal(err)
	}

	if len(nodes) != size {
		t.Fatalf("padding produced %d bytes, wanted %d", len(nodes), size)
	}

	return fmt.Sprintf(`{"actionNodes":%s,"rules":[]}`, nodes)
}

func TestParseFlowRejectsActionNodesOverTheApisSizeCap(t *testing.T) {
	_, errs := parseFlow(flowWithActionNodesOfSize(t, flowMaxActionNodesBytes+1))
	if len(errs) != 1 {
		t.Fatalf("expected one error, got %v", errs)
	}

	if errs[0].Path != "actionNodes" || !strings.Contains(errs[0].Message, "at most 300000 bytes") {
		t.Fatalf("expected a size error at actionNodes, got %v", errs[0])
	}
}

func TestParseFlowAcceptsActionNodesAtTheApisSizeCap(t *testing.T) {
	if _, errs := parseFlow(flowWithActionNodesOfSize(t, flowMaxActionNodesBytes)); len(errs) > 0 {
		t.Fatalf("the API accepts nodes of exactly %d bytes, so must the validator: %v", flowMaxActionNodesBytes, errs)
	}
}
