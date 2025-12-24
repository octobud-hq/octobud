<script lang="ts">
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

	// ============================================================================
	// IMPORTS
	// ============================================================================

	// CSS
	import "../app.css";

	// Framework & Core
	import { getContext, onDestroy, onMount, setContext } from "svelte";
	import { get } from "svelte/store";
	import { browser } from "$app/environment";
	import { goto, afterNavigate } from "$app/navigation";
	import { resolve } from "$app/paths";
	import { page as pageStore } from "$app/stores";
	import { invalidateAll } from "$app/navigation";
	import type { NavigationState } from "$lib/state/interfaces/common";
	import type { LayoutData } from "./$types";
	import {
		hasSyncSettingsConfigured,
		fetchUserInfo,
		hasGitHubConnected,
	} from "$lib/stores/authStore";

	// Components - Dialogs
	import ViewDialog from "$lib/components/dialogs/ViewDialog.svelte";
	import ConfirmDialog from "$lib/components/dialogs/ConfirmDialog.svelte";
	import BulkActionConfirmDialog from "$lib/components/dialogs/BulkActionConfirmDialog.svelte";
	import SyntaxGuideDialog from "$lib/components/dialogs/SyntaxGuideDialog.svelte";
	import CustomSnoozeDateDialog from "$lib/components/dialogs/CustomSnoozeDateDialog.svelte";

	// Components - Layout
	import PageSidebar from "$lib/components/sidebar/PageSidebar.svelte";
	import PageHeader from "$lib/components/shared/PageHeader.svelte";
	import ToastContainer from "$lib/components/shared/ToastContainer.svelte";
	import LayoutSkeleton from "$lib/components/shared/LayoutSkeleton.svelte";
	import SyncStatusBanner from "$lib/components/shared/SyncStatusBanner.svelte";
	import SWHealthBanner from "$lib/components/shared/SWHealthBanner.svelte";
	import UpdateBanner from "$lib/components/shared/UpdateBanner.svelte";
	import RestartBanner from "$lib/components/shared/RestartBanner.svelte";
	import { getUpdateStore } from "$lib/stores/updateStore";

	// State Controllers
	import { createViewDialogController } from "$lib/state/viewDialogController";
	import { createNotificationPageController } from "$lib/state/notificationPageController";

	// Stores
	import { toastStore } from "$lib/stores/toastStore";

	// API & Types
	import { fetchViews } from "$lib/api/views";
	import type { Tag } from "$lib/api/tags";

	// Stores
	import { currentTime } from "$lib/stores/timeStore";

	// Utilities
	import {
		registerServiceWorker,
		requestNotificationPermission,
		getNotificationPermission,
		sendMessageToSW,
		sendNotificationSettingToSW,
		forceSWPoll,
		ensureSWPolling,
	} from "$lib/utils/serviceWorkerRegistration";
	import {
		getNotificationSettingsStore,
		isNotificationEnabled,
	} from "$lib/stores/notificationSettings";
	import { getThemeStore } from "$lib/stores/themeStore";
	import { getSWHealthStore } from "$lib/stores/swHealthStore";
	import { setupServiceWorkerHandlers } from "$lib/utils/serviceWorkerMessageHandler";
	import { NavigationEventSource } from "$lib/api/navigation";
	import { initializeFavicon, updateFaviconBadge } from "$lib/utils/faviconBadge";

	// Props
	export let data: LayoutData;

	// Component refs
	let pageSidebarComponent: any = null;
	let pageHeaderComponent: any = null;

	// ============================================================================
	// UNIFIED PAGE CONTROLLER - SINGLE SOURCE OF TRUTH
	// ============================================================================

	// Store current tags for getTags function
	let currentTags: Tag[] = data.tags ?? [];

	// Update tags when data changes
	$: currentTags = data.tags ?? [];

	// ============================================================================
	// SERVICE WORKER MESSAGE LISTENER - Set up immediately, not in onMount
	// ============================================================================

	// Store reference to pageController for service worker handlers
	// This will be set after pageController is created
	let pageControllerRef: ReturnType<typeof createNotificationPageController> | null = null;

	// Service worker handler cleanup function
	let swCleanup: (() => void) | null = null;

	// Navigation event source for tray menu navigation
	let navEventSource: NavigationEventSource | null = null;

	// Set up the service worker message handler at top level
	if (browser) {
		swCleanup = setupServiceWorkerHandlers({
			getPageController: () => pageControllerRef,
		});
	}

	// ============================================================================
	// INITIAL SYNC BANNER STATE
	// ============================================================================
	// Simple approach: store timestamp in localStorage when setup completes.
	// Show banner if: timestamp exists AND no notifications AND < 2 minutes passed.

	const SYNC_STARTED_KEY = "octobud:sync-started-at";
	const SYNC_TIMEOUT_MS = 2 * 60 * 1000; // 2 minutes

	let syncStartTime: number | null = null;

	function readSyncStartTime() {
		if (browser) {
			const stored = localStorage.getItem(SYNC_STARTED_KEY);
			syncStartTime = stored ? parseInt(stored, 10) : null;
		}
	}

	function clearSyncStartTime() {
		if (browser) {
			localStorage.removeItem(SYNC_STARTED_KEY);
		}
		syncStartTime = null;
	}

	function dismissSyncBanner() {
		clearSyncStartTime();
	}

	// Read sync start time on initial load
	if (browser) {
		readSyncStartTime();
	}

	// ============================================================================
	// SERVICE WORKER HEALTH BANNER STATE
	// ============================================================================

	// Get the health store (singleton) for reactive access to showBanner
	const swHealthStoreForBanner = browser ? getSWHealthStore() : null;
	const swHealthBannerStore = swHealthStoreForBanner?.showBanner;
	$: showSWHealthBanner = swHealthBannerStore ? $swHealthBannerStore : false;

	function dismissSWHealthBanner() {
		swHealthStoreForBanner?.dismissBanner();
	}

	function refreshPage() {
		if (browser) {
			window.location.reload();
		}
	}

	// Initialize update store
	const updateStore = browser ? getUpdateStore() : null;

	const pageController = createNotificationPageController(
		{
			views: data.views,
			selectedViewSlug: null,
			selectedViewId: null,
			baseFilters: [],
			initialPage: { items: [], total: 0, page: 1, pageSize: 50 },
			initialSearchTerm: "",
			initialQuickFilters: [],
			initialPageNumber: 1,
			initialQuery: "",
			viewQuery: "",
			apiError: null,
		} as any,
		{
			onRefresh: async () => {
				await invalidateAll();
			},
			onRefreshViewCounts: async () => {
				await invalidateAll();
			},
			navigateToUrl: async (
				url: string,
				navOptions?: import("$lib/state/interfaces/common").NavigateOptions
			) => {
				// Extract pathname and search from URL
				const urlObj = new URL(
					url,
					typeof window !== "undefined" ? window.location.origin : "http://localhost"
				);
				await goto(resolve(urlObj.pathname as any) + urlObj.search, {
					replaceState: navOptions?.replace ?? true, // Default to replace for backwards compat
					noScroll: true,
					keepFocus: true,
					state: navOptions?.state,
				});
			},
			requestBulkConfirmation: async (action: string, count: number) => {
				return new Promise((resolve) => {
					bulkConfirmResolve = resolve;
					bulkConfirmDialogOpen = true;
					bulkConfirmAction = action;
					bulkConfirmCount = count;
				});
			},
			getTags: () => currentTags,
		}
	);

	// Set context immediately
	setContext("notificationPageController", pageController);

	// Update the service worker message handler with the pageController reference
	pageControllerRef = pageController;

	// Expose view dialog actions via context for child pages
	setContext("viewDialogActions", {
		startEditingWithQuery: null, // Will be set after viewDialogController is created
		openNewDialogWithQuery: null,
	});

	// Expose layout functions via context for child pages
	setContext("layoutFunctions", {
		showMoreShortcuts: null, // Will be set below
		showQueryGuide: null,
		isAnyDialogOpen: null,
		isShortcutsModalOpen: null,
		toggleShortcutsModal: null,
		getCommandPalette: () => pageHeaderComponent?.getCommandPalette(),
	});

	// ============================================================================
	// EXTRACT STORES FROM PAGE CONTROLLER
	// ============================================================================

	const {
		views,
		selectedViewId,
		quickQuery,
		hydrated,
		sidebarCollapsed,
		customDateDialogOpen,
		bulkUpdating,
	} = pageController.stores;

	const { builtInViewList, selectedViewSlug, defaultViewSlug, defaultViewDisplayName } =
		pageController.derived;

	// ============================================================================
	// BULK ACTION CONFIRMATION DIALOG STATE
	// ============================================================================

	let bulkConfirmDialogOpen = false;
	let bulkConfirmAction = "";
	let bulkConfirmCount = 0;
	let bulkConfirmResolve: ((confirmed: boolean) => void) | null = null;

	// ============================================================================
	// VIEW DIALOG CONTROLLER
	// ============================================================================

	async function refreshViewCounts() {
		if (typeof window === "undefined") {
			return;
		}

		try {
			const updatedViews = await fetchViews();
			pageController.actions.setViews(updatedViews);
		} catch (error) {
			console.error("Failed to refresh view counts:", error);
		}
	}

	async function selectViewBySlug(slug: string, shouldInvalidate: boolean = false) {
		await pageController.actions.selectViewBySlug(slug, shouldInvalidate);
	}

	const {
		stores: {
			open: viewDialogOpen,
			saving: viewDialogSaving,
			editing: editingView,
			confirmDeleteOpen,
			linkedRulesConfirmOpen,
			linkedRuleCount,
			draft: viewDraft,
			error: viewDialogError,
		},
		actions: {
			openNewDialog,
			openNewDialogWithQuery,
			startEditing,
			startEditingWithQuery,
			closeDialog: closeViewDialog,
			handleSave: handleViewSave,
			clearError: clearViewError,
			requestDelete: requestViewDelete,
			cancelDelete: cancelViewDelete,
			confirmDelete: confirmDeleteView,
			confirmLinkedRulesDelete,
			cancelLinkedRulesDelete,
		},
	} = createViewDialogController({
		views,
		setViews: pageController.actions.setViews,
		selectedViewId,
		invalidateViews: refreshViewCounts,
		navigateToSlug: selectViewBySlug,
		refreshNotifications: pageController.actions.refresh,
		quickQuery,
	});

	// ============================================================================
	// UPDATE CONTEXT WITH VIEW DIALOG ACTIONS
	// ============================================================================

	const viewDialogActionsContext = getContext("viewDialogActions") as any;
	viewDialogActionsContext.startEditingWithQuery = startEditingWithQuery;
	viewDialogActionsContext.openNewDialogWithQuery = openNewDialogWithQuery;

	// ============================================================================
	// OTHER DIALOG STATE
	// ============================================================================

	let syntaxGuideOpen = false;

	// ============================================================================
	// UPDATE CONTEXT WITH LAYOUT FUNCTIONS
	// ============================================================================

	const layoutFunctionsContext = getContext("layoutFunctions") as any;
	layoutFunctionsContext.showMoreShortcuts = handleShowMoreShortcuts;
	layoutFunctionsContext.showQueryGuide = () => {
		syntaxGuideOpen = true;
	};
	layoutFunctionsContext.isAnyDialogOpen = isAnyDialogOpen;
	layoutFunctionsContext.isShortcutsModalOpen = isShortcutsModalOpen;
	layoutFunctionsContext.toggleShortcutsModal = toggleShortcutsModal;

	// ============================================================================
	// HANDLER FUNCTIONS
	// ============================================================================

	async function handleLogoClick() {
		// Always navigate to inbox without query params
		if (typeof window === "undefined") return;

		const url = new URL(window.location.href);
		url.pathname = `/views/inbox`;
		url.search = "";

		await goto(resolve(url.pathname as any) + url.search, {
			keepFocus: true,
			noScroll: true,
			replaceState: false,
			invalidateAll: false,
		});
	}

	function handleCustomSnoozeConfirm(until: string) {
		const notificationId = get(pageController.stores.customDateDialogNotificationId);
		if (!notificationId) return;

		const pageData = get(pageController.stores.pageData);
		const notification = pageData.items.find((n) => (n.githubId ?? n.id) === notificationId);
		if (notification) {
			void pageController.actions.snooze(notification, until);
		}
		pageController.actions.closeCustomSnoozeDialog();
	}

	function handleBulkConfirm() {
		if (bulkConfirmResolve) {
			bulkConfirmResolve(true);
			bulkConfirmResolve = null;
		}
		bulkConfirmDialogOpen = false;
	}

	function handleBulkCancel() {
		if (bulkConfirmResolve) {
			bulkConfirmResolve(false);
			bulkConfirmResolve = null;
		}
		bulkConfirmDialogOpen = false;
	}

	function handleShowMoreShortcuts() {
		pageSidebarComponent?.openShortcutsModal();
	}

	function handleShowQueryGuide() {
		syntaxGuideOpen = true;
	}

	function toggleShortcutsModal(): boolean {
		pageSidebarComponent?.toggleShortcutsModal();
		return true;
	}

	function isShortcutsModalOpen(): boolean {
		return pageSidebarComponent?.isShortcutsModalOpen() ?? false;
	}

	function isAnyDialogOpen(): boolean {
		const commandPalette = pageHeaderComponent?.getCommandPalette();
		return (
			$viewDialogOpen ||
			$confirmDeleteOpen ||
			$linkedRulesConfirmOpen ||
			bulkConfirmDialogOpen ||
			$customDateDialogOpen ||
			isShortcutsModalOpen() ||
			commandPalette?.isPaletteOpen() === true
		);
	}

	// ============================================================================
	// KEYBOARD SHORTCUTS - Will be scoped based on route
	// ============================================================================

	let unregisterShortcuts: (() => void) | null = null;

	// Initialize time store to keep timestamps refreshed
	// Subscribe to currentTime store to ensure it stays active
	$: _timeStoreValue = $currentTime;

	onMount(() => {
		// Hide splash screen once component is mounted
		if (browser) {
			document.body.classList.remove("loading");

			// Initialize favicon for badge rendering
			initializeFavicon();

			// Set up navigation event source for tray menu navigation
			navEventSource = new NavigationEventSource({
				onNavigate: (url: string) => {
					try {
						// Ensure URL starts with / for proper routing
						const normalizedUrl = url.startsWith("/") ? url : `/${url}`;
						void goto(normalizedUrl, {
							replaceState: false,
							noScroll: false,
							keepFocus: false,
						}).catch((err) => {
							console.error("Navigation failed:", err, "URL was:", normalizedUrl);
						});
					} catch (err) {
						console.error("Error during navigation:", err, "URL was:", url);
					}
				},
				onError: (error) => {
					console.error("Navigation event source error", error);
				},
			});
			navEventSource.connect();

			// Ensure service worker message handler is set up (in case it wasn't set up at top level)
			if (!swCleanup && "serviceWorker" in navigator) {
				swCleanup = setupServiceWorkerHandlers({
					getPageController: () => pageControllerRef,
				});
			}

			// Async initialization wrapped in IIFE
			(async () => {
				// Check GitHub connection and redirect if needed
				const isSetupPage = window.location.pathname === "/setup";

				// Fetch user info to check GitHub connection and sync settings
				try {
					await fetchUserInfo();
					const isConnected = hasGitHubConnected();
					const hasConfigured = hasSyncSettingsConfigured();

					if (!isConnected || !hasConfigured) {
						// Not connected or not configured, redirect to setup
						if (!isSetupPage) {
							void goto(resolve("/setup" as any));
							return;
						}
					} else if (isSetupPage && hasConfigured) {
						// Already configured, redirect to main app
						void goto(resolve("/views/inbox" as any));
						return;
					}
				} catch (err) {
					console.error("Failed to check auth status:", err);
					// If we can't check, redirect to setup to be safe
					if (!isSetupPage) {
						void goto(resolve("/setup" as any));
						return;
					}
				}

				// Only initialize app features if not on setup page
				if (!isSetupPage) {
					void refreshViewCounts();
					// Initialize update store and start polling
					if (updateStore) {
						await updateStore.initialize();
					}
				}
			})();
		}

		// Initialize theme store (only if not on setup page)
		if (browser && window.location.pathname !== "/setup") {
			getThemeStore();
		}

		// Register service worker for background sync (only if not on setup page)
		let controllerChangeHandler: (() => Promise<void>) | null = null;
		let swPollingCheckInterval: ReturnType<typeof setInterval> | null = null;
		let swHealthStore: ReturnType<typeof getSWHealthStore> | null = null;

		if (browser && window.location.pathname !== "/setup") {
			const notificationSettings = getNotificationSettingsStore();
			swHealthStore = getSWHealthStore();

			const sendSettingsToSW = async () => {
				const permission = getNotificationPermission();
				const enabled = isNotificationEnabled();
				await sendMessageToSW({
					type: "NOTIFICATION_PERMISSION",
					permission,
				});
				await sendNotificationSettingToSW(enabled);
			};

			void registerServiceWorker().then(async (registration) => {
				if (registration) {
					// Send current settings to SW and ensure polling starts
					const ensurePollingStarted = async () => {
						await sendSettingsToSW();
						// Ensure polling starts - this is especially important after setup/login
						await forceSWPoll();
					};

					if (navigator.serviceWorker.controller) {
						// Controller already available - start polling immediately
						await ensurePollingStarted();
					} else {
						// Wait for controller to be available
						controllerChangeHandler = async () => {
							await ensurePollingStarted();
							if (controllerChangeHandler) {
								navigator.serviceWorker.removeEventListener(
									"controllerchange",
									controllerChangeHandler
								);
								controllerChangeHandler = null;
							}
						};
						navigator.serviceWorker.addEventListener("controllerchange", controllerChangeHandler);

						// Also check if controller becomes available shortly (polling fallback)
						// Sometimes the controller is available immediately after registration
						setTimeout(async () => {
							if (navigator.serviceWorker.controller && controllerChangeHandler) {
								await ensurePollingStarted();
								navigator.serviceWorker.removeEventListener(
									"controllerchange",
									controllerChangeHandler
								);
								controllerChangeHandler = null;
							}
						}, 500);
					}

					// Send initial notification setting to SW
					const enabled = isNotificationEnabled();
					await sendNotificationSettingToSW(enabled);

					// Request permission if notifications are enabled and permission not granted
					if (enabled) {
						await requestNotificationPermission();
					}
				}
			});

			// Start SW health checks
			swHealthStore?.startHealthChecks();
		}

		// Periodic check to ensure SW polling is active when app is ready
		// This acts as a catchall for various routing scenarios (e.g., after setup redirect)
		// The SW will ignore the message if it's already polling
		const SW_POLLING_CHECK_INTERVAL = 30000; // Check every 30 seconds
		swPollingCheckInterval = setInterval(async () => {
			// Only check if not on setup page
			const isSetupPage = window.location.pathname === "/setup";

			if (!isSetupPage && hasGitHubConnected()) {
				// ensureSWPolling is safe to call repeatedly - SW ignores if already polling
				await ensureSWPolling();
			}
		}, SW_POLLING_CHECK_INTERVAL);

		// Notification permission is handled by service worker registration
		// Keyboard shortcuts are registered per-route (see views/[slug]/+page.svelte, settings/+page.svelte)

		return () => {
			if (unregisterShortcuts) {
				unregisterShortcuts();
			}
			// Clean up service worker event listener if it was added
			if (browser && controllerChangeHandler && "serviceWorker" in navigator) {
				navigator.serviceWorker.removeEventListener("controllerchange", controllerChangeHandler);
				controllerChangeHandler = null;
			}
			// Clean up SW polling check interval
			if (swPollingCheckInterval) {
				clearInterval(swPollingCheckInterval);
				swPollingCheckInterval = null;
			}
			// Clean up SW health checks
			if (swHealthStore) {
				swHealthStore.stopHealthChecks();
			}
		};
	});

	onDestroy(() => {
		// Clean up navigation event source
		if (navEventSource) {
			navEventSource.disconnect();
			navEventSource = null;
		}

		// Clean up service worker message listener
		if (swCleanup) {
			swCleanup();
		}

		// Only destroy page controller if it was initialized (not on login page)
		if (!isLoginRoute) {
			pageController.actions.destroy();
		}
		// Clean up timeStore interval
		if (currentTime && typeof currentTime.destroy === "function") {
			currentTime.destroy();
		}

		// Clean up update store
		if (updateStore) {
			updateStore.destroy();
		}
	});

	// Restore UI state from browser history when navigating back/forward
	afterNavigate(({ from, to, type }) => {
		// Re-read sync start time on every navigation
		// This handles the case where setup page sets localStorage then navigates
		readSyncStartTime();

		// When navigating from setup to the main app, ensure SW polling is active
		// This handles the case where SW wasn't started during setup flow
		const fromSetup = from?.url?.pathname === "/setup";
		const toProtectedRoute = to?.url?.pathname !== "/setup";

		if (fromSetup && toProtectedRoute && hasGitHubConnected()) {
			// Small delay to let the navigation settle, then ensure SW is polling
			setTimeout(() => {
				void ensureSWPolling();
			}, 100);
		}

		// Only restore state on browser back/forward (popstate) navigation
		if (type !== "popstate") return;

		const navState = $pageStore.state as NavigationState | undefined;
		if (!navState) return;

		// Restore sidebar state
		if (navState.sidebarCollapsed !== undefined) {
			sidebarCollapsed.set(navState.sidebarCollapsed);
		}

		// Restore split mode state
		if (navState.splitModeEnabled !== undefined) {
			pageController.stores.splitModeEnabled.set(navState.splitModeEnabled);
		}

		// Restore scroll position (for SingleMode list view)
		if (navState.savedScrollPosition !== undefined) {
			pageController.stores.savedListScrollPosition.set(navState.savedScrollPosition);
		}

		// Restore keyboard focus index
		if (navState.keyboardFocusIndex !== undefined) {
			pageController.stores.keyboardFocusIndex.set(navState.keyboardFocusIndex);
		}
	});

	// ============================================================================
	// REACTIVE: Update views when data changes
	// ============================================================================

	$: if (!isLoginRoute) {
		pageController.actions.setViews(data.views);
	}

	// ============================================================================
	// COMPUTED VALUES FOR SIDEBAR
	// ============================================================================

	$: systemViewsBySlug = new Map($views.filter((v) => v.systemView).map((v) => [v.slug, v]));

	$: inboxView = systemViewsBySlug.get("inbox");

	// ============================================================================
	// DOCUMENT TITLE
	// ============================================================================

	$: pageTitle = isSetupRoute
		? "Setup - Octobud"
		: (() => {
				const unreadCount = inboxView?.unreadCount ?? 0;
				return unreadCount > 0 ? `Octobud (${unreadCount})` : "Octobud";
			})();

	// ============================================================================
	// FAVICON BADGE
	// ============================================================================

	// Update favicon badge whenever unread count changes
	$: if (browser && !isLoginRoute && !isSetupRoute) {
		const unreadCount = inboxView?.unreadCount ?? 0;
		updateFaviconBadge(unreadCount);
	}

	// ============================================================================
	// ROUTE-SPECIFIC LOGIC
	// ============================================================================

	$: isSettingsRoute = $pageStore.url.pathname.startsWith("/settings");
	$: isViewsRoute =
		$pageStore.url.pathname.startsWith("/views/") || $pageStore.url.pathname === "/";
	const isLoginRoute = false; // No longer have a login route
	$: isSetupRoute = $pageStore.url.pathname === "/setup";

	// Close detail modal when navigating to settings
	$: if (!isLoginRoute && isSettingsRoute) {
		const detailOpen = get(pageController.stores.detailOpen);
		if (detailOpen) {
			pageController.actions.closeDetail();
		}
	}

	// ============================================================================
	// INITIAL SYNC BANNER - Reactive logic
	// ============================================================================
	// Show syncing banner if:
	// - We have a sync start timestamp
	// - AND no notifications in page data yet
	// - AND less than 2 minutes have passed
	// When notifications appear, clear the timestamp and hide banner.

	// Get current page data for banner logic
	const { pageData } = pageController.stores;

	// Reactive: compute banner visibility based on timestamp and page data
	$: hasNotifications = ($pageData?.items?.length ?? 0) > 0;
	$: timeSinceSync = syncStartTime ? Date.now() - syncStartTime : 0;
	$: syncTimedOut = syncStartTime !== null && timeSinceSync >= SYNC_TIMEOUT_MS;
	$: showSyncBanner =
		syncStartTime !== null &&
		!hasNotifications &&
		!syncTimedOut &&
		isViewsRoute &&
		!isLoginRoute &&
		!isSetupRoute;
	$: showSyncTimedOut = syncStartTime !== null && !hasNotifications && syncTimedOut;

	// When notifications appear, clear the sync timestamp
	$: if (hasNotifications && syncStartTime !== null) {
		clearSyncStartTime();
	}
