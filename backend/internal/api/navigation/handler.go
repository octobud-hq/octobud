// Copyright (C) 2025 Austin Beattie
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package navigation

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

// Handler handles navigation-related HTTP routes.
type Handler struct {
	logger      *zap.Logger
	broadcaster *Broadcaster
}

// NewHandler creates a new navigation handler.
func NewHandler(logger *zap.Logger, broadcaster *Broadcaster) *Handler {
	return &Handler{
		logger:      logger,
		broadcaster: broadcaster,
	}
}

// Register registers navigation routes on the provided router.
func (h *Handler) Register(r chi.Router) {
	r.Get("/navigation-events", h.handleSSE)
}

// handleSSE handles Server-Sent Events connections for navigation commands.
func (h *Handler) handleSSE(w http.ResponseWriter, r *http.Request) {
	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // Disable nginx buffering

	// Flush headers immediately
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}

	// Subscribe to events
	eventCh := h.broadcaster.Subscribe()
	defer h.broadcaster.Unsubscribe(eventCh)

	// Send initial connection message
	err := h.sendEvent(w, "connected", map[string]interface{}{
		"message": "Connected to navigation events",
	})
	if err != nil {
		h.logger.Error("Failed to send initial connected event", zap.Error(err))
		return // Exit early, defer will unsubscribe
	}
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}

	// Keep connection alive with periodic pings
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	ctx := r.Context()

	for {
		select {
		case event := <-eventCh:
			// Check if context is done before trying to send
			select {
			case <-ctx.Done():
				return
			default:
			}

			err := h.sendEvent(w, "navigate", event)
			if err != nil {
				h.logger.Error(
					"Failed to send navigation event",
					zap.String("url", event.URL),
					zap.Error(err),
				)
				return // Exit handler loop, defer will unsubscribe
			}

			if flusher, ok := w.(http.Flusher); ok {
				flusher.Flush()
				// Check if context was cancelled after flush
				select {
				case <-ctx.Done():
					return
				default:
				}
			}

		case <-ticker.C:
			// Send keepalive ping
			select {
			case <-ctx.Done():
				return
			default:
			}

			err := h.sendEvent(w, "ping", map[string]interface{}{
				"timestamp": time.Now().Unix(),
			})
			if err != nil {
				h.logger.Warn("Failed to send keepalive ping", zap.Error(err))
				return // Exit handler loop, defer will unsubscribe
			}

			if flusher, ok := w.(http.Flusher); ok {
				flusher.Flush()
			}

		case <-ctx.Done():
			return
		}
	}
}

// sendEvent sends an SSE-formatted event to the client.
// Returns an error if the write fails, which indicates the connection is broken.
func (h *Handler) sendEvent(
	w http.ResponseWriter,
	eventType string,
	data interface{},
	_ ...string, // urlForLogging - reserved for future logging enhancements
) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		h.logger.Error(
			"Failed to marshal SSE event",
			zap.String("event_type", eventType),
			zap.Error(err),
		)
		return err
	}

	// SSE format: "event: <type>\ndata: <json>\n\n"
	_, err = w.Write([]byte("event: " + eventType + "\n"))
	if err != nil {
		h.logger.Error(
			"Failed to write SSE event type",
			zap.String("event_type", eventType),
			zap.Error(err),
		)
		return err
	}
	_, err = w.Write([]byte("data: " + string(jsonData) + "\n\n"))
	if err != nil {
		h.logger.Error(
			"Failed to write SSE event data",
			zap.String("event_type", eventType),
			zap.Error(err),
		)
		return err
	}

	return nil
}
