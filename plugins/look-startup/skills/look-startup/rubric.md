# Startup look rubric

Checks for a visual review of a built page against this look. Each line is a yes or no; a no is a
finding with the file to fix. The reasons are in `SKILL.md`.

## Every page

- The heading font is loaded (headings are drawn in the heading face, not the fallback), with the
  tight tracking and semibold weight the look sets.
- The page colour comes from `look.css`: a near-neutral page with white surfaces, not pure white
  everywhere and not a colour set in the app's own CSS.
- The teal accent is on actions, marks and links only; it is not a large fill.
- The second colour is visible in the data (a meter, a sparkline, a highlight card, a tinted
  current item in the nav), not only in the nav.
- Where the app has navigation, it has an icon beside each destination and the look's tinted
  current item, whatever shell the app picked; on a phone every destination is reachable.
- Surfaces are layered: a 1px hairline and a soft shadow on raised cards, no heavy outline, no
  glow, no offset shadow.
- Density is comfortable: generous space between groups, rows that are airy rather than packed.

## An overview or a list

- Headline figures, wherever they appear, carry an icon and a change in words, with a meter or a
  sparkline where a trend or a share exists, not bare number boxes.
- The page has one primary action, in the header beside the title.
- A status is a pill with a glyph and a word, never colour alone.

## First run and empty states

- A first-run page has a designed state: the look's art used for first-run states, one plain
  sentence and the one action that fixes it.
- An error or an empty search result has no art, only a sentence and a way forward.
