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
 * Service Worker for Octobud
 * Handles background sync and desktop notifications
 */

const SW_VERSION = '1.0.4';
const CACHE_NAME = `octobud-sw-${SW_VERSION}`;
const NOTIFICATIONS_URL = '/api/notifications';
const POLL_NOTIFICATIONS_URL = '/api/notifications/poll'; // Poll endpoint for service worker
const POLL_INTERVAL = 10000; // 10 seconds
const DB_NAME = 'OctobudDB';
const DB_VERSION = 5; // Stores: pollState, queryConfig

// Debug flag - stored in memory and can be updated via messages from main app
let debugEnabled = false;

/**
 * Debug logger - only logs if debug mode is enabled
 * Can be enabled by the main app via message
 */
function debugLog(...args) {
    if (debugEnabled) {
        console.log(...args);
    }
}

// State
let lastMaxEffectiveSortDate = null; // Track the latest effectiveSortDate from first page
let notificationQuery = 'in:inbox'; // Configurable query (default: inbox)
let pollTimer = null;
let isPolling = false;
let isStartingPolling = false; // Prevent race condition during async startup
let notificationPermission = 'default'; // Track permission status
let notificationsEnabled = true; // Default to enabled

// IndexedDB helpers
function openDB() {
    return new Promise((resolve, reject) => {
        const request = indexedDB.open(DB_NAME, DB_VERSION);

        request.onerror = () => reject(request.error);
        request.onsuccess = async () => {
            const db = request.result;

            // Verify all required stores exist
            const requiredStores = ['pollState', 'queryConfig'];
            const missingStores = requiredStores.filter(storeName => !db.objectStoreNames.contains(storeName));

            if (missingStores.length > 0) {
                // Stores are missing - this shouldn't happen if upgrade worked, but handle it
                console.warn('[SW] Missing object stores:', missingStores, '- closing and deleting database to recreate');
                db.close();

                // Delete the database and reopen it (which will trigger onupgradeneeded)
                try {
                    const deleteRequest = indexedDB.deleteDatabase(DB_NAME);
                    await new Promise((deleteResolve, deleteReject) => {
                        deleteRequest.onsuccess = () => deleteResolve(undefined);
                        deleteRequest.onerror = () => deleteReject(deleteRequest.error);
                        deleteRequest.onblocked = () => {
                            console.warn('[SW] Database delete blocked, will retry');
                            // Wait a bit and retry
                            setTimeout(() => deleteResolve(undefined), 100);
                        };
                    });

                    // Now reopen the database (which will trigger onupgradeneeded)
                    const reopenedDb = await new Promise((reopenResolve, reopenReject) => {
                        const reopenRequest = indexedDB.open(DB_NAME, DB_VERSION);
                        reopenRequest.onerror = () => reopenReject(reopenRequest.error);
                        reopenRequest.onsuccess = () => reopenResolve(reopenRequest.result);
                        reopenRequest.onupgradeneeded = (event) => {
                            const newDb = event.target.result;

                            // Create all required stores
                            if (!newDb.objectStoreNames.contains('pollState')) {
                                newDb.createObjectStore('pollState', { keyPath: 'id' });
                            }
                            if (!newDb.objectStoreNames.contains('queryConfig')) {
                                newDb.createObjectStore('queryConfig', { keyPath: 'id' });
                            }
                        };
                    });

                    // Return the newly opened database
                    resolve(reopenedDb);
                } catch (error) {
                    console.error('[SW] Failed to recreate database:', error);
                    reject(error);
                }
            } else {
                // All stores exist, return the database
                resolve(db);
            }
        };

        request.onupgradeneeded = (event) => {
            const db = event.target.result;

            // Store for poll state (last max effectiveSortDate)
            if (!db.objectStoreNames.contains('pollState')) {
                db.createObjectStore('pollState', { keyPath: 'id' });
            }

            // Store for notification query configuration
            if (!db.objectStoreNames.contains('queryConfig')) {
                db.createObjectStore('queryConfig', { keyPath: 'id' });
            }
        };
    });
}

