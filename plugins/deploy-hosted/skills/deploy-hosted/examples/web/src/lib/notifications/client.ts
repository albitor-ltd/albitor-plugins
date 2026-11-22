// The Notifications page's client: which optional and consent emails a person gets, and their
// choices. The API (api/internal/notifications/notifications.go) forwards both to the gateway, which keeps
// them; nothing is stored in the browser except OptInPrompt's "already asked" flag.
//
// Every call answers a result and never throws on a refusal, so the page shows what happened
// instead of an error page.
import { ApiError, NetworkError, get, put } from '../shell/api';

export interface Notification {
	template_id: string;
	/** The email in the person's words. */
	name: string;
	/** Why it is sent: one plain sentence, shown beside its switch. */
	reason: string;
	/** True when it is sent only if the person agrees. Consent mail starts off, optional mail on. */
	consent: boolean;
	subscribed: boolean;
}

export type Result<T = object> = ({ ok: true } & T) | { ok: false; message: string };

const UNAVAILABLE = { ok: false, message: 'Email settings are unavailable right now. Try again shortly.' } as const;
const OFFLINE = { ok: false, message: 'You are offline: check your connection and try again.' } as const;

async function guarded<T extends object>(run: () => Promise<T>): Promise<Result<T>> {
	try {
		return { ok: true, ...(await run()) };
	} catch (err) {
		if (err instanceof NetworkError) return err.offline ? OFFLINE : UNAVAILABLE;
		if (err instanceof ApiError) {
			const b = (err.body ?? {}) as { error?: unknown };
			if (err.status < 500 && typeof b.error === 'string') return { ok: false, message: b.error };
			return UNAVAILABLE;
		}
		throw err;
	}
}

/** The optional and consent emails this app sends, with the person's choice on each. */
export function loadNotifications(): Promise<Result<{ notifications: Notification[] }>> {
	return guarded(async () => {
		const body = await get<{ notifications?: Notification[] }>('/notifications');
		return { notifications: body.notifications ?? [] };
	});
}

/**
 * Save choices by template id. Only what the person changed is sent, and only from something they
 * did (a switch or a ticked box): the platform records an opt-in as the person's own act.
 */
export function saveNotifications(choices: Record<string, boolean>): Promise<Result> {
	if (Object.keys(choices).length === 0) return Promise.resolve({ ok: true });
	return guarded(async () => {
		await put('/notifications', { choices });
		return {};
	});
}

/** The choices that differ from what was loaded: the whole of what a save sends. */
export function changedChoices(loaded: Notification[], now: Record<string, boolean>): Record<string, boolean> {
	const out: Record<string, boolean> = {};
	for (const n of loaded) {
		if (n.template_id in now && now[n.template_id] !== n.subscribed) out[n.template_id] = now[n.template_id]!;
	}
	return out;
}

/** Whether the app has any email that is sent only with a person's agreement. */
export function hasConsentEmail(notifications: Notification[]): boolean {
	return notifications.some((n) => n.consent);
}
