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
 * V2 Stores - Composed store architecture
 */

export { createNotificationStore } from "./notificationStore";
export { createPaginationStore } from "./paginationStore";
export { createDetailViewStore } from "./detailViewStore";
export { createSelectionStore } from "./selectionStore";
export { createKeyboardNavigationStore } from "./keyboardNavigationStore";
export { createUIStateStore } from "./uiStateStore";
export { createQueryStore } from "./queryStore";
export { createViewStore } from "./viewStore";
export { createEventBus } from "./eventBus";

export type { NotificationStore } from "./notificationStore";
export type { PaginationStore } from "./paginationStore";
export type { DetailStore } from "./detailViewStore";
export type { SelectionStore } from "./selectionStore";
export type { KeyboardStore } from "./keyboardNavigationStore";
export type { UIStore } from "./uiStateStore";
export type { QueryStore } from "./queryStore";
export type { ViewStore } from "./viewStore";
export type { EventBus, NotificationEvent } from "./eventBus";