async function loadState() {
    try {
        const db = await openDB();

        // Check if required object stores exist
        if (!db.objectStoreNames.contains('pollState')) {
            console.warn('[SW] Required object stores missing, skipping state load');
            return Promise.resolve();
        }

        // Load last max effectiveSortDate
        return new Promise((resolve, reject) => {
            const tx = db.transaction('pollState', 'readonly');
            const store = tx.objectStore('pollState');
            const request = store.get('latest');

            request.onsuccess = () => {
                const data = request.result;
                if (data && data.maxEffectiveSortDate) {
                    lastMaxEffectiveSortDate = data.maxEffectiveSortDate;
                    debugLog('[SW] Loaded last max effectiveSortDate from IndexedDB:', lastMaxEffectiveSortDate);
                } else {
                    debugLog('[SW] No previous poll state found in IndexedDB');
                }

                // Load notification query config
                const queryTx = db.transaction('queryConfig', 'readonly');
                const queryStore = queryTx.objectStore('queryConfig');
                const queryRequest = queryStore.get('notificationQuery');

                queryRequest.onsuccess = () => {
                    const queryData = queryRequest.result;
                    if (queryData && queryData.query) {
                        notificationQuery = queryData.query;
                        debugLog('[SW] Loaded notification query from IndexedDB:', notificationQuery);
                    } else {
                        debugLog('[SW] No notification query config found, using default: in:inbox');
                    }
                    resolve();
                };

                queryRequest.onerror = () => {
                    // If query config doesn't exist, that's fine - use default
                    debugLog('[SW] No notification query config found, using default: in:inbox');
                    resolve();
                };
            };

            request.onerror = () => {
                // If poll state doesn't exist, that's fine - start fresh
                // Still need to load query config
                const queryTx = db.transaction('queryConfig', 'readonly');
                const queryStore = queryTx.objectStore('queryConfig');
                const queryRequest = queryStore.get('notificationQuery');

                queryRequest.onsuccess = () => {
                    const queryData = queryRequest.result;
                    if (queryData && queryData.query) {
                        notificationQuery = queryData.query;
                        debugLog('[SW] Loaded notification query from IndexedDB:', notificationQuery);
                    } else {
                        debugLog('[SW] No notification query config found, using default: in:inbox');
                    }
                    resolve();
                };

                queryRequest.onerror = () => {
                    debugLog('[SW] No notification query config found, using default: in:inbox');
                    resolve();
                };
            };
        });
    } catch (error) {
        console.error('[SW] Failed to load state:', error);
        return Promise.resolve();
    }
}

async function savePollState(maxEffectiveSortDate) {
    try {
        const db = await openDB();

        // Check if object store exists
        if (!db.objectStoreNames.contains('pollState')) {
            console.warn('[SW] pollState object store missing, skipping save');
            return;
        }

        return new Promise((resolve, reject) => {
            const tx = db.transaction('pollState', 'readwrite');
            const store = tx.objectStore('pollState');
            const request = store.put({
                id: 'latest',
                maxEffectiveSortDate: maxEffectiveSortDate,
                timestamp: Date.now()
            });

            request.onsuccess = () => {
                lastMaxEffectiveSortDate = maxEffectiveSortDate;
                debugLog('[SW] Saved poll state to IndexedDB - max effectiveSortDate:', maxEffectiveSortDate);
                resolve();
            };

            request.onerror = () => reject(request.error);
        });
    } catch (error) {
        console.error('[SW] Failed to save poll state:', error);
    }
}

async function saveNotificationQuery(query) {
    try {
        const db = await openDB();

        // Check if object store exists
        if (!db.objectStoreNames.contains('queryConfig')) {
            console.warn('[SW] queryConfig object store missing, skipping save');
            return;
        }

        return new Promise((resolve, reject) => {
            const tx = db.transaction('queryConfig', 'readwrite');
            const store = tx.objectStore('queryConfig');
            const request = store.put({
                id: 'notificationQuery',
                query: query,
                timestamp: Date.now()
            });

            request.onsuccess = () => {
                notificationQuery = query;
                debugLog('[SW] Saved notification query to IndexedDB:', query);
                resolve();
            };

            request.onerror = () => reject(request.error);
        });
    } catch (error) {
        console.error('[SW] Failed to save notification query:', error);
    }
}


// Fetch notifications using the configured query
async function fetchNotifications() {
    try {
        // Use poll endpoint for much smaller payload
        const url = `${POLL_NOTIFICATIONS_URL}?page=1&pageSize=10&query=${encodeURIComponent(notificationQuery)}`;
        debugLog('[SW] Fetching poll notifications from:', url);
        const response = await fetch(url, {
            credentials: 'include',
            cache: 'no-cache',
            headers: {
                'Cache-Control': 'no-cache'
            }
        });

        if (!response.ok) {
            throw new Error(`Notifications fetch failed: ${response.status}`);
        }

        const data = await response.json();
        const notifications = data.notifications || [];
        debugLog('[SW] Received', notifications.length, 'notifications from API');
        return notifications;
    } catch (error) {
        console.error('[SW] Failed to fetch notifications:', error);
        debugLog('[SW] Fetch error:', error);
        return [];
    }
}

