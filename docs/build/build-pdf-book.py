from __future__ import annotations

import shutil
import subprocess
import sys
from pathlib import Path

from playwright.sync_api import (
    Error as PlaywrightError,
    TimeoutError as PlaywrightTimeoutError,
    sync_playwright,
)


# ---------------------------------------------------------------------------
# Project paths
# ---------------------------------------------------------------------------

# This script is expected at:
#     docs/build/build-pdf.py
#
# Therefore parents[1] points to:
#     docs/
ROOT = Path(__file__).resolve().parents[1]

REFERENCE_FILE = ROOT / "build/book/preface.md"
CSS_FILE = ROOT / "language-ref.pdf.css"


BUILD_DIR = ROOT / "build"
COMBINED_FILE = BUILD_DIR / "book/tlfolang.md"
HTML_FILE = BUILD_DIR / "book/tlfolang.html"
PDF_FILE = BUILD_DIR / "book/tlfolang.pdf"


# ---------------------------------------------------------------------------
# Configuration
# ---------------------------------------------------------------------------

# Use screen CSS so the output more closely resembles VS Code/browser preview.
#
# Change to False when the stylesheet primarily uses @media print rules.
USE_SCREEN_CSS = True

# Used only when the CSS does not provide an @page size.
DEFAULT_PDF_FORMAT = "A4"


# ---------------------------------------------------------------------------
# Utility functions
# ---------------------------------------------------------------------------

def require_file(path: Path) -> None:
    """Raise a clear error when a required input file is missing."""
    if not path.is_file():
        raise FileNotFoundError(f"Required file not found: {path}")


def require_program(name: str) -> str:
    """
    Find an external executable in PATH.

    Returns the resolved command path.
    """
    resolved = shutil.which(name)

    if resolved is None:
        raise RuntimeError(
            f"Required program {name!r} was not found in PATH.\n"
            f"Install it, close PowerShell, reopen PowerShell, and retry."
        )

    return resolved


def read_utf8(path: Path) -> str:
    """Read a UTF-8 text file."""
    require_file(path)
    return path.read_text(encoding="utf-8")


def write_utf8(path: Path, content: str) -> None:
    """Write UTF-8 text without introducing a BOM."""
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(content, encoding="utf-8", newline="\n")


def run_command(
    command: list[str],
    *,
    cwd: Path | None = None,
) -> None:
    """Run an external command and stop on failure."""
    print("Running:", subprocess.list2cmdline(command))

    subprocess.run(
        command,
        cwd=str(cwd) if cwd is not None else None,
        check=True,
    )


# ---------------------------------------------------------------------------
# Markdown processing
# ---------------------------------------------------------------------------

def normalize_horizontal_rules(markdown: str) -> str:
    """
    Replace standalone Markdown horizontal rules written as `---` with `***`.

    Replacement is not performed inside fenced code blocks.

    This avoids Pandoc treating a later standalone `---` line as the beginning
    of a YAML metadata block.
    """
    output: list[str] = []

    inside_fence = False
    active_fence_character: str | None = None
    active_fence_length = 0

    for line in markdown.splitlines(keepends=True):
        stripped = line.lstrip()

        # Detect fenced code blocks beginning with at least three backticks
        # or at least three tildes.
        if stripped.startswith("```") or stripped.startswith("~~~"):
            fence_character = stripped[0]
            fence_length = 0

            for character in stripped:
                if character == fence_character:
                    fence_length += 1
                else:
                    break

            if fence_length >= 3:
                if not inside_fence:
                    inside_fence = True
                    active_fence_character = fence_character
                    active_fence_length = fence_length

                elif (
                    fence_character == active_fence_character
                    and fence_length >= active_fence_length
                ):
                    inside_fence = False
                    active_fence_character = None
                    active_fence_length = 0

                output.append(line)
                continue

        if not inside_fence and line.strip() == "---":
            if line.endswith("\r\n"):
                newline = "\r\n"
            elif line.endswith("\n"):
                newline = "\n"
            else:
                newline = ""

            output.append(f"***{newline}")
        else:
            output.append(line)

    return "".join(output)


def indent_heading_levels(
    markdown: str,
    levels: int = 1,
) -> str:
    """
    Increase Markdown heading levels outside fenced code blocks.

    This is useful when inserting grammar-decisions.md beneath an appendix
    heading already present in language-ref.md.

    For example:
        # Decision Register
    becomes:
        ## Decision Register

    Set levels=0 to disable this behavior.
    """
    if levels <= 0:
        return markdown

    output: list[str] = []

    inside_fence = False
    active_fence_character: str | None = None
    active_fence_length = 0

    for line in markdown.splitlines(keepends=True):
        stripped = line.lstrip()

        if stripped.startswith("```") or stripped.startswith("~~~"):
            fence_character = stripped[0]
            fence_length = 0

            for character in stripped:
                if character == fence_character:
                    fence_length += 1
                else:
                    break

            if fence_length >= 3:
                if not inside_fence:
                    inside_fence = True
                    active_fence_character = fence_character
                    active_fence_length = fence_length

                elif (
                    fence_character == active_fence_character
                    and fence_length >= active_fence_length
                ):
                    inside_fence = False
                    active_fence_character = None
                    active_fence_length = 0

                output.append(line)
                continue

        if not inside_fence:
            leading_spaces = len(line) - len(line.lstrip(" "))
            content = line[leading_spaces:]

            hash_count = 0
            for character in content:
                if character == "#":
                    hash_count += 1
                else:
                    break

            if (
                1 <= hash_count <= 6
                and len(content) > hash_count
                and content[hash_count] == " "
            ):
                new_level = min(6, hash_count + levels)

                line = (
                    " " * leading_spaces
                    + "#" * new_level
                    + content[hash_count:]
                )

        output.append(line)

    return "".join(output)


