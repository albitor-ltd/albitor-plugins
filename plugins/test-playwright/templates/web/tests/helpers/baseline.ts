// Starting point (test-playwright).

// THE DEMO ACCOUNT IS SHARED AND PERSISTENT. Every walk of every spec, and every earlier walk of the
// same preview, signs in as the one demo identity, so its lists already hold other records: other
// specs' leftovers, the seeded record, what the last walk made. An absolute count or "the list is
// empty" is true on a fresh account and false on the next walk. Read the value before acting, then
// assert the CHANGE from that baseline (and that the record the test made is present by its own
// `fixtureScope.uniqueName`), never the total.
//
// Take the baseline after the first sync pull has landed: on an offline-first app wait for the
// sync status to settle first, or the "baseline" is the empty local store and the count jumps when
// the pull arrives.

/** Poll for the change; well inside a test's 30 s, so a miss fails here with a message. */
export const BASELINE_POLL_MS = 15_000;
const POLL_INTERVAL_MS = 250;

/**
 * Read a number, run an action, then poll (no fixed sleep) until the number has changed by `delta`
 * from what it was before the action. Throws with both numbers if it never does. Returns the baseline.
 */
export async function expectChangeFromBaseline(
  read: () => Promise<number>,
  act: () => Promise<void>,
  delta: number,
  timeoutMs = BASELINE_POLL_MS,
): Promise<number> {
  const before = await read();
  await act();
  const deadline = Date.now() + timeoutMs;
  let change = (await read()) - before;
  while (change !== delta && Date.now() < deadline) {
    await new Promise((r) => setTimeout(r, POLL_INTERVAL_MS));
    change = (await read()) - before;
  }
  if (change !== delta) {
    throw new Error(`expected a change of ${delta} from a baseline of ${before}, saw ${change}`);
  }
  return before;
}
