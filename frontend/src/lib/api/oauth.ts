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

import { fetchWithAuth } from "./fetch";

export interface DeviceFlowResponse {
	userCode: string;
	verificationUri: string;
	expiresIn: number;
	interval: number;
	deviceCode: string;
}

export interface PollResponse {
	status: "pending" | "complete" | "expired" | "denied";
	githubUsername?: string;
}

/**
 * Start the GitHub OAuth Device Flow.
 * Returns device code info including the verification URL to open.
 */
export async function startDeviceFlow(fetchImpl?: typeof fetch): Promise<DeviceFlowResponse> {
	const response = await fetchWithAuth(
		"/api/oauth/device",
		{
			method: "POST",
		},
		fetchImpl
	);

	if (!response.ok) {
		const error = await response
			.json()
			.catch(() => ({ error: "Failed to start GitHub authorization" }));
		throw new Error(error.error || "Failed to start GitHub authorization");
	}

	return response.json();
}

/**
 * Poll for the OAuth access token.
 * Call this repeatedly until status is "complete", "expired", or "denied".
 */
export async function pollForToken(
	deviceCode: string,
	fetchImpl?: typeof fetch
): Promise<PollResponse> {
	const response = await fetchWithAuth(
		"/api/oauth/poll",
		{
			method: "POST",
			headers: {
				"Content-Type": "application/json",
			},
			body: JSON.stringify({ deviceCode }),
		},
		fetchImpl
	);

	if (!response.ok) {
		const error = await response
			.json()
			.catch(() => ({ error: "Failed to check authorization status" }));
		throw new Error(error.error || "Failed to check authorization status");
	}

	return response.json();
}

/**
 * Run the complete OAuth Device Flow.
 * Opens the verification URL in a new window and polls until complete.
 *
 * @param onStatusChange - Called with status updates during the flow
 * @returns The GitHub username on success
 * @throws Error if the flow fails, expires, or is denied
 */
export async function runOAuthFlow(
	onStatusChange?: (status: "starting" | "waiting" | "complete" | "error", message?: string) => void
): Promise<string> {
	onStatusChange?.("starting", "Starting GitHub authorization...");

	// Start the device flow
	const deviceFlow = await startDeviceFlow();

	// Open the verification URL with the code pre-filled
	const verificationUrl = `${deviceFlow.verificationUri}?user_code=${deviceFlow.userCode}`;
	window.open(verificationUrl, "_blank", "noopener,noreferrer");

	onStatusChange?.("waiting", `Enter code: ${deviceFlow.userCode}`);

	// Poll for completion
	const pollInterval = Math.max(deviceFlow.interval, 5) * 1000; // At least 5 seconds
	const expiresAt = Date.now() + deviceFlow.expiresIn * 1000;

	while (Date.now() < expiresAt) {
		await new Promise((resolve) => setTimeout(resolve, pollInterval));

		const pollResult = await pollForToken(deviceFlow.deviceCode);

		switch (pollResult.status) {
			case "complete":
				onStatusChange?.("complete", `Connected as ${pollResult.githubUsername}`);
				return pollResult.githubUsername!;

			case "expired":
				throw new Error("Authorization expired. Please try again.");

			case "denied":
				throw new Error("Authorization was denied.");

			case "pending":
				// Continue polling
				break;
		}
	}

	throw new Error("Authorization timed out. Please try again.");
}
