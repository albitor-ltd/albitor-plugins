# Business look rubric

Checks for a visual review of a built page against this look. Each line is a yes or no; a no is a
finding with the file to fix. The reasons are in `SKILL.md`.

## Every page

- The type is the look's system stack at the small, exact sizes it sets; headings differ by size
  and weight, not by a second family.
- The page colour comes from `look.css`: a cool near-white page with navy-tinted text, not a colour
  set in the app's own CSS.
- The blue accent is on primary actions, links and the selected state only, never a large fill.
- The second colour is visible in the data (a selected-row tint, a bulk bar, a highlighted
  figure), not only in the nav.
- Where the app has navigation, it has an icon beside each destination and the compact styling
  (tint plus a 2px bar on the current item), whatever shell the app picked.
- Corners are 2px and there is no elevation: regions are separated by a 1px line or a step to the
  surface colour.
- Density is compact: rows and controls at the look's small target height, with columns added
  rather than type scaled up to fill space.

## Tables and numbers

- Numbers are right-aligned in tabular figures, and their column headers align the same way.
- A page of records is one dense table with its filters above it, not a stack of cards.
- A list that people work through has saved views and a search, and a bulk bar when rows can be
  selected.
- A selected row is a tint plus a 2px left bar, and the checkbox says it too.
- A status is a pill with a glyph and a word, never colour alone.
- A summary of headline figures, wherever it appears, is one bordered strip of exact values, not
  large decorated tiles.

## First run and empty states

- A first-run page has a designed state: the look's art used for first-run states, small and calm,
  one plain sentence and the one action that fixes it.
- Inside a dense table or a multi-column view the plain sentence alone is the empty state.
