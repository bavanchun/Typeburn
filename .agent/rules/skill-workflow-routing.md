# Skill Workflow Routing

When orchestrating multi-step tasks, consider these workflow sequences. Skills are listed in typical execution order.

## Core Development Workflow

```
/ck:plan → /ck:cook → /ck:test → /ck:code-review → /ck:ship → /ck:journal
```

| User Intent | Suggested Start |
|-------------|----------------|
| "implement feature X", "build X", "add X" | `` then `` |
| "execute this plan" | ` <plan-path>` |
| "quick implementation" | ` --fast` |

## Bugfix Workflow

```
/ck:scout → /ck:debug → /ck:fix → /ck:test → /ck:code-review
```

| User Intent | Suggested Start |
|-------------|----------------|
| "X is broken", "error in X", "bug in X" | `` (auto-scouts internally) |
| "CI is failing", "tests broken" | ` --auto` |
| "investigate why X happens" | `` then `` |

## Investigation Workflow

```
/ck:scout → /ck:debug → /ck:brainstorm → /ck:plan
```

| User Intent | Suggested Start |
|-------------|----------------|
| "understand how X works" | `` |
| "why is X happening" | `` |
| "explore options for X" | `` then `` |

## Post-Implementation Checklist

After completing implementation work, consider:
- `` — review changes before merging
- `` — run full shipping pipeline (tests, review, version, PR)
- `` — document decisions and lessons learned

## Setup Skills

Before starting implementation in a shared codebase:
- `` — create isolated worktree for the feature/fix
- `` — discover relevant files and code patterns