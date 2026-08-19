from pypdf import PdfReader
from pathlib import Path

for pdf in Path(".").glob("*.pdf"):
    print("Checking:", pdf)
    try:
        r = PdfReader(pdf, strict=False)
        print("  pages:", len(r.pages))
    except Exception as e:
        print("  ERROR:", e)