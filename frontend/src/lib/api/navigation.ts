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

/**
 * Navigation Event Source
 * Manages SSE connection for receiving navigation commands from the tray menu.
 * Handles auto-reconnection and cleanup.
 */

import { debugLog } from "$lib/utils/debug";

const LOG_PREFIX = "[SSE Nav]";

interface NavigationEvent {
	url: string;
	timestamp: string;
}

interface NavigationEventSourceOptions {
	onNavigate: (url: string) => void;
	onError?: (error: Event) => void;
	reconnectDelay?: number;
	maxReconnectAttempts?: number;
}

export class NavigationEventSource {
	private eventSource: EventSource | null = null;
	private reconnectTimeoutId: ReturnType<typeof setTimeout> | null = null;
	private healthCheckInterval: ReturnType<typeof setInterval> | null = null;
	private visibilityChangeHandler: (() => void) | null = null;
	private reconnectAttempts = 0;
	private isIntentionallyClosed = false;
	private options: Required<NavigationEventSourceOptions>;

	constructor(options: NavigationEventSourceOptions) {
		this.options = {
			reconnectDelay: options.reconnectDelay ?? 3000,
			maxReconnectAttempts: options.maxReconnectAttempts ?? 10,
			onNavigate: options.onNavigate,
			onError: options.onError ?? (() => {}),
		};
	}

	/**
	 * Connects to the navigation events SSE endpoint.
	 */
	connect(): void {
		if (typeof window === "undefined") {
			return;
		}

		// If we have an existing connection, check if it's still valid
		if (this.eventSource) {
			if (this.eventSource.readyState === EventSource.OPEN) {
				// Already connected and open
				return;
			} else if (this.eventSource.readyState === EventSource.CONNECTING) {
				// Already connecting, wait for it
				return;
			} else {
				// Connection is closed, clean it up before reconnecting
				this.eventSource.close();
				this.eventSource = null;
			}
		}

		this.isIntentionallyClosed = false;
		this.reconnectAttempts = 0;

		try {
			this.eventSource = new EventSource("/api/navigation-events");

			this.eventSource.addEventListener("navigate", (event: MessageEvent) => {
				try {
					// Check if connection is still valid before processing
					if (!this.eventSource) {
						console.warn(
							`${LOG_PREFIX} Received navigate event but eventSource is null, reconnecting...`
						);
						this.connect();
						return;
					}

					const readyState = this.eventSource.readyState;
					if (readyState === EventSource.CLOSED) {
						console.warn(`${LOG_PREFIX} Connection closed, reconnecting...`);
						this.connect();
					}

					const data: NavigationEvent = JSON.parse(event.data);
					this.options.onNavigate(data.url);
				} catch (err) {
					console.error(
						`${LOG_PREFIX} Failed to parse navigation event`,
						err,
						"Event data:",
						event.data
					);
				}
			});

			this.eventSource.addEventListener("connected", () => {
				debugLog(`${LOG_PREFIX} Connected`);
				this.reconnectAttempts = 0; // Reset on successful connection
			});

			let lastPingTime = Date.now();
			this.eventSource.addEventListener("ping", () => {
				lastPingTime = Date.now();
			});

			// Monitor for missing pings - if we don't receive a ping within 60 seconds, reconnect
			const pingMonitor = setInterval(() => {
				const timeSinceLastPing = Date.now() - lastPingTime;
				if (timeSinceLastPing > 60000 && !this.isIntentionallyClosed) {
					console.warn(`${LOG_PREFIX} No ping received in 60s, reconnecting...`);
					clearInterval(pingMonitor);
					this.handleError();
				}
			}, 10000); // Check every 10 seconds

			// Store ping monitor for cleanup
			(this as any)._pingMonitor = pingMonitor;

			this.eventSource.onerror = (error) => {
				// EventSource onerror fires for various reasons, not just fatal errors
				// Check the readyState to determine if this is a real error
				if (!this.eventSource) {
					return;
				}

				const readyState = this.eventSource.readyState;

				if (readyState === EventSource.CLOSED) {
					// Connection is actually closed - this is a real error
					console.error(`${LOG_PREFIX} Connection closed`, error);
					// Use setTimeout to avoid closing during event processing
					setTimeout(() => {
						if (
							!this.isIntentionallyClosed &&
							this.eventSource?.readyState === EventSource.CLOSED
						) {
							this.handleError();
						}
					}, 100);
				}
				// Other states (CONNECTING, OPEN) are normal - don't log
			};

			this.eventSource.onopen = () => {
				this.reconnectAttempts = 0;
				this.startHealthCheck();
				this.setupVisibilityListener();
			};
		} catch (err) {
			console.error(`${LOG_PREFIX} Failed to create event source`, err);
			this.scheduleReconnect();
		}
	}

