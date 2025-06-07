package main

import (
	"testing"
)

func TestApplyRules_StringContains(t *testing.T) {
	rules = RuleConfig{
		Rules: []Rule{
			{
				RuleType:     "stringContains",
				TargetFields: []string{"title"},
				Pattern:      "GoLang",
				Ref:          "match1",
			},
		},
	}

	entry := Entry{
		ID:    "1",
		Title: "Learning GoLang with fun",
		Author: Author{
			Name: "Jane Doe",
		},
	}

	matched, groups := applyRules(entry)

	if len(matched) != 1 || matched[0] != "stringContains:GoLang" {
		t.Errorf("Expected one stringContains match, got: %v", matched)
	}

	if len(groups) != 0 {
		t.Errorf("Expected no groups, got: %v", groups)
	}
}

func TestApplyRules_RegexNamedCapture(t *testing.T) {
	rules = RuleConfig{
		Rules: []Rule{
			{
				RuleType:     "regex_named_capture",
				TargetFields: []string{"title"},
				Pattern:      `(?P<tag>Go\d+)`,
				Ref:          "capture1",
			},
		},
	}

	entry := Entry{
		ID:    "2",
		Title: "Release note: Go123 is here",
		Author: Author{
			Name: "Go Team",
		},
	}

	matched, groups := applyRules(entry)

	if len(matched) != 1 || matched[0] != "regex_named_capture:(?P<tag>Go\\d+)" {
		t.Errorf("Expected one named capture match, got: %v", matched)
	}

	if val, ok := groups["tag"]; !ok || val != "Go123" {
		t.Errorf("Expected capture group 'tag' = Go123, got: %v", groups)
	}
}

func TestApplyRules_StringDistance(t *testing.T) {
	rules = RuleConfig{
		Rules: []Rule{
			{
				RuleType:          "string_distance",
				TargetFields:      []string{"title"},
				Pattern:           "Crown corporations",
				DistanceThreshold: 5,
				Ref:               "fuzzy",
			},
		},
	}

	entry := Entry{
		ID:    "3",
		Title: "Crown corporatons", // typo, missing 'i' in 'corporations'
		Author: Author{
			Name: "John",
		},
	}

	matched, _ := applyRules(entry)

	if len(matched) != 1 {
		t.Errorf("Expected fuzzy match, got: %v", matched)
	}
}

func TestApplyRules_ChainedTargetRef(t *testing.T) {
	rules = RuleConfig{
		Rules: []Rule{
			{
				RuleType:     "regex",
				TargetFields: []string{"author"},
				Pattern:      "(?i)Fancy",
				Ref:          "fancy_author",
			},
			{
				RuleType:  "regex",
				TargetRef: "fancy_author",
				Pattern:   "^/u/FancyNewMe$",
				Ref:       "exact_fancy_author",
			},
		},
	}

	entry := Entry{
		ID:    "chain1",
		Title: "Some title",
		Author: Author{
			Name: "/u/FancyNewMe",
		},
	}

	matched, _ := applyRules(entry)

	if len(matched) != 2 {
		t.Errorf("Expected 2 matched rules, got: %v", matched)
	}
	if matched[1] != "regex:^/u/FancyNewMe$" {
		t.Errorf("Expected second rule to match exact author, got: %v", matched)
	}
}

func TestApplyRules_ChainedTargetCaptureGroup(t *testing.T) {
	rules = RuleConfig{
		Rules: []Rule{
			{
				RuleType:     "regex_named_capture",
				TargetFields: []string{"title"},
				Pattern:      "(?P<bill>S-\\d+)",
				Ref:          "bill_capture",
			},
			{
				RuleType:           "regex",
				TargetCaptureGroup: "bill",
				Pattern:            "^S-218$",
				Ref:                "is_bill_218",
			},
		},
	}

	entry := Entry{
		ID:    "chain2",
		Title: "Parliament just passed bill S-218",
		Author: Author{
			Name: "ReporterX",
		},
	}

	matched, groups := applyRules(entry)

	if len(matched) != 2 {
		t.Errorf("Expected 2 matched rules, got: %v", matched)
	}
	if groups["bill"] != "S-218" {
		t.Errorf("Expected captured group 'bill' to be 'S-218', got: %v", groups["bill"])
	}
}
