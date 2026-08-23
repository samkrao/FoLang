from __future__ import annotations

"""
Shared cover + body PDF merge.

Chromium records every internal Markdown anchor link (`[text](#heading)`) as a
link annotation whose `/Dest` is a NAMED destination, resolved through the
document catalog's `/Dests` dictionary. The table of contents and the outline
(`/Outlines`) are stored the same way.

Copying pages one at a time into a fresh PdfWriter carries the annotations but
NOT the catalog that gives their names meaning, so every internal link in the
merged file points at nothing and the reader refuses to follow it. Only the
handful of absolute `/URI` links keep working, because those are self-contained.

The body PDF is therefore cloned whole - catalog included - and the cover pages
are inserted in front of it. Named destinations hold indirect page references,
not page numbers, so prepending pages leaves them pointing at the right pages.
"""

from pathlib import Path

from pypdf import PdfReader, PdfWriter


def _internal_link_targets(reader: PdfReader) -> list[str]:
    """Collect the named destination each internal link annotation refers to."""
    targets: list[str] = []

    for page in reader.pages:
        for annotation_reference in page.get("/Annots") or []:
            annotation = annotation_reference.get_object()

            if annotation.get("/Subtype") != "/Link":
                continue

            destination = annotation.get("/Dest")

            if destination is None:
                action = annotation.get("/A")

                if action is not None and action.get("/S") == "/GoTo":
                    destination = action.get("/D")

            if destination is not None:
                targets.append(str(destination).lstrip("/"))

    return targets


def verify_links(output: Path) -> None:
    """
    Fail when an internal link in the merged PDF resolves to nothing.

    A dangling link is silent in every PDF reader: the text still looks like a
    link and simply does nothing when clicked, so it has to be checked here.
    """
    reader = PdfReader(str(output), strict=False)

    known_destinations = {
        name.lstrip("/") for name in reader.named_destinations
    }

    targets = _internal_link_targets(reader)
    dangling = sorted({t for t in targets if t not in known_destinations})

    if dangling:
        raise RuntimeError(
            f"{len(dangling)} internal link target(s) in {output.name} resolve "
            f"to no destination: " + ", ".join(dangling[:10])
        )

    print(
        f"[check] {len(targets)} internal links resolve, "
        f"{len(known_destinations)} named destinations, "
        f"{len(reader.pages)} pages"
    )


def merge_with_cover(
    cover: Path,
    body: Path,
    output: Path,
) -> None:
    """Prepend `cover` to `body`, preserving the body's links and outline."""
    for required_file in (cover, body):
        if not required_file.is_file():
            raise FileNotFoundError(f"Required PDF not found: {required_file}")

    # Clone rather than copy pages: this carries /Dests, /Outlines and the
    # structure tree that the link annotations depend on.
    writer = PdfWriter(clone_from=str(body))

    # Insert the cover pages ahead of page 0 of the cloned body.
    writer.merge(0, str(cover))

    output.parent.mkdir(parents=True, exist_ok=True)
    writer.write(str(output))

    print(f"[write] {output}")

    verify_links(output)
