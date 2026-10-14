// Managed by Albitor (test-playwright). Run with `npm run unit` from web/tests.
import assert from "node:assert/strict";
import { describe, it } from "node:test";
import { WCAG_TAGS, checkWaivers, describeViolations, unwaived } from "../helpers/a11y-report.ts";

const violation = (id: string, impact = "serious", targets = ["#a"]) => ({
	id,
	impact,
	help: `${id} help`,
	helpUrl: `https://dequeuniversity.com/rules/axe/4/${id}`,
	nodes: targets.map((t) => ({ target: [t], html: `<div id="${t.slice(1)}">` })),
});

describe("WCAG_TAGS", () => {
	it("covers every Level A and AA criterion through 2.2 and nothing weaker or stronger", () => {
		assert.deepEqual([...WCAG_TAGS], ["wcag2a", "wcag2aa", "wcag21a", "wcag21aa", "wcag22aa"]);
	});
});

describe("checkWaivers", () => {
	it("accepts a rule with a reason", () => {
		assert.deepEqual(checkWaivers([{ rule: "color-contrast", reason: "third-party map tiles" }]), []);
	});
	it("refuses a waiver with no reason, a blank reason, or a non-rule id", () => {
		assert.match(checkWaivers([{ rule: "color-contrast", reason: "" }]).join(), /needs a reason/);
		assert.match(checkWaivers([{ rule: "color-contrast", reason: "   " }]).join(), /needs a reason/);
		assert.match(checkWaivers([{ rule: "Color Contrast!", reason: "x" }]).join(), /not an axe rule id/);
	});
	it("refuses a duplicate waiver", () => {
		const w = { rule: "region", reason: "r" };
		assert.match(checkWaivers([w, w]).join(), /twice/);
	});
});

describe("unwaived", () => {
	it("drops only the waived rules", () => {
		const left = unwaived([violation("region"), violation("label")], [{ rule: "region", reason: "r" }]);
		assert.deepEqual(left.map((v) => v.id), ["label"]);
	});
});

describe("describeViolations", () => {
	it("names the rule, impact, help url and the offending elements", () => {
		const text = describeViolations([violation("label", "critical", ["#email", "#name"])]);
		assert.match(text, /label \(critical\)/);
		assert.match(text, /dequeuniversity\.com\/rules\/axe\/4\/label/);
		assert.match(text, /#email/);
		assert.match(text, /#name/);
	});
	it("caps how many elements it prints, so a page of failures stays readable", () => {
		const many = violation("label", "minor", Array.from({ length: 40 }, (_, i) => `#n${i}`));
		const text = describeViolations([many]);
		assert.match(text, /and 30 more/);
		assert.doesNotMatch(text, /#n39/);
	});
});