// Check if all windows are hidden
async function areAllWindowsHidden() {
    try {
        const clients = await self.clients.matchAll({ type: 'window', includeUncontrolled: true });
        debugLog('[SW] Found', clients.length, 'client window(s)');
        if (clients.length === 0) {
            debugLog('[SW] No windows found - treating as hidden');
            return true; // No windows = hidden
        }
        const visibilityStates = clients.map(client => ({
            url: client.url,
            visibilityState: client.visibilityState
        }));
        debugLog('[SW] Window visibility states:', visibilityStates);
        const allHidden = clients.every(client => client.visibilityState === 'hidden');
        debugLog('[SW] All windows hidden:', allHidden);
        return allHidden;
    } catch (error) {
        console.error('[SW] Failed to check window visibility:', error);
        debugLog('[SW] Window visibility check error:', error);
        // Default to showing notifications if we can't check
        return false;
    }
}

// Check if we should show desktop notifications
// Returns true if:
// - All windows are hidden, OR
// - Any visible window is NOT on the inbox route (/views/inbox)
async function shouldShowDesktopNotifications() {
    try {
        const clients = await self.clients.matchAll({ type: 'window', includeUncontrolled: true });
        debugLog('[SW] Found', clients.length, 'client window(s)');

        if (clients.length === 0) {
            debugLog('[SW] No windows found - should show notifications');
            return true; // No windows = show notifications
        }

        // Check if all windows are hidden
        const allHidden = clients.every(client => client.visibilityState === 'hidden');
        if (allHidden) {
            debugLog('[SW] All windows hidden - should show notifications');
            return true;
        }

        // If any window is visible, check if any visible window is NOT on inbox route
        const visibleClients = clients.filter(client => client.visibilityState === 'visible');
        debugLog('[SW] Found', visibleClients.length, 'visible window(s)');

        // Log all client details for debugging
        const clientDetails = clients.map(client => ({
            url: client.url,
            visibilityState: client.visibilityState,
            focused: client.focused
        }));
        debugLog('[SW] All client details:', JSON.stringify(clientDetails, null, 2));

        for (const client of visibleClients) {
            try {
                const clientUrl = new URL(client.url);
                const pathname = clientUrl.pathname;
                const fullUrl = client.url;

                debugLog('[SW] Checking visible client - full URL:', fullUrl);
                debugLog('[SW] Checking visible client - pathname:', pathname);
                debugLog('[SW] Checking visible client - startsWith /views/inbox?', pathname.startsWith('/views/inbox'));

                // If any visible window is NOT on /views/inbox, show notifications
                // This allows notifications when user is on other views (archive, custom views, etc.)
                if (!pathname.startsWith('/views/inbox')) {
                    debugLog('[SW] Visible window not on inbox route:', pathname, '- should show notifications');
                    return true;
                } else {
                    debugLog('[SW] Visible window IS on inbox route:', pathname);
                }
            } catch (urlError) {
                // If we can't parse the URL, be conservative and show notifications
                console.error('[SW] Failed to parse client URL:', client.url, urlError);
                debugLog('[SW] Failed to parse client URL:', client.url, '- error:', urlError.message, '- should show notifications');
                return true;
            }
        }

        // All visible windows are on /views/inbox - don't show notifications
        debugLog('[SW] All visible windows are on inbox route - should NOT show notifications');
        return false;
    } catch (error) {
        console.error('[SW] Failed to check if should show notifications:', error);
        debugLog('[SW] Error checking notification visibility:', error);
        // Default to showing notifications if we can't check
        return true;
    }
}

// Check if notification should trigger desktop notification
// Note: This only checks basic filters - the main logic is in pollForUpdates
function shouldShowNotification(notification) {
    // Don't show if archived or muted
    if (notification.archived || notification.muted) {
        return false;
    }

    return true;
}

// Check if notifications are globally muted
let isGloballyMuted = false;
let muteCheckTime = 0;
const MUTE_CHECK_INTERVAL = 30000; // Check every 30 seconds

