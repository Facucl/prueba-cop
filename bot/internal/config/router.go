package config

import (
	"fmt"
	"regexp"
)

const ActionDrop = "drop"

type Router struct {
	rules         []compiledRule
	defaultAction string
	secrets       map[string]string
}

type compiledRule struct {
	pattern *regexp.Regexp
	secret  string
}

// NewRouter compiles routing.rules and validates that every referenced secret
// is present in the loaded secrets map. Fail-fast at boot beats discovering
// the missing webhook on the first POST /notify in production.
func NewRouter(rules []Rule, defaultAction string, secrets map[string]string) (*Router, error) {
	compiled := make([]compiledRule, 0, len(rules))
	for i, r := range rules {
		re, err := regexp.Compile(r.ServicePattern)
		if err != nil {
			return nil, fmt.Errorf("rule[%d] servicePattern %q: %w", i, r.ServicePattern, err)
		}
		if _, ok := secrets[r.TeamsWebhookSecret]; !ok {
			return nil, fmt.Errorf("rule[%d] references secret %q which is not mounted", i, r.TeamsWebhookSecret)
		}
		compiled = append(compiled, compiledRule{pattern: re, secret: r.TeamsWebhookSecret})
	}
	if defaultAction != ActionDrop {
		return nil, fmt.Errorf("defaultAction %q not supported (only %q)", defaultAction, ActionDrop)
	}
	return &Router{rules: compiled, defaultAction: defaultAction, secrets: secrets}, nil
}

// Resolve finds the first matching rule for service and returns its webhook URL.
// When no rule matches, returns ("", false) so the caller can drop silently.
func (r *Router) Resolve(service string) (string, bool) {
	for _, c := range r.rules {
		if c.pattern.MatchString(service) {
			return r.secrets[c.secret], true
		}
	}
	return "", false
}
