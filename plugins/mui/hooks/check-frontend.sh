#!/usr/bin/env bash
#
# MUI plugin advisory hook: lightweight, high-confidence accessibility and
# Material UI usage checks on frontend files after they are written or edited.
#
# It NEVER blocks. It runs only on frontend file types, only flags issues with a
# low false-positive rate, and surfaces them to Claude as additional context so
# they can be fixed. A clean file produces no output. This is a nudge, not a
# substitute for `/mui:audit` or the mui-accessibility-auditor agent.
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
  *.html|*.htm|*.svelte|*.vue|*.jsx|*.tsx|*.astro|*.mdx) ;;
  *) exit 0 ;;
esac

# Is this a React/JSX-style component file? MUI-component nudges only make sense
# there (MUI is a React library); raw HTML controls in a plain .html file are
# expected and correct.
is_jsx=0
case "$file_path" in
  *.jsx|*.tsx|*.mdx) is_jsx=1 ;;
esac

findings=""
add() { findings="${findings}$1"$'\n'; }

# --- WCAG checks (all frontend file types) ---------------------------------

# 1.1.1 Non-text Content: <img> without an alt attribute (alt="" is allowed for decorative images).
while IFS=: read -r ln _; do
  [ -n "$ln" ] && add "  - line ${ln}: <img> without an alt attribute (WCAG 1.1.1). Add descriptive alt text, or alt=\"\" if decorative. (MUI: an <Avatar> image also needs alt.)"
done < <(grep -niE '<img\b' "$file_path" 2>/dev/null | grep -ivE 'alt[[:space:]]*=' | cut -d: -f1 | sed 's/$/:/')

# 3.1.1 Language of Page: <html> element with no lang attribute.
if grep -qiE '<html\b' "$file_path" 2>/dev/null && ! grep -iE '<html\b' "$file_path" 2>/dev/null | grep -qiE 'lang[[:space:]]*='; then
  add "  - <html> element has no lang attribute (WCAG 3.1.1). Add e.g. lang=\"en\"."
fi

# 2.4.3 / keyboard: positive tabindex disrupts natural focus order.
while IFS=: read -r ln _; do
  [ -n "$ln" ] && add "  - line ${ln}: positive tabIndex (WCAG 2.4.3). Use only 0 or -1; positive values break focus order."
done < <(grep -niE 'tabindex[[:space:]]*=[[:space:]]*["'\''{]?[1-9]' "$file_path" 2>/dev/null | cut -d: -f1 | sed 's/$/:/')

# 2.1.1 Keyboard: click handlers on non-interactive elements (div/span) are not keyboard-operable.
while IFS=: read -r ln _; do
  [ -n "$ln" ] && add "  - line ${ln}: onClick on a <div>/<span> (WCAG 2.1.1). Use a real <button>/MUI <Button> (or <a>/MUI <Link> for navigation) so it is keyboard-operable."
done < <(grep -niE '<(div|span)\b[^>]*onclick' "$file_path" 2>/dev/null | cut -d: -f1 | sed 's/$/:/')

# 2.1.1 Keyboard: href="#" with a click handler is a fake button.
while IFS=: read -r ln _; do
  [ -n "$ln" ] && add "  - line ${ln}: href=\"#\" used as a button (WCAG 2.1.1). Use a real <button>/MUI <Button> for actions."
done < <(grep -niE 'href[[:space:]]*=[[:space:]]*["'\'']#["'\''][^>]*onclick' "$file_path" 2>/dev/null | cut -d: -f1 | sed 's/$/:/')

# --- MUI-conformance nudges (JSX/TSX only) ---------------------------------
# Lowercase raw HTML controls in a component file often mean a hand-rolled
# control where a vetted MUI component exists. MUI components are capitalised
# (<Button>, <TextField>...), so a lowercase tag is a reliable signal.
if [ "$is_jsx" -eq 1 ]; then
  while IFS=: read -r ln _; do
    [ -n "$ln" ] && add "  - line ${ln}: raw <button> — prefer MUI <Button> (contained/outlined/text) for consistent styling, states, focus and ripple."
  done < <(grep -niE '<button\b' "$file_path" 2>/dev/null | cut -d: -f1 | sed 's/$/:/')

  while IFS=: read -r ln _; do
    [ -n "$ln" ] && add "  - line ${ln}: raw <input> — prefer MUI <TextField> (or <Checkbox>/<Radio>/<Switch>) so the label, helper text and error state are wired up accessibly."
  done < <(grep -niE '<input\b' "$file_path" 2>/dev/null | cut -d: -f1 | sed 's/$/:/')

  while IFS=: read -r ln _; do
    [ -n "$ln" ] && add "  - line ${ln}: raw <select> — prefer MUI <Select> (with <MenuItem>) or <Autocomplete> for consistent behaviour and accessibility."
  done < <(grep -niE '<select\b' "$file_path" 2>/dev/null | cut -d: -f1 | sed 's/$/:/')

  while IFS=: read -r ln _; do
    [ -n "$ln" ] && add "  - line ${ln}: raw <textarea> — prefer MUI <TextField multiline> so labelling and styling match the rest of the form."
  done < <(grep -niE '<textarea\b' "$file_path" 2>/dev/null | cut -d: -f1 | sed 's/$/:/')
fi

[ -n "$findings" ] || exit 0

context="MUI accessibility / component check flagged possible issues in $(basename "$file_path") — please review and fix where appropriate:
${findings}
This is an advisory, high-confidence subset only. For a full review run /mui:audit or use the mui-accessibility-auditor agent."

jq -n --arg ctx "$context" \
  '{hookSpecificOutput: {hookEventName: "PostToolUse", additionalContext: $ctx}}'
exit 0
