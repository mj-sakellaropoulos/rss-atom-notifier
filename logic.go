package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/agnivade/levenshtein"
	"net/http"
	"regexp"
	"strings"
	"text/template"
)

func notifyMatch(entry Entry, matched []string, namedGroups map[string]string) {
	rulesLock.RLock()
	outputs := rules.MatchOutputs
	rulesLock.RUnlock()

	for _, out := range outputs {
		switch {
		case out.Stdout != nil:
			fmt.Printf("MATCH: Entry %s matched rules: %v, groups: %v\n", entry.ID, matched, namedGroups)

		case out.Gotify != nil:
			go func(cfg *GotifyOutput) {
				payload := map[string]interface{}{
					"title":   "RSS Match",
					"message": fmt.Sprintf("Matched: %v\nTitle: %s\nURL: %s", matched, entry.Title, entry.Link.Href),

					"priority": 5,
				}
				data, _ := json.Marshal(payload)
				req, err := http.NewRequest("POST", fmt.Sprintf("%s?token=%s", cfg.URL, cfg.Token), bytes.NewBuffer(data))
				if err != nil {
					debugLog("Failed to create Gotify request: %v", err)
					return
				}
				req.Header.Set("Content-Type", "application/json")
				resp, err := http.DefaultClient.Do(req)
				if err != nil {
					debugLog("Gotify request failed: %v", err)
					return
				}
				resp.Body.Close()
			}(out.Gotify)

		case out.HTTP != nil:
			go func(cfg *HTTPOutput) {
				tmpl, err := template.New("payload").Parse(cfg.PayloadTmpl)
				if err != nil {
					debugLog("Failed to parse template: %v", err)
					return
				}
				var buf bytes.Buffer
				tmpl.Execute(&buf, map[string]interface{}{
					"Entry":   entry,
					"Groups":  namedGroups,
					"Matched": matched,
				})
				req, err := http.NewRequest(cfg.Method, cfg.URL, &buf)
				if err != nil {
					debugLog("Failed to create HTTP request: %v", err)
					return
				}
				req.Header.Set("Content-Type", "application/json")
				resp, err := http.DefaultClient.Do(req)
				if err != nil {
					debugLog("HTTP notification failed: %v", err)
					return
				}
				resp.Body.Close()
			}(out.HTTP)
		}
	}
}

func min(a, b, c int) int {
	if a < b && a < c {
		return a
	}
	if b < c {
		return b
	}
	return c
}

/*
applyRules evaluates all rules against an entry and returns match results.
*/
func applyRules(entry Entry) (matchedRules []string, namedGroups map[string]string) {
	matchedRules = []string{}
	namedGroups = make(map[string]string)
	refResults := make(map[string]string)

	fields := map[string]string{
		"id":     entry.ID,
		"author": entry.Author.Name,
		"title":  entry.Title,
		"raw":    entry.Raw,
	}

	rulesLock.RLock()
	defer rulesLock.RUnlock()

	for _, rule := range rules.Rules {
		targets := resolveTargets(rule, fields, namedGroups, refResults)
		for _, text := range targets {
			if applyRule(rule, text, matchedRules, namedGroups, refResults) {
				matchedRules = append(matchedRules, fmt.Sprintf("%s:%s", rule.RuleType, rule.Pattern))
			}
		}
	}
	return matchedRules, namedGroups
}

/*
resolveTargets determines the actual strings to evaluate for a given rule.
*/
func resolveTargets(rule Rule, fields map[string]string, groups, refs map[string]string) []string {
	if rule.TargetRef != "" {
		if val, ok := refs[rule.TargetRef]; ok {
			return []string{val}
		}
		return nil
	}
	if rule.TargetCaptureGroup != "" {
		if val, ok := groups[rule.TargetCaptureGroup]; ok {
			return []string{val}
		}
		return nil
	}
	if rule.MatchRaw {
		return []string{fields["raw"]}
	}
	if rule.TargetField != "" {
		return []string{fields[rule.TargetField]}
	}
	var result []string
	for _, f := range rule.TargetFields {
		result = append(result, fields[f])
	}
	return result
}

/*
applyRule runs the specific rule logic and updates state if matched.
*/
func applyRule(rule Rule, text string, matched []string, groups, refs map[string]string) bool {
	switch rule.RuleType {
	case "stringContains":
		if strings.Contains(text, rule.Pattern) {
			if rule.Ref != "" {
				refs[rule.Ref] = text
			}
			return true
		}
	case "regex":
		if matched, _ := regexp.MatchString(rule.Pattern, text); matched {
			if rule.Ref != "" {
				refs[rule.Ref] = text
			}
			return true
		}
	case "regex_named_capture":
		re := regexp.MustCompile(rule.Pattern)
		match := re.FindStringSubmatch(text)
		if len(match) > 0 {
			for i, name := range re.SubexpNames() {
				if i > 0 && name != "" {
					groups[name] = match[i]
				}
			}
			if rule.Ref != "" {
				refs[rule.Ref] = text
			}
			return true
		}
	case "string_distance":
		distance := levenshtein.ComputeDistance(text, rule.Pattern)
		if distance <= rule.DistanceThreshold {
			if rule.Ref != "" {
				refs[rule.Ref] = text
			}
			return true
		}
	}
	return false
}
