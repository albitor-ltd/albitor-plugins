# Albitor plugins

The marketplace catalogue for [Albitor Ltd](https://github.com/albitor-ltd)'s [Claude Code](https://code.claude.com/docs/en/plugins) plugins.

Each plugin lives in its own `albitor-plugin-*` repository; this repo's [`marketplace.json`](.claude-plugin/marketplace.json) lists them, pinned to a released version.

## Install

```bash
claude plugin marketplace add albitor-ltd/albitor-plugins
claude plugin install gds@albitor-plugins
```

Then enable/list/update from the Claude Code plugin UI or CLI:

```bash
claude plugin list
claude plugin enable gds@albitor-plugins
```

## Plugins

| Plugin | What it does | Source |
| --- | --- | --- |
| `gds` | GOV.UK Design System + WCAG 2.2 AA accessibility — skills, commands (`/gds:component`, `/gds:audit`, `/gds:accessibility-statement`, `/gds:review`), an accessibility-audit agent, and an advisory frontend hook. Needs `jq` for the hook. | [`albitor-plugin-gds`](https://github.com/albitor-ltd/albitor-plugin-gds) |

## How versions work

Plugins are referenced by **github source pinned to a release tag** (e.g. `v0.1.0`). To ship an update: tag a new release in the plugin's repo, then bump the `ref` (and/or `version`) for that entry in `marketplace.json`. Until the pinned `ref` moves, installs stay reproducible.

See the Albitor docs for the full workflow:
- How-to: *Publish a plugin to the marketplace*
- Reference: *Plugin governance model*
- Explanation: *Plugin distribution and governance*

## Air-gapped / offline use

Internet-connected installs use the command above. Air-gapped or regulated deployments are handled per engagement: clone the specific plugin repos needed, assemble a local marketplace (relative-path sources) or load via `claude --plugin-dir`, and sign as required.

## Licence

This catalogue (the `marketplace.json` and docs) is MIT — see [`LICENCE`](LICENCE). Each plugin carries its own licence; bundled standards content (e.g. GOV.UK / WCAG material in `gds`) is redistributed under the Open Government Licence v3.0 and the W3C Document Licence — see that plugin's `NOTICE`.
