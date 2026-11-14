// Starting point (test-playwright).

// THE DEMO ACCOUNT IS SHARED AND PERSISTENT. Every walk of every spec, and every earlier walk of the
// same preview, signs in as the one demo identity, so its lists already hold other records: other
// specs' leftovers, the seeded record, what the last walk made. An absolute count or "the list is
// empty" is true on a fresh account and false on the next walk. Read the value before acting, then
// assert the CHANGE from that baseline (and that the record the test made is present by its own
// `fixtureScope.uniqueName`), never the total.
//
// Take the baseline AFTER the first sync pull has landed: on an offline-first app wait for the
// app's own saved/synced indicator first, or the "baseline" is the empty local store and the count
// jumps when the pull arrives.
//
// The default is "at least": another spec or a late sync can add records to a shared account while
// this one runs, so an exact delta flakes. Ask for `{ exact: true }` only when nothing else can
// write to what you read.

/** Poll for the change; well inside a test's 30 s, so a miss fails here with a message. */
export const BASELINE_POLL_MS = 10_000;
// The gap between reads while polling, not a sleep to copy into a spec: a spec waits on a ready
// signal (an `expect` on what the person sees), never on a delay.
const POLL_INTERVAL_MS = 250;

export interface BaselineOptions {
  /** Require exactly `delta`; the default accepts `delta` or more. */
  exact?: boolean;
  timeoutMs?: number;
}

/**
 * Read a number, run an action, then poll until the number has changed by `delta` (at least, unless
 * `exact`) from what it was before the action. Throws with both numbers if it never does. Returns
 * the baseline.
 */
export async function expectChangeFromBaseline(
  read: () => Promise<number>,
  act: () => Promise<void>,
  delta: number,
  { exact = false, timeoutMs = BASELINE_POLL_MS }: BaselineOptions = {},
): Promise<number> {
  const before = await read();
  await act();
  const met = (change: number) => (exact ? change === delta : change >= delta);
  const deadline = Date.now() + timeoutMs;
  let change = (await read()) - before;
  while (!met(change) && Date.now() < deadline) {
    await new Promise((r) => setTimeout(r, POLL_INTERVAL_MS));
    change = (await read()) - before;
  }
  if (!met(change)) {
    throw new Error(`expected a change of ${exact ? "" : "at least "}${delta} from a baseline of ${before}, saw ${change}`);
  }
  return before;
}
