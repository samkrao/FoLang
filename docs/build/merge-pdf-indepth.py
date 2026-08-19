from pypdf import PdfReader, PdfWriter

writer = PdfWriter()

for pdf in ["cover/folang_in_depth_cover_a4.pdf", "folangid/tlfolang.pdf"]:
    reader = PdfReader(pdf)
    for page in reader.pages:
        writer.add_page(page)

with open("folangindepth.pdf", "wb") as f:
    writer.write(f)