// Managed by Albitor (test-playwright).
import { appendFileSync, mkdirSync, readFileSync } from "node:fs";
import { join } from "node:path";
import { test as base, type BrowserContext, type Page } from "@playwright/test";
import { expectNoA11yViolations, type A11yOptions } from "./a11y";
import { authHeader, localIssuerRunning, signInAs } from "./local-auth";
import { onApp, titleProblem } from "./screen-rules";

// One step of a journey: run it, photograph the screen it left behind, and record
// that the step completed.
//
// WHY A HELPER AND NOT FIVE LINES IN EVERY SPEC. The photographs are shown to the
// app's owner in step order, and the ledger is what says which step a walk
// reached. Both have to be right in every spec, in every journey, for the review
// to mean anything — so they are written once here and called everywhere, rather
// than copied and slowly diverging.
//
// THE LEDGER IS APPEND-ONLY AND RECORDS ONLY WHAT COMPLETED, and that is the
// whole trick. One line is appended as each step finishes. A step that throws
// appends nothing, so the walk's reader sees exactly the steps that worked and
// infers the stopping point from the first one missing — without needing a report
// from a process that has already crashed. A spec that assembled the file in
// memory and wrote it at the end would be correct on every run that finished and
// useless on the one that did not, which is the run worth reading: "the walk
// stopped at step 3" is a defect report, where a missing photograph is a gap.
// Nothing here rewrites, truncates or reorders the file.
//
// OUTSIDE A WALK THIS IS A test.step THAT ALSO JUDGES THE SCREEN. An ordinary
// local run or CI run sets none of the variables below, so no screenshot is
// taken and no ledger is written; the screen checks further down run everywhere,
// so a broken title or a blocked script fails locally before it deploys. That is
// deliberate: these specs are this repository's, and they must run for whoever
// checks it out with nothing else in place.

// Set by the automated walk, on the spec's own process. Absent everywhere else.
const shotDir = process.env.ALBITOR_JOURNEY_SHOT_DIR || "";
const ledgerPath = process.env.ALBITOR_JOURNEY_STEPS || "";

// ONE SEQUENCE PER WALK, NOT PER TEST. A walk is the whole spec file: every test
// in it appends to the same ledger, and the photographs are shown in ledger
// order. So a step's number is its place in the ledger — the count of lines
// already there, plus one — and it runs on across tests: a spec with two tests
// of two steps each records 1, 2, 3, 4, never 1, 2, 1, 2. Numbering per test
// made the second test's first step a second "step 1", which the walk's reader
// dropped as a duplicate, and could give its photograph the same file name as
// the first test's.
//
// The number is read off the FILE, not a counter in this process, because
// Playwright starts a fresh worker after a failing test and a counter would
// restart with it. And it is taken when the step COMPLETES, not when it starts:
// a step nested inside another finishes first and is appended first, so it must
// also be numbered first. The walk runs one worker, so nothing else appends
// between the read and the write.
//
// Outside a walk there is no ledger to count, and a per-process counter only
// keeps file names apart.
let localSteps = 0;

function nextStep(): number {
  if (!ledgerPath) {
    localSteps += 1;
    return localSteps;
  }
  let recorded = 0;
  try {
    recorded = readFileSync(ledgerPath, "utf8")
      .split("\n")
      .filter((line) => line.trim() !== "").length;
  } catch {
    // Not written yet: this is the first step.
  }
  return recorded + 1;
}

// A file name that sorts in step order and survives any step name a person writes.
// It carries no directory: the walk reads the file from its own screenshot
// directory and refuses a name with a path separator in it.
function shotName(step: number, name: string): string {
  const slug =
    name
      .toLowerCase()
      .replace(/[^a-z0-9]+/g, "-")
      .replace(/^-+|-+$/g, "")
      .slice(0, 60) || "step";
  return `step-${String(step).padStart(2, "0")}-${slug}.png`;
}

