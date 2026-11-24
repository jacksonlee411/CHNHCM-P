package bootstrap

import "net/http"

// Handler exposes process liveness only. It does not claim database readiness
// or any implemented business capability.
func Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		_, _ = w.Write([]byte(`{"status":"ok","component":"chnhcm-server","check":"liveness"}` + "\n"))
	})
	return mux
}
