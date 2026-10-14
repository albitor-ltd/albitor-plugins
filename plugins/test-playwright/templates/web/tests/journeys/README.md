# Journey specs

One Playwright spec per journey, named for it: `web/tests/journeys/<journey>.spec.ts`. The first
line declares the journey it checks, `// @journey <id>`.

Import `test` and `expect` from `../helpers/journey` (or `../helpers/seed`, which adds
`fixtureScope`), never from `@playwright/test`, and wrap each screen in `journeyStep`. Read
`web/tests/helpers/journey.ts` and the `test-playwright` skill for the rules.

Specs here are written before the screens exist and then pinned: builders make them pass and cannot
change them.
