# Note Taxonomy

Use this reference when deciding where a note belongs.

## Folder Rules

Never create professional durable notes in vault root.

Recommended layout:

```text
<Domain>/
  <Domain> MOC.md
  Concepts/
    <Concept>.md
  Deep Dives/
    <Focused Deep Dive>.md
  <Topic-Slug>/
    <Deep Dive>.md
  Projects/
    <Project>.md
  Sources/
    <Source Note>.md
  Inbox/
    <Temporary Capture>.md
```

## Note Types

### concept-note

Reusable atomic concept.

Path:

```text
<Domain>/Concepts/<Concept>.md
```

Use for:

- definitions
- mental models
- reusable principles
- terms that many notes link to

Length:

```text
300-900 words. Split if longer.
```

### hub-note

Short navigation note for a broad topic.

Path:

```text
<Domain>/<Topic-Slug>/<Topic Overview>.md
```

Use for:

- executive summary
- mental model
- links to concept notes
- learning path
- topic map

Avoid deep explanations. Link out instead.

### literature-note

Source-based note.

Path:

```text
<Domain>/Sources/<Source Title>.md
```

Use for:

- articles
- papers
- books
- videos
- lectures

Must include `source`.

### project-note

Implementation or active project knowledge.

Path:

```text
<Domain>/Projects/<Project Name>.md
```

Use for:

- architecture decisions
- implementation notes
- project-specific lessons

### pipeline-note

End-to-end workflow explanation.

Path:

```text
<Domain>/<Topic-Slug>/<Pipeline Name>.md
```

Use for:

- "from X to Y" flows
- system diagrams
- architecture breakdowns

If pipeline is complex, keep the pipeline note as a map and split stages into concept notes.

### moc

Map of content.

Path:

```text
<Domain>/<Domain> MOC.md
```

Use for:

- curated navigation
- topic overview
- linking clusters of notes
