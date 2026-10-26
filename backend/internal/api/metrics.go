package api

import (
	"fmt"
	"net/http"

	"github.com/olegrand1976/mammomesh/backend/internal/realtime"
)

// MetricsHandler expose les indicateurs au format texte de Prometheus, sur un port interne.
func MetricsHandler(hub *realtime.Hub) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4")
		st := &hub.Stats
		write := func(name, kind, help string, v int64) {
			fmt.Fprintf(w, "# HELP %s %s\n# TYPE %s %s\n%s %d\n", name, help, name, kind, name, v)
		}
		write("mammomesh_sse_clients", "gauge", "Navigateurs connectés au flux temps réel.", st.Clients.Load())
		write("mammomesh_sse_messages_total", "counter", "Messages envoyés aux navigateurs.", st.Sent.Load())
		write("mammomesh_sse_coalesced_total", "counter", "Avis de progression fusionnés avant envoi.", st.Coalesced.Load())
		write("mammomesh_sse_slow_disconnects_total", "counter", "Navigateurs déconnectés pour lenteur.", st.SlowDisconnect.Load())
		write("mammomesh_notify_lag_ms", "gauge", "Dernier délai entre l'écriture d'un événement et sa diffusion.", st.NotifyLagMs.Load())
	})
}
