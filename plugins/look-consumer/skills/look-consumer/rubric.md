# Consumer look rubric

Checks for a visual review of a built page against this look. Each line is a yes or no; a no is a
finding with the file to fix. The reasons are in `SKILL.md`.

## Every page

- The heading font is loaded and heavy (headings are drawn in the rounded heading face, not the
  fallback), in sentence case with plain, short labels.
- The page colour comes from `look.css`: a warm white page with white cards, not a flat white set
  in the app's own CSS.
- The berry accent marks the main action and links; the second colour appears in the data and on
  small highlights, not only in the nav.
- Where the app has navigation, it has an icon beside each destination and a fully rounded current
  item with a visible label, whatever shell the app picked.
- Corners are large (24px on cards and sheets), cards are raised with a soft shadow, and there is
  no hard outline or offset shadow.
- Density is spacious: a few things at a time, one column, targets at least 48px.

## Screens with figures or a next step

- Where a screen opens with a heading band, it carries the look's art, sized down on a phone.
- The one thing to do next is one focus card with a progress ring or bar, not a grid of equal boxes.
- Figures are friendly tiles with an icon, not bare numbers.
- On a phone the main action sits within thumb reach, above any tab bar.
- A status is a pill with a glyph and a word, never colour alone.

## First run and empty states

- A first-run page has a designed state: the look's art used for first-run states, one plain
  sentence and one large action.
- An error or an empty search result has no art, only a sentence and a way forward.