async function checkGlobalMute() {
    const now = Date.now();
    // Only check if it's been more than MUTE_CHECK_INTERVAL since last check
    if (now - muteCheckTime < MUTE_CHECK_INTERVAL && muteCheckTime > 0) {
        return isGloballyMuted;
    }

    try {
        const response = await fetch('/api/user/mute-status', {
            credentials: 'include',
            cache: 'no-cache'
        });

        if (response.ok) {
            const status = await response.json();
            isGloballyMuted = status.isMuted && status.mutedUntil && new Date(status.mutedUntil) > new Date();
            muteCheckTime = now;
            return isGloballyMuted;
        }
    } catch (error) {
        console.error('[SW] Failed to check mute status:', error);
        // On error, don't block notifications
    }

    return false;
}

// Check if we have permission to show notifications
async function hasNotificationPermission() {
    // First check if notifications are enabled in settings
    if (!notificationsEnabled) {
        return false;
    }

    // Check if globally muted
    if (await checkGlobalMute()) {
        return false;
    }

    // Check if registration and showNotification are available
    if (!self.registration || typeof self.registration.showNotification !== 'function') {
        return false;
    }

    // If we know permission is denied, don't try
    if (notificationPermission === 'denied') {
        return false;
    }

    // If we know permission is granted, we're good
    if (notificationPermission === 'granted') {
        return true;
    }

    // Default: try anyway and let it fail gracefully
    return true;
}

// Show summary notification when there are many new notifications
async function showSummaryNotification(count) {
    const title = 'New notifications';
    const body = `${count} new notification${count === 1 ? '' : 's'} in your inbox`;

    const options = {
        body: body,
        icon: '/favicon.png',
        tag: 'summary-' + Date.now(), // Unique tag for summary notifications
        renotify: true,
        silent: false,
        data: {
            notificationId: 'summary',
            githubId: 'summary',
            isSummary: true
        }
    };

    try {
        const timeoutPromise = new Promise((_, reject) =>
            setTimeout(() => reject(new Error('showNotification timeout after 5s')), 5000)
        );

        const notificationPromise = self.registration.showNotification(title, options);
        await Promise.race([notificationPromise, timeoutPromise]);
        return true;
    } catch (error) {
        console.error('[SW] ❌ Failed to show summary notification:', error);
        return false;
    }
}

// Show desktop notification
// Returns true if notification was successfully shown, false otherwise
async function showDesktopNotification(notification) {
    // Check if we can show notifications
    if (!(await hasNotificationPermission())) {
        console.warn('[SW] Cannot show notification:', {
            hasRegistration: !!self.registration,
            hasShowNotification: typeof self.registration?.showNotification,
            permission: notificationPermission
        });
        return false;
    }

    // Handle both BackendNotificationResponse (from API) and Notification (from main app)
    // Use githubId if it exists and is non-empty, otherwise fall back to stringified id
    const id = (notification.githubId && notification.githubId.trim() !== '')
        ? String(notification.githubId)
        : String(notification.id);
    debugLog('[SW] Preparing to show desktop notification for', id, 'githubId:', notification.githubId, 'id:', notification.id);

    // BackendNotificationResponse has repository.fullName, Notification has repoFullName
    const repoName = notification.repository?.fullName || notification.repoFullName || 'Unknown repository';
    const subjectTitle = notification.subjectTitle || 'New notification';
    const subjectTypeRaw = notification.subjectType || 'notification';
    const reason = notification.reason || 'notification';
    debugLog('[SW] Notification details - repo:', repoName, 'title:', subjectTitle, 'type:', subjectTypeRaw, 'reason:', reason);

    // Format subject type for display (pull_request -> Pull Request, issue -> Issue)
    const formatSubjectType = (type) => {
        const normalized = type.toLowerCase();
        if (normalized === 'pull_request' || normalized === 'pullrequest') {
            return 'Pull Request';
        }
        if (normalized === 'issue') {
            return 'Issue';
        }
        // Capitalize first letter of each word
        return type.split('_').map(word =>
            word.charAt(0).toUpperCase() + word.slice(1).toLowerCase()
        ).join(' ');
    };

    const subjectType = formatSubjectType(subjectTypeRaw);

    // Title is repo name
    const title = repoName;

    // Body: [subject title truncated] + [subject type] · [reason]
    // Truncate subject title to reasonable length (e.g., 60 chars)
    const truncatedTitle = subjectTitle.length > 60
        ? subjectTitle.substring(0, 57) + '...'
        : subjectTitle;

    // Format: "subject title" + "subjectType · reason" (using middle dot separator)
    const body = `${truncatedTitle}\n${subjectType} · ${reason}`;

    // Build options - keep it simple to avoid browser compatibility issues
    const options = {
        body: body,
        icon: '/favicon.png',
        tag: id, // Prevent duplicate notifications
        renotify: true, // Re-alert even for notifications with same tag
        // requireInteraction: true, // Set to true to encourage macOS to show as alert instead of just notification center
        silent: false, // Ensure sound plays
        data: {
            notificationId: id,
            githubId: (notification.githubId && notification.githubId.trim() !== '')
                ? String(notification.githubId)
                : String(notification.id)
            // Note: Not including URL - clicking notification will just focus the app
        }
    };

    try {
        // Add timeout to detect if showNotification hangs
        const timeoutPromise = new Promise((_, reject) =>
            setTimeout(() => reject(new Error('showNotification timeout after 5s')), 5000)
        );

        const notificationPromise = self.registration.showNotification(title, options);

        // Race between notification and timeout
        await Promise.race([notificationPromise, timeoutPromise]);

        // Notification successfully shown
        debugLog('[SW] Successfully showed notification for', id);
        return true;
    } catch (error) {
        const errorDetails = {
            error: error?.message || String(error),
            errorName: error?.name,
            stack: error?.stack,
            id,
            title,
            hasRegistration: !!self.registration,
            registrationType: typeof self.registration,
            showNotificationType: typeof self.registration?.showNotification,
            permission: notificationPermission,
            options: JSON.stringify(options)
        };
        console.error('[SW] ❌ Failed to show notification:', errorDetails);

        // Don't re-throw - just log the error and continue
        // This allows other notifications to still be shown
        return false;
    }
}

