from __future__ import annotations

"""Merge the language-reference cover and body into docs/language-ref.pdf."""

import sys
from pathlib import Path

from pdfmerge import merge_with_cover


# This script is expected at docs/build/merge-pdf.py, so parents[1] is docs/.
ROOT = Path(__file__).resolve().parents[1]
BUILD_DIR = ROOT / "build"

COVER_FILE = BUILD_DIR / "cover" / "lrm_cover_a4.pdf"
BODY_FILE = BUILD_DIR / "folang-language-reference.pdf"
OUTPUT_FILE = ROOT / "language-ref.pdf"


if __name__ == "__main__":
    try:
        merge_with_cover(COVER_FILE, BODY_FILE, OUTPUT_FILE)
    except (FileNotFoundError, RuntimeError) as error:
        print(f"Merge failed: {error}", file=sys.stderr)
        raise SystemExit(1)