</script>

<svelte:head>
	<title>{pageTitle}</title>
</svelte:head>

{#if $hydrated}
	{#if isLoginRoute || isSetupRoute}
		<!-- Login and setup pages use their own layout - just render slot -->
		<slot />
	{:else}
		<div class="h-screen flex flex-col bg-white dark:bg-gray-950">
			<!-- Full-width header at top - always use PageHeader -->
			<PageHeader
				bind:this={pageHeaderComponent}
				defaultViewSlug={$defaultViewSlug}
				defaultViewDisplayName={$defaultViewDisplayName}
				detailOpen={false}
				sidebarCollapsed={$sidebarCollapsed}
				onToggleSidebar={pageController.actions.toggleSidebar}
				builtInViews={$builtInViewList}
				views={$views}
				tags={data.tags}
				selectedViewSlug={$selectedViewSlug}
				onLogoClick={handleLogoClick}
				onSelectView={selectViewBySlug}
				onShowMoreShortcuts={handleShowMoreShortcuts}
				onShowQueryGuide={handleShowQueryGuide}
			/>

			<!-- Sync status banner -->
			<SyncStatusBanner
				show={showSyncBanner}
				timedOut={showSyncTimedOut}
				onDismiss={dismissSyncBanner}
			/>

			<!-- Service worker health banner -->
			<SWHealthBanner
				show={showSWHealthBanner}
				onDismiss={dismissSWHealthBanner}
				onRefresh={refreshPage}
			/>

			<!-- Restart needed banner (shows when new version is installed) -->
			{#if !isLoginRoute && !isSetupRoute}
				<RestartBanner />
			{/if}

			<!-- Update available banner -->
			{#if !isLoginRoute && !isSetupRoute}
				<UpdateBanner />
			{/if}

			<!-- Sidebar and content area below header -->
			<div class="flex-1 flex min-h-0 overflow-hidden">
				<!-- Sidebar -->
				<PageSidebar
					bind:this={pageSidebarComponent}
					builtInViewList={$builtInViewList}
					views={$views}
					tags={data.tags}
					selectedViewId={$selectedViewId}
					selectedViewSlug={$selectedViewSlug}
					{inboxView}
					collapsed={$sidebarCollapsed}
					onNewView={openNewDialog}
					onEditView={startEditing}
				/>

				<!-- Main content area -->
				<div class="flex-1 flex flex-col overflow-hidden">
					<!-- Child routes render in slot -->
					<slot />
				</div>
			</div>
		</div>
	{/if}
{:else if isLoginRoute || isSetupRoute}
	<!-- Login and setup pages - no skeleton needed -->
	<slot />
{:else}
	<LayoutSkeleton />
{/if}

{#if !isLoginRoute}
	<!-- Dialogs and modals - only show when not on login page -->
	<ViewDialog
		open={$viewDialogOpen}
		saving={$viewDialogSaving}
		initialValue={$viewDraft}
		error={$viewDialogError}
		onDelete={$editingView ? requestViewDelete : null}
		onClose={closeViewDialog}
		onSave={handleViewSave}
		onClearError={clearViewError}
	/>

	<ConfirmDialog
		open={$confirmDeleteOpen}
		title="Delete view"
		body="Are you sure you want to delete this view? This action cannot be undone."
		confirmLabel="Delete"
		cancelLabel="Cancel"
		confirmTone="danger"
		confirming={$viewDialogSaving}
		onCancel={cancelViewDelete}
		onConfirm={confirmDeleteView}
	/>

	<ConfirmDialog
		open={$linkedRulesConfirmOpen}
		title="Delete view with linked rules"
		body="This view has {$linkedRuleCount} linked rule{$linkedRuleCount === 1
			? ''
			: 's'}. Deleting the view will also delete {$linkedRuleCount === 1
			? 'this rule'
			: 'these rules'}. Continue?"
		confirmLabel="Delete view and rules"
		cancelLabel="Cancel"
		confirmTone="danger"
		confirming={$viewDialogSaving}
		onCancel={cancelLinkedRulesDelete}
		onConfirm={confirmLinkedRulesDelete}
	/>

	<CustomSnoozeDateDialog
		isOpen={$customDateDialogOpen}
		onConfirm={handleCustomSnoozeConfirm}
		onClose={pageController.actions.closeCustomSnoozeDialog}
	/>

	<BulkActionConfirmDialog
		open={bulkConfirmDialogOpen}
		action={bulkConfirmAction}
		count={bulkConfirmCount}
		onConfirm={handleBulkConfirm}
		onCancel={handleBulkCancel}
		confirming={$bulkUpdating}
	/>

	<SyntaxGuideDialog bind:open={syntaxGuideOpen} onClose={() => (syntaxGuideOpen = false)} />

	<ToastContainer />
{/if}
