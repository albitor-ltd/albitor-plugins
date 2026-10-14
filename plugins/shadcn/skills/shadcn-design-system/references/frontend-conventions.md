Source: shadcn/ui installation & CLI docs (https://ui.shadcn.com/docs/installation, /docs/cli, /docs/components-json), Tailwind CSS, Radix UI. MIT-licensed. Distilled reference — verify against the live source for current detail.

# shadcn/ui conventions: how to actually build it

shadcn/ui is a **copy-in** design system, not an npm component library. The CLI writes component source into your project; you own and edit it. This file covers initialising a project, the CLI, `components.json`, Tailwind v3 vs v4, the ownership model, React Server vs Client Components, and porting to non-React stacks.

## The mental model (don't skip this)

- You do **not** `import { Button } from "shadcn-ui"`. You run `npx shadcn@latest add button`, which drops `components/ui/button.tsx` into your repo. You then `import { Button } from "@/components/ui/button"`.
- There is **no upstream package to upgrade**. Updates are re-pulled per component (`add` again) and reconciled by hand. This is deliberate — you own the code and its accessibility.
- The *dependencies* it installs are real npm packages: the Radix primitives (`@radix-ui/react-*`), `class-variance-authority`, `clsx`, `tailwind-merge`, `lucide-react`, and per-component libs (`react-hook-form`, `zod`, `cmdk`, `sonner`, `vaul`, `embla-carousel-react`, `react-day-picker`, `@tanstack/react-table`, `recharts`, `input-otp`, `react-resizable-panels`).

## Initialising a project

Prerequisite: a React app with Tailwind CSS configured (Next.js, Vite, Astro, Remix, etc.). Then:

```bash
npx shadcn@latest init
```

`init` asks for the **style** (`new-york` is the current default), the **base colour** (Zinc/Slate/Stone/Gray/Neutral), and whether to use **CSS variables** for theming (recommended — yes). It writes `components.json`, sets up the CSS-variable tokens in your global stylesheet, adds the `cn()` util (`lib/utils.ts`), and wires Tailwind. Then add components as you need them:

```bash
npx shadcn@latest add button input card dialog form
```

(Run `add` with no args for an interactive picker; `npx shadcn@latest add` also accepts a registry URL.)

## `components.json`

The config the CLI reads. Key fields:

```json
{
  "$schema": "https://ui.shadcn.com/schema.json",
  "style": "new-york",
  "rsc": true,
  "tsx": true,
  "tailwind": {
    "config": "tailwind.config.ts",
    "css": "app/globals.css",
    "baseColor": "zinc",
    "cssVariables": true
  },
  "aliases": {
    "components": "@/components",
    "ui": "@/components/ui",
    "utils": "@/lib/utils",
    "lib": "@/lib",
    "hooks": "@/hooks"
  }
}
```

- `rsc: true` marks the project as React Server Components (Next.js App Router) so the CLI adds `"use client"` where needed.
- `aliases` must match your `tsconfig`/`jsconfig` path aliases (e.g. `@/*` → `./src/*` or `./`).
- `cssVariables: true` = theme via CSS variables (the default, recommended); `false` = utility-class theming (harder to theme/dark-mode).

## Tailwind v3 vs v4

shadcn/ui supports both:
- **v3:** tokens are HSL channels in `:root`/`.dark`, mapped in `tailwind.config.{js,ts}` via `hsl(var(--token))`. Uses `tailwindcss-animate` for the enter/exit animations.
- **v4:** no JS config; tokens (often `oklch()`) live in CSS, exposed with `@theme inline`, and dark mode via `@custom-variant dark (&:is(.dark *))`. Uses `tw-animate-css`.

Let `init` generate the right setup for your Tailwind version. The class names you author (`bg-primary`, `rounded-lg`, `focus-visible:ring-ring`) are identical across versions — only the token *plumbing* differs. See `styles.md` for both token formats.

## React Server Components vs Client Components

In the Next.js App Router, interactive Radix-based components (Dialog, Dropdown Menu, Select, Tabs, Accordion, Popover, Form, etc.) are **Client Components** and carry `"use client"` at the top of their file. Purely presentational ones (Card, Badge, Separator, Skeleton, Table markup) can render on the server. Practical rules:
- Keep pages/layouts as Server Components where possible; push interactivity into small client leaf components.
- Don't add `"use client"` to a whole page just to use one interactive component — isolate it.
- Server-render the initial HTML so the page works before hydration (progressive enhancement); Radix enhances the already-rendered, accessible DOM.

## Progressive enhancement and accessibility posture

- Radix primitives ship the keyboard, focus-management, and ARIA behaviour — **use them and don't re-implement**. That is the single biggest accessibility win.
- Radix is a **baseline, not a guarantee**: you still own labels (`Label`/`Form`), dialog titles (`DialogTitle`), accessible names for icon-only buttons, colour contrast of your tokens, heading order, and alt text. A perfectly-wired Radix Dialog with no `DialogTitle` and unlabelled inputs still fails.
- Prefer real semantic elements and SSR over client-only interactions where the feature must work without JS. Validate on the server; Zod client validation is an enhancement.
- Keep the `focus-visible:ring` styles; never `outline-none` alone.

## The `cn()` utility and variants

Every component merges classes with `cn()` (clsx + tailwind-merge) and defines its variants with `cva` — see `styles.md`. When you consume a component, pass overrides via `className` (they merge safely) and prefer the component's `variant`/`size` props over ad-hoc classes.

## Consuming shadcn/ui from Svelte / Vue / other stacks

shadcn/ui itself is React. Community ports carry the same philosophy (copy-in, Tailwind tokens, `cn()`), built on the framework's own accessible-primitive library instead of React Radix:

- **Svelte:** **shadcn-svelte** (<https://www.shadcn-svelte.com/>) — same tokens/CLI feel, built on **Bits UI** / Melt UI (the Svelte equivalent of Radix). (Albitor's own web app uses Bits UI + Tailwind, which is this stack in Svelte form.)
- **Vue:** **shadcn-vue** (<https://www.shadcn-vue.com/>) — built on **Radix Vue / Reka UI**.

For any port, keep the contract that matters: **use the accessible primitive** (Bits UI / Radix Vue) rather than hand-rolling, theme through the CSS-variable tokens, and merge classes with `cn()`. The component names and token classes mirror the React set, so `components.md`/`styles.md`/`patterns.md` still apply — only the import paths and the primitive library change.

If you are hand-authoring plain HTML (no framework), you lose the Radix behaviour entirely — you must supply the keyboard handling, focus management, and ARIA yourself, which is exactly what these primitives exist to prevent. Prefer a supported stack.

## Coding in the open

shadcn/ui, Radix UI, and Tailwind CSS are all MIT-licensed open source. Because you copy the code in, your components are yours under your project's licence — keep secrets and personal data out of the repo, and attribute per the NOTICE where you redistribute the bundled reference content.
