# agent-browser and Browser MCP

Guidance for choosing between `agent-browser` and MCP-based browser diagnostics.

## Use Case Decision Tree

```
Need browser automation?
|
+-- Long autonomous AI session?
|   +-- YES --> agent-browser (better context efficiency)
|   +-- NO --> Continue
|
+-- Need video recording?
|   +-- YES --> agent-browser (built-in)
|   +-- NO --> Continue
|
+-- Cloud browser (CI/CD)?
|   +-- YES --> agent-browser (Browserbase native)
|   +-- NO --> Continue
|
+-- Low-level Chrome DevTools Protocol inspection?
|   +-- YES --> chrome-devtools-mcp through ck:use-mcp
|   +-- NO --> Continue
|
+-- Ad-hoc page driving, snapshots, screenshots, forms?
|   +-- YES --> agent-browser
+-- Otherwise --> ck:web-testing for test strategy/runners
```

## Primary Patterns

```bash
# Long autonomous session
agent-browser --session test1 open https://example.com
agent-browser snapshot -i
agent-browser click @e1
agent-browser close
```

## MCP Pattern

Use `ck:use-mcp` when the project has `chrome-devtools-mcp` configured and the task specifically needs MCP/CDP tools. Claude Code Tool Search defer-loads that MCP server, so the local skill fallback is no longer shipped.

## Migration Notes

| Old local script habit | Current route |
|------------------------|---------------|
| `node navigate.js --url X` | `agent-browser open X` |
| `node aria-snapshot.js --url X` | `agent-browser open X && agent-browser snapshot -i` |
| `node select-ref.js --ref e5 --action click` | `agent-browser click @e5` |
| `node fill.js --selector "#email" --value "X"` | `agent-browser fill @e1 "X"` |
| `node screenshot.js --output X.png` | `agent-browser screenshot -o X.png` |
| `node console.js --types error` | `ck:use-mcp` with browser console tools, or a project-local Playwright test |
| `node network.js` | `ck:use-mcp` with network tools, or a project-local Playwright test |
