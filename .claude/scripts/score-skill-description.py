#!/usr/bin/env python3
"""
Skill description format compliance scorer and dependency graph validator.

Scores SKILL.md descriptions on 5 structural criteria (deterministic, no LLM).
Also detects confusable skill pairs via Jaccard similarity and dependency cycles via DFS.

Note: This measures FORMAT COMPLIANCE (structure), not semantic effectiveness.
A/B testing for actual model routing accuracy is separate future work.
"""

from __future__ import annotations

import re
from dataclasses import dataclass, field


# ── Constants ────────────────────────────────────────────────────────────────

# Common action verbs that start well-structured skill descriptions.
ACTION_VERBS: frozenset[str] = frozenset({
    "add", "analyze", "answer", "apply", "automate",
    "build", "check", "configure", "create", "debug",
    "deploy", "design", "discover", "execute", "extract",
    "fix", "generate", "implement", "integrate", "manage",
    "monitor", "optimize", "orchestrate", "organize", "pack",
    "plan", "process", "research", "review", "run",
    "scan", "search", "set", "ship", "simplify",
    "stage", "style", "test", "track", "transform",
    "validate", "view", "visualize", "write",
})

# Regex: matches "Use for", "Use when", or broad "for " with trailing context.
_TRIGGER_RE = re.compile(r"\bUse\s+(for|when)\b", re.IGNORECASE)
_BROAD_FOR_RE = re.compile(r"\bfor\b.{8,}", re.IGNORECASE)

# Stop words excluded from Jaccard similarity to avoid inflating overlap.
_STOP_WORDS: frozenset[str] = frozenset({
    "a", "an", "and", "are", "as", "at", "be", "by", "do", "for",
    "from", "has", "have", "if", "in", "is", "it", "not", "of", "on",
    "or", "so", "the", "to", "up", "use", "via", "vs", "was", "we",
    "when", "with", "you", "your",
})

# Description quality template shown when a skill fails the gate.
TEMPLATE_HINT = (
    'Expected pattern: "{Action verb} {what}. '
    'Use for/when {2-3 use cases}. Supports {tech if relevant}."'
)

# Scoring weights (sum = 1.0).
W_LENGTH = 0.30
W_VERB = 0.10
W_TRIGGER = 0.25
W_USECASE = 0.20
W_BOUNDARY = 0.15

# Thresholds.
PASS_THRESHOLD = 0.6
CONFUSABLE_THRESHOLD = 0.80


# ── Score result ─────────────────────────────────────────────────────────────

@dataclass
class FormatScore:
    """Result of scoring a single skill description."""

    skill_name: str
    description: str
    length_score: float = 0.0
    verb_score: float = 0.0
    trigger_score: float = 0.0
    usecase_score: float = 0.0
    boundary_score: float = 1.0  # default pass; caller sets 0 if confusable
    issues: list[str] = field(default_factory=list)

    @property
    def total(self) -> float:
        return (
            self.length_score * W_LENGTH
            + self.verb_score * W_VERB
            + self.trigger_score * W_TRIGGER
            + self.usecase_score * W_USECASE
            + self.boundary_score * W_BOUNDARY
        )

    @property
    def passed(self) -> bool:
        return self.total >= PASS_THRESHOLD


# ── Scoring functions ────────────────────────────────────────────────────────

def _score_length(desc: str) -> tuple[float, list[str]]:
    """Score description length. Optimal: 80-200 chars."""
    n = len(desc)
    issues: list[str] = []
    if n < 20:
        issues.append(f"Too short ({n} chars, min 20)")
        return 0.0, issues
    if n < 80:
        issues.append(f"Short ({n} chars, target 80-200)")
        return 0.5, issues
    if n <= 200:
        return 1.0, issues
    if n <= 300:
        issues.append(f"Long ({n} chars, target 80-200)")
        return 0.8, issues
    issues.append(f"Very long ({n} chars, max ~300)")
    return 0.5, issues


def _score_verb(desc: str) -> tuple[float, list[str]]:
    """Score whether description starts with an action verb."""
    first_word = desc.split()[0].lower().rstrip(".,;:!") if desc.strip() else ""
    if first_word in ACTION_VERBS:
        return 1.0, []
    # Check second word too — some start with "ALWAYS" or similar.
    words = desc.split()
    if len(words) > 1 and words[1].lower().rstrip(".,;:!") in ACTION_VERBS:
        return 0.5, [f'Starts with "{words[0]}" not an action verb']
    return 0.0, [f'No action verb start (first word: "{first_word}")']


def _score_trigger(desc: str) -> tuple[float, list[str]]:
    """Score whether description contains trigger/use-case phrase."""
    if _TRIGGER_RE.search(desc):
        return 1.0, []
    if _BROAD_FOR_RE.search(desc):
        return 0.7, ["Has 'for' context but missing explicit 'Use for/when'"]
    return 0.0, ["Missing trigger phrase ('Use for/when ...')"]


def _score_usecases(desc: str) -> tuple[float, list[str]]:
    """Score number of use cases listed (comma-separated items)."""
    # Find text after trigger phrase or after first period.
    trigger_match = _TRIGGER_RE.search(desc)
    if trigger_match:
        after = desc[trigger_match.end():]
    else:
        dot = desc.find(".")
        after = desc[dot + 1:] if dot >= 0 else desc

    # Count comma-separated segments as proxy for use-case count.
    segments = [s.strip() for s in after.split(",") if s.strip()]
    n = len(segments)
    if n == 0:
        return 0.0, ["No use cases listed"]
    if n == 1:
        return 0.5, ["Only 1 use case (target 2-4)"]
    if n <= 4:
        return 1.0, []
    return 0.8, [f"{n} use cases (target 2-4, may be too many)"]


