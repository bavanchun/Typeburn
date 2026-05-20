#!/usr/bin/env python3
"""
Validate skill cross-references and workflow chain integrity.

Scans SKILL.md bodies for /ck: references, builds a directed graph,
checks expected workflow chains, reports orphans/hubs/broken refs.

Usage:
  python3 validate-skill-crossrefs.py <skills-dir>          # audit mode
  python3 validate-skill-crossrefs.py <skills-dir> --json    # JSON output
  python3 validate-skill-crossrefs.py --self-test             # run self-tests
"""
from __future__ import annotations

import json
import os
import re
import sys
import tempfile
from collections import defaultdict
from pathlib import Path

try:
    import yaml as _yaml
    _HAS_YAML = True
except ImportError:
    _HAS_YAML = False

# ── Constants ────────────────────────────────────────────────────────────────

FRONTMATTER_RE = re.compile(r"^---\s*\n(.*?)\n---\s*\n", re.DOTALL)
# Match /ck:skill-name but NOT inside code fences
CODE_FENCE_RE = re.compile(r"```[\s\S]*?```", re.MULTILINE)
SKILL_REF_RE = re.compile(r"/ck:([a-z0-9][\w-]*)")

SKIP_DIRS = frozenset({"_shared", "template-skill", "common", ".venv",
                        "node_modules", "__pycache__"})

# Expected workflow chains — consecutive pairs must have edges.
# Note: test and journal are intentionally omitted from chains because they're
# optional steps (--fast skips test, journal is terminal/post-hoc).
EXPECTED_CHAINS = {
    "development": ["ck-plan", "cook", "code-review", "ship"],
    "bugfix": ["scout", "ck-debug", "fix", "code-review"],
    "investigation": ["scout", "ck-debug", "brainstorm"],
}


# ── Scanning ─────────────────────────────────────────────────────────────────

def _parse_frontmatter(content: str) -> dict:
    """Extract frontmatter dict from SKILL.md content."""
    m = FRONTMATTER_RE.match(content)
    if not m:
        return {}
    raw = m.group(1)
    if _HAS_YAML:
        try:
            return _yaml.safe_load(raw) or {}
        except Exception:
            return {}
    # Minimal fallback: extract name only
    nm = re.search(r"^name:\s*[\"']?(.+?)[\"']?\s*$", raw, re.M)
    return {"name": nm.group(1).strip() if nm else ""}


def _extract_body_refs(content: str, own_name: str) -> set[str]:
    """Extract /ck: skill references from body text, excluding code fences."""
    m = FRONTMATTER_RE.match(content)
    body = content[m.end():] if m else content
    # Remove code fences to avoid false positives
    body = CODE_FENCE_RE.sub("", body)
    refs = set(SKILL_REF_RE.findall(body))
    # Normalize: strip ck: prefix if present in the captured group
    # The regex captures what's after /ck: so "plan" from "/ck:plan"
    refs.discard(own_name)
    # Also discard dir-name form of own name (e.g. "ck-plan" vs "plan")
    refs.discard(own_name.replace("ck:", ""))
    return refs


def scan_all_skills(skills_dir: Path) -> dict[str, dict]:
    """Scan all SKILL.md files, return {dir_name: {name, refs, requires, related}}."""
    skills = {}
    for entry in sorted(skills_dir.iterdir()):
        if not entry.is_dir() or entry.name in SKIP_DIRS or entry.name.startswith("."):
            continue
        skill_file = entry / "SKILL.md"
        if not skill_file.exists():
            continue
        content = skill_file.read_text(encoding="utf-8", errors="replace")
        fm = _parse_frontmatter(content)
        name = fm.get("name", entry.name) or entry.name
        # Normalize dir name for graph node
        dir_name = entry.name
        body_refs = _extract_body_refs(content, name)
        requires = fm.get("requires") or []
        related = fm.get("related") or []
        if isinstance(requires, str):
            requires = [requires]
        if isinstance(related, str):
            related = [related]
        skills[dir_name] = {
            "name": name,
            "body_refs": body_refs,
            "requires": requires,
            "related": related,
        }
    return skills


# ── Graph Analysis ───────────────────────────────────────────────────────────

