// Starting point (test-playwright). Run with `npm run unit` from web/tests.
import assert from "node:assert/strict";
import { describe, it } from "node:test";
import { expectChangeFromBaseline } from "../helpers/baseline.ts";

describe("expectChangeFromBaseline", () => {
  it("passes on a change of delta whatever the starting total", async () => {
    for (const start of [0, 7]) {
      let n = start;
      const before = await expectChangeFromBaseline(
        async () => n,
        async () => {
          n += 2;
        },
        2,
      );
      assert.equal(before, start);
    }
  });
  it("sees a change that lands after the action returns", async () => {
    let n = 3;
    const act = async () => {
      setTimeout(() => {
        n += 1;
      }, 100);
    };
    assert.equal(await expectChangeFromBaseline(async () => n, act, 1), 3);
  });
  it("fails, naming both numbers, when the change never comes", async () => {
    await assert.rejects(
      expectChangeFromBaseline(async () => 5, async () => {}, 1, 300),
      /change of 1 from a baseline of 5, saw 0/,
    );
  });
});
