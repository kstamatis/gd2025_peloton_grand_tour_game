package main

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"mime"
	"net/http"
	"os"
	"os/signal"
	"sort"
	"sync"
	"syscall"
	"time"
)

type ScoreSubmission struct {
	PlayerName string    `json:"player_name"`
	Score      int       `json:"score"`
	Stage      int       `json:"stage"`
	CreatedAt  time.Time `json:"created_at"`
}

type LeaderboardStore struct {
	mu     sync.RWMutex
	scores []ScoreSubmission
}

func NewLeaderboardStore() *LeaderboardStore {
	return &LeaderboardStore{
		scores: []ScoreSubmission{
			{PlayerName: "T. Pogacar", Score: 5400, Stage: 3, CreatedAt: time.Now().UTC()},
			{PlayerName: "J. Vingegaard", Score: 5150, Stage: 3, CreatedAt: time.Now().UTC()},
			{PlayerName: "M. van der Poel", Score: 4850, Stage: 3, CreatedAt: time.Now().UTC()},
		},
	}
}

func (s *LeaderboardStore) Top(limit int) []ScoreSubmission {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if len(s.scores) < limit {
		limit = len(s.scores)
	}
	out := make([]ScoreSubmission, limit)
	copy(out, s.scores[:limit])
	return out
}

func (s *LeaderboardStore) Add(entry ScoreSubmission) {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry.CreatedAt = time.Now().UTC()
	s.scores = append(s.scores, entry)
	sort.Slice(s.scores, func(i, j int) bool {
		return s.scores[i].Score > s.scores[j].Score
	})
	if len(s.scores) > 100 {
		s.scores = s.scores[:100]
	}
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	_ = mime.AddExtensionType(".wasm", "application/wasm")
	store := NewLeaderboardStore()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("GET /api/v1/scores", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(store.Top(10))
	})
	mux.HandleFunc("POST /api/v1/scores", func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, 4096)
		var sub ScoreSubmission
		if err := json.NewDecoder(r.Body).Decode(&sub); err != nil {
			http.Error(w, "invalid JSON payload", http.StatusBadRequest)
			return
		}
		if len(sub.PlayerName) < 1 || len(sub.PlayerName) > 24 || sub.Score < 0 || sub.Score > 1_000_000 {
			http.Error(w, "validation failed", http.StatusUnprocessableEntity)
			return
		}
		store.Add(sub)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(sub)
	})
	mux.Handle("/", http.FileServer(http.Dir("./public")))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		slog.Info("starting Peloton Grand Tour web server", "port", port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server crashed", "error", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
}
