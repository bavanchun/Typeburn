---
name: ck:agent-browser
description: Browser and desktop automation through the agent-browser CLI. Use for long autonomous browsing, compact page snapshots, screenshots, form filling, login flows, scraping, exploratory QA, cloud browsers, and Electron app automation when a live browser or app must be operated by the agent.
user-invocable: true
when_to_use: "Invoke for browser/app automation that needs snapshots or clicks."
category: dev-tools
keywords: [browser, automation, playwright, testing, e2e, browserbase, autonomous, headless, electron, slack, dogfood, agentcore, vercel-sandbox]
license: Apache-2.0
allowed-tools: Bash(agent-browser:*), Bash(npx agent-browser:*)
argument-hint: "[url or task]"
metadata:
  author: claudekit
  version: "2.0.0"
  upstream: "vercel-labs/agent-browser"
---

# agent-browser Skill

Fast browser automation CLI for AI agents. Chrome/Chromium via CDP with accessibility-tree snapshots and compact `@eN` element refs (~280 chars/snapshot vs 8K+ for Playwright MCP).

## Install / Upgrade

```bash
npm install -g agent-browser     # install (or upgrade) to latest
agent-browser install            # download Chromium (one-time)
agent-browser install --with-deps  # Linux: include system deps
agent-browser upgrade            # self-upgrade the binary
agent-browser --version          # verify
```

Re-run `npm install -g agent-browser` (or `agent-browser upgrade`) periodically — new commands and skills ship with the binary.

## Start here — load live workflow content

This file is a discovery stub, not the usage guide. Before running any `agent-browser` command, load workflow content from the installed CLI so it always matches your version:

```bash
agent-browser skills get core             # workflows, common patterns, troubleshooting
agent-browser skills get core --full      # full command reference + templates
agent-browser skills list                 # see everything available on this version
```

The CLI serves skill content from the installed binary, so instructions never go stale between releases. Prefer `skills get` over memorized command lists in this file.

## Specialized skills

Load when the task falls outside browser web pages:

```bash
agent-browser skills get electron          # Electron apps (VS Code, Slack, Discord, Figma, Notion, Spotify)
agent-browser skills get slack             # Slack workspace automation
agent-browser skills get dogfood           # Exploratory testing / QA / bug hunts
agent-browser skills get vercel-sandbox    # agent-browser inside Vercel Sandbox microVMs
agent-browser skills get agentcore         # AWS Bedrock AgentCore cloud browsers
```

## When to use

Default for any live-browser interaction — autonomous sessions, ad-hoc navigation, screenshots, form fills, scraping, multi-tab work, self-verifying build loops, Electron desktop apps, Slack automation.

For low-level Chrome DevTools Protocol diagnostics, use `chrome-devtools-mcp` via `/ck:use-mcp`. See `references/agent-browser-vs-chrome-devtools.md` for the trade-off.

## Cloud browsers

For CI/CD or environments without a local browser:

```bash
export BROWSERBASE_API_KEY="..."
export BROWSERBASE_PROJECT_ID="..."
agent-browser -p browserbase open https://example.com
```

See `references/browserbase-cloud-setup.md` for detailed setup. For AWS Bedrock AgentCore or Vercel Sandbox, run `agent-browser skills get agentcore` / `agent-browser skills get vercel-sandbox`.

## Troubleshooting

| Issue | Solution |
|-------|----------|
| Command not found | `npm install -g agent-browser` |
| Chromium missing | `agent-browser install` |
| Linux deps missing | `agent-browser install --with-deps` |
| Stale commands / missing flags | `npm install -g agent-browser` then `agent-browser skills get core --full` |
| Session stale | `agent-browser close` |
| Element not found | Re-run `agent-browser snapshot -i` after page changes |

## Resources

- Upstream: https://github.com/vercel-labs/agent-browser
- Browserbase: https://docs.browserbase.com/
