---
name: obsidian-second-brain-note
description: Create professional Obsidian second-brain notes. Use when saving knowledge, summaries, concepts, lectures, images, or important info into an Obsidian vault with tags, links, MOCs, and clean folders.
version: "1.0.0"
---

# Obsidian Second-Brain Note

Use this skill to turn raw information into professional Obsidian notes that behave like a real second brain: atomic notes, clean folders, YAML properties, tags, aliases, backlinks, related concept notes, and map-of-content (MOC) links.

This skill handles: knowledge capture, lecture notes, concept notes, image-to-note summaries, technical learning notes, professional metadata, and graph-friendly linking.

This skill does NOT handle: private credential storage, secret management, legal/medical/financial advice as authority, or random root-vault dumping.

Core rule: keep notes readable. Do not create one giant wall-of-text note. Use a short hub note plus linked atomic notes when the topic is broad.

## Security Policy

Never write secrets, API keys, passwords, tokens, private keys, recovery phrases, or credentials into notes. If source material contains sensitive data, summarize safely and redact the secret. Do not invent source claims. Mark uncertainty explicitly.

## Workflow

1. **Find Vault Context**
   - Work from the Obsidian vault Git root when possible.
   - Inspect existing folder conventions before creating new folders.
   - Never place new durable notes in vault root unless user explicitly asks.

2. **Classify Note Type**
   - Use `concept-note` for reusable ideas.
   - Use `literature-note` for source/book/video/article notes.
   - Use `project-note` for implementation/project knowledge.
   - Use `moc` for index/map pages.
   - Use `daily-capture` only for temporary inbox content.

3. **Choose Professional Location**
   - Put domain notes under a domain folder, e.g. `AI/`, `Programming/`, `DevOps/`.
   - Put reusable concepts under `<Domain>/Concepts/`.
   - Put pipelines/deep dives under `<Domain>/<Topic-Slug>/`.
   - Put MOCs under `<Domain>/<Domain> MOC.md` or `<Domain>/MOCs/`.
   - If unclear, create an `Inbox/` note and mark `status: seedling`.

4. **Create YAML Frontmatter**
   - Include:
     - `title`
     - `created`
     - `updated`
     - `status`
     - `type`
     - `domain`
     - `aliases`
     - `tags`
     - `related`
     - `source` when applicable
   - Use `status: seedling | budding | evergreen`.
   - Use nested tags, e.g. `ai/speech`, `second-brain/concept`.

5. **Write Atomic Content**
   - One note should teach one durable idea.
   - Keep normal notes short enough to read without fatigue.
   - Target 300-900 words for concept notes.
   - If content grows beyond one focused idea, split it into linked notes.
   - Use concise sections:
     - `Definition`
     - `Why It Matters`
     - `Mental Model`
     - `Key Components`
     - `Workflow`
     - `Failure Modes`
     - `Best Practices`
     - `Related Notes`
   - For lecture-like output, use examples and practical checklists.

6. **Split Long Content Professionally**
   - Create a hub note when source material covers many concepts.
   - Hub note should include executive summary, mental model, map of linked notes, key takeaways, and next learning path.
   - Move deep detail into `Concepts/`, `Deep Dives/`, `Sources/`, or `Projects/`.
   - Link detail notes from the hub instead of making readers scroll forever.

7. **Build Neural Links**
   - Add `related:` YAML links using full Obsidian paths with aliases:
     - `[[AI/Concepts/Voice Cloning|Voice Cloning]]`
   - Add visible body links under `Related Notes`.
   - Create missing concept notes only when useful and not too many.
   - Link both directions where practical.
   - Prefer explicit path links to avoid duplicate ambiguous notes.

8. **Create or Update MOC**
   - If a domain has many related notes, create/update a MOC.
   - MOC should group notes by concept, pipeline, project, and safety.
   - Keep MOC short, navigational, and curated.

9. **Quality Check**
   - Confirm no durable note was created in root vault.
   - Confirm no note is too long for its purpose.
   - Confirm broad topics are split into linked docs.
   - Confirm tags are meaningful, not spam.
   - Confirm links resolve to existing files or intentional future notes.
   - Confirm note can stand alone without chat context.
   - Check Git diff before final answer.

## Output Rules

When finished, report:

- files created/updated
- folder structure
- key tags added
- unresolved links, if any
- whether changes were committed, only if user asked

## References

Load only as needed:

- `references/note-taxonomy.md` for note types and folder placement
- `references/frontmatter-schema.md` for YAML schema
- `references/linking-rules.md` for backlinks, aliases, MOCs
- `references/quality-checklist.md` for final review
