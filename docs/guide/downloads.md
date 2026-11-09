# Downloads & sync

![The Download menu: plain text, JSON, Markdown and Word](/screenshots/downloads.webp)

**Download PDF** is its own button; the other formats are under **Download ▾** next to it. On the **Cover letters** tab
they export the letter instead. In the Mac app each one asks where to save the file.

| Format       | What you get                                                                                                     |
| ------------ | ---------------------------------------------------------------------------------------------------------------- |
| **PDF**      | Exactly the page in the preview: tagged, PDF/UA-1, with your content attached as JSON                            |
| **Word**     | The same content and layout, built from native Word features so it stays editable and parser-friendly            |
| **Markdown** | Bold, italic and links kept; everything else plain. The preview's **Markdown** tab shows it live                 |
| **Text**     | The PDF's text layer, as in **What machines read**                                                               |
| **JSON**     | The resume as saved, style and template included. Re-import it with **Import JSON…**                             |

## Word that behaves like Word

The `.docx` isn't a picture of the PDF. Bullets are list formatting, dot leaders are tab leaders, capitals are the caps
effect, and columns stack vertically instead of hiding in tables, so recruiters can edit it and parsers can read it.
Julia's icons are pictures marked decorative, with a PNG copy for programs that can't show SVG, such as Google Docs.

## Importing

**Import JSON…** in the resume menu turns a JSON download, or any resume's `resume.json`, into a new resume. Missing
fields are filled in. If the file names a template this version doesn't have, it opens on Classic and the toolbar says
so.

## Sync files

**Sync files**, next to the save status, writes a Markdown, a Word and a PDF copy beside the resume and each cover
letter in your data folder, and the same inside each version. Handy for attaching to an email or keeping in a synced
folder.

- Pending edits are saved first.
- Copies already up to date are skipped.
- A file that fails to render is reported, and the rest still sync.
- The button reads **Files synced** until the resume next changes.