// The capture is the WHOLE PAGE, top to bottom, not the fixed viewport: a viewport
// photograph cut off whatever sat below the fold, and the owner reviewing the
// step could not see it. The gallery shows the top of a tall image and lets the
// opened view scroll the rest.
//
// THE HEIGHT IS CAPPED AT MAX_CAPTURE_HEIGHT CSS PIXELS. The runner refuses any
// one image over 8 MB, and a very long list photographed in full would be
// dropped altogether — worse than a cut-off one. 6000px of a typical page is
// well under that. `scale: "css"` keeps one image pixel per CSS pixel whatever
// the project's deviceScaleFactor, so a high-density phone project cannot
// multiply the file size past the cap. A page taller than the cap is clipped and
// the ledger line says so (`clipped: true`), so a reader knows the picture is not
// the whole page. Width stays the viewport's: horizontal overflow is a layout
// fault the checks above report, not part of the screen.
const MAX_CAPTURE_HEIGHT = 6000;

type Capture = { file: string; width: number; height: number; clipped: boolean };

async function capture(page: Page, step: number, name: string): Promise<Capture | null> {
  if (!shotDir) {
    return null;
  }
  const file = shotName(step, name);
  mkdirSync(shotDir, { recursive: true });
  const width = page.viewportSize()?.width ?? 0;
  let pageHeight = 0;
  try {
    pageHeight = await page.evaluate(() =>
      Math.max(document.documentElement.scrollHeight, document.body?.scrollHeight ?? 0),
    );
  } catch {
    // No document to measure (mid-navigation): fall back to the viewport.
  }
  const viewportHeight = page.viewportSize()?.height ?? 0;
  const full = Math.max(Math.ceil(pageHeight), viewportHeight);
  const height = Math.min(full, MAX_CAPTURE_HEIGHT);
  const clipped = full > MAX_CAPTURE_HEIGHT;
  if (width > 0 && height > 0) {
    await page.screenshot({
      path: join(shotDir, file),
      fullPage: true,
      scale: "css",
      clip: { x: 0, y: 0, width, height },
    });
    return { file, width, height, clipped };
  }
  await page.screenshot({ path: join(shotDir, file), fullPage: true, scale: "css" });
  return { file, width, height: viewportHeight, clipped: false };
}

// One ledger line per completed step. `route` is the path the step ended on and
// must start with "/" — that is what labels the tile — and `file` is the bare
// image name in the screenshot directory.
function record(page: Page, step: number, name: string, shot: Capture | null): void {
  if (!ledgerPath || !shot) {
    return;
  }
  let route = "/";
  try {
    route = new URL(page.url()).pathname || "/";
  } catch {
    // A page that never navigated has no URL to read; the landing route is the
    // honest fallback and keeps the line renderable.
  }
  appendFileSync(
    ledgerPath,
    `${JSON.stringify({
      step,
      route,
      title: name,
      file: shot.file,
      // The image's own pixel size, which is the page's height up to the cap.
      width: shot.width,
      height: shot.height,
      // Additive: present only when the page was taller than the cap.
      ...(shot.clipped ? { clipped: true } : {}),
    })}\n`,
  );
}

// --- What every screen must be, whatever the journey ---------------------------
//
// A journey asserts what the person must see. Three things are true of EVERY
// screen of this app, so a spec does not have to remember them: the helper
// checks them after each step and fails the step — before its photograph is
// taken or its ledger line written — when one does not hold. Because the walk
// drives the deployed app through its edge, these are also this app's
// post-deploy checks, made by a real browser rather than a curl script that
// could only approximate one.
//
// THE PAGE HAS A TITLE THAT IS A NAME. The browser tab, a bookmark and a screen
// reader's page-change announcement (WCAG 2.4.2) all read document.title, and a
// live app was once seen shipping a whole sentence of leaked model prose there.
// So it must be non-empty, at most 60 characters, and not read as more than one
// sentence. page.title() is what the browser parsed, so an HTML comment that
// happens to contain "<title>" cannot be mistaken for the title — a regex over
// the raw shell once was, and failed a healthy deploy over it.
//
// THE CONTENT-SECURITY-POLICY BLOCKED NOTHING. A framework shell whose inline
// bootstrap the policy refuses still answers 200 text/html with a <title>, and
// renders nothing. The browser reports every enforced refusal as a
// `securitypolicyviolation` event, so no code here re-implements how a browser
// matches a script against a policy.
//
// NOTHING THREW UNCAUGHT. An exception nothing caught is the other way a page
// renders blank or half-built while answering 200.
//
// Console errors are deliberately NOT a failure: a sign-in journey legitimately
// logs a 401, and a failed request the app handles is the app working. Only
// pages on the app's own origin are judged — a step that ends on an identity
// provider's hosted page is that provider's business.

