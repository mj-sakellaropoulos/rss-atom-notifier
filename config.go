package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v2"
)

type RuleConfig struct {
	APIVersion   string         `yaml:"apiVersion"`
	LogLevel     string         `yaml:"loglevel"`
	Rules        []Rule         `yaml:"rules"`
	HTTP         HTTPConfig     `yaml:"http"`
	MatchOutputs []MatchOutput  `yaml:"match_outputs"`
	Database     DatabaseConfig `yaml:"database"`
}

type DatabaseConfig struct {
	Path string `yaml:"path"`
}

type MatchOutput struct {
	Gotify *GotifyOutput `yaml:"gotify,omitempty"`
	Stdout *struct{}     `yaml:"stdout,omitempty"`
	HTTP   *HTTPOutput   `yaml:"http,omitempty"`
}

type GotifyOutput struct {
	URL   string `yaml:"url"`
	Token string `yaml:"token"`
}

type HTTPOutput struct {
	Method      string `yaml:"method"`
	URL         string `yaml:"url"`
	PayloadTmpl string `yaml:"payload_tmpl"`
}

type HTTPConfig struct {
	UserAgent         string `yaml:"userAgent"`
	PollingIntervalMs int    `yaml:"pollingIntervalMs"`
	RSSURL            string `yaml:"rss_url"`
}

type Rule struct {
	RuleType           string   `yaml:"ruleType"`
	TargetFields       []string `yaml:"targetFields"`
	TargetField        string   `yaml:"targetField"`
	MatchRaw           bool     `yaml:"matchRaw"`
	DistanceThreshold  int      `yaml:"distanceThreshold"`
	Pattern            string   `yaml:"pattern"`
	Ref                string   `yaml:"ref"`
	TargetRef          string   `yaml:"targetRef"`
	TargetCaptureGroup string   `yaml:"targetCaptureGroup"`
}

var (
	rules     RuleConfig
	rulesLock sync.RWMutex
	debug     bool
)

/*
loadRules reads and parses the YAML config from disk.
*/
func loadRules(path string) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	var newRules RuleConfig
	if err := yaml.Unmarshal(data, &newRules); err != nil {
		log.Fatalf("Failed to parse YAML: %v", err)
	}
	if err := validateRules(newRules.Rules); err != nil {
		log.Fatalf("Invalid rule config: %v", err)
	}
	rulesLock.Lock()
	rules = newRules
	debug = strings.ToLower(rules.LogLevel) == "debug"
	rulesLock.Unlock()
	debugLog("Rules reloaded with debug=%v", debug)
}

/*
watchConfig monitors the YAML file for changes and reloads it.
*/
func watchConfig(path string) {
	go func() {
		lastMod := time.Time{}
		for {
			info, err := os.Stat(path)
			if err == nil && info.ModTime().After(lastMod) {
				lastMod = info.ModTime()
				loadRules(path)
			}
			time.Sleep(5 * time.Second)
		}
	}()
}

/*
validateRules checks that targetRefs point to valid previous refs.
*/
func validateRules(rules []Rule) error {
	refNames := make(map[string]bool)
	for i, rule := range rules {
		if rule.Ref != "" {
			refNames[rule.Ref] = true
		}
		if rule.TargetRef != "" && !refNames[rule.TargetRef] {
			return fmt.Errorf("rule %d: targetRef '%s' not defined before use", i, rule.TargetRef)
		}
	}
	return nil
}

/*
debugLog prints a debug line to stderr if debug mode is active.
*/
func debugLog(format string, args ...interface{}) {
	if debug {
		log.Printf("[DEBUG] "+format, args...)
	}
}
