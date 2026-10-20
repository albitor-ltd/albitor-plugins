# Design system notes

What the shadcn design system asks of a screen, restated for this kit (Svelte, Bits UI, scoped CSS
over tokens). The kit's own pieces are in `components.md` and the colour roles in `tokens.md`.

## The ownership model

shadcn is not a runtime library. Components are source in the app, copied in once and edited
freely, and there is no upstream package to upgrade. The dependencies that remain are real: Bits
UI for behaviour. Behaviour (focus trap, keyboard, ARIA, portals) is never rewritten by hand;
labels, titles, accessible names, contrast and heading order stay the app's job, because a
primitive is a baseline and not a guarantee. A Bits UI dialog with no title still fails.

## Beyond the kit: the Bits UI primitive for each shadcn component

The kit ships the shell pieces. For anything else, import the primitive from `bits-ui`, style it
with tokens, and keep the `data-` and `aria-` attributes it sets. shadcn-svelte
(https://www.shadcn-svelte.com/docs/components) has the designs to copy from; its Tailwind classes
become scoped CSS over tokens here.

| Need | Primitive | Notes |
|---|---|---|
| Confirm a destructive action | `AlertDialog` | Interrupts; focus lands on the safe action |
| Expand and collapse | `Accordion`, `Collapsible` | |
| Checkbox, radio group, switch | `Checkbox`, `RadioGroup`, `Switch` | Wrap in `Field`-style label wiring |
| Dropdown select | `Select` | Typeahead and keyboard included |
| Searchable select, command palette | `Combobox`, `Command` | |
| Tabs | `Tabs` | Arrow keys move, panels are labelled by their tab |
| Menus | `DropdownMenu`, `ContextMenu`, `Menubar`, `NavigationMenu` | |
| Floating content | `Popover`, `Tooltip`, `LinkPreview` | A tooltip never holds essential information |
| Date entry | `DateField`, `DatePicker`, `Calendar` | Uses `@internationalized/date`, already a kit dependency |
| Progress, meter, slider | `Progress`, `Meter`, `Slider` | Give each an accessible name |
| Pagination, separator, toggle | `Pagination`, `Separator`, `Toggle`, `ToggleGroup` | |
| Avatar, scroll region, aspect ratio | `Avatar`, `ScrollArea`, `AspectRatio` | |

Purely presentational pieces (card, badge, alert, skeleton, breadcrumb) need no primitive: use real
elements (`nav` with an `ol` for a breadcrumb, `role="alert"` for a callout) styled with tokens.

## Variants

Name the looks a component offers (`variant`, and `size` if sizes differ) and implement each as a
class over tokens in the component's `<style>`. Pass a variant, not a one-off override. This keeps a
restyle to one file and keeps the contrast audit's list of pairs complete.

## Focus

Every interactive element shows a visible focus indicator. Use `:focus-visible`, so keyboard users
see it and a mouse click does not trigger it. Removing an outline is safe only when a replacement is
at least as visible. `--ring` must keep 3:1 against the background and against `--primary`
(`contrast.test.ts` checks both); a changed ring colour is re-checked in both schemes.

## Contrast traps

The shipped pairs pass. Your additions are yours to check:

- Secondary text (`--muted-foreground`) on a tinted surface (`--muted`) is the common failure.
- A brand colour used for `--primary` needs `--primary-foreground` at 4.5:1 and the fill itself at
  3:1 against the page.
- A disabled control that is dimmed with opacity can fall below AA; do not let it be the only
  place information appears.
- A new semantic token (a `--success`, say) is declared as a foreground and surface pair and added
  to `contrast.test.ts` in the same change.
- Colour is never the only signal (WCAG 1.4.1): an error or success state also has text or an icon.

## Icons

A decorative icon is `aria-hidden="true"`. An icon-only control gets its name on the control
(`aria-label` or visually hidden text), not on the icon. Do not use an image of text where real
text works. Prefer inline SVG over an icon font, and size icons in `rem`/`em` so they scale with
text.

## Type and spacing

Use the `--text-*` tokens and `rem`-based sizes so text resizes (WCAG 1.4.4). Space with `gap` on
flex or grid rather than margins, and keep one rhythm across a view.
