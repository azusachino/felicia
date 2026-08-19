#!/usr/bin/env python3
"""Verify the fork-safety invariants of the GitHub Pages workflow."""

from __future__ import annotations

from pathlib import Path


ROOT = Path(__file__).resolve().parent.parent
WORKFLOW = ROOT / ".github" / "workflows" / "pages.yml"


def main() -> None:
    workflow = WORKFLOW.read_text(encoding="utf-8")
    required = (
        'branches: [main]',
        "workflow_dispatch:",
        "contents: read",
        "pages: write",
        "id-token: write",
        "github.event.repository.name",
        "actions/checkout@v7",
        "cachix/install-nix-action@v31",
        "nix develop --command bun install --frozen-lockfile",
        "nix develop --command uv run python scripts/felicia.py preview",
        "nix develop --command uv run python scripts/verify_static_artifact.py",
        "path: apps/felicia-public-site/dist",
        "actions/configure-pages@v6",
        "actions/upload-pages-artifact@v5",
        "actions/deploy-pages@v5",
    )
    missing = [value for value in required if value not in workflow]
    assert not missing, f"workflow invariants missing: {missing}"
    assert "azusachino/felicia" not in workflow, "workflow contains owner-specific repository data"
    assert "localhost" not in workflow, "workflow contains local-only host data"
    print(f"Pages workflow verified: {WORKFLOW}")


if __name__ == "__main__":
    main()
