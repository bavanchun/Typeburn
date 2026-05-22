# UI Testing Workflow

Use `ck:agent-browser` for live browser interaction and `ck:web-testing` or project-native Playwright/Vitest/k6 commands for repeatable test runs.

## Purpose
Run comprehensive UI tests on a website and generate a detailed report.

## Arguments
- $1: URL - The URL of the website to test
- $2: OPTIONS - Optional test configuration (e.g., --headless, --mobile, --auth)

## Testing Protected Routes (Authentication)

### Step 1: User Manual Login
Instruct the user to:
1. Open the target site in their browser
2. Log in manually with their credentials
3. Open browser DevTools (F12) → Application tab → Cookies/Storage

### Step 2: Persist Auth State
Prefer project-native auth helpers. For ad-hoc browser driving, use `agent-browser` state commands after manual login when available.

### Step 3: Run Tests
After auth is available, run tests normally:
```bash
agent-browser open https://example.com/dashboard
agent-browser screenshot -o profile.png
```

## Workflow
- Use `ck:plan` skill to organize the test plan & report
- All screenshots saved in the same report directory
- Browse URL, discover all pages, components, endpoints
- Create test plan based on discovered structure
- Use multiple `tester` subagents in parallel for: pages, forms, navigation, user flows, accessibility, responsive layouts, performance, security, seo
- Use `ck:ai-multimodal` to analyze all screenshots
- Generate comprehensive Markdown report
- Ask user if they want to preview with `/ck:preview`

## Output Requirements
- Clear, structured Markdown with headers, lists, code blocks
- Include test results summary, key findings, screenshot references
- Ensure token efficiency while maintaining high quality
- Sacrifice grammar for concision

**Do not** start implementing fixes.
