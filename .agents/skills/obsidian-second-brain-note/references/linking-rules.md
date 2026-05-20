# Linking Rules

Use links to make notes work as a second brain, not just storage.

Rule: prefer a readable network of focused notes over one long exhaustive note.

## Link Types

### YAML Related Links

Use for machine-readable relationships.

```yaml
related:
  - "[[AI/Concepts/Automatic Speech Recognition|Automatic Speech Recognition]]"
```

### Body Links

Use for visible reading flow.

```md
This concept connects to [[AI/Concepts/Voice Cloning|Voice Cloning]].
```

### Backlinks

When creating a new concept note for an existing main note, link back to the main note.

Example:

```md
## Related Notes

- [[AI/From-ASR-to-Emotion-to-voice-cloing/From ASR to Emotion to Voice Cloning|From ASR to Emotion to Voice Cloning]]
```

## MOC Rules

Create/update a MOC when a domain has 5+ related notes.

MOC structure:

```md
# AI MOC

## Speech AI

- [[AI/Concepts/Automatic Speech Recognition|Automatic Speech Recognition]]
- [[AI/Concepts/Text to Speech|Text to Speech]]

## Pipelines

- [[AI/From-ASR-to-Emotion-to-voice-cloing/From ASR to Emotion to Voice Cloning|From ASR to Emotion to Voice Cloning]]
```

## Hub and Detail Pattern

For broad topics:

```text
Hub note = overview, mental model, map, learning path.
Concept notes = reusable atomic ideas.
Deep dives = detailed explanations that would make the hub too long.
Source notes = article/video/book-specific extraction.
```

Example:

```md
# Speech AI Pipeline

## Summary

Short explanation.

## Map

- [[AI/Concepts/Automatic Speech Recognition|Automatic Speech Recognition]]
- [[AI/Concepts/Emotion Recognition|Emotion Recognition]]
- [[AI/Concepts/Text to Speech|Text to Speech]]

## Deep Dives

- [[AI/Deep Dives/ASR Evaluation Metrics|ASR Evaluation Metrics]]
```

## Duplicate Prevention

Before creating a note:

1. Search exact title.
2. Search aliases.
3. Search likely folder.
4. Reuse existing note if present.
5. If moving a root note, update links to path aliases.
