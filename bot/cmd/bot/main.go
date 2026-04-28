package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"copernico/deploy-notify-bot/internal/argo"
	"copernico/deploy-notify-bot/internal/config"
	"copernico/deploy-notify-bot/internal/message"
	"copernico/deploy-notify-bot/internal/tag"
	"copernico/deploy-notify-bot/internal/teams"
)

const (
	defaultListenAddr = ":8080"
	defaultConfigPath = "/etc/deploy-notify-bot/config.yaml"
	defaultSecretsDir = "/etc/deploy-notify-bot/secrets"

	shutdownGracePeriod = 25 * time.Second
)

type notifyRequest struct {
	Service        string `json:"service"`
	Branch         string `json:"branch"`
	Release        string `json:"release"`
	Envs           string `json:"envs"`
	CommitMessage  string `json:"commitMessage"`
	PipelineStatus string `json:"pipelineStatus"`
}

type server struct {
	cfg    *config.Config
	router *config.Router
	argo   *argo.Client

	bgCtx    context.Context
	bgCancel context.CancelFunc
	wg       sync.WaitGroup
	ready    atomic.Bool
}

func main() {
	listenAddr := envDefault("BOT_LISTEN_ADDR", defaultListenAddr)
	configPath := envDefault("BOT_CONFIG_PATH", defaultConfigPath)
	secretsDir := envDefault("BOT_SECRETS_DIR", defaultSecretsDir)

	cfg, err := config.LoadFile(configPath)
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	secrets, err := config.LoadSecrets(secretsDir, cfg.TeamsWebhook.SecretKey)
	if err != nil {
		log.Fatalf("secrets: %v", err)
	}

	router, err := config.NewRouter(cfg.Routing.Rules, cfg.Routing.DefaultAction, secrets)
	if err != nil {
		log.Fatalf("router: %v", err)
	}

	argoClient, err := argo.New(cfg.Argo.Namespace)
	if err != nil {
		log.Fatalf("argo client: %v", err)
	}

	bgCtx, bgCancel := context.WithCancel(context.Background())
	s := &server{
		cfg:      cfg,
		router:   router,
		argo:     argoClient,
		bgCtx:    bgCtx,
		bgCancel: bgCancel,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/notify", s.handleNotify)
	mux.HandleFunc("/healthz", s.handleHealthz)
	mux.HandleFunc("/readyz", s.handleReadyz)

	httpSrv := &http.Server{
		Addr:              listenAddr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("listening on %s", listenAddr)
		s.ready.Store(true)
		if err := httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("listen: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	log.Printf("shutdown signal received")
	s.ready.Store(false)

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownGracePeriod)
	defer cancel()
	if err := httpSrv.Shutdown(shutdownCtx); err != nil {
		log.Printf("http shutdown: %v", err)
	}

	bgCancel()
	done := make(chan struct{})
	go func() { s.wg.Wait(); close(done) }()
	select {
	case <-done:
		log.Printf("background notifications drained cleanly")
	case <-shutdownCtx.Done():
		log.Printf("shutdown timeout — some background notifications may have been dropped")
	}
}

func (s *server) handleHealthz(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (s *server) handleReadyz(w http.ResponseWriter, _ *http.Request) {
	if !s.ready.Load() {
		http.Error(w, "not ready", http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (s *server) handleNotify(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req notifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json: "+err.Error(), http.StatusBadRequest)
		return
	}

	if missing := req.missingFields(); len(missing) > 0 {
		http.Error(w, "missing required fields: "+strings.Join(missing, ", "), http.StatusBadRequest)
		return
	}

	if !config.IsSecurityFix(req.CommitMessage) {
		log.Printf("[%s/%s] skipped (not a security fix)", req.Service, req.Release)
		writeJSON(w, http.StatusOK, map[string]string{"status": "skipped", "reason": "not a security fix"})
		return
	}

	webhookURL, ok := s.router.Resolve(req.Service)
	if !ok {
		log.Printf("[%s/%s] dropped (no routing rule matched)", req.Service, req.Release)
		writeJSON(w, http.StatusOK, map[string]string{"status": "dropped", "reason": "no routing rule matched"})
		return
	}

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		s.processNotify(req, webhookURL)
	}()

	writeJSON(w, http.StatusAccepted, map[string]string{"status": "accepted"})
}

func (s *server) processNotify(req notifyRequest, webhookURL string) {
	description := firstLine(req.CommitMessage)
	if description == "" {
		description = config.SecurityFixMarker
	}

	ctx, cancel := context.WithTimeout(s.bgCtx, s.cfg.Poll.Timeout+30*time.Second)
	defer cancel()

	teamsClient := teams.New(webhookURL)

	if !isPipelineOK(req.PipelineStatus) {
		p := message.Build(message.Input{
			Service:     req.Service,
			Branch:      req.Branch,
			Release:     req.Release,
			Description: description,
			PipelinesOK: false,
		})
		if err := teamsClient.Send(ctx, p); err != nil {
			log.Printf("[%s/%s] teams send (scenario 5): %v", req.Service, req.Release, err)
			return
		}
		log.Printf("[%s/%s] notified scenario 5 (pipeline ERROR)", req.Service, req.Release)
		return
	}

	requested, err := tag.ParseList(req.Envs)
	if err != nil {
		log.Printf("[%s/%s] parse envs %q: %v", req.Service, req.Release, req.Envs, err)
		return
	}
	log.Printf("[%s/%s] envs requested: %v", req.Service, req.Release, requested)

	succeeded := s.pollAll(ctx, req.Service, requested)

	p := message.Build(message.Input{
		Service:     req.Service,
		Branch:      req.Branch,
		Release:     req.Release,
		Description: description,
		Requested:   requested,
		Succeeded:   succeeded,
		PipelinesOK: true,
	})
	if err := teamsClient.Send(ctx, p); err != nil {
		log.Printf("[%s/%s] teams send: %v", req.Service, req.Release, err)
		return
	}
	log.Printf("[%s/%s] notified: requested=%v succeeded=%v", req.Service, req.Release, requested, succeeded)
}

func (s *server) pollAll(ctx context.Context, service string, envs []string) []string {
	since := time.Now().Add(-1 * time.Minute)

	oks := make(map[string]bool, len(envs))
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, env := range envs {
		env := env
		wg.Add(1)
		go func() {
			defer wg.Done()
			appName := s.cfg.AppName(service, env)
			res, err := s.argo.WaitForSync(ctx, appName, since, s.cfg.Poll.Interval, s.cfg.Poll.Timeout)
			if err != nil {
				log.Printf("[%s/%s] argo: %v", env, appName, err)
			}
			log.Printf("[%s/%s] result: %s", env, appName, res)
			if res == argo.ResultOK {
				mu.Lock()
				oks[env] = true
				mu.Unlock()
			}
		}()
	}
	wg.Wait()

	out := make([]string, 0, len(oks))
	for e := range oks {
		out = append(out, e)
	}
	return tag.SortCanonical(out)
}

func (r notifyRequest) missingFields() []string {
	var out []string
	if r.Service == "" {
		out = append(out, "service")
	}
	if r.Release == "" {
		out = append(out, "release")
	}
	if r.PipelineStatus == "" {
		out = append(out, "pipelineStatus")
	}
	return out
}

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

func envDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
