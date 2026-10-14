#!/usr/bin/env bash
#
# GDS plugin advisory hook: lightweight, high-confidence accessibility / GOV.UK
# checks on frontend files after they are written or edited.
#
# It NEVER blocks. It runs only on frontend file types, only flags issues with a
# low false-positive rate, and surfaces them to Claude as additional context so
# they can be fixed. A clean file produces no output. This is a nudge, not a
# substitute for `/gds:audit` or the accessibility-auditor agent.
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
  *.html|*.htm|*.njk|*.nunjucks|*.svelte|*.vue|*.jsx|*.tsx|*.erb|*.ejs|*.hbs|*.handlebars|*.twig|*.liquid|*.astro|*.php) ;;
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
  [ -n "$ln" ] && add "  - line ${ln}: positive tabindex (WCAG 2.4.3). Use only tabindex=\"0\" or \"-1\"; positive values break focus order."
done < <(grep -niE 'tabindex[[:space:]]*=[[:space:]]*["'\'']?[1-9]' "$file_path" 2>/dev/null | cut -d: -f1 | sed 's/$/:/')

# 2.1.1 Keyboard: click handlers on non-interactive elements (div/span) are not keyboard-operable.
while IFS=: read -r ln _; do
  [ -n "$ln" ] && add "  - line ${ln}: onclick on a <div>/<span> (WCAG 2.1.1). Use <button> (or <a> for navigation) so it is keyboard-operable."
done < <(grep -niE '<(div|span)\b[^>]*onclick' "$file_path" 2>/dev/null | cut -d: -f1 | sed 's/$/:/')

# 2.1.1 Keyboard: href="#" with a click handler is a fake button.
while IFS=: read -r ln _; do
  [ -n "$ln" ] && add "  - line ${ln}: href=\"#\" used as a button (WCAG 2.1.1). Use a real <button> for actions."
done < <(grep -niE 'href[[:space:]]*=[[:space:]]*["'\'']#["'\''][^>]*onclick' "$file_path" 2>/dev/null | cut -d: -f1 | sed 's/$/:/')

[ -n "$findings" ] || exit 0

context="GDS accessibility check flagged possible issues in $(basename "$file_path") — please review and fix where appropriate:
${findings}
This is an advisory, high-confidence subset only. For a full review run /gds:audit or use the accessibility-auditor agent."

jq -n --arg ctx "$context" \
  '{hookSpecificOutput: {hookEventName: "PostToolUse", additionalContext: $ctx}}'
exit 0