def build_reference_graph(skills: dict[str, dict]) -> dict:
    """Build directed graph from skill cross-references."""
    all_dirs = set(skills.keys())
    edges = defaultdict(set)       # src → {dst, ...}
    in_degree = defaultdict(int)
    out_degree = defaultdict(int)

    for dir_name, data in skills.items():
        for ref in data["body_refs"]:
            # Resolve ref to dir name (refs are like "cook", "ck-plan", etc.)
            target = ref if ref in all_dirs else None
            if not target:
                # Try with ck- prefix removed or added
                for candidate in [ref, f"ck-{ref}", ref.replace("ck-", "")]:
                    if candidate in all_dirs:
                        target = candidate
                        break
            if target and target != dir_name:
                edges[dir_name].add(target)
                out_degree[dir_name] += 1
                in_degree[target] += 1

    # Identify orphans (no body refs in or out) and hubs (in_degree >= 3)
    orphans = [d for d in all_dirs if in_degree[d] == 0 and out_degree[d] == 0]
    hubs = [(d, in_degree[d]) for d in all_dirs if in_degree[d] >= 3]
    hubs.sort(key=lambda x: -x[1])

    # Find broken refs (refs to nonexistent skills)
    broken = []
    for dir_name, data in skills.items():
        for ref in data["body_refs"]:
            found = ref in all_dirs or f"ck-{ref}" in all_dirs or ref.replace("ck-", "") in all_dirs
            if not found:
                broken.append((dir_name, ref))

    return {
        "edges": {k: sorted(v) for k, v in edges.items()},
        "orphans": sorted(orphans),
        "hubs": hubs,
        "broken": broken,
        "in_degree": dict(in_degree),
        "out_degree": dict(out_degree),
    }


def check_expected_workflows(graph: dict, skills: dict) -> list[dict]:
    """Check expected workflow chains for missing consecutive edges."""
    edges = graph["edges"]
    all_dirs = set(skills.keys())
    missing = []
    for chain_name, chain in EXPECTED_CHAINS.items():
        for i in range(len(chain) - 1):
            src, dst = chain[i], chain[i + 1]
            # Check if edge exists in either direction
            fwd = dst in edges.get(src, set())
            rev = src in edges.get(dst, set())
            if not fwd and not rev:
                missing.append({"chain": chain_name, "from": src, "to": dst})
    return missing


# ── Output ───────────────────────────────────────────────────────────────────

def print_report(skills: dict, graph: dict, missing: list[dict]):
    """Print human-readable audit report."""
    print(f"\n=== Skill Cross-Reference Audit ===")
    print(f"Total skills scanned: {len(skills)}")
    print(f"Skills with outward refs: {len(graph['edges'])}")
    print(f"Orphaned skills: {len(graph['orphans'])}")
    print(f"Hub skills: {len(graph['hubs'])}")
    print(f"Broken references: {len(graph['broken'])}")
    print(f"Missing workflow edges: {len(missing)}")

    if graph["hubs"]:
        print(f"\n--- Hub Skills (in_degree >= 3) ---")
        for name, count in graph["hubs"]:
            print(f"  {name}: {count} inbound refs")

    if missing:
        print(f"\n--- Missing Workflow Edges ---")
        for m in missing:
            print(f"  [{m['chain']}] {m['from']} → {m['to']}")

    if graph["broken"]:
        print(f"\n--- Broken References ---")
        for src, ref in graph["broken"]:
            print(f"  {src} → /ck:{ref} (not found)")

    if graph["orphans"]:
        print(f"\n--- Orphaned Skills ({len(graph['orphans'])}) ---")
        for name in graph["orphans"]:
            print(f"  {name}")


# ── Self-Tests ───────────────────────────────────────────────────────────────

