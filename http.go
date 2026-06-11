package stellar

import (
	"encoding/json"
	"net/http"
)

type StatusResponse struct {
	Service     string   `json:"service"`
	Framework   string   `json:"framework"`
	Environment string   `json:"environment"`
	Zone        string   `json:"zone,omitempty"`
	Modules     []string `json:"modules"`
}

func (a *App) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", a.onlyMethod(http.MethodGet, a.handleHealth))
	mux.HandleFunc("/stellar/status", a.onlyMethod(http.MethodGet, a.handleStatus))
	return mux
}

func (a *App) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status":    "ok",
		"framework": "stellar",
	})
}

func (a *App) handleStatus(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, StatusResponse{
		Service:     a.config.AppName,
		Framework:   "stellar",
		Environment: string(a.config.Environment),
		Zone:        a.config.Zone,
		Modules:     a.Modules(),
	})
}

func (a *App) onlyMethod(method string, handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != method {
			w.Header().Set("Allow", method)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		handler(w, r)
	}
}

func writeJSON(w http.ResponseWriter, statusCode int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}
