from pypdf import PdfReader, PdfWriter

writer = PdfWriter()

for pdf in ["cover/lrm_cover_a4.pdf", "folang-language-reference.pdf"]:
    reader = PdfReader(pdf)
    for page in reader.pages:
        writer.add_page(page)

with open("language-ref.pdf", "wb") as f:
    writer.write(f)