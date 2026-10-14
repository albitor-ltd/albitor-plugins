#!/usr/bin/env bash
#
# shadcn plugin advisory hook: lightweight, high-confidence accessibility and
# shadcn/ui conformance checks on frontend files after they are written or edited.
#
# It NEVER blocks. It runs only on frontend file types, only flags issues with a
# low false-positive rate, and surfaces them to Claude as additional context so
# they can be fixed. A clean file produces no output. This is a nudge, not a
# substitute for /shadcn:audit, /shadcn:review, or the shadcn-accessibility-auditor
# agent.
#
# Reads the PostToolUse payload as JSON on stdin.

set -u

# jq is required to parse the hook payload; if it's missing, do nothing.
command -v jq >/dev/null 2>&1 || exit 0

payload="$(cat)"
file_path="$(printf '%s' "$payload" | jq -r '.tool_input.file_path // .tool_input.path // empty' 2>/dev/null)"

# Nothing to check, or file no longer present.
[ -n "$file_path" ] || exit 0
[ -f "$file_path" ] || exit 0

# Only inspect frontend template / component file types.
case "$file_path" in
  *.tsx|*.jsx|*.ts|*.js|*.svelte|*.vue|*.astro|*.html|*.htm|*.mdx) ;;
  *) exit 0 ;;
esac

findings=""
add() { findings="${findings}$1"$'\n'; }

# 1.1.1 Non-text Content: <img> without an alt attribute (alt="" is allowed for decorative images).
while IFS=: read -r ln _; do
  [ -n "$ln" ] && add "  - line ${ln}: <img> without an alt attribute (WCAG 1.1.1). Add descriptive alt text, or alt=\"\" if decorative."
done < <(grep -niE '<img\b' "$file_path" 2>/dev/null | grep -ivE 'alt[[:space:]]*=' | cut -d: -f1 | sed 's/$/:/')

# 3.1.1 Language of Page: <html> element with no lang attribute.
if grep -qiE '<html\b' "$file_path" 2>/dev/null && ! grep -iE '<html\b' "$file_path" 2>/dev/null | grep -qiE 'lang[[:space:]]*='; then
  add "  - <html> element has no lang attribute (WCAG 3.1.1). Add e.g. lang=\"en\"."
fi

# 2.4.3 / keyboard: positive tabindex disrupts natural focus order.
while IFS=: read -r ln _; do
  [ -n "$ln" ] && add "  - line ${ln}: positive tabindex (WCAG 2.4.3). Use only tabIndex 0 or -1; positive values break focus order."
done < <(grep -niE 'tab[iI]ndex[[:space:]]*=[[:space:]]*[{"'\'']?[1-9]' "$file_path" 2>/dev/null | cut -d: -f1 | sed 's/$/:/')

# 2.1.1 Keyboard: click handlers on non-interactive elements (div/span) are not keyboard-operable.
while IFS=: read -r ln _; do
  [ -n "$ln" ] && add "  - line ${ln}: onClick on a <div>/<span> (WCAG 2.1.1). Use shadcn <Button> (or an <a>/<Link> for navigation) so it is keyboard-operable."
done < <(grep -niE '<(div|span)\b[^>]*[oO]n[cC]lick' "$file_path" 2>/dev/null | cut -d: -f1 | sed 's/$/:/')

# --- shadcn/ui conformance nudges (React JSX only; skip the shadcn primitive
#     definitions themselves, which legitimately render raw elements). ---
case "$file_path" in
  *.tsx|*.jsx)
    case "$file_path" in
      # shadcn generates its primitives into components/ui/ — those files ARE the
      # raw <button>/<input> implementations, so don't nudge them.
      */components/ui/*|*/ui/*) ;;
      *)
        # Ad-hoc button where the shadcn <Button> primitive exists.
        while IFS=: read -r ln _; do
          [ -n "$ln" ] && add "  - line ${ln}: raw <button> element (shadcn conformance). Use the shadcn <Button> primitive, not an ad-hoc component — it carries the focus, variant, and accessibility baseline."
        done < <(grep -niE '<button[[:space:]>]' "$file_path" 2>/dev/null | cut -d: -f1 | sed 's/$/:/')

        # Ad-hoc form controls where shadcn primitives exist.
        while IFS=: read -r ln _; do
          [ -n "$ln" ] && add "  - line ${ln}: raw <input>/<select>/<textarea> (shadcn conformance). Use the shadcn <Input>/<Select>/<Textarea> primitive so labelling, focus rings, and Radix behaviour come for free."
        done < <(grep -niE '<(input|select|textarea)[[:space:]>/]' "$file_path" 2>/dev/null | cut -d: -f1 | sed 's/$/:/')
        ;;
    esac
    ;;
esac

[ -n "$findings" ] || exit 0

context="shadcn accessibility/conformance check flagged possible issues in $(basename "$file_path") — please review and fix where appropriate:
${findings}
This is an advisory, high-confidence subset only. For a full review run /shadcn:audit or /shadcn:review, or use the shadcn-accessibility-auditor agent."

jq -n --arg ctx "$context" \
  '{hookSpecificOutput: {hookEventName: "PostToolUse", additionalContext: $ctx}}'
exit 0
