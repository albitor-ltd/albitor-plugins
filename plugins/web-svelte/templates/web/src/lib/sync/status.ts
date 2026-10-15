// Managed by Albitor (web-svelte, offline data layer): what to tell the person about their data, in plain words.
import type { SyncStatus } from './types';

export type StatusTone = 'ok' | 'busy' | 'warn';

export function statusText(status: SyncStatus, online: boolean): { text: string; tone: StatusTone } {
	const n = (count: number, noun: string) => `${count} ${noun}${count === 1 ? '' : 's'}`;
	if (status.owner === null) return { text: '', tone: 'ok' };
	if (status.needsSignIn) {
		return { text: 'Sign in again to save your changes. They are kept on this device.', tone: 'warn' };
	}
	if (status.failed > 0) {
		return { text: `${n(status.failed, 'change')} could not be saved. Review, retry or discard.`, tone: 'warn' };
	}
	if (status.pending > 0) {
		return online
			? { text: status.syncing ? 'Saving…' : `${n(status.pending, 'change')} waiting to save.`, tone: 'busy' }
			: { text: `You are offline. ${n(status.pending, 'change')} will save when you reconnect.`, tone: 'busy' };
	}
	return online
		? { text: 'All changes saved.', tone: 'ok' }
		: { text: 'You are offline. Your data is on this device.', tone: 'ok' };
}
