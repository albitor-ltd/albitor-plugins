# WCAG 2.2 Success Criteria (Level A and AA)

> **Sources:** [WCAG 2.2 (W3C Recommendation)](https://www.w3.org/TR/WCAG22/) and [How to Meet WCAG 2.2 Quick Reference](https://www.w3.org/WAI/WCAG22/quickref/).
> © W3C, used under the W3C Document Licence. Distilled reference — verify against the live source for current detail. "How to meet" tips are shadcn/ui + Radix-aware where relevant.

## What you need to know

WCAG 2.2 organises requirements under **four principles (POUR)** — Perceivable, Operable, Understandable, Robust — each broken into **guidelines** containing **success criteria** graded **A** (minimum), **AA**, or **AAA**.

**The target is every Level A and Level AA criterion (50 total: 31 A + 19 AA).** Level AAA is not required and is not listed here.

Radix (under shadcn/ui) satisfies many *Robust* and *Operable* criteria for its components out of the box — but only when used correctly and composed with labels, titles, and sufficient contrast.

### New in WCAG 2.2

| Criterion | Name | Level |
| --- | --- | --- |
| 2.4.11 | Focus Not Obscured (Minimum) | AA |
| 2.5.7 | Dragging Movements | AA |
| 2.5.8 | Target Size (Minimum) | AA |
| 3.2.6 | Consistent Help | A |
| 3.3.7 | Redundant Entry | A |
| 3.3.8 | Accessible Authentication (Minimum) | AA |

(2.4.12, 2.4.13, and 3.3.9 are also new but are AAA and not required.)

**Removed in WCAG 2.2:** **4.1.1 Parsing** was removed entirely — obsolete because modern browsers and assistive tech handle markup parsing reliably. Do not test against it.

---

## 1. Perceivable

Information and UI components must be presentable to users in ways they can perceive.

### Guideline 1.1 Text Alternatives

| # | Name | Level | What it requires | How to meet it |
| --- | --- | --- | --- | --- |
| 1.1.1 | Non-text Content | A | All non-text content (images, icons, charts) has a text alternative; decorative items are hidden from AT. | Give meaningful `<img>`/`next/image` an `alt` describing purpose; `alt=""` for decorative. Icon-only `<Button size="icon">` needs `aria-label`/`sr-only` text. lucide icons are `aria-hidden` by default — the name goes on the control. `Chart`/recharts needs a text/table alternative. |

### Guideline 1.2 Time-based Media

| # | Name | Level | What it requires | How to meet it |
| --- | --- | --- | --- | --- |
| 1.2.1 | Audio-only and Video-only (Prerecorded) | A | Transcript for audio-only; transcript or audio for video-only. | Publish a transcript next to audio; describe silent video in text. |
| 1.2.2 | Captions (Prerecorded) | A | Synchronised media has captions. | Add accurate synchronised captions (`<track kind="captions">`). |
| 1.2.3 | Audio Description or Media Alternative (Prerecorded) | A | Prerecorded video has audio description or full text alternative. | Provide a descriptive transcript or an audio-described track. |
| 1.2.4 | Captions (Live) | AA | Live synchronised media has captions. | Arrange real-time captioning for live streams. |
| 1.2.5 | Audio Description (Prerecorded) | AA | Prerecorded video has an audio description. | Add an audio-description track for key visual info. |

### Guideline 1.3 Adaptable

| # | Name | Level | What it requires | How to meet it |
| --- | --- | --- | --- | --- |
| 1.3.1 | Info and Relationships | A | Visual structure/relationships are also available programmatically. | Semantic HTML: real `<h1>`–`<h6>`, shadcn `Table` (`<th scope>`), lists, `Label`↔input, `RadioGroup`/`fieldset`. Don't fake structure with styled `<div>`s. |
| 1.3.2 | Meaningful Sequence | A | Reading/navigation order is programmatically correct. | Keep DOM order = reading order; don't reorder with CSS in a way that desyncs from the DOM. |
| 1.3.3 | Sensory Characteristics | A | Instructions don't rely solely on shape, size, location, or sound. | Write "select Save" not "click the button on the right". |
| 1.3.4 | Orientation | AA | Content isn't locked to one orientation unless essential. | Don't force portrait/landscape; responsive Tailwind layouts reflow both ways. |
| 1.3.5 | Identify Input Purpose | AA | Purpose of common inputs is programmatically determinable. | Set `autoComplete` tokens (`email`, `name`, `tel`, `new-password`) on `Input`s. |

### Guideline 1.4 Distinguishable

| # | Name | Level | What it requires | How to meet it |
| --- | --- | --- | --- | --- |
| 1.4.1 | Use of Color | A | Colour isn't the only means of conveying information. | Pair `destructive`/`success` colour with text or an icon; don't show status by colour alone (Badge text carries the meaning). |
| 1.4.2 | Audio Control | A | Auto-playing audio >3s can be paused/stopped/muted. | Avoid autoplay; provide a visible control if unavoidable. |
| 1.4.3 | Contrast (Minimum) | AA | Text contrast ≥ 4.5:1 (≥ 3:1 large: 18.66px bold / 24px). | Check the token pairs — especially `text-muted-foreground` on tinted `card`/`muted` surfaces, and your brand `primary`/`primary-foreground` — in **both** light and dark themes. |
| 1.4.4 | Resize Text | AA | Text scales to 200% without loss. | Use Tailwind's `rem`-based scale; avoid fixed pixel heights that clip text. |
| 1.4.5 | Images of Text | AA | Real text, not images of text (except logos/essential). | Style text with CSS/Tailwind, not baked into an image. |
| 1.4.10 | Reflow | AA | Reflows to 320px (400% zoom) without 2-D scrolling. | Responsive flex/grid + relative units; avoid fixed-width containers; let tables scroll within a `ScrollArea`/`overflow-x` wrapper. |
| 1.4.11 | Non-text Contrast | AA | UI components and meaningful graphics ≥ 3:1 against adjacent colours. | Ensure `border-input`, button edges, icons, and the `--ring` focus colour meet 3:1 in both themes. |
| 1.4.12 | Text Spacing | AA | No loss when users override line/letter/word/paragraph spacing. | Don't fix container heights around text; allow growth. |
| 1.4.13 | Content on Hover or Focus | AA | Hover/focus content is dismissible, hoverable, persistent. | `Tooltip`/`HoverCard`/`Popover` content: dismissible with `Esc`, pointer can move onto it, stays until dismissed. Don't put essential info only in a Tooltip. |

---

## 2. Operable

UI components and navigation must be operable.

### Guideline 2.1 Keyboard Accessible

| # | Name | Level | What it requires | How to meet it |
| --- | --- | --- | --- | --- |
| 2.1.1 | Keyboard | A | All functionality is operable by keyboard. | Use native controls / Radix primitives (they handle keys). No `onClick` on `<div>`/`<span>` — use `Button` (or `<a>`/`Link` for navigation). |
| 2.1.2 | No Keyboard Trap | A | Focus can always move away. | Radix `Dialog`/`Sheet`/`Popover` trap focus only while open and release on close — don't break that with manual focus hacks. |
| 2.1.4 | Character Key Shortcuts | A | Single-key shortcuts can be turned off, remapped, or are focus-only. | Scope command-palette/single-letter shortcuts to focus, or make them modified (⌘K), or allow disabling. |

### Guideline 2.2 Enough Time

| # | Name | Level | What it requires | How to meet it |
| --- | --- | --- | --- | --- |
| 2.2.1 | Timing Adjustable | A | Time limits can be turned off, adjusted, or extended. | Warn before session timeout and offer "extend"; avoid arbitrary limits. |
| 2.2.2 | Pause, Stop, Hide | A | Moving/auto-updating content >5s can be paused/stopped/hidden. | Give `Carousel`/tickers a pause control; respect `prefers-reduced-motion` (Tailwind `motion-reduce:`). |

### Guideline 2.3 Seizures and Physical Reactions

| # | Name | Level | What it requires | How to meet it |
| --- | --- | --- | --- | --- |
| 2.3.1 | Three Flashes or Below Threshold | A | Nothing flashes more than three times per second. | Avoid rapid flashing/strobe animations. |

### Guideline 2.4 Navigable

| # | Name | Level | What it requires | How to meet it |
| --- | --- | --- | --- | --- |
| 2.4.1 | Bypass Blocks | A | A way to skip repeated blocks. | Add a "Skip to main content" link (first focusable) + landmarks (`<nav>`, `<main id="main-content">`). |
| 2.4.2 | Page Titled | A | Pages have descriptive titles. | Set a unique `<title>` per route (Next.js `metadata`/`<title>`). |
| 2.4.3 | Focus Order | A | Focus order preserves meaning. | Keep DOM order logical; avoid positive `tabIndex`. |
| 2.4.4 | Link Purpose (In Context) | A | Link purpose clear from text/context. | Descriptive link text ("View invoice"), not "click here". |
| 2.4.5 | Multiple Ways | AA | More than one way to find a page (except in a process). | Provide nav + search/Command palette or a sitemap. |
| 2.4.6 | Headings and Labels | AA | Headings and labels describe topic/purpose. | Clear `CardTitle`/headings and specific `Label`s. |
| 2.4.7 | Focus Visible | AA | Keyboard focus indicator is visible. | Keep shadcn's `focus-visible:ring-2 ring-ring`; never `outline-none` without a replacement. |
| 2.4.11 | Focus Not Obscured (Minimum) **[NEW 2.2]** | AA | A focused element isn't entirely hidden by author content. | Manage z-index / `scroll-padding` so sticky headers, `Sheet`s, and cookie bars don't fully cover the focused control. |

### Guideline 2.5 Input Modalities

| # | Name | Level | What it requires | How to meet it |
| --- | --- | --- | --- | --- |
| 2.5.1 | Pointer Gestures | A | Multipoint/path gestures have a single-pointer alternative. | Provide button/tap alternatives to swipe/pinch/drag. |
| 2.5.2 | Pointer Cancellation | A | Actions complete on up-event or can be aborted/undone. | Trigger on click/up-event; allow moving off to cancel (Radix defaults do this). |
| 2.5.3 | Label in Name | A | Accessible name contains the visible label text. | Ensure `aria-label` on a labelled control includes the visible text (voice-control users say it). |
| 2.5.4 | Motion Actuation | A | Motion-triggered functions have a UI alternative and can be disabled. | Provide a button for any "shake"/tilt action. |
| 2.5.7 | Dragging Movements **[NEW 2.2]** | AA | Drag operations have a single-pointer non-drag alternative. | `Slider`/reorder/`Resizable`/drag-and-drop need tap/click or arrow-key alternatives. |
| 2.5.8 | Target Size (Minimum) **[NEW 2.2]** | AA | Pointer targets ≥ 24×24 CSS px, or adequately spaced. | shadcn `Button size="icon"` is 40×40 (fine); check dense icon rows, table row actions, and small `Badge`/close buttons meet 24px + spacing. |

---

## 3. Understandable

Information and operation of the UI must be understandable.

### Guideline 3.1 Readable

| # | Name | Level | What it requires | How to meet it |
| --- | --- | --- | --- | --- |
| 3.1.1 | Language of Page | A | Default language is programmatically set. | Add `lang` to `<html>` (Next.js: `<html lang="en">` in the root layout). |
| 3.1.2 | Language of Parts | AA | Language changes within content are marked up. | Wrap foreign-language phrases with the correct `lang`. |

### Guideline 3.2 Predictable

| # | Name | Level | What it requires | How to meet it |
| --- | --- | --- | --- | --- |
| 3.2.1 | On Focus | A | Focus doesn't cause an unexpected context change. | Don't auto-submit/navigate/open on focus alone. |
| 3.2.2 | On Input | A | Changing a setting doesn't auto-change context unless warned. | Don't auto-submit a form when a `Select`/`Checkbox` changes; require an explicit action or warn. |
| 3.2.3 | Consistent Navigation | AA | Repeated navigation is in the same relative order across pages. | Keep the `Sidebar`/top-nav order consistent site-wide. |
| 3.2.4 | Consistent Identification | AA | Same-function components identified consistently. | Same icon/label for the same action everywhere; "delete" is always the same destructive pattern. |
| 3.2.6 | Consistent Help **[NEW 2.2]** | A | Help mechanisms appear in the same relative order across pages. | Put help/contact/chat in a consistent location on every page. |

### Guideline 3.3 Input Assistance

| # | Name | Level | What it requires | How to meet it |
| --- | --- | --- | --- | --- |
| 3.3.1 | Error Identification | A | Errors identified and described in text. | shadcn `Form` `FormMessage` renders text errors linked to the field (not colour alone). |
| 3.3.2 | Labels or Instructions | A | Labels/instructions provided for input. | Every control has a `Label`/`FormLabel`; add `FormDescription` hint for format. |
| 3.3.3 | Error Suggestion | A | If a fix is known, suggest it. | Zod messages that say how to fix ("Enter a valid email"). |
| 3.3.4 | Error Prevention (Legal, Financial, Data) | A | Consequential submissions are reversible/checked/confirmed. | Add a review/confirm step; use `AlertDialog` to confirm destructive actions. |
| 3.3.7 | Redundant Entry **[NEW 2.2]** | A | Info already entered isn't requested again in the same process. | Carry forward/pre-fill data across steps; offer "same as above". |
| 3.3.8 | Accessible Authentication (Minimum) **[NEW 2.2]** | AA | Auth doesn't require a cognitive-function test without an alternative. | Allow password managers/paste; offer email/magic-link/passkey; don't require solving a puzzle CAPTCHA with no alternative. |

---

## 4. Robust

Content must be robust enough to be interpreted reliably by user agents and assistive technologies.

### Guideline 4.1 Compatible

> **Note:** 4.1.1 Parsing was **removed** in WCAG 2.2.

| # | Name | Level | What it requires | How to meet it |
| --- | --- | --- | --- | --- |
| 4.1.2 | Name, Role, Value | A | All UI components expose name, role, state, value to AT. | Prefer Radix primitives (they set role/state/`data-state`+ARIA). For a custom widget, wire correct ARIA and keep it updated. Give dialogs a `DialogTitle` and icon buttons an accessible name. |
| 4.1.3 | Status Messages | A | Status messages exposed to AT without moving focus. | Use `aria-live` / `role="status"` — Sonner toasts and shadcn `Form` errors do this; announce "Saved", "3 results". |

---

## Quick reference: counts

- **Level A:** 31 criteria
- **Level AA:** 19 criteria
- **Total target (A + AA):** 50 criteria

Radix/shadcn helps most with 2.1.1, 2.1.2, 2.4.3, 2.4.7, 4.1.2, and 4.1.3 for its components — but 1.1.1, 1.4.3, 2.4.6, 3.3.2, and 4.1.2's *naming* half remain the author's responsibility on every screen.