// The page-side name the violation reporter calls. Exposed once per page.
const CSP_BINDING = "__reportCspViolation";

type CspViolation = {
  documentURI: string;
  blockedURI: string;
  effectiveDirective: string;
  sample: string;
};

// What each page has done wrong since its last step was judged.
const pageProblems = new WeakMap<Page, string[]>();

// The origin under test: the walk's deployed app, else the suite's own
// baseURL. Empty when neither is known, and then every http(s) page is judged.
function appOrigin(): string {
  const target =
    process.env.ALBITOR_JOURNEY_BASE_URL ||
    (base.info().project?.use?.baseURL as string | undefined) ||
    "";
  try {
    return target ? new URL(target).origin : "";
  } catch {
    return "";
  }
}

function pathOf(url: string): string {
  try {
    return new URL(url).pathname || "/";
  } catch {
    return url;
  }
}

// The browser side of the watch: report every ENFORCED CSP refusal to the
// binding. It runs as an init script, so it is in place for every document
// loaded after it is installed and for none loaded before — which is why the
// watch is installed on the browser CONTEXT before the test body runs: every
// page the test opens there, a second page or a popup included, is then
// watched from its very first navigation.
function cspReporter(binding: string): void {
  document.addEventListener(
    "securitypolicyviolation",
    (e: SecurityPolicyViolationEvent) => {
      // A report-only policy blocks nothing; only an enforced refusal breaks a page.
      if (e.disposition !== "enforce") {
        return;
      }
      const report = (window as unknown as Record<string, unknown>)[binding];
      if (typeof report === "function") {
        report({
          documentURI: e.documentURI,
          blockedURI: e.blockedURI,
          effectiveDirective: e.effectiveDirective,
          sample: e.sample,
        });
      }
    },
    true,
  );
}

function reportViolation(problems: string[], origin: string, v: CspViolation): void {
  if (!onApp(v.documentURI, origin)) {
    return;
  }
  const what = v.blockedURI && v.blockedURI !== "inline" ? v.blockedURI : "an inline script or style";
  const sample = v.sample ? ` (${v.sample})` : "";
  problems.push(
    `the Content-Security-Policy blocked ${what} under ${v.effectiveDirective} on ${pathOf(v.documentURI)}${sample}`,
  );
}

// problemsOf returns a page's problem list, creating it — and starting to hear
// the page's uncaught errors — the first time the page is seen. `all`, when
// given, collects every list so the fixture can flush them at teardown.
function problemsOf(page: Page, origin: string, all?: Set<string[]>): string[] {
  let problems = pageProblems.get(page);
  if (!problems) {
    const list: string[] = [];
    problems = list;
    pageProblems.set(page, list);
    page.on("pageerror", (err) => {
      if (onApp(page.url(), origin)) {
        list.push(`an uncaught error on ${pathOf(page.url())}: ${err.message}`);
      }
    });
  }
  all?.add(problems);
  return problems;
}

// watchContext watches every page of a browser context: those already open,
// and every one opened later — context.newPage(), a popup — before its first
// navigation.
async function watchContext(context: BrowserContext, origin: string): Promise<Set<string[]>> {
  const all = new Set<string[]>();
  context.on("page", (page) => {
    problemsOf(page, origin, all);
  });
  for (const page of context.pages()) {
    problemsOf(page, origin, all);
  }
  await context.exposeBinding(CSP_BINDING, (source, v: CspViolation) => {
    if (source.page) {
      reportViolation(problemsOf(source.page, origin, all), origin, v);
    }
  });
  await context.addInitScript(cspReporter, CSP_BINDING);
  return all;
}

// watchPage is the fallback for a page outside a watched context — a spec that
// takes `test` from @playwright/test. It can only watch documents loaded from
// now on, so journeyStep calls it only on a page that has not navigated yet.
async function watchPage(page: Page, origin: string): Promise<string[]> {
  const problems = problemsOf(page, origin);
  await page.exposeBinding(CSP_BINDING, (_source, v: CspViolation) => {
    reportViolation(problems, origin, v);
  });
  await page.addInitScript(cspReporter, CSP_BINDING);
  return problems;
}