// Notify all clients about new notifications
async function notifyClients(newNotifications) {
    try {
        const clients = await self.clients.matchAll({
            includeUncontrolled: true,
            type: 'window'
        });

        if (clients.length > 0) {
            clients.forEach(client => {
                client.postMessage({
                    type: 'NEW_NOTIFICATIONS',
                    count: newNotifications.length,
                    timestamp: Date.now()
                });
            });
        }
    } catch (error) {
        console.error('[SW] Failed to notify clients:', error);
    }
}

// Main polling function
async function pollForUpdates() {
    if (isPolling) {
        debugLog('[SW] Poll already in progress, skipping');
        return; // Prevent concurrent polls
    }

    isPolling = true;
    const pollStartTime = Date.now();

    try {
        debugLog('[SW] ========== Starting poll ==========');
        debugLog('[SW] Query:', notificationQuery);
        debugLog('[SW] Last max effectiveSortDate:', lastMaxEffectiveSortDate);

        // Always fetch notifications on each poll
        const notifications = await fetchNotifications();
        debugLog('[SW] Fetched', notifications.length, 'notifications from inbox');

        if (notifications.length > 0) {
            // Find the max effectiveSortDate from the first page
            let currentMaxEffectiveSortDate = null;
            for (const notification of notifications) {
                const effectiveSortDate = notification.effectiveSortDate;
                if (effectiveSortDate) {
                    const date = new Date(effectiveSortDate);
                    if (!currentMaxEffectiveSortDate || date > new Date(currentMaxEffectiveSortDate)) {
                        currentMaxEffectiveSortDate = effectiveSortDate;
                    }
                }
            }
            debugLog('[SW] Current max effectiveSortDate from first page:', currentMaxEffectiveSortDate);

            // Find new notifications: those with effectiveSortDate > lastMaxEffectiveSortDate
            const newNotifications = notifications.filter(n => {
                const effectiveSortDate = n.effectiveSortDate;
                if (!effectiveSortDate) {
                    const logId = (n.githubId && n.githubId.trim() !== '') ? n.githubId : String(n.id);
                    debugLog('[SW] Notification', logId, 'has no effectiveSortDate, skipping');
                    return false;
                }

                const date = new Date(effectiveSortDate);
                const isNew = !lastMaxEffectiveSortDate || date > new Date(lastMaxEffectiveSortDate);
                const shouldShow = shouldShowNotification(n);

                if (isNew) {
                    const logId = (n.githubId && n.githubId.trim() !== '') ? n.githubId : String(n.id);
                    debugLog('[SW] Notification', logId, '- effectiveSortDate:', effectiveSortDate, 'isNew:', isNew, 'shouldShow:', shouldShow, 'archived:', n.archived, 'muted:', n.muted);
                }

                return isNew && shouldShow;
            });

            debugLog('[SW] Filtered to', newNotifications.length, 'notifications that are new and should be shown');

            if (newNotifications.length > 0) {
                // Check if we should show desktop notifications
                // Show if: all windows hidden OR any visible window is NOT on inbox route
                const shouldShow = await shouldShowDesktopNotifications();
                debugLog('[SW] Should show desktop notifications:', shouldShow);
                debugLog('[SW] Notifications enabled:', notificationsEnabled);

                // Show desktop notifications if enabled in settings AND should show
                if (notificationsEnabled && shouldShow) {
                    if (newNotifications.length > 3) {
                        // Show summary notification if more than 3
                        debugLog('[SW] Showing summary notification for', newNotifications.length, 'notifications');
                        await showSummaryNotification(newNotifications.length);
                    } else {
                        // Show individual notifications (up to 3)
                        debugLog('[SW] Showing', newNotifications.length, 'individual notifications');
                        for (const notification of newNotifications) {
                            try {
                                const id = (notification.githubId && notification.githubId.trim() !== '')
                                    ? String(notification.githubId)
                                    : String(notification.id);
                                debugLog('[SW] Showing desktop notification for', id);
                                await showDesktopNotification(notification);
                                debugLog('[SW] Successfully showed notification for', id);
                                // Small delay between notifications to avoid overwhelming the user
                                await new Promise(resolve => setTimeout(resolve, 100));
                            } catch (error) {
                                const errorId = (notification.githubId && notification.githubId.trim() !== '')
                                    ? notification.githubId
                                    : notification.id;
                                console.error('[SW] Failed to show notification for:', errorId, {
                                    error: error?.message || String(error),
                                    errorName: error?.name
                                });
                            }
                        }
                    }
                } else {
                    // Notifications disabled or shouldn't show (user is on inbox) - don't show but still update state
                    debugLog('[SW] Not showing notifications - disabled or user viewing inbox.');
                }

                // Notify clients regardless of notification setting or window visibility
                // This allows the app to refresh its UI when new notifications appear
                debugLog('[SW] Notifying clients about', newNotifications.length, 'new notifications');
                await notifyClients(newNotifications);

                // Update last max effectiveSortDate ONLY when we found new notifications
                // This ensures we don't advance the timestamp prematurely when polling while window is visible
                if (currentMaxEffectiveSortDate) {
                    debugLog('[SW] Updating last max effectiveSortDate to:', currentMaxEffectiveSortDate, '(found new notifications)');
                    await savePollState(currentMaxEffectiveSortDate);
                } else {
                    debugLog('[SW] No effectiveSortDate found in notifications, keeping previous max');
                }
            } else {
                debugLog('[SW] No notifications to show after filtering');
                // Don't update timestamp when no new notifications found
                // This prevents the timestamp from advancing when the window is visible
                // and the user has already seen all current notifications
                debugLog('[SW] Keeping last max effectiveSortDate unchanged (no new notifications)');
            }
        } else {
            // No notifications in inbox - keep the last max date (don't reset it)
            debugLog('[SW] No notifications in inbox - keeping last max effectiveSortDate');
        }

        const pollDuration = Date.now() - pollStartTime;
        debugLog('[SW] ========== Poll complete in', pollDuration, 'ms ==========');

        // Send heartbeat to all clients so they know the SW is alive and polling
        await sendHeartbeat();
    } catch (error) {
        console.error('[SW] Error during poll:', error);
        debugLog('[SW] Poll error:', error);
    } finally {
        isPolling = false;
    }
}