	/**
	 * Disconnects from the navigation events endpoint.
	 */
	disconnect(): void {
		this.isIntentionallyClosed = true;

		this.stopHealthCheck();
		this.removeVisibilityListener();

		// Clear ping monitor if it exists
		if ((this as any)._pingMonitor) {
			clearInterval((this as any)._pingMonitor);
			(this as any)._pingMonitor = null;
		}

		if (this.reconnectTimeoutId) {
			clearTimeout(this.reconnectTimeoutId);
			this.reconnectTimeoutId = null;
		}

		if (this.eventSource) {
			this.eventSource.close();
			this.eventSource = null;
		}
	}

	/**
	 * Starts periodic health checks to ensure connection is still alive.
	 */
	private startHealthCheck(): void {
		this.stopHealthCheck(); // Clear any existing interval

		// Check connection health every 5 seconds (more frequent to catch issues faster)
		this.healthCheckInterval = setInterval(() => {
			if (this.isIntentionallyClosed) {
				this.stopHealthCheck();
				return;
			}

			if (!this.eventSource) {
				this.handleError();
				return;
			}

			const readyState = this.eventSource.readyState;
			if (readyState === EventSource.CLOSED) {
				this.handleError();
			}
			// Other states are normal - don't log
		}, 5000);
	}

	/**
	 * Stops the health check interval.
	 */
	private stopHealthCheck(): void {
		if (this.healthCheckInterval) {
			clearInterval(this.healthCheckInterval);
			this.healthCheckInterval = null;
		}
	}

	/**
	 * Handles connection errors and schedules reconnection if needed.
	 */
	private handleError(): void {
		if (this.isIntentionallyClosed) {
			return;
		}

		this.stopHealthCheck();

		// Clear ping monitor if it exists
		if ((this as any)._pingMonitor) {
			clearInterval((this as any)._pingMonitor);
			(this as any)._pingMonitor = null;
		}

		// Close the current connection
		if (this.eventSource) {
			this.eventSource.close();
			this.eventSource = null;
		}

		// Check if we should reconnect
		if (this.reconnectAttempts < this.options.maxReconnectAttempts) {
			this.scheduleReconnect();
		} else {
			console.error(
				`${LOG_PREFIX} Max reconnection attempts (${this.options.maxReconnectAttempts}) reached`
			);
		}
	}

	/**
	 * Schedules a reconnection attempt.
	 */
	private scheduleReconnect(): void {
		if (this.isIntentionallyClosed) {
			return;
		}

		this.reconnectAttempts++;
		const delay = this.options.reconnectDelay * this.reconnectAttempts; // Exponential backoff

		if (this.reconnectTimeoutId) {
			clearTimeout(this.reconnectTimeoutId);
		}

		this.reconnectTimeoutId = setTimeout(() => {
			this.reconnectTimeoutId = null;
			this.connect();
		}, delay);
	}

	/**
	 * Returns whether the event source is currently connected.
	 */
	isConnected(): boolean {
		return this.eventSource !== null && this.eventSource.readyState === EventSource.OPEN;
	}

	/**
	 * Sets up a visibility change listener to reconnect when the page becomes visible again.
	 * This handles cases where the device goes to sleep and the connection is lost.
	 */
	private setupVisibilityListener(): void {
		this.removeVisibilityListener(); // Remove any existing listener

		if (typeof document === "undefined") {
			return;
		}

		this.visibilityChangeHandler = () => {
			if (document.visibilityState === "visible") {
				// Page became visible - check connection and reconnect if needed
				// Use a small delay to allow the browser to detect connection state after wake
				setTimeout(() => {
					if (!this.isIntentionallyClosed && !this.isConnected()) {
						debugLog(`${LOG_PREFIX} Page visible but connection not open, reconnecting...`);
						this.handleError();
					}
				}, 500);
			}
		};

		document.addEventListener("visibilitychange", this.visibilityChangeHandler);
	}

	/**
	 * Removes the visibility change listener.
	 */
	private removeVisibilityListener(): void {
		if (this.visibilityChangeHandler && typeof document !== "undefined") {
			document.removeEventListener("visibilitychange", this.visibilityChangeHandler);
			this.visibilityChangeHandler = null;
		}
	}
}
