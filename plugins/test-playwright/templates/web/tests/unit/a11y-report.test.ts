// Managed by Albitor (test-playwright). Run with `npm run unit` from web/tests.
import assert from "node:assert/strict";
import { describe, it } from "node:test";
import { WCAG_TAGS, checkSelectors, checkWaivers, describeViolations, unwaived } from "../helpers/a11y-report.ts";

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

describe("checkSelectors", () => {
	it("accepts a selector for one element", () => {
		assert.deepEqual(checkSelectors(["#map-embed", ".third-party iframe"], "exclude"), []);
	});
	it("refuses a selector that is the whole page or its landmark, which would switch the audit off", () => {
		for (const s of ["body", "html", "main", ":root", "*", "BODY", " main ", "body *", "[role=main]", '[role="main"]']) {
			assert.match(checkSelectors([s], "exclude").join(), /whole page/, s);
		}
	});
	it("refuses lists and descendants made only of page roots", () => {
		for (const s of ["html, body", "body,main", "body > *", "html > body", "body main", ":root *", "[role=main] *", "body, .ok"]) {
			assert.match(checkSelectors([s], "exclude").join(), /whole page/, s);
		}
	});
	it("refuses a page root wrapped in :not() or :is()", () => {
		for (const s of ["body:not(.x)", ":root:not([hidden])", "*:not(.x)", ":is(body)", ":where(html, body)", ":is(:is(main))", "body:not(.x):not(.y) *"]) {
			assert.match(checkSelectors([s], "exclude").join(), /whole page/, s);
		}
	});
	it("allows a root as the start of a selector that names one element", () => {
		assert.deepEqual(checkSelectors(["body .banner", "main > #map", "html:lang(en) .x", "body:not(.x) .banner", ":is(.a, .b) iframe"], "exclude"), []);
	});
	it("refuses an empty selector", () => {
		assert.match(checkSelectors([""], "include").join(), /empty/);
	});
});