// Send heartbeat to all clients to indicate SW is alive and polling
async function sendHeartbeat() {
    try {
        const clients = await self.clients.matchAll({
            includeUncontrolled: true,
            type: 'window'
        });

        if (clients.length > 0) {
            const timestamp = Date.now();
            clients.forEach(client => {
                client.postMessage({
                    type: 'SW_HEARTBEAT',
                    timestamp: timestamp
                });
            });
            debugLog('[SW] Sent heartbeat to', clients.length, 'client(s)');
        }
    } catch (error) {
        console.error('[SW] Failed to send heartbeat:', error);
    }
}

// Start polling
// If skipImmediatePoll is true, don't do an immediate poll - just set up the interval
// This is useful when the main app has already fetched notifications
function startPolling(skipImmediatePoll = false) {
    if (pollTimer || isStartingPolling) {
        return; // Already polling or starting up
    }

    // Set flag immediately to prevent race conditions during async startup
    isStartingPolling = true;

    // Load initial state
    loadState().then(() => {
        debugLog('[SW] State loaded - lastMaxEffectiveSortDate:', lastMaxEffectiveSortDate);
        debugLog('[SW] State loaded - notificationQuery:', notificationQuery);
        // Do initial poll unless skipped
        if (!skipImmediatePoll) {
            debugLog('[SW] Starting initial poll');
            pollForUpdates();
        } else {
            debugLog('[SW] Skipping initial poll');
        }

        // Set up periodic polling
        debugLog('[SW] Setting up periodic polling every', POLL_INTERVAL, 'ms');
        pollTimer = setInterval(() => {
            pollForUpdates();
        }, POLL_INTERVAL);

        // Clear the starting flag now that we're fully set up
        isStartingPolling = false;
    }).catch((error) => {
        console.error('[SW] Failed to start polling:', error);
        isStartingPolling = false;
    });
}

