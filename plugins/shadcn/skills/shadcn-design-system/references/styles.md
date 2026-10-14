Source: shadcn/ui theming (https://ui.shadcn.com/docs/theming), Tailwind CSS (https://tailwindcss.com/docs), and Radix UI. MIT-licensed. Distilled reference — verify against the live source for current detail (shadcn/ui supports both Tailwind v3 and v4; token formats differ).

# shadcn/ui styles: tokens, theming, variants, focus

Covers the CSS-variable design tokens, colour/dark mode, the `cva` variant system, the `cn()` helper, radius/typography/spacing, and focus states.

Golden rule: **style through the semantic tokens, never hardcoded hex or arbitrary values.** `bg-primary`, `text-muted-foreground`, `border-input`, `ring-ring` all resolve to CSS variables, so themes and dark mode work by swapping variables — not by editing every component. `bg-[#0f172a]` or an inline `style` breaks that and is a conformance smell.

## The token system

shadcn/ui defines a small set of **semantic** design tokens as CSS variables in your global stylesheet (`globals.css` / `app.css`), and maps them to Tailwind colour utilities. Every colour is a foreground/background **pair** so text-on-surface contrast is defined together.

### Core token pairs

| Token | Tailwind classes | Role |
| --- | --- | --- |
| `background` / `foreground` | `bg-background` / `text-foreground` | Page base surface + text |
| `card` / `card-foreground` | `bg-card` / `text-card-foreground` | Card surface + text |
| `popover` / `popover-foreground` | `bg-popover` / `text-popover-foreground` | Popover/menu surface + text |
| `primary` / `primary-foreground` | `bg-primary` / `text-primary-foreground` | Primary actions/brand |
| `secondary` / `secondary-foreground` | `bg-secondary` / … | Secondary actions |
| `muted` / `muted-foreground` | `bg-muted` / `text-muted-foreground` | Subtle surfaces + secondary text |
| `accent` / `accent-foreground` | `bg-accent` / … | Hover/active accents |
| `destructive` / `destructive-foreground` | `bg-destructive` / … | Dangerous/negative actions |
| `border` | `border-border` | Default borders |
| `input` | `border-input` | Form-control borders |
| `ring` | `ring-ring` | Focus ring colour |
| `radius` | `rounded-*` (via `--radius`) | Global corner radius |

Chart tokens (`--chart-1`…`--chart-5`) and Sidebar tokens (`--sidebar-*`) exist for those components. Add your own semantic tokens (e.g. `--success`) the same way rather than hardcoding.

### Defining the tokens (Tailwind v3 — HSL channels)

In v3 the values are bare HSL channels and Tailwind reads them via `hsl(var(--token))`:

```css
:root {
  --background: 0 0% 100%;
  --foreground: 240 10% 3.9%;
  --card: 0 0% 100%;
  --card-foreground: 240 10% 3.9%;
  --popover: 0 0% 100%;
  --popover-foreground: 240 10% 3.9%;
  --primary: 240 5.9% 10%;
  --primary-foreground: 0 0% 98%;
  --secondary: 240 4.8% 95.9%;
  --secondary-foreground: 240 5.9% 10%;
  --muted: 240 4.8% 95.9%;
  --muted-foreground: 240 3.8% 46.1%;
  --accent: 240 4.8% 95.9%;
  --accent-foreground: 240 5.9% 10%;
  --destructive: 0 84.2% 60.2%;
  --destructive-foreground: 0 0% 98%;
  --border: 240 5.9% 90%;
  --input: 240 5.9% 90%;
  --ring: 240 10% 3.9%;
  --radius: 0.5rem;
}
.dark {
  --background: 240 10% 3.9%;
  --foreground: 0 0% 98%;
  --card: 240 10% 3.9%;
  --card-foreground: 0 0% 98%;
  --popover: 240 10% 3.9%;
  --popover-foreground: 0 0% 98%;
  --primary: 0 0% 98%;
  --primary-foreground: 240 5.9% 10%;
  --secondary: 240 3.7% 15.9%;
  --secondary-foreground: 0 0% 98%;
  --muted: 240 3.7% 15.9%;
  --muted-foreground: 240 5% 64.9%;
  --accent: 240 3.7% 15.9%;
  --accent-foreground: 0 0% 98%;
  --destructive: 0 62.8% 30.6%;
  --destructive-foreground: 0 0% 98%;
  --border: 240 3.7% 15.9%;
  --input: 240 3.7% 15.9%;
  --ring: 240 4.9% 83.9%;
  --radius: 0.5rem;
}
```

`tailwind.config.*` (v3) then maps them:

```js
theme: {
  extend: {
    colors: {
      background: "hsl(var(--background))",
      foreground: "hsl(var(--foreground))",
      primary: { DEFAULT: "hsl(var(--primary))", foreground: "hsl(var(--primary-foreground))" },
      // …destructive, muted, accent, card, popover, border, input, ring…
    },
    borderRadius: { lg: "var(--radius)", md: "calc(var(--radius) - 2px)", sm: "calc(var(--radius) - 4px)" },
  },
}
```

### Tailwind v4 (`@theme inline`)
In Tailwind v4 there is no JS config; tokens live in CSS. shadcn/ui's v4 setup declares the variables (commonly in the `oklch()` colour space) under `:root`/`.dark` and exposes them to Tailwind with `@theme inline` (e.g. `--color-primary: var(--primary)`), plus `@custom-variant dark (&:is(.dark *))`. Use `npx shadcn@latest init` to generate the correct v4 or v3 setup for your project; the *class names you write* (`bg-primary`, etc.) are the same across both.

### Base colour
`init` picks a base neutral (Zinc, Slate, Stone, Gray, Neutral) that seeds the grey scale. All examples here use a Zinc-ish neutral. The brand colour is `--primary`; change it in one place to rebrand.

## Dark mode
Theming is driven by a `.dark` class on the `<html>` (or a root) element that swaps the variable block. A library like `next-themes` toggles it and respects the OS preference. Because components reference tokens, no component needs dark-specific classes — but **test both themes for contrast**; the common failure is `text-muted-foreground` on a `bg-muted`/tinted card dropping below 4.5:1.

## Variants — `cva` (class-variance-authority)
shadcn components define their look with `cva`: a base class string plus named `variants` and a `defaultVariants`. Example (Button):

```ts
const buttonVariants = cva(
  "inline-flex items-center justify-center rounded-md text-sm font-medium ring-offset-background transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:pointer-events-none disabled:opacity-50",
  {
    variants: {
      variant: {
        default: "bg-primary text-primary-foreground hover:bg-primary/90",
        destructive: "bg-destructive text-destructive-foreground hover:bg-destructive/90",
        outline: "border border-input bg-background hover:bg-accent hover:text-accent-foreground",
        secondary: "bg-secondary text-secondary-foreground hover:bg-secondary/80",
        ghost: "hover:bg-accent hover:text-accent-foreground",
        link: "text-primary underline-offset-4 hover:underline",
      },
      size: { default: "h-10 px-4 py-2", sm: "h-9 px-3", lg: "h-11 px-8", icon: "h-10 w-10" },
    },
    defaultVariants: { variant: "default", size: "default" },
  }
);
```

Add a new look by adding a variant here, not by passing a long one-off `className` at every call site.

## `cn()` — conflict-safe class merging
Every shadcn component merges classes with `cn(...inputs)` = `twMerge(clsx(inputs))`:

```ts
import { clsx, type ClassValue } from "clsx";
import { twMerge } from "tailwind-merge";
export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}
```

`clsx` handles conditional classes; `tailwind-merge` ensures a later utility wins over an earlier conflicting one (`cn("px-2", "px-4")` → `px-4`). Always merge through `cn()` so consumer `className` overrides resolve predictably — never string-concatenate class names.

## Radius, typography, spacing
- **Radius:** one `--radius` variable feeds `rounded-lg/md/sm` (via `calc`). Change it once to make the whole UI sharper or rounder.
- **Typography:** shadcn ships no opinionated type scale beyond Tailwind's (`text-sm`, `text-base`, `text-lg`, `font-medium`, `tracking-*`). Body default is usually `text-sm`/`text-base`; headings use Tailwind sizes + `font-semibold`. For rich prose, the `@tailwindcss/typography` `prose` classes pair well. Use relative units (Tailwind's `rem`-based scale) so text resizes (WCAG 1.4.4).
- **Spacing:** Tailwind's spacing scale (`p-*`, `gap-*`, `space-y-*`). Prefer `gap`/`space-*` with flex/grid over manual margins. Keep rhythm consistent across a view.
- **The `new-york` vs `default` style:** `init` offers two visual styles (`new-york` is the current default) — they differ in default sizing, shadows, and icon set (`new-york` uses lucide with slightly tighter spacing). Pick one per project and keep it.

