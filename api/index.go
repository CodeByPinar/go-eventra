package handler

import (
  "context"
  "encoding/json"
  "net/http"
  "os"
  "strings"
  "sync"

  "eventra/bootstrap"
)

var (
  initOnce sync.Once
  initErr  error
  router   http.Handler
)

func Handler(w http.ResponseWriter, r *http.Request) {
  applyCORS(w, r)
  if r.Method == http.MethodOptions {
    w.WriteHeader(http.StatusNoContent)
    return
  }

  if r.URL.Path == "/health" {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    _ = json.NewEncoder(w).Encode(map[string]string{"status": "ok", "service": "eventra-auth"})
    return
  }

  initOnce.Do(func() {
    router, initErr = bootstrap.NewHTTPHandler(context.Background())
  })

  if initErr != nil {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusInternalServerError)
    _ = json.NewEncoder(w).Encode(map[string]string{
      "error": "backend initialization failed",
    })
    return
  }

  router.ServeHTTP(w, r)
}

func applyCORS(w http.ResponseWriter, r *http.Request) {
  origin := strings.TrimSpace(r.Header.Get("Origin"))
  if origin == "" {
    return
  }

  if !isAllowedOrigin(origin) {
    return
  }

  w.Header().Set("Access-Control-Allow-Origin", origin)
  w.Header().Set("Vary", "Origin")
  w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
  w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
}

func isAllowedOrigin(origin string) bool {
  allowed := []string{"http://localhost:5173", "http://127.0.0.1:5173", "https://eventra-auth.vercel.app"}

  if raw := strings.TrimSpace(os.Getenv("CORS_ALLOWED_ORIGINS")); raw != "" {
    allowed = allowed[:0]
    for _, part := range strings.Split(raw, ",") {
      trimmed := strings.TrimSpace(part)
      if trimmed != "" {
        allowed = append(allowed, trimmed)
      }
    }
  }

  for _, candidate := range allowed {
    if strings.EqualFold(strings.TrimSpace(candidate), origin) {
      return true
    }
  }

  return false
}