// Stop polling
function stopPolling() {
    if (pollTimer) {
        clearInterval(pollTimer);
        pollTimer = null;
    }
    isStartingPolling = false;
}

// Service Worker Lifecycle Events

self.addEventListener('install', (event) => {
    debugLog('[SW] Installing service worker v' + SW_VERSION);
    // Skip waiting to activate immediately
    self.skipWaiting();
});

self.addEventListener('activate', (event) => {
    debugLog('[SW] Activating service worker v' + SW_VERSION);
    debugLog('[SW] Registration available:', !!self.registration);
    debugLog('[SW] showNotification available:', typeof self.registration?.showNotification);

    event.waitUntil(
        Promise.all([
            self.clients.claim(),
            // Clean up old caches if needed
            caches.keys().then(cacheNames => {
                return Promise.all(
                    cacheNames
                        .filter(cacheName => cacheName.startsWith('octobud-sw-') && cacheName !== CACHE_NAME)
                        .map(cacheName => caches.delete(cacheName))
                );
            })
        ]).then(async () => {
            // Main app will send the current setting via message
            // Default to enabled until we receive the actual setting

            startPolling();
        })
    );
});

// Handle notification clicks
self.addEventListener('notificationclick', (event) => {
    event.notification.close();

    const notificationId = event.notification.data?.notificationId;
    const githubId = event.notification.data?.githubId;

    event.waitUntil(
        (async () => {
            try {
                // Try to focus existing window first
                const clients = await self.clients.matchAll({
                    type: 'window',
                    includeUncontrolled: true
                });

                // Prefer focusing existing app window
                const appClient = clients.find(client => {
                    try {
                        const clientUrl = new URL(client.url);
                        return clientUrl.origin === self.location.origin;
                    } catch {
                        return client.url.includes(self.location.origin);
                    }
                });

                // Prefer githubId if it exists and is non-empty, otherwise use notificationId
                const idToOpen = (githubId && githubId.trim() !== '') ? githubId : notificationId;
                debugLog('[SW] Opening notification - githubId:', githubId, 'notificationId:', notificationId, 'idToOpen:', idToOpen);
                const isSummary = event.notification.data?.isSummary === true;

                if (appClient) {
                    // Focus existing window and open the notification in the app
                    await appClient.focus();

                    // Post message to open the notification in the app
                    // For summary notifications, just open inbox without specific ID
                    if (!isSummary && idToOpen) {
                        debugLog('[SW] Sending OPEN_NOTIFICATION message to client:', {
                            type: 'OPEN_NOTIFICATION',
                            notificationId: idToOpen,
                            clientUrl: appClient.url
                        });
                        try {
                            appClient.postMessage({
                                type: 'OPEN_NOTIFICATION',
                                notificationId: idToOpen
                            });
                        } catch (error) {
                            console.error('[SW] Error sending OPEN_NOTIFICATION message:', error);
                        }
                    } else {
                        // Summary notification or no ID - just refresh the app
                        appClient.postMessage({
                            type: 'NEW_NOTIFICATIONS',
                            count: 0,
                            timestamp: Date.now()
                        });
                    }
                } else {
                    // No app window open, open the app
                    // For summary notifications or no ID, just open inbox
                    const url = (!isSummary && idToOpen) ? `/views/inbox?id=${encodeURIComponent(idToOpen)}` : '/views/inbox';
                    debugLog('[SW] Opening new window with URL:', url);
                    await self.clients.openWindow(url);
                }
            } catch (error) {
                console.error('[SW] Error handling notification click:', error);
                // Fallback: just open the app without notification ID if there's an error
                try {
                    const clients = await self.clients.matchAll({
                        type: 'window',
                        includeUncontrolled: true
                    });
                    if (clients.length > 0) {
                        await clients[0].focus();
                    } else {
                        await self.clients.openWindow('/');
                    }
                } catch (fallbackError) {
                    console.error('[SW] Fallback error handling notification click:', fallbackError);
                }
            }
        })()
    );
});

