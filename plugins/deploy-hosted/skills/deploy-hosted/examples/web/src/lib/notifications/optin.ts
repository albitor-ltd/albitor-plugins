// The first-sign-in opt-in: an app with consent email asks once, after the person first signs in,
// whether to send it. The hosted sign-in page belongs to the platform and cannot carry a box of
// the app's, so this is the app's own first screen after it. Consent starts off: the box is
// unticked, and closing the prompt records nothing.
import type { Notification } from './client';

/** Per browser, per person: the one thing kept locally is that the question was already put. */
const KEY = 'notifications.asked';

function storage(): Storage | null {
	try {
		return typeof localStorage === 'undefined' ? null : localStorage;
	} catch {
		return null;
	}
}

/** The consent emails to ask about: those the person has not already said yes to. */
export function consentToAsk(notifications: Notification[]): Notification[] {
	return notifications.filter((n) => n.consent && !n.subscribed);
}

/** Whether to show the prompt: there is something to ask, and it has not been asked here. */
export function shouldAsk(notifications: Notification[], who: string): boolean {
	if (consentToAsk(notifications).length === 0) return false;
	try {
		return storage()?.getItem(`${KEY}:${who}`) !== '1';
	} catch {
		return false; // storage is unreadable: do not nag, the Notifications page is always there
	}
}

/** Remember the question was put, whatever the answer. */
export function markAsked(who: string): void {
	try {
		storage()?.setItem(`${KEY}:${who}`, '1');
	} catch {
		/* private mode: the page still works, and the person may be asked again */
	}
}
