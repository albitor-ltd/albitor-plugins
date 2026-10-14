// Seeded by Albitor (kit-shadcn), then yours: the toast queue the Toaster renders.
//
// Module state is fine here because the app is a client-rendered SPA: one browser, one queue.
// If you ever prerender, move the queue into context.

export type ToastTone = 'info' | 'success' | 'error';

export interface Toast {
	id: string;
	message: string;
	tone: ToastTone;
}

/** Most toasts shown at once; the oldest is dropped, so a loop cannot flood the page. */
export const MAX_TOASTS = 5;
/** Longest message. A toast is a sentence, not a log. */
export const MAX_MESSAGE = 500;

class ToastQueue {
	items = $state<Toast[]>([]);
	#next = 0;

	/** Queue a message and return its id. The message is shown as text, never as markup. */
	push({ message, tone = 'info' }: { message: string; tone?: ToastTone }): string {
		if (typeof message !== 'string' || message.length === 0 || message.length > MAX_MESSAGE) {
			throw new Error(`a toast message is 1 to ${MAX_MESSAGE} characters`);
		}
		const id = `toast-${++this.#next}`;
		this.items = [...this.items, { id, message, tone }].slice(-MAX_TOASTS);
		return id;
	}

	dismiss(id: string): void {
		this.items = this.items.filter((t) => t.id !== id);
	}

	clear(): void {
		this.items = [];
	}
}

export const toast = new ToastQueue();
