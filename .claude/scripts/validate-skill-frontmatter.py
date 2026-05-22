#!/usr/bin/env python3
"""
Schema loading, category constants, and frontmatter validation for SKILL.md files.

Validates frontmatter dicts against claude/schemas/skill-schema.json.
Designed to be imported by scan_skills.py or called standalone.
"""

from __future__ import annotations

import json
import re
import sys
from pathlib import Path
from typing import Any

# ── Schema location ──────────────────────────────────────────────────────────

_SCHEMA_PATH = Path(__file__).resolve().parents[1] / "schemas" / "skill-schema.json"

# ── Category constants (shared with scan_skills.py) ──────────────────────────

CATEGORY_ORDER = [
    "ai-ml", "frontend", "backend", "infrastructure", "database",
    "dev-tools", "multimedia", "frameworks", "security", "utilities", "other",
]
CATEGORY_NAMES = {
    "ai-ml": "AI & Machine Learning",
    "frontend": "Frontend & Design",
    "backend": "Backend Development",
    "infrastructure": "Infrastructure & DevOps",
    "database": "Database & Storage",
    "dev-tools": "Development Tools",
    "multimedia": "Multimedia & Processing",
    "frameworks": "Frameworks & Platforms",
    "security": "Security & Intelligence",
    "utilities": "Utilities & Helpers",
    "other": "Other",
}

# Exact name→category mappings for high-signal CK skills (avoids falling into "other")
EXACT_CATEGORY_MAP: dict[str, str] = {
    "ask": "utilities", "bootstrap": "utilities", "brainstorm": "utilities",
    "ck-autoresearch": "utilities", "ck-debug": "utilities", "ck-loop": "utilities",
    "ck-predict": "utilities", "ck-scenario": "utilities", "code-review": "utilities",
    "coding-level": "utilities", "context-engineering": "utilities", "cook": "utilities",
    "copywriting": "utilities", "debug": "utilities", "docs": "utilities",
    "fix": "utilities", "journal": "utilities", "markdown-novel-viewer": "utilities",
    "mermaidjs-v11": "utilities", "plan": "utilities", "ck-plan": "utilities",
    "preview": "utilities", "problem-solving": "utilities",
    "project-management": "utilities", "project-organization": "utilities",
    "research": "utilities", "retro": "utilities", "sequential-thinking": "utilities",
    "test": "utilities", "watzup": "utilities",
    "find-skills": "dev-tools", "git": "dev-tools", "gkg": "dev-tools",
    "kanban": "dev-tools", "llms": "dev-tools", "mcp-builder": "dev-tools",
    "mintlify": "dev-tools", "plans-kanban": "dev-tools", "scout": "dev-tools",
    "ship": "dev-tools", "team": "dev-tools", "use-mcp": "dev-tools",
    "worktree": "dev-tools", "xia": "dev-tools",
    "react-best-practices": "frontend", "remotion": "frontend",
    "shader": "frontend", "stitch": "frontend", "web-design-guidelines": "frontend",
    "tanstack": "frameworks",
    "deploy": "infrastructure",
    "agent-browser": "multimedia", "web-testing": "multimedia",
    "ck-security": "security", "cti-expert": "security", "security-scan": "security",
}

# Category enum for fast validation without jsonschema dep
VALID_CATEGORIES = frozenset(CATEGORY_NAMES.keys())
VALID_MATURITIES = frozenset(["experimental", "beta", "stable"])
REQUIRES_RELATED_RE = re.compile(r"^[a-z0-9][a-z0-9:-]*$")


def load_schema() -> dict[str, Any]:
    """Load and return the skill JSON Schema. Raises FileNotFoundError if missing."""
    if not _SCHEMA_PATH.exists():
        raise FileNotFoundError(f"Skill schema not found: {_SCHEMA_PATH}")
    with _SCHEMA_PATH.open(encoding="utf-8") as fh:
        return json.load(fh)


