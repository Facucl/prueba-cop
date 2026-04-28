package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"sigs.k8s.io/yaml"
)

const SecurityFixMarker = "Mantenimiento: Fixes de seguridad y reparación automática de dependencias."

type Config struct {
	Argo         ArgoConfig    `json:"argo"`
	Poll         PollConfig    `json:"poll"`
	Routing      RoutingConfig `json:"routing"`
	TeamsWebhook TeamsWebhook  `json:"teamsWebhook"`
}

type ArgoConfig struct {
	Namespace  string `json:"namespace"`
	AppPattern string `json:"appPattern"`
}

type PollConfig struct {
	Interval time.Duration `json:"-"`
	Timeout  time.Duration `json:"-"`

	IntervalRaw string `json:"interval"`
	TimeoutRaw  string `json:"timeout"`
}

type RoutingConfig struct {
	Rules         []Rule `json:"rules"`
	DefaultAction string `json:"defaultAction"`
}

type Rule struct {
	ServicePattern     string `json:"servicePattern"`
	TeamsWebhookSecret string `json:"teamsWebhookSecret"`
}

type TeamsWebhook struct {
	SecretKey string `json:"secretKey"`
}

func LoadFile(path string) (*Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	var c Config
	if err := yaml.Unmarshal(raw, &c); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	if c.Poll.Interval, err = time.ParseDuration(c.Poll.IntervalRaw); err != nil {
		return nil, fmt.Errorf("poll.interval %q: %w", c.Poll.IntervalRaw, err)
	}
	if c.Poll.Timeout, err = time.ParseDuration(c.Poll.TimeoutRaw); err != nil {
		return nil, fmt.Errorf("poll.timeout %q: %w", c.Poll.TimeoutRaw, err)
	}
	if c.Argo.Namespace == "" {
		return nil, fmt.Errorf("argo.namespace is required")
	}
	if c.Argo.AppPattern == "" {
		return nil, fmt.Errorf("argo.appPattern is required")
	}
	if c.TeamsWebhook.SecretKey == "" {
		return nil, fmt.Errorf("teamsWebhook.secretKey is required")
	}
	if len(c.Routing.Rules) == 0 {
		return nil, fmt.Errorf("routing.rules cannot be empty")
	}
	return &c, nil
}

func (c *Config) AppName(service, env string) string {
	r := strings.NewReplacer("{service}", service, "{env}", env)
	return r.Replace(c.Argo.AppPattern)
}

// IsSecurityFix returns true when the commit message contains the required
// marker. Same filter as v1 — only commits with the exact subject trigger
// notifications.
func IsSecurityFix(commitMessage string) bool {
	return strings.Contains(commitMessage, SecurityFixMarker)
}