def build_combined_markdown() -> None:
    """Merge the language reference, grammar, and decisions."""
    reference = read_utf8(REFERENCE_FILE)
   
   
    combined = normalize_horizontal_rules(reference)

   
    write_utf8(COMBINED_FILE, combined)

    print(f"Combined Markdown generated: {COMBINED_FILE}")


# ---------------------------------------------------------------------------
# Pandoc HTML generation
# ---------------------------------------------------------------------------

def build_html(pandoc_executable: str) -> None:
    """Generate self-contained HTML using Pandoc."""
    command = [
        pandoc_executable,
        str(COMBINED_FILE),

        # Disable Pandoc YAML metadata block detection.
        #"--from=markdown-yaml_metadata_block",
        "--from=markdown-yaml_metadata_block-tex_math_dollars",
        "--to=html5",
        "--standalone",
        "--section-divs",

        # Embed CSS and images in the generated HTML.
        "--embed-resources",

        # Preserve source wrapping rather than reflowing generated HTML.
        "--wrap=none",

        # Syntax highlighting for code blocks.
        "--highlight-style=pygments",

        f"--css={CSS_FILE}",

        # Pandoc searches this directory for relative images and resources.
        f"--resource-path={ROOT}",

        "--output",
        str(HTML_FILE),
    ]

    run_command(command, cwd=ROOT)

    if not HTML_FILE.is_file():
        raise RuntimeError(
            f"Pandoc completed without creating the expected HTML file: "
            f"{HTML_FILE}"
        )

    print(f"HTML generated: {HTML_FILE}")


# ---------------------------------------------------------------------------
# Chromium PDF generation
# ---------------------------------------------------------------------------

def build_pdf() -> None:
    """Render the generated HTML to PDF using Playwright Chromium."""
    html_uri = HTML_FILE.resolve().as_uri()

    with sync_playwright() as playwright:
        browser = playwright.chromium.launch(headless=True)

        try:
            page = browser.new_page(
                viewport={
                    "width": 1440,
                    "height": 1000,
                },
                device_scale_factor=1,
            )

            console_errors: list[str] = []
            failed_requests: list[str] = []

            def handle_console(message) -> None:
                if message.type == "error":
                    console_errors.append(message.text)

            def handle_request_failed(request) -> None:
                failure = request.failure
                failure_text = str(failure) if failure else "Unknown failure"
                failed_requests.append(
                    f"{request.url}: {failure_text}"
                )

            page.on("console", handle_console)
            page.on("requestfailed", handle_request_failed)

            page.goto(
                html_uri,
                wait_until="load",
                timeout=120_000,
            )

            # Wait until fonts are fully loaded.
            page.evaluate(
                """
                async () => {
                    if (document.fonts && document.fonts.ready) {
                        await document.fonts.ready;
                    }
                }
                """
            )

            if USE_SCREEN_CSS:
                page.emulate_media(media="screen")
            else:
                page.emulate_media(media="print")

            page.add_style_tag(
                content="""
                    html,
                    body {
                        -webkit-print-color-adjust: exact !important;
                        print-color-adjust: exact !important;
                    }

                    img,
                    svg {
                        max-width: 100%;
                    }

                    pre {
                        overflow-wrap: anywhere;
                    }
                """
            )

            page.pdf(
                path=str(PDF_FILE),
                format=DEFAULT_PDF_FORMAT,
                print_background=True,
                prefer_css_page_size=True,
                display_header_footer=False,
                outline=True,
                tagged=True,
                margin={
                    "top": "0",
                    "right": "0",
                    "bottom": "0",
                    "left": "0",
                },
            )

            if console_errors:
                print("\nChromium console errors:")

                for error in console_errors:
                    print(f"  - {error}")

            if failed_requests:
                print("\nFailed resource requests:")

                for failure in failed_requests:
                    print(f"  - {failure}")

        finally:
            browser.close()

    if not PDF_FILE.is_file():
        raise RuntimeError(
            "Chromium completed without creating the expected "
            f"PDF file: {PDF_FILE}"
        )

    print(f"PDF generated:  {PDF_FILE}")

# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------

def main() -> None:
    BUILD_DIR.mkdir(parents=True, exist_ok=True)

    for required_file in (
        REFERENCE_FILE,
        CSS_FILE,
    ):
        require_file(required_file)

    pandoc_executable = require_program("pandoc")

    build_combined_markdown()
    build_html(pandoc_executable)
    build_pdf()

    print()
    print("Build completed successfully.")
    print(f"Markdown: {COMBINED_FILE}")
    print(f"HTML:     {HTML_FILE}")
    print(f"PDF:      {PDF_FILE}")


if __name__ == "__main__":
    try:
        main()

    except FileNotFoundError as error:
        print(f"Build failed: {error}", file=sys.stderr)
        raise SystemExit(1)

    except RuntimeError as error:
        print(f"Build failed: {error}", file=sys.stderr)
        raise SystemExit(1)

    except subprocess.CalledProcessError as error:
        print(
            f"Build failed: external command returned exit code "
            f"{error.returncode}.",
            file=sys.stderr,
        )
        raise SystemExit(error.returncode)

    except PlaywrightTimeoutError as error:
        print(
            f"Build failed: Chromium timed out while loading the document: "
            f"{error}",
            file=sys.stderr,
        )
        raise SystemExit(1)

    except PlaywrightError as error:
        print(
            f"Build failed: Playwright/Chromium error: {error}",
            file=sys.stderr,
        )
        raise SystemExit(1)

    except OSError as error:
        print(f"Build failed: operating-system error: {error}", file=sys.stderr)
        raise SystemExit(1)