def validate_frontmatter(frontmatter: dict[str, Any], skill_path: str = "") -> list[str]:
    """
    Validate frontmatter dict against skill schema rules.

    Returns a list of error strings (empty = valid).
    Does NOT require jsonschema package — validates manually against known rules.

    Args:
        frontmatter: Parsed frontmatter dict from extract_frontmatter().
        skill_path: Optional path string used in error messages.

    Returns:
        List of human-readable error strings. Empty list means valid.
    """
    errors: list[str] = []
    prefix = f"{skill_path}: " if skill_path else ""

    # ── Required fields ──────────────────────────────────────────────────────
    name = frontmatter.get("name")
    if not name:
        errors.append(f"{prefix}missing required field 'name'")
    elif not isinstance(name, str) or not (1 <= len(name) <= 100):
        errors.append(f"{prefix}'name' must be 1-100 chars, got {len(str(name))}")

    description = frontmatter.get("description")
    if not description:
        errors.append(f"{prefix}missing required field 'description'")
    elif not isinstance(description, str) or not (10 <= len(description) <= 512):
        errors.append(
            f"{prefix}'description' must be 10-512 chars, got {len(str(description))}"
        )

    # ── Required shipped-skill routing metadata ───────────────────────────────
    when_to_use = frontmatter.get("when_to_use")
    if not when_to_use:
        errors.append(f"{prefix}missing required field 'when_to_use'")
    elif not isinstance(when_to_use, str) or len(when_to_use) > 1024:
        errors.append(f"{prefix}'when_to_use' must be a string <= 1024 chars")

    # ── Optional scalar fields ────────────────────────────────────────────────
    argument_hint = frontmatter.get("argument-hint")
    if argument_hint is not None:
        if not isinstance(argument_hint, str) or len(argument_hint) > 200:
            errors.append(f"{prefix}'argument-hint' must be a string <= 200 chars")

    arguments = frontmatter.get("arguments")
    if arguments is not None:
        if isinstance(arguments, str):
            pass
        elif isinstance(arguments, list) and all(isinstance(item, str) for item in arguments):
            if len(arguments) > 20:
                errors.append(f"{prefix}'arguments' exceeds max 20 items ({len(arguments)} given)")
        else:
            errors.append(f"{prefix}'arguments' must be a string or string array")

    allowed_tools = frontmatter.get("allowed-tools")
    if allowed_tools is not None:
        if isinstance(allowed_tools, str):
            pass
        elif isinstance(allowed_tools, list) and all(isinstance(item, str) for item in allowed_tools):
            if len(allowed_tools) > 50:
                errors.append(
                    f"{prefix}'allowed-tools' exceeds max 50 items ({len(allowed_tools)} given)"
                )
        else:
            errors.append(f"{prefix}'allowed-tools' must be a string or string array")

    for field in ("model", "context", "agent", "shell"):
        value = frontmatter.get(field)
        if value is not None and not isinstance(value, str):
            errors.append(f"{prefix}'{field}' must be a string")

    effort = frontmatter.get("effort")
    if effort is not None and effort not in {"low", "medium", "high", "xhigh", "max"}:
        errors.append(f"{prefix}'effort' must be one of ['low', 'medium', 'high', 'xhigh', 'max']")

    paths = frontmatter.get("paths")
    if paths is not None:
        if isinstance(paths, str):
            pass
        elif isinstance(paths, list) and all(isinstance(item, str) for item in paths):
            if len(paths) > 50:
                errors.append(f"{prefix}'paths' exceeds max 50 items ({len(paths)} given)")
        else:
            errors.append(f"{prefix}'paths' must be a string or string array")

    hooks = frontmatter.get("hooks")
    if hooks is not None and not isinstance(hooks, dict):
        errors.append(f"{prefix}'hooks' must be an object")

    category = frontmatter.get("category")
    if category is not None:
        if category not in VALID_CATEGORIES:
            errors.append(
                f"{prefix}'category' must be one of {sorted(VALID_CATEGORIES)}, got '{category}'"
            )

    maturity = frontmatter.get("maturity")
    if maturity is not None:
        if maturity not in VALID_MATURITIES:
            errors.append(
                f"{prefix}'maturity' must be one of {sorted(VALID_MATURITIES)}, got '{maturity}'"
            )

    user_invocable = frontmatter.get("user-invocable")
    if user_invocable is not True:
        errors.append(f"{prefix}'user-invocable' must be true for shipped ClaudeKit skills")

    disable_model_invocation = frontmatter.get("disable-model-invocation")
    if disable_model_invocation is not None:
        if not isinstance(disable_model_invocation, bool):
            errors.append(f"{prefix}'disable-model-invocation' must be a boolean")
        elif disable_model_invocation:
            errors.append(
                f"{prefix}'disable-model-invocation' must be false for shipped ClaudeKit skills"
            )

    # ── Array fields ─────────────────────────────────────────────────────────
    keywords = frontmatter.get("keywords")
    if keywords is not None:
        if not isinstance(keywords, list):
            errors.append(f"{prefix}'keywords' must be an array")
        elif len(keywords) > 15:
            errors.append(f"{prefix}'keywords' exceeds max 15 items ({len(keywords)} given)")

    for field, max_items in [("requires", 10), ("related", 10)]:
        value = frontmatter.get(field)
        if value is None:
            continue
        if not isinstance(value, list):
            errors.append(f"{prefix}'{field}' must be an array")
            continue
        if len(value) > max_items:
            errors.append(f"{prefix}'{field}' exceeds max {max_items} items ({len(value)} given)")
        for item in value:
            if not isinstance(item, str) or not REQUIRES_RELATED_RE.match(item):
                errors.append(
                    f"{prefix}'{field}' items must match ^[a-z0-9][a-z0-9:-]*$, got '{item}'"
                )

    # ── Metadata block ────────────────────────────────────────────────────────
    metadata = frontmatter.get("metadata")
    if metadata is not None and not isinstance(metadata, dict):
        errors.append(f"{prefix}'metadata' must be an object")

    # ── Unknown fields (additionalProperties: false at top level) ────────────
    _KNOWN_KEYS = frozenset({
        "name", "description", "when_to_use", "argument-hint", "arguments", "license",
        "languages", "allowed-tools", "model", "effort", "context", "agent", "shell",
        "paths", "hooks", "category", "keywords", "requires", "related",
        "maturity", "metadata", "user-invocable", "disable-model-invocation",
    })
    unknown = set(frontmatter.keys()) - _KNOWN_KEYS
    if unknown:
        errors.append(f"{prefix}unknown fields not in schema: {sorted(unknown)}")

    return errors


