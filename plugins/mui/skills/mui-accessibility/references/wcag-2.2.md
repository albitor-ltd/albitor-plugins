# WCAG 2.2 Success Criteria (Level A and AA)

> **Sources:** [WCAG 2.2 (W3C Recommendation)](https://www.w3.org/TR/WCAG22/) and [How to Meet WCAG 2.2 Quick Reference](https://www.w3.org/WAI/WCAG22/quickref/), plus [MUI accessibility guidance](https://mui.com/).
> WCAG 2.2 © W3C, used under the W3C Document Licence. Distilled reference — verify against the live source for current detail. The "MUI note" rows show how Material UI helps meet a criterion (and how misusing it breaks the criterion).

## What you need to know

WCAG 2.2 organises requirements under **four principles (POUR)** — Perceivable, Operable, Understandable, Robust — each broken into **guidelines** containing **success criteria** graded **A** (minimum), **AA**, or **AAA**.

**The common conformance target is every Level A and Level AA criterion** — this is what EN 301 549 / the European Accessibility Act, Section 508, and ADA settlements reference. Level AAA is not required and is not listed here.

This page covers **all 50 Level A and AA success criteria** (31 at Level A, 19 at Level AA).

### New in WCAG 2.2

| Criterion | Name | Level |
| --- | --- | --- |
| 2.4.11 | Focus Not Obscured (Minimum) | AA |
| 2.5.7 | Dragging Movements | AA |
| 2.5.8 | Target Size (Minimum) | AA |
| 3.2.6 | Consistent Help | A |
| 3.3.7 | Redundant Entry | A |
| 3.3.8 | Accessible Authentication (Minimum) | AA |

(2.4.12 Focus Not Obscured (Enhanced), 2.4.13 Focus Appearance, and 3.3.9 Accessible Authentication (Enhanced) are also new but are AAA and not required.)

**Removed in WCAG 2.2:** **4.1.1 Parsing** was removed entirely — it is obsolete because modern browsers and assistive tech handle markup parsing reliably. Do not test against it.

---

## 1. Perceivable

Information and UI components must be presentable to users in ways they can perceive.

### Guideline 1.1 Text Alternatives

| # | Name | Level | What it requires | How to meet it (incl. MUI) |
| --- | --- | --- | --- | --- |
| 1.1.1 | Non-text Content | A | All non-text content (images, icons, charts) has a text alternative serving the equivalent purpose; decorative items are hidden from assistive tech. | Give meaningful images an `alt` (`<Avatar alt="Jane Doe">`, `<CardMedia image=… title=…>`); use `alt=""`/`aria-hidden` for decorative icons; an icon-only `IconButton`/`Fab` needs `aria-label`; provide a data-table alternative for MUI X charts. |

### Guideline 1.2 Time-based Media

| # | Name | Level | What it requires | How to meet it |
| --- | --- | --- | --- | --- |
| 1.2.1 | Audio-only and Video-only (Prerecorded) | A | Prerecorded audio-only has a transcript; prerecorded video-only has a transcript or audio track. | Publish a full transcript next to podcasts; describe silent video in text or an audio narration. |
| 1.2.2 | Captions (Prerecorded) | A | Synchronised media (video with audio) has captions. | Add accurate, synchronised closed captions to all prerecorded video. |
| 1.2.3 | Audio Description or Media Alternative (Prerecorded) | A | Prerecorded video has audio description or a full text alternative. | Provide a transcript including descriptions of important visual detail, or an audio-described track. |
| 1.2.4 | Captions (Live) | AA | Live synchronised media has captions. | Arrange live (real-time) captioning for webcasts and live streams. |
| 1.2.5 | Audio Description (Prerecorded) | AA | All prerecorded video in synchronised media has an audio description. | Add an audio-description track narrating key visual information during natural pauses. |

### Guideline 1.3 Adaptable

| # | Name | Level | What it requires | How to meet it (incl. MUI) |
| --- | --- | --- | --- | --- |
| 1.3.1 | Info and Relationships | A | Structure and relationships conveyed visually are also available programmatically. | Use semantic HTML: real headings (`<Typography component="h1">`), `<Table>` with `<th scope>`, lists (`<List>`), `<label>` tied to inputs (`TextField label`), `FormLabel`/`FormControl` for grouped controls. |
| 1.3.2 | Meaningful Sequence | A | The reading/navigation order is programmatically correct where order affects meaning. | Make the DOM order match the logical reading order; don't reorder with CSS `order`/absolute positioning in a way that breaks it. |
| 1.3.3 | Sensory Characteristics | A | Instructions don't rely solely on shape, size, location, orientation or sound. | Write "select Save" not "click the round button on the right". |
| 1.3.4 | Orientation | AA | Content isn't locked to one orientation unless essential. | Don't force portrait or landscape; let MUI's responsive layout reflow in both. |
| 1.3.5 | Identify Input Purpose | AA | The purpose of common input fields is programmatically determinable. | Set `autoComplete` on `TextField`s (`autoComplete="email"`, `"name"`, `"tel"`). |

### Guideline 1.4 Distinguishable

| # | Name | Level | What it requires | How to meet it (incl. MUI) |
| --- | --- | --- | --- | --- |
| 1.4.1 | Use of Color | A | Colour is not the only means of conveying information or distinguishing elements. | Pair colour with text/icon: `Alert` and `Chip` carry an icon/label, not colour alone; don't signal errors with red only — set `error` **and** a `helperText` message. |
| 1.4.2 | Audio Control | A | Audio that plays automatically for >3s can be paused, stopped or muted. | Avoid autoplay audio; if unavoidable, provide a visible pause/stop/volume control. |
| 1.4.3 | Contrast (Minimum) | AA | Text contrast ≥ 4.5:1 (≥ 3:1 for large text, 18.66px bold / 24px regular). | Check colours with a contrast checker; **re-check after theming** — a brand `primary.main` with white `contrastText` can fall below 4.5:1. |
| 1.4.4 | Resize Text | AA | Text can scale to 200% without loss of content or function. | Use MUI's `rem`-based type scale; avoid fixed pixel heights that clip text. |
| 1.4.5 | Images of Text | AA | Use real text, not images of text, except logos or where essential. | Render text with `Typography`, not baked into an image. |
| 1.4.10 | Reflow | AA | Content reflows to a 320px-wide viewport (400% zoom) without two-dimensional scrolling. | Use responsive `Grid`/`Stack`/`sx` breakpoints; avoid fixed-width layouts; let content go single-column on small screens. |
| 1.4.11 | Non-text Contrast | AA | UI components and meaningful graphics have ≥ 3:1 contrast against adjacent colours. | Ensure `TextField`/`OutlinedInput` borders, button edges, icons and focus indicators meet 3:1 after theming. |
| 1.4.12 | Text Spacing | AA | No loss of content when users override line height, paragraph, letter and word spacing. | Don't fix container heights around text; allow text to grow. |
| 1.4.13 | Content on Hover or Focus | AA | Hover/focus-triggered content is dismissible, hoverable and persistent. | MUI `Tooltip` is dismissible with Esc and focus-triggerable — keep those; make popovers stay until dismissed. |

---

## 2. Operable

UI components and navigation must be operable.

### Guideline 2.1 Keyboard Accessible

| # | Name | Level | What it requires | How to meet it (incl. MUI) |
| --- | --- | --- | --- | --- |
| 2.1.1 | Keyboard | A | All functionality is operable by keyboard. | Use native/MUI interactive elements (`Button`, `IconButton`, `Link`, `ListItemButton`, `CardActionArea`); never a `<div onClick>` — for a clickable list row use `ListItemButton`, not `ListItem` + `onClick`. |
| 2.1.2 | No Keyboard Trap | A | Keyboard focus can always move away from a component. | MUI `Dialog`/`Drawer`/`Menu` trap focus while open and release on close — keep that; test Tab/Shift+Tab through everything. |
| 2.1.4 | Character Key Shortcuts | A | Single-character shortcuts can be turned off, remapped, or are focus-only. | Avoid lone-letter shortcuts, or make them active only when a component has focus / offer a way to disable them. |

### Guideline 2.2 Enough Time

| # | Name | Level | What it requires | How to meet it |
| --- | --- | --- | --- | --- |
| 2.2.1 | Timing Adjustable | A | Time limits can be turned off, adjusted or extended (with exceptions). | Warn before a session times out and offer "extend"; avoid arbitrary limits. |
| 2.2.2 | Pause, Stop, Hide | A | Moving/blinking/auto-updating content lasting >5s can be paused, stopped or hidden. | Add pause controls to carousels/tickers; respect `prefers-reduced-motion`; don't auto-advance a `Snackbar`-driven carousel without control. |

### Guideline 2.3 Seizures and Physical Reactions

| # | Name | Level | What it requires | How to meet it |
| --- | --- | --- | --- | --- |
| 2.3.1 | Three Flashes or Below Threshold | A | Nothing flashes more than three times per second. | Avoid rapid flashing; keep any flashing below the general and red flash thresholds. |

### Guideline 2.4 Navigable

| # | Name | Level | What it requires | How to meet it (incl. MUI) |
| --- | --- | --- | --- | --- |
| 2.4.1 | Bypass Blocks | A | A way exists to skip blocks repeated across pages. | Add a "Skip to main content" link and use landmarks (`<nav>`, `<main>`); wrap the `AppBar`/`Drawer` and main content in the right landmarks. |
| 2.4.2 | Page Titled | A | Pages have descriptive titles. | Set a unique, meaningful `<title>` per page/route. |
| 2.4.3 | Focus Order | A | Focus moves in an order that preserves meaning and operability. | Keep DOM order logical; avoid positive `tabIndex`. |
| 2.4.4 | Link Purpose (In Context) | A | A link's purpose is clear from its text or surrounding context. | Write descriptive `Link` text ("View invoice"), not "click here". |
| 2.4.5 | Multiple Ways | AA | More than one way to find a page exists (except within a process). | Provide navigation plus search, a sitemap, or related links. |
| 2.4.6 | Headings and Labels | AA | Headings and labels describe topic or purpose. | Write clear headings (`Typography`) and specific form labels. |
| 2.4.7 | Focus Visible | AA | The keyboard focus indicator is visible. | Keep MUI's focus indicators; never `outline: none` without a visible replacement. |
| 2.4.11 | Focus Not Obscured (Minimum) **[NEW 2.2]** | AA | A focused element is not *entirely* hidden by author content. | A fixed `AppBar`/sticky footer must not fully cover the focused control — use `scroll-padding` and manage z-index. |

### Guideline 2.5 Input Modalities

| # | Name | Level | What it requires | How to meet it (incl. MUI) |
| --- | --- | --- | --- | --- |
| 2.5.1 | Pointer Gestures | A | Multipoint/path gestures have a single-pointer alternative (unless essential). | Provide buttons/taps as alternatives to swipe/pinch/drag-path gestures. |
| 2.5.2 | Pointer Cancellation | A | Single-pointer actions complete on up-event, or can be aborted/undone. | Trigger actions on click/up-event, not `mousedown`; allow moving off to cancel. |
| 2.5.3 | Label in Name | A | A control's accessible name contains its visible label text. | Ensure `aria-label`/`aria-labelledby` includes the visible text so voice-control users can say it (watch icon-`IconButton`s where the `aria-label` may differ from a nearby visible label). |
| 2.5.4 | Motion Actuation | A | Functions triggered by device motion have a UI alternative and can be disabled. | Provide a button for any "shake to…" action and let users turn motion control off. |
| 2.5.7 | Dragging Movements **[NEW 2.2]** | AA | Drag operations have a single-pointer non-drag alternative. | For drag-and-drop reordering (or a `Slider`), offer tap/click or keyboard alternatives (up/down buttons, arrow keys). |
| 2.5.8 | Target Size (Minimum) **[NEW 2.2]** | AA | Pointer targets are ≥ 24×24 CSS px, or have adequate spacing (exceptions: inline links, UA-default sizing). | MUI default control sizes meet this; be careful shrinking `IconButton`/`Chip` (`size="small"`) below 24×24 or crowding them. |

---

## 3. Understandable

Information and operation of the UI must be understandable.

### Guideline 3.1 Readable

| # | Name | Level | What it requires | How to meet it |
| --- | --- | --- | --- | --- |
| 3.1.1 | Language of Page | A | The default human language of the page is programmatically set. | Add `lang` to `<html>` (e.g. `<html lang="en">`). |
| 3.1.2 | Language of Parts | AA | Language changes within content are marked up. | Wrap foreign-language phrases in an element with the correct `lang`. |

### Guideline 3.2 Predictable

| # | Name | Level | What it requires | How to meet it (incl. MUI) |
| --- | --- | --- | --- | --- |
| 3.2.1 | On Focus | A | Receiving focus doesn't cause an unexpected change of context. | Don't auto-submit/navigate/open a popup just because an element is focused. |
| 3.2.2 | On Input | A | Changing a setting doesn't auto-change context unless the user is warned. | Don't auto-submit a form when a `Select`/`Switch` changes; require an explicit action or warn first. |
| 3.2.3 | Consistent Navigation | AA | Repeated navigation appears in the same relative order across pages. | Keep the `AppBar`/`Drawer`/footer order consistent site-wide. |
| 3.2.4 | Consistent Identification | AA | Components with the same function are identified consistently. | Use the same label/icon for the same action everywhere. |
| 3.2.6 | Consistent Help **[NEW 2.2]** | A | Help mechanisms appear in the same relative order across pages. | Place help links / contact / chat in a consistent location on every page. |

### Guideline 3.3 Input Assistance

| # | Name | Level | What it requires | How to meet it (incl. MUI) |
| --- | --- | --- | --- | --- |
| 3.3.1 | Error Identification | A | Input errors are identified and described in text. | Set `error` **and** a text `helperText` on the field (not colour alone); link/summarise errors. |
| 3.3.2 | Labels or Instructions | A | Labels/instructions are provided for user input. | Give every `TextField` a `label`; add `helperText` for format hints. |
| 3.3.3 | Error Suggestion | A | If an error is detected and a fix is known, suggest it. | Tell users how to fix it in `helperText` ("Enter a date as DD/MM/YYYY"). |
| 3.3.4 | Error Prevention (Legal, Financial, Data) | A | Legal/financial/data submissions are reversible, checked, or confirmed. | Add a review step (or a confirmation `Dialog`) before a consequential submit. |
| 3.3.7 | Redundant Entry **[NEW 2.2]** | A | Information already entered isn't requested again in the same process (with exceptions). | Pre-fill/carry forward data across steps of a `Stepper`; offer "same as above". |
| 3.3.8 | Accessible Authentication (Minimum) **[NEW 2.2]** | AA | Authentication doesn't require a cognitive function test without an alternative. | Allow password managers/paste on the password `TextField`; offer email/magic-link instead of CAPTCHAs or memory puzzles. |

---

## 4. Robust

Content must be robust enough to be interpreted reliably by user agents and assistive technologies.

### Guideline 4.1 Compatible

> **Note:** 4.1.1 Parsing was **removed** in WCAG 2.2.

| # | Name | Level | What it requires | How to meet it (incl. MUI) |
| --- | --- | --- | --- | --- |
| 4.1.2 | Name, Role, Value | A | All UI components expose a name, role, state and value to assistive tech. | Prefer MUI components (correct roles/states built in); label them (`TextField label`, icon-`IconButton` `aria-label`, `Dialog` `aria-labelledby`); if you drop to `Modal`/`Popper`, re-implement ARIA yourself. |
| 4.1.3 | Status Messages | A | Status messages are exposed to assistive tech without moving focus. | `Snackbar`/`Alert` use live regions; for custom updates use `aria-live="polite"` (or `role="status"`/`"alert"`) for "3 results found" / "Saved". |

---

## Quick reference: counts

- **Level A:** 31 criteria
- **Level AA:** 19 criteria
- **Total for AA conformance (A + AA):** 50 criteria