## Focus states (do not remove)
shadcn components include a visible focus ring: `focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2`. The `outline-none` is only safe **because** it is paired with a `ring` replacement. **Never strip the ring** or set `outline-none` without an equivalent visible indicator — that fails WCAG 2.4.7 Focus Visible. `--ring` must keep ≥3:1 contrast against adjacent colours (1.4.11); check it in both themes. `:focus-visible` (not `:focus`) means the ring shows for keyboard users without firing on mouse click, which is the intended, accessible behaviour.

## Contrast
Meet WCAG 2.2 AA 1.4.3: 4.5:1 for normal text, 3:1 for large text (≥24px, or ≥18.66px bold) and for UI components/graphics (1.4.11). The default token pairs are chosen to pass on their own surface, but **your brand `--primary` and any custom tokens are your responsibility** — check `primary-foreground` on `primary`, muted text on tinted cards, and disabled states (`opacity-50` can drop contrast below AA; ensure disabled controls aren't the only way to convey required information). Never use colour as the only signal (1.4.1) — pair `destructive`/`success` colour with text or an icon.

## Icons
shadcn/ui uses **lucide-react** icons. Decorative icons need `aria-hidden` (lucide sets this by default when there's no label); an icon that *is* the control's meaning (icon-only button) needs an accessible name on the button (`aria-label`/`sr-only`), not on the icon. Don't embed text as an icon/image where real text works (1.4.5).
