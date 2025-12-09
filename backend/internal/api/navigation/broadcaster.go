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

// Package navigation implements a broadcaster for navigation events.
package navigation

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"
)

// NavigationEvent represents a navigation command to be sent to clients.
type NavigationEvent struct { //nolint:revive // It's okay that it repeats.
	URL       string    `json:"url"`
	Timestamp time.Time `json:"timestamp"`
}

// Broadcaster manages SSE connections and broadcasts navigation events.
type Broadcaster struct {
	logger  *zap.Logger
	mu      sync.RWMutex
	clients map[chan NavigationEvent]bool
}

// NewBroadcaster creates a new navigation event broadcaster.
func NewBroadcaster(logger *zap.Logger) *Broadcaster {
	return &Broadcaster{
		logger:  logger,
		clients: make(map[chan NavigationEvent]bool),
	}
}

// Broadcast sends a navigation event to all connected clients.
func (b *Broadcaster) Broadcast(ctx context.Context, url string) {
	event := NavigationEvent{
		URL:       url,
		Timestamp: time.Now(),
	}

	b.mu.RLock()
	clients := make([]chan NavigationEvent, 0, len(b.clients))
	for ch := range b.clients {
		clients = append(clients, ch)
	}
	clientCount := len(b.clients)
	b.mu.RUnlock()

	if clientCount == 0 {
		b.logger.Debug("No clients connected for navigation broadcast", zap.String("url", url))
		return
	}

	sentCount := 0
	for _, ch := range clients {
		select {
		case ch <- event:
			sentCount++
		case <-ctx.Done():
			return
		default:
			// Client channel is full, skip (shouldn't happen with buffered channel)
			b.logger.Warn("Client channel full, skipping broadcast", zap.String("url", url))
		}
	}
}

// Subscribe adds a new client and returns their event channel.
func (b *Broadcaster) Subscribe() chan NavigationEvent {
	ch := make(chan NavigationEvent, 1)
	b.mu.Lock()
	b.clients[ch] = true
	b.mu.Unlock()
	return ch
}

// Unsubscribe removes a client.
func (b *Broadcaster) Unsubscribe(ch chan NavigationEvent) {
	b.mu.Lock()
	delete(b.clients, ch)
	close(ch)
	b.mu.Unlock()
}