def validate_skill_file(skill_path: Path, frontmatter: dict[str, Any]) -> bool:
    """
    Validate a skill file's frontmatter and print any errors to stderr.

    Args:
        skill_path: Path to the SKILL.md file.
        frontmatter: Parsed frontmatter dict.

    Returns:
        True if valid, False if errors found.
    """
    errors = validate_frontmatter(frontmatter, str(skill_path))
    for error in errors:
        print(f"[WARN] {error}", file=sys.stderr)
    return len(errors) == 0


def main() -> None:
    """Standalone validator: scan all SKILL.md files and report issues."""
    repo_root = Path(__file__).resolve().parents[2]
    base_path = repo_root / "claude" / "skills"
    if not base_path.exists():
        raise SystemExit(f"Error: {base_path} not found")

    # Lazy import to avoid circular dependency if used as module
    try:
        from scan_skills import extract_frontmatter  # type: ignore[import]
    except ImportError:
        print("Error: must run from the scripts directory or with scan_skills.py on sys.path",
              file=sys.stderr)
        raise SystemExit(1)

    total = 0
    invalid = 0
    for skill_file in sorted(base_path.rglob("SKILL.md")):
        if skill_file.is_symlink():
            continue
        content = skill_file.read_text(encoding="utf-8")
        fm = extract_frontmatter(content)
        rel = skill_file.relative_to(repo_root)
        is_valid = validate_skill_file(rel, fm)
        total += 1
        if not is_valid:
            invalid += 1

    print(f"\nValidated {total} skills: {total - invalid} valid, {invalid} with errors")
    if invalid:
        raise SystemExit(1)


if __name__ == "__main__":
    main()