def score_description(name: str, description: str) -> FormatScore:
    """Score a skill description on 5 structural criteria. Deterministic."""
    result = FormatScore(skill_name=name, description=description)

    result.length_score, length_issues = _score_length(description)
    result.verb_score, verb_issues = _score_verb(description)
    result.trigger_score, trigger_issues = _score_trigger(description)
    result.usecase_score, usecase_issues = _score_usecases(description)
    # boundary_score set externally by check_confusable_pairs()

    result.issues = length_issues + verb_issues + trigger_issues + usecase_issues
    return result


# ── Confusable pair detection ────────────────────────────────────────────────

def _tokenize(text: str) -> set[str]:
    """Tokenize text into lowercase words, excluding stop words."""
    return {
        w for w in re.findall(r"[a-z0-9]+", text.lower())
        if w not in _STOP_WORDS and len(w) > 1
    }


def check_confusable_pairs(
    skills: list[dict],
) -> list[tuple[str, str, float]]:
    """Find skill pairs with description similarity above threshold.

    Uses Jaccard index on word tokens (excluding stop words).
    Returns list of (name_a, name_b, similarity).
    """
    pairs: list[tuple[str, str, float]] = []
    tokenized = [
        (str(s.get("name", "")), _tokenize(str(s.get("description", ""))))
        for s in skills
    ]
    for i in range(len(tokenized)):
        name_a, tokens_a = tokenized[i]
        if not tokens_a:
            continue
        for j in range(i + 1, len(tokenized)):
            name_b, tokens_b = tokenized[j]
            if not tokens_b:
                continue
            intersection = tokens_a & tokens_b
            union = tokens_a | tokens_b
            sim = len(intersection) / len(union) if union else 0.0
            if sim >= CONFUSABLE_THRESHOLD:
                pairs.append((name_a, name_b, sim))
    return pairs


# ── Dependency cycle detection ───────────────────────────────────────────────

def validate_dependency_graph(skills: list[dict]) -> list[str]:
    """Detect cycles in the skill dependency graph (requires field).

    Builds directed graph from requires arrays, runs DFS cycle detection.
    Returns list of error strings (empty = no cycles).
    """
    # Build adjacency list from requires fields.
    graph: dict[str, list[str]] = {}
    for skill in skills:
        name = str(skill.get("name", ""))
        requires = skill.get("requires", [])
        if isinstance(requires, list) and requires:
            graph[name] = [str(r) for r in requires]

    if not graph:
        return []

    errors: list[str] = []
    WHITE, GRAY, BLACK = 0, 1, 2
    color: dict[str, int] = {n: WHITE for n in graph}

    # Iterative DFS to avoid RecursionError on deep chains.
    for start in list(graph.keys()):
        if color.get(start, WHITE) != WHITE:
            continue
        stack: list[tuple[str, int]] = [(start, 0)]  # (node, neighbor_index)
        path: list[str] = []
        while stack:
            node, idx = stack.pop()
            if idx == 0:
                color[node] = GRAY
                path.append(node)
            neighbors = graph.get(node, [])
            if idx < len(neighbors):
                stack.append((node, idx + 1))
                neighbor = neighbors[idx]
                if neighbor not in color:
                    color[neighbor] = WHITE
                if color[neighbor] == GRAY:
                    cycle_start = path.index(neighbor)
                    cycle = path[cycle_start:] + [neighbor]
                    errors.append(f"Cycle: {' -> '.join(cycle)}")
                elif color[neighbor] == WHITE:
                    stack.append((neighbor, 0))
            else:
                path.pop()
                color[node] = BLACK

    return errors


# ── Report printing ──────────────────────────────────────────────────────────

def print_format_compliance_report(
    scores: list[FormatScore],
    confusable: list[tuple[str, str, float]],
    cycles: list[str],
) -> None:
    """Print format compliance report to stdout."""
    print("\n" + "=" * 70)
    print("FORMAT COMPLIANCE REPORT")
    print("(Structural format check — not semantic effectiveness)")
    print("=" * 70)

    # Sort by score ascending (worst first).
    sorted_scores = sorted(scores, key=lambda s: s.total)

    passed = sum(1 for s in scores if s.passed)
    failed = len(scores) - passed

    for s in sorted_scores:
        status = "[OK]" if s.passed else "[!!]"
        score_str = f"{s.total:.2f}"
        print(f"  {status} {score_str}  {s.skill_name}")
        if not s.passed and s.issues:
            for issue in s.issues[:3]:
                print(f"           {issue}")
            print(f"           {TEMPLATE_HINT}")

    print(f"\nSummary: {passed} passed, {failed} failed (threshold {PASS_THRESHOLD})")

    if confusable:
        print(f"\nConfusable Pairs ({len(confusable)}):")
        for name_a, name_b, sim in confusable:
            print(f"  [!] {name_a} <-> {name_b} (similarity {sim:.2f})")

    if cycles:
        print(f"\nDependency Cycles ({len(cycles)}):")
        for err in cycles:
            print(f"  [X] {err}")

    print("=" * 70)