def _run_self_tests():
    """Run 8 self-tests with temp SKILL.md fixtures."""
    passed = 0
    failed = 0

    def _assert(cond, label):
        nonlocal passed, failed
        if cond:
            passed += 1
            print(f"  [OK] {label}")
        else:
            failed += 1
            print(f"  [X]  {label}")

    with tempfile.TemporaryDirectory() as tmp:
        base = Path(tmp)
        # Create fixture skills
        fixtures = {
            "cook": '---\nname: ck:cook\ndescription: "Cook things"\ncategory: utilities\nkeywords: [impl]\n---\n# Cook\nAfter planning, run /ck:scout then /ck:code-review.\n',
            "plan": '---\nname: ck:plan\ndescription: "Plan things"\ncategory: utilities\nkeywords: [plan]\n---\n# Plan\nThis skill creates plans.\n',
            "scout": '---\nname: ck:scout\ndescription: "Scout"\ncategory: dev-tools\nkeywords: [scout]\nrequires: [ck:scout]\n---\n# Scout\nExplore code. See /ck:cook for next step.\n```\n/ck:plan should not match inside fence\n```\n',
            "orphan-skill": '---\nname: ck:orphan\ndescription: "Orphan"\ncategory: other\nkeywords: []\n---\n# Orphan\nNo refs here.\n',
            "broken-ref": '---\nname: ck:broken\ndescription: "Broken"\ncategory: other\nkeywords: []\n---\n# Broken\nSee /ck:nonexistent for help.\nAlso /ck:cook is good.\n',
        }
        for dname, content in fixtures.items():
            d = base / dname
            d.mkdir()
            (d / "SKILL.md").write_text(content)

        skills = scan_all_skills(base)
        graph = build_reference_graph(skills)

        print("Self-tests:")
        # T1: Detects /ck:scout in cook's body
        _assert("scout" in graph["edges"].get("cook", []), "T1: Detects /ck:scout ref in body")
        # T2: cook→code-review edge exists
        _assert("code-review" in skills["cook"]["body_refs"] or "code-review" in graph["edges"].get("cook", []),
                "T2: Detects /ck:code-review ref in body")
        # T3: orphan-skill has no refs
        _assert("orphan-skill" in graph["orphans"], "T3: Orphan detection")
        # T4: cook is referenced by scout and broken-ref → check if hub-like
        cook_in = graph["in_degree"].get("cook", 0)
        _assert(cook_in >= 2, f"T4: Hub-like detection (cook in_degree={cook_in})")
        # T5: Missing chain edge detection
        test_chains_bak = EXPECTED_CHAINS.copy()
        missing = check_expected_workflows(graph, skills)
        _assert(len(missing) > 0, "T5: Missing chain edges detected")
        # T6: Broken reference detection
        broken_refs = [r for s, r in graph["broken"] if r == "nonexistent"]
        _assert(len(broken_refs) > 0, "T6: Broken reference /ck:nonexistent detected")
        # T7: Self-reference ignored (scout mentions itself via requires)
        _assert("scout" not in graph["edges"].get("scout", []), "T7: Self-reference ignored")
        # T8: Code fence exclusion (scout has /ck:plan in code fence — should NOT count)
        scout_refs = skills["scout"]["body_refs"]
        _assert("plan" not in scout_refs, "T8: Code fence refs excluded")

        print(f"\nResults: {passed}/8 passed, {failed}/8 failed")
        return failed == 0


# ── Main ─────────────────────────────────────────────────────────────────────

def main():
    if "--self-test" in sys.argv:
        ok = _run_self_tests()
        sys.exit(0 if ok else 1)

    if len(sys.argv) < 2:
        print("Usage: validate-skill-crossrefs.py <skills-dir> [--json]")
        sys.exit(2)

    skills_dir = Path(sys.argv[1])
    if not skills_dir.is_dir():
        print(f"Error: {skills_dir} is not a directory")
        sys.exit(2)

    use_json = "--json" in sys.argv
    skills = scan_all_skills(skills_dir)
    graph = build_reference_graph(skills)
    missing = check_expected_workflows(graph, skills)

    if use_json:
        result = {
            "total_skills": len(skills),
            "edges": graph["edges"],
            "orphans": graph["orphans"],
            "hubs": [{"name": n, "in_degree": c} for n, c in graph["hubs"]],
            "broken": [{"from": s, "ref": r} for s, r in graph["broken"]],
            "missing_workflow_edges": missing,
        }
        print(json.dumps(result, indent=2))
    else:
        print_report(skills, graph, missing)

    # Exit 1 if there are missing workflow edges (gaps to fix)
    sys.exit(1 if missing else 0)


if __name__ == "__main__":
    main()
