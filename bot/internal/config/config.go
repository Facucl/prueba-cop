package config

import (
	"fmt"
	"os"
	"strings"
	"time"
)

type Config struct {
	Service         string
	Branch          string
	Release         string
	Envs            string // raw string: "desa test preprod"
	CommitMessage   string
	PipelineStatus  string
	TeamsWebhookURL string

	ArgoNamespace  string
	ArgoAppPattern string // ej. "{service}-{env}"
	PollInterval   time.Duration
	PollTimeout    time.Duration
}

// SecurityFixMarker is the exact commit message substring required to trigger
// a notification. Anything else is ignored silently.
const SecurityFixMarker = "Mantenimiento: Fixes de seguridad y reparación automática de dependencias."

func FromEnv() (*Config, error) {
	c := &Config{
		Service:         os.Getenv("SERVICE"),
		Branch:          envDefault("BRANCH", "master"),
		Release:         os.Getenv("TAG"),
		Envs:            os.Getenv("ENVS"),
		CommitMessage:   os.Getenv("COMMIT_MESSAGE"),
		PipelineStatus:  os.Getenv("PIPELINE_STATUS"),
		TeamsWebhookURL: os.Getenv("TEAMS_WEBHOOK_URL"),

		ArgoNamespace:  envDefault("ARGO_NAMESPACE", "openshift-gitops"),
		ArgoAppPattern: envDefault("ARGO_APP_PATTERN", "{service}-{env}"),
		PollInterval:   envDuration("POLL_INTERVAL", 15*time.Second),
		PollTimeout:    envDuration("POLL_TIMEOUT", 15*time.Minute),
	}

	var missing []string
	if c.Service == "" {
		missing = append(missing, "SERVICE")
	}
	if c.Release == "" {
		missing = append(missing, "TAG")
	}
	if c.PipelineStatus == "" {
		missing = append(missing, "PIPELINE_STATUS")
	}
	if c.TeamsWebhookURL == "" {
		missing = append(missing, "TEAMS_WEBHOOK_URL")
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("missing required env vars: %s", strings.Join(missing, ", "))
	}
	return c, nil
}

func (c *Config) AppName(env string) string {
	r := strings.NewReplacer("{service}", c.Service, "{env}", env)
	return r.Replace(c.ArgoAppPattern)
}

// IsSecurityFix returns true when the commit message contains the required
// marker. A missing COMMIT_MESSAGE env var is treated as a non-security-fix.
func (c *Config) IsSecurityFix() bool {
	return strings.Contains(c.CommitMessage, SecurityFixMarker)
}

func envDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envDuration(key string, def time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return def
	}
	return d
}
