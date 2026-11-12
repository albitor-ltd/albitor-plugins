import type { Page } from "@playwright/test";
import { CODE_CONTEXT_SELECTOR, sourceTextProblem } from "./screen-rules";

// renderedText returns the page's rendered text (innerText: what a person sees, so hidden and
// display:none content is left out), minus everything a person is meant to read as code or type
// into (CODE_CONTEXT_SELECTOR, including the `data-allow-source-text` opt-out). Those elements are
// hidden for the duration of one synchronous read and restored.
export async function renderedText(page: Page): Promise<string> {
  return page.evaluate((skip) => {
    const hidden: Array<[HTMLElement, string, string]> = [];
    for (const el of Array.from(document.body.querySelectorAll<HTMLElement>(skip))) {
      hidden.push([el, el.style.getPropertyValue("display"), el.style.getPropertyPriority("display")]);
      el.style.setProperty("display", "none", "important");
    }
    try {
      return document.body.innerText;
    } finally {
      for (const [el, value, priority] of hidden) {
        el.style.setProperty("display", value, priority);
      }
    }
  }, CODE_CONTEXT_SELECTOR);
}

// A page that navigates while it is read loses its execution context. That is the one expected
// failure; the next step judges whatever the page lands on.
const MID_NAVIGATION = /execution context was destroyed|navigat|target (page, context or browser )?has been closed/i;

// pageSourceTextProblem is sourceTextProblem for a live page: "" when the screen shows none.
export async function pageSourceTextProblem(page: Page): Promise<string> {
  try {
    return sourceTextProblem([await renderedText(page)]);
  } catch (err) {
    if (err instanceof Error && MID_NAVIGATION.test(err.message)) {
      return "";
    }
    throw err;
  }
}
