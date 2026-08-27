#!/usr/bin/env python3
"""Merge the Markdown files in a folder into one Markdown document."""

from __future__ import annotations

import argparse
from pathlib import Path


def markdown_files(folder: Path, recursive: bool) -> list[Path]:
    """Return Markdown files in deterministic, case-insensitive path order."""
    candidates = folder.rglob("*.md") if recursive else folder.glob("*.md")
    return sorted(
        (path for path in candidates if path.is_file()),
        key=lambda path: path.relative_to(folder).as_posix().casefold(),
    )


def merge_markdown(folder: Path, output: Path, recursive: bool) -> int:
    """Merge files from folder into output and return the number merged."""
    folder = folder.resolve()
    output = output.resolve()

    if not folder.is_dir():
        raise ValueError(f"input folder does not exist or is not a directory: {folder}")

    files = [path for path in markdown_files(folder, recursive) if path.resolve() != output]
    if not files:
        raise ValueError(f"no Markdown files found in: {folder}")

    output.parent.mkdir(parents=True, exist_ok=True)
    with output.open("w", encoding="utf-8", newline="\n") as merged:
        for index, path in enumerate(files):
            if index:
                merged.write("\n\n")
            relative = path.relative_to(folder).as_posix()
            merged.write(f"<!-- Source: {relative} -->\n\n")
            merged.write(path.read_text(encoding="utf-8").rstrip())
            merged.write("\n")

    return len(files)


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Merge Markdown files from a folder in sorted path order."
    )
    parser.add_argument("folder", type=Path, help="folder containing Markdown files")
    parser.add_argument(
        "-o",
        "--output",
        type=Path,
        default=Path("merged.md"),
        help="output file (default: merged.md in the current directory)",
    )
    parser.add_argument(
        "-r",
        "--recursive",
        action="store_true",
        help="include Markdown files in subdirectories",
    )
    return parser.parse_args()


def main() -> int:
    args = parse_args()
    try:
        count = merge_markdown(args.folder, args.output, args.recursive)
    except (OSError, UnicodeError, ValueError) as error:
        print(f"merge-md: {error}")
        return 1

    print(f"merged {count} Markdown file(s) into {args.output.resolve()}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