// Handle messages from main app
self.addEventListener('message', async (event) => {
    // Ensure polling is active whenever we receive a message
    // This handles the case where the SW was terminated and woke up from a message
    const needsPollingRestart = !pollTimer;

    if (event.data.type === 'SKIP_WAITING') {
        self.skipWaiting();
        if (needsPollingRestart) startPolling(true);
    } else if (event.data.type === 'FORCE_POLL') {
        // FORCE_POLL should trigger immediate poll after state is loaded
        if (needsPollingRestart) {
            // If polling wasn't active, start it (which loads state and polls)
            startPolling();
        } else {
            // If polling was active, just trigger an immediate poll
            pollForUpdates();
        }
    } else if (event.data.type === 'NOTIFICATION_PERMISSION') {
        // Update permission status from main app
        notificationPermission = event.data.permission || 'default';
        // Skip immediate poll - this is just a settings update
        if (needsPollingRestart) startPolling(true);
    } else if (event.data.type === 'NOTIFICATION_SETTING_CHANGED') {
        // Update notification enabled setting from main app
        const newValue = event.data.enabled !== false; // Default to true
        if (notificationsEnabled !== newValue) {
            notificationsEnabled = newValue;
        }
        // Skip immediate poll - this is just a settings update
        if (needsPollingRestart) startPolling(true);
    } else if (event.data.type === 'TEST_NOTIFICATION') {
        // Show a test notification
        const testNotification = {
            id: 'test-' + Date.now(),
            githubId: 'test-' + Date.now(),
            repository: { fullName: 'octobud/test' },
            subjectTitle: 'This is a test notification',
            subjectType: 'issue',
            reason: 'test',
            archived: false,
            muted: false
        };
        showDesktopNotification(testNotification).catch(error => {
            console.error('[SW] Failed to show test notification:', error);
        });
        if (needsPollingRestart) startPolling(true);
    } else if (event.data.type === 'DEBUG_ENABLED') {
        // Toggle debug logging
        debugEnabled = event.data.enabled !== false;
    } else if (event.data.type === 'SET_NOTIFICATION_QUERY') {
        // Update notification query configuration
        const newQuery = event.data.query || 'in:inbox';
        if (notificationQuery !== newQuery) {
            notificationQuery = newQuery;
            await saveNotificationQuery(newQuery);
            // Trigger immediate poll with new query
            if (needsPollingRestart) {
                startPolling();
            } else {
                pollForUpdates();
            }
        } else {
            // Query unchanged, just restart polling if needed
            if (needsPollingRestart) startPolling(true);
        }
    } else if (event.data.type === 'RESET_POLL_STATE') {
        // Reset the poll state - used after login or initial setup
        // This clears the lastMaxEffectiveSortDate so all notifications are treated as new
        debugLog('[SW] Resetting poll state');
        lastMaxEffectiveSortDate = null;
        // Also clear from IndexedDB
        try {
            const db = await openDB();
            if (db.objectStoreNames.contains('pollState')) {
                const transaction = db.transaction('pollState', 'readwrite');
                const store = transaction.objectStore('pollState');
                store.delete('latest');
                debugLog('[SW] Cleared poll state from IndexedDB');
            }
        } catch (error) {
            console.error('[SW] Failed to clear poll state from IndexedDB:', error);
        }
        // Restart polling if needed
        if (needsPollingRestart) startPolling(true);
    } else {
        // Unknown message type, but still restart polling if needed
        if (needsPollingRestart) startPolling(true);
    }
});

// No fetch event listener needed - service worker only handles background polling
// and desktop notifications. All requests (images, API calls, etc.) are handled
// normally by the browser without service worker interception.

