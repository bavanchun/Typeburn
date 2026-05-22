# Frontmatter Schema

Use YAML frontmatter for every durable Obsidian note.

## Standard Schema

```yaml
---
title: Note Title
created: YYYY-MM-DD
updated: YYYY-MM-DD
status: seedling
type: concept-note
domain: artificial-intelligence
aliases:
  - Short Name
tags:
  - ai
  - ai/topic
  - second-brain/concept
related:
  - "[[AI/Concepts/Example|Example]]"
source:
---
```

## Status

- `seedling`: new idea, incomplete
- `budding`: useful but still growing
- `evergreen`: polished durable note

## Type

Use one:

- `concept-note`
- `literature-note`
- `project-note`
- `pipeline-note`
- `moc`
- `daily-capture`

## Tags

Use nested tags:

```yaml
tags:
  - ai
  - ai/speech
  - ai/asr
  - second-brain/concept
```

Avoid:

```yaml
tags:
  - important
  - note
  - random
```

## Related Links

Prefer path links with aliases:

```yaml
related:
  - "[[AI/Concepts/Voice Cloning|Voice Cloning]]"
```

This prevents duplicate files and keeps root vault clean.

