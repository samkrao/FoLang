from pypdf import PdfReader, PdfWriter

writer = PdfWriter()

for pdf in ["cover/lrm_cover_a4.pdf", "book/tlfolang.pdf"]:
    reader = PdfReader(pdf)
    for page in reader.pages:
        writer.add_page(page)

with open("tlfolang.pdf", "wb") as f:
    writer.write(f)