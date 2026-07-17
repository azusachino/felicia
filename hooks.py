"""MkDocs hooks for rendering structured page metadata."""

from __future__ import annotations

from collections.abc import Mapping, Sequence
from typing import Any


def _display(value: Any) -> str:
    if isinstance(value, Sequence) and not isinstance(value, (str, bytes)):
        return ", ".join(str(item) for item in value)
    return str(value)


def on_page_markdown(markdown: str, *, page: Any, **_: Any) -> str:
    """Render ADR frontmatter as a visible admonition above the page body."""

    meta: Mapping[str, Any] = page.meta
    if "id" not in meta or "status" not in meta:
        return markdown

    fields = (
        ("ID", "id"),
        ("Status", "status"),
        ("Date", "date"),
        ("Decisions", "decisions"),
        ("Related", "related"),
        ("Supersedes", "supersedes"),
    )
    lines = ["!!! info \"ADR metadata\""]
    for label, key in fields:
        value = meta.get(key)
        if value in (None, "", []):
            continue
        lines.append(f"    **{label}:** {_display(value)}")

    return "\n".join(lines) + "\n\n" + markdown