export { titleProblem };

function brokenScreen(found: string[]): Error {
  return new Error(`this screen is broken: ${found.join("; ")}`);
}

// judgeScreen fails the step on anything the page did wrong since the last one.
async function judgeScreen(page: Page, problems: string[], origin: string): Promise<void> {
  const found = problems.splice(0, problems.length);
  if (onApp(page.url(), origin)) {
    let title: string | undefined;
    try {
      title = await page.title();
    } catch {
      // The page is mid-navigation and has no document to ask. The next step
      // judges whatever it lands on.
    }
    const problem = title === undefined ? "" : titleProblem(title);
    if (problem) {
      found.push(problem);
    }
  }
  if (found.length > 0) {
    throw brokenScreen(found);
  }
}


// `test`, with a browser context whose every page is watched from before its
// first navigation: the `page` fixture, a second page from context.newPage(),
// and a popup alike. A journey spec imports `test` and `expect` from this file
// (or from ./seed, which extends it), not from @playwright/test.
//
// What a page does AFTER the last step was judged — a script blocked as the
// final screen settles — is reported when the test ends, so it fails the test
// rather than vanishing with the context. One round-trip to each open page
// first lets a report already on its way arrive.
type JourneyFixtures = {
  appAuthHeaders: Record<string, string>;
};

export const test = base.extend<JourneyFixtures>({
  // Ordinary journeys run as the fixed local alice identity when the local
  // issuer exists. Deployed runs keep the demo session loaded by Playwright.
  // Sign-in and signed-out specs import @playwright/test directly instead.
  appAuthHeaders: [async ({ context, request }, use) => {
    if (!(await localIssuerRunning(request))) {
      await use({});
      return;
    }
    const tokens = await signInAs(context, request, "alice", appOrigin());
    await use(authHeader(tokens));
    // A request the auth route is still answering when the test ends must not fail it.
    await context.unrouteAll({ behavior: "ignoreErrors" });
  }, { auto: true }],
  context: async ({ context }, use) => {
    const origin = appOrigin();
    const all = await watchContext(context, origin);
    await use(context);
    for (const page of context.pages()) {
      try {
        await page.evaluate(() => 0);
      } catch {
        // Closed or mid-navigation: nothing more will arrive from it.
      }
    }
    const found = [...all].flatMap((problems) => problems.splice(0, problems.length));
    if (found.length > 0) {
      throw brokenScreen(found);
    }
  },
});

export { expect } from "@playwright/test";

// journeyStep wraps one screen of a journey. Name it for what the person is
// doing — "open the claims queue", not "assert list renders" — because the name
// is the caption the owner reads beside the photograph.
//
// Pass `{ a11y: true }` (or axe options) to also run the accessibility audit on the screen the
// step left behind: it fails the step on any WCAG 2.2 A or AA violation axe finds (see
// ./a11y.ts), before the photograph is taken.
export async function journeyStep(
  page: Page,
  name: string,
  body: () => Promise<void>,
  options: { a11y?: boolean | A11yOptions } = {},
): Promise<void> {
  await base.step(name, async () => {
    const origin = appOrigin();
    let problems = pageProblems.get(page);
    if (!problems) {
      if (page.url() !== "about:blank") {
        throw new Error(
          `journeyStep was handed a page that is not watched and has already navigated (to ${page.url()}), so a script the Content-Security-Policy blocked there would go unreported. Only pages in the browser context of a \`test\` imported from tests/helpers/journey (or tests/helpers/seed) are watched from their first navigation: take \`test\` from there, and open extra pages and popups from that test's own page or context, not from a new browser context.`,
        );
      }
      problems = await watchPage(page, origin);
    }
    await body();
    await judgeScreen(page, problems, origin);
    if (options.a11y && onApp(page.url(), origin)) {
      await expectNoA11yViolations(page, options.a11y === true ? {} : options.a11y);
    }
    const step = nextStep();
    record(page, step, name, await capture(page, step, name));
  });
}
