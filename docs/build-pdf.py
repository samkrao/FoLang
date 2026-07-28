from pathlib import Path
import shutil
import subprocess
import sys

ROOT = Path(__file__).resolve().parents[1]

REFERENCE_FILE = ROOT / "language-ref.md"
GRAMMAR_FILE = ROOT / "grammar" / "folang.ebnf"
DECISIONS_FILE = ROOT / "grammar" / "grammar-decisions.md"
CSS_FILE = ROOT / "language-ref.pdf.css"

BUILD_DIR = ROOT / "build"
COMBINED_FILE = BUILD_DIR / "language-ref-complete.md"
HTML_FILE = BUILD_DIR / "folang-language-reference.html"
PDF_FILE = BUILD_DIR / "folang-language-reference.pdf"


def require_file(path: Path) -> None:
    if not path.is_file():
        raise FileNotFoundError(f"Required file not found: {path}")


def require_program(name: str) -> None:
    if shutil.which(name) is None:
        raise RuntimeError(
            f"{name!r} was not found in PATH. "
            f"Install it and reopen PowerShell."
        )


def run(command: list[str]) -> None:
    print("Running:", subprocess.list2cmdline(command))
    subprocess.run(command, check=True)


def main() -> None:
    for path in (
        REFERENCE_FILE,
        GRAMMAR_FILE,
        DECISIONS_FILE,
        CSS_FILE,
    ):
        require_file(path)

    require_program("pandoc")
    require_program("weasyprint")

    BUILD_DIR.mkdir(parents=True, exist_ok=True)

    reference = REFERENCE_FILE.read_text(encoding="utf-8")
    grammar = GRAMMAR_FILE.read_text(encoding="utf-8").rstrip()
    decisions = DECISIONS_FILE.read_text(encoding="utf-8").rstrip()

    combined = reference.replace(
        "{{FOLANG_EBNF}}",
        f"```ebnf\n{grammar}\n```",
    ).replace(
        "{{GRAMMAR_DECISIONS}}",
        decisions,
    )

    if "{{FOLANG_EBNF}}" in combined:
        raise RuntimeError("Missing or unreplaced FOLANG_EBNF placeholder.")

    if "{{GRAMMAR_DECISIONS}}" in combined:
        raise RuntimeError("Missing or unreplaced GRAMMAR_DECISIONS placeholder.")

    COMBINED_FILE.write_text(combined, encoding="utf-8")

    run([
        "pandoc",
        str(COMBINED_FILE),
        "--from=markdown",
        "--to=html5",
        "--standalone",
        "--toc",
        "--number-sections",
        f"--css={CSS_FILE}",
        f"--resource-path={ROOT}",
        "--output",
        str(HTML_FILE),
    ])

    run([
        "weasyprint",
        str(HTML_FILE),
        str(PDF_FILE),
    ])

    print(f"HTML generated: {HTML_FILE}")
    print(f"PDF generated:  {PDF_FILE}")


if __name__ == "__main__":
    try:
        main()
    except (OSError, RuntimeError, subprocess.CalledProcessError) as error:
        print(f"Build failed: {error}", file=sys.stderr)
        raise SystemExit(1)