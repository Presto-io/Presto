export type NotificationType = 'success' | 'info' | 'warning' | 'error';

export interface NotificationAction {
	label: string;
	run: () => void | Promise<void>;
}

export interface NotificationItem {
	id: number;
	message: string;
	type: NotificationType;
	durationMs: number;
	source?: string;
	groupKey?: string;
	detail?: string;
	action?: NotificationAction;
	visibleSince?: number;
}

export interface NotificationInput {
	message: string;
	type?: NotificationType;
	durationMs?: number;
	source?: string;
	groupKey?: string;
	detail?: string;
	action?: NotificationAction;
}

const DEFAULT_DURATION_MS: Record<NotificationType, number> = {
	success: 2800,
	info: 4000,
	warning: 5200,
	error: 4500,
};

export const MAX_VISIBLE = 3;
const EXIT_TRANSITION_MS = 260;

let counter = 0;
let visibleNotifications = $state<NotificationItem[]>([]);
let queuedNotifications = $state<NotificationItem[]>([]);
let pendingNotifications = $state<NotificationItem[]>([]);

let activeDismissTimer: ReturnType<typeof setTimeout> | null = null;
let releaseTimer: ReturnType<typeof setTimeout> | null = null;
let dismissingId: number | null = null;
let dismissalQueue: number[] = [];

function hasFocus(): boolean {
	return typeof document === 'undefined' || document.hasFocus();
}

function getDurationMs(type: NotificationType, durationMs?: number): number {
	if (typeof durationMs === 'number') {
		return Math.max(0, durationMs);
	}
	return DEFAULT_DURATION_MS[type];
}

function normalizeNotification(
	input: string | NotificationInput,
	type: NotificationType = 'info',
): NotificationItem {
	if (typeof input === 'string') {
		return {
			id: ++counter,
			message: input,
			type,
			durationMs: getDurationMs(type),
		};
	}

	const notificationType = input.type ?? type;
	return {
		id: ++counter,
		message: input.message,
		type: notificationType,
		durationMs: getDurationMs(notificationType, input.durationMs),
		source: input.source,
		groupKey: input.groupKey,
		detail: input.detail,
		action: input.action,
	};
}

function removeById(items: NotificationItem[], id: number): NotificationItem[] {
	return items.filter((item) => item.id !== id);
}

function clearDismissTimer() {
	if (activeDismissTimer) {
		clearTimeout(activeDismissTimer);
		activeDismissTimer = null;
	}
}

function clearReleaseTimer() {
	if (releaseTimer) {
		clearTimeout(releaseTimer);
		releaseTimer = null;
	}
}

function promoteNextQueued() {
	if (visibleNotifications.length >= MAX_VISIBLE || queuedNotifications.length === 0) {
		return;
	}

	const [next, ...rest] = queuedNotifications;
	queuedNotifications = rest;
	next.visibleSince = Date.now();
	visibleNotifications = [next, ...visibleNotifications];
}

function flushQueuedDismissals() {
	while (dismissalQueue.length > 0) {
		const nextId = dismissalQueue.shift();
		if (!nextId) continue;
		if (visibleNotifications.some((item) => item.id === nextId)) {
			beginDismiss(nextId);
			return;
		}
	}
	scheduleDismissal();
}

function completeDismissal() {
	dismissingId = null;
	promoteNextQueued();
	flushQueuedDismissals();
}

function beginDismiss(id: number) {
	const target = visibleNotifications.find((item) => item.id === id);
	if (!target) {
		dismissalQueue = dismissalQueue.filter((queuedId) => queuedId !== id);
		scheduleDismissal();
		return;
	}

	if (dismissingId !== null && dismissingId !== id) {
		if (!dismissalQueue.includes(id)) {
			dismissalQueue = [...dismissalQueue, id];
		}
		return;
	}

	clearDismissTimer();
	clearReleaseTimer();
	dismissingId = id;
	visibleNotifications = removeById(visibleNotifications, id);
	releaseTimer = setTimeout(() => {
		clearReleaseTimer();
		completeDismissal();
	}, EXIT_TRANSITION_MS);
}

function scheduleDismissal() {
	clearDismissTimer();
	if (dismissingId !== null) return;

	const nextAutoDismiss = [...visibleNotifications]
		.reverse()
		.find((item) => item.durationMs > 0);

	if (!nextAutoDismiss) return;

	const elapsed = nextAutoDismiss.visibleSince ? Date.now() - nextAutoDismiss.visibleSince : 0;
	const remaining = Math.max(0, nextAutoDismiss.durationMs - elapsed);
	activeDismissTimer = setTimeout(() => {
		beginDismiss(nextAutoDismiss.id);
	}, remaining);
}

function enqueueVisible(notification: NotificationItem) {
	notification.visibleSince = Date.now();
	visibleNotifications = [notification, ...visibleNotifications];

	if (visibleNotifications.length > MAX_VISIBLE) {
		const overflow = visibleNotifications.at(-1);
		if (overflow) {
			visibleNotifications = visibleNotifications.slice(0, MAX_VISIBLE);
			overflow.visibleSince = undefined;
			queuedNotifications = [overflow, ...queuedNotifications];
		}
	}

	scheduleDismissal();
}

function showInternal(input: string | NotificationInput, type: NotificationType = 'info') {
	const notification = normalizeNotification(input, type);
	if (!hasFocus()) {
		pendingNotifications = [...pendingNotifications, notification];
		return notification.id;
	}
	enqueueVisible(notification);
	return notification.id;
}

export const notificationStore = {
	get items() {
		return visibleNotifications;
	},

	get visibleItems() {
		return visibleNotifications;
	},

	get queuedItems() {
		return queuedNotifications;
	},

	get pendingItems() {
		return pendingNotifications;
	},

	show(input: string | NotificationInput, type: NotificationType = 'info') {
		return showInternal(input, type);
	},

	dismiss(id: number) {
		if (pendingNotifications.some((item) => item.id === id)) {
			pendingNotifications = removeById(pendingNotifications, id);
			return;
		}

		if (queuedNotifications.some((item) => item.id === id)) {
			queuedNotifications = removeById(queuedNotifications, id);
			return;
		}

		beginDismiss(id);
	},

	dismissGroup(groupKey: string) {
		pendingNotifications = pendingNotifications.filter((item) => item.groupKey !== groupKey);
		queuedNotifications = queuedNotifications.filter((item) => item.groupKey !== groupKey);

		const ids = visibleNotifications
			.filter((item) => item.groupKey === groupKey)
			.map((item) => item.id);

		for (const id of ids) {
			this.dismiss(id);
		}
	},

	async runAction(id: number) {
		const notification =
			visibleNotifications.find((item) => item.id === id) ??
			queuedNotifications.find((item) => item.id === id) ??
			pendingNotifications.find((item) => item.id === id);

		if (!notification?.action) return;
		await notification.action.run();
	},

	flushPending() {
		if (pendingNotifications.length === 0) {
			return;
		}

		const pending = [...pendingNotifications];
		pendingNotifications = [];

		for (const notification of pending) {
			enqueueVisible(notification);
		}
	},
};
