#!/usr/bin/env python3
"""
repair-pdf.py

Repair PDFs that pypdf can recover in non-strict mode but rejects in strict mode,
for example:
    incorrect startxref pointer(...)
    broken xref table
    malformed cross-reference offsets

The repair strategy is safe for normal use:
  1. Read with PdfReader(strict=False), allowing pypdf to reconstruct/recover.
  2. Clone the recovered document into a new PdfWriter.
  3. Write a completely new PDF so xref/startxref offsets are recalculated.
  4. Re-open the new PDF with strict=True as a structural sanity check.

IMPORTANT:
Do NOT "fix" these PDFs by globally replacing CRLF with LF in the raw PDF bytes.
That can alter compressed/binary stream data. Rewriting through a PDF parser/writer
is the safer general solution.
"""

from __future__ import annotations

import argparse
import logging
import sys
from pathlib import Path

from pypdf import PdfReader, PdfWriter
from pypdf.errors import PdfReadError


def default_output_path(src: Path) -> Path:
    return src.with_name(f"{src.stem}_repaired{src.suffix}")


def strict_check(path: Path, password: str | None = None) -> tuple[bool, str]:
    """Return (ok, details) for a strict pypdf parse."""
    try:
        reader = PdfReader(str(path), strict=True)

        if reader.is_encrypted:
            if password is None:
                return False, "PDF is encrypted; provide --password."
            result = reader.decrypt(password)
            if result == 0:
                return False, "Password did not decrypt the PDF."

        page_count = len(reader.pages)
        return True, f"strict parse OK, pages={page_count}"
    except Exception as exc:
        return False, f"{type(exc).__name__}: {exc}"


def repair_pdf(
    src: Path,
    dst: Path,
    password: str | None = None,
    overwrite: bool = False,
    quiet_recovery_warnings: bool = False,
) -> None:
    if not src.exists():
        raise FileNotFoundError(f"Input PDF does not exist: {src}")

    if src.resolve() == dst.resolve():
        raise ValueError(
            "Input and output paths are the same. "
            "Use a different output path; replace the original only after verification."
        )

    if dst.exists() and not overwrite:
        raise FileExistsError(
            f"Output already exists: {dst}\n"
            "Use --force to overwrite it."
        )

    # First show whether the original is structurally clean in strict mode.
    ok_before, details_before = strict_check(src, password)
    print(f"[check] original: {details_before}")

    # pypdf emits recovery diagnostics through its logger.
    pypdf_logger = logging.getLogger("pypdf")
    old_level = pypdf_logger.level
    if quiet_recovery_warnings:
        pypdf_logger.setLevel(logging.ERROR)

    try:
        # Non-strict mode is intentional here: this is the recovery step.
        reader = PdfReader(str(src), strict=False)

        if reader.is_encrypted:
            if password is None:
                raise PdfReadError(
                    "Input PDF is encrypted. Supply --password."
                )
            result = reader.decrypt(password)
            if result == 0:
                raise PdfReadError("Incorrect PDF password.")

        # Force page-tree access while the recovered reader is alive.
        page_count = len(reader.pages)
        print(f"[read] recovered document, pages={page_count}")

        writer = PdfWriter()

        # Clone the recovered document instead of copying pages only.
        # This preserves substantially more document structure such as metadata,
        # outlines/catalog entries, named destinations, etc., when pypdf supports them.
        writer.clone_document_from_reader(reader)

        dst.parent.mkdir(parents=True, exist_ok=True)
        with dst.open("wb") as fh:
            writer.write(fh)

    finally:
        if quiet_recovery_warnings:
            pypdf_logger.setLevel(old_level)

    print(f"[write] repaired PDF: {dst}")

    # The new file should parse strictly because PdfWriter rebuilt offsets/xref.
    ok_after, details_after = strict_check(dst, password=None)
    print(f"[check] repaired: {details_after}")

    if not ok_after:
        raise PdfReadError(
            "The rewritten PDF still fails strict validation. "
            "Try qpdf or Ghostscript as a second-stage repair."
        )

    if ok_before:
        print(
            "[note] The original already passed pypdf strict parsing. "
            "The output is a normalized rewrite rather than a required repair."
        )
    else:
        print("[done] Structural rewrite completed successfully.")


def main() -> int:
    parser = argparse.ArgumentParser(
        description="Recover and rewrite a malformed PDF using pypdf."
    )
    parser.add_argument("input", type=Path, help="Input PDF")
    parser.add_argument(
        "output",
        nargs="?",
        type=Path,
        help="Output PDF (default: <input>_repaired.pdf)",
    )
    parser.add_argument(
        "--password",
        help="Password for an encrypted input PDF",
    )
    parser.add_argument(
        "--force",
        action="store_true",
        help="Overwrite the output file if it already exists",
    )
    parser.add_argument(
        "--quiet-recovery-warnings",
        action="store_true",
        help="Suppress pypdf recovery warnings while reading the malformed input",
    )
    parser.add_argument(
        "--check-only",
        action="store_true",
        help="Only perform a strict structural check; do not rewrite",
    )

    args = parser.parse_args()

    src: Path = args.input

    if args.check_only:
        ok, details = strict_check(src, args.password)
        print(details)
        return 0 if ok else 2

    dst = args.output or default_output_path(src)

    try:
        repair_pdf(
            src=src,
            dst=dst,
            password=args.password,
            overwrite=args.force,
            quiet_recovery_warnings=args.quiet_recovery_warnings,
        )
        return 0
    except Exception as exc:
        print(f"[error] {type(exc).__name__}: {exc}", file=sys.stderr)
        return 1


if __name__ == "__main__":
    raise SystemExit(main())
