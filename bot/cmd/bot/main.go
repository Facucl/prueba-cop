package main

import (
	"context"
	"log"
	"strings"
	"sync"
	"time"

	"copernico/deploy-notify-bot/internal/argo"
	"copernico/deploy-notify-bot/internal/config"
	"copernico/deploy-notify-bot/internal/message"
	"copernico/deploy-notify-bot/internal/tag"
	"copernico/deploy-notify-bot/internal/teams"
)

func main() {
	cfg, err := config.FromEnv()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	if !cfg.IsSecurityFix() {
		log.Printf("commit no contiene la marca de seguridad — no se notifica (servicio=%s tag=%s)", cfg.Service, cfg.Release)
		return
	}

	description := firstLine(cfg.CommitMessage)
	if description == "" {
		description = config.SecurityFixMarker
	}

	ctx, cancel := context.WithTimeout(context.Background(), cfg.PollTimeout+30*time.Second)
	defer cancel()

	teamsClient := teams.New(cfg.TeamsWebhookURL)
	pipelineOK := isPipelineOK(cfg.PipelineStatus)

	if !pipelineOK {
		p := message.Build(message.Input{
			Service:     cfg.Service,
			Branch:      cfg.Branch,
			Release:     cfg.Release,
			Description: description,
			PipelinesOK: false,
		})
		if err := teamsClient.Send(ctx, p); err != nil {
			log.Fatalf("teams send: %v", err)
		}
		log.Printf("notificado escenario 5 (pipeline ERROR) para %s %s", cfg.Service, cfg.Release)
		return
	}

	requested, err := tag.ParseList(cfg.Envs)
	if err != nil {
		log.Fatalf("parse envs: %v", err)
	}
	log.Printf("ambientes solicitados: %v", requested)

	argoClient, err := argo.New(cfg.ArgoNamespace)
	if err != nil {
		log.Fatalf("argo client: %v", err)
	}

	succeeded := pollAll(ctx, argoClient, cfg, requested)

	p := message.Build(message.Input{
		Service:     cfg.Service,
		Branch:      cfg.Branch,
		Release:     cfg.Release,
		Description: description,
		Requested:   requested,
		Succeeded:   succeeded,
		PipelinesOK: true,
	})
	if err := teamsClient.Send(ctx, p); err != nil {
		log.Fatalf("teams send: %v", err)
	}
	log.Printf("notificación enviada: solicitados=%v succeeded=%v", requested, succeeded)
}

// isPipelineOK acepta los valores que Tekton expone en $(tasks.status):
// "Succeeded" (todas OK), más tolera "OK"/"success" por compatibilidad.
func isPipelineOK(s string) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "succeeded", "completed", "ok", "success":
		return true
	}
	return false
}

func firstLine(s string) string {
	s = strings.TrimLeft(s, "\n\r \t")
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = s[:i]
	}
	return strings.TrimRight(s, "\r \t")
}

func pollAll(ctx context.Context, c *argo.Client, cfg *config.Config, envs []string) []string {
	// Tolerancia por si la reconciliación arrancó justo antes que este pod.
	since := time.Now().Add(-1 * time.Minute)

	oks := make(map[string]bool, len(envs))
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, env := range envs {
		env := env
		wg.Add(1)
		go func() {
			defer wg.Done()
			appName := cfg.AppName(env)
			res, err := c.WaitForSync(ctx, appName, since, cfg.PollInterval, cfg.PollTimeout)
			if err != nil {
				log.Printf("[%s/%s] error: %v", env, appName, err)
			}
			log.Printf("[%s/%s] resultado: %s", env, appName, res)
			if res == argo.ResultOK {
				mu.Lock()
				oks[env] = true
				mu.Unlock()
			}
		}()
	}
	wg.Wait()

	succeeded := make([]string, 0, len(oks))
	for e := range oks {
		succeeded = append(succeeded, e)
	}
	return tag.SortCanonical(succeeded)
}
