# Tour of the editor

The editor is one window: forms on the left, the page on the right. Everything you type is saved as you go.

![The editor, with the Content tab on the left and the rendered page on the right](/screenshots/editor.webp)

## First run

The first time it opens, the editor asks where to keep your resumes. Pick any folder (it's created if it doesn't exist);
`~/resume-builder/` is the default. A folder with no resumes in it starts you off with a sample resume to edit. You can
change the folder later with the gear button at the top right.

## The toolbar

| Control                   | What it does                                                                                       |
| ------------------------- | -------------------------------------------------------------------------------------------------- |
| **Resume name ▾**         | Switch resumes, or **New resume…**, **Import JSON…**, **Duplicate…**, **Rename…**, **Delete…**     |
| Folder button             | Open the resume's folder in Finder (Explorer on Windows)                                           |
| **Saved**                 | Save status. Edits autosave a moment after you stop typing                                         |
| **Sync files**            | Write Markdown, Word and PDF copies next to the resume ([more](./downloads#sync-files))            |
| Undo / redo arrows        | Same as <kbd>⌘Z</kbd> and <kbd>⇧⌘Z</kbd> (<kbd>Ctrl</kbd>+<kbd>Z</kbd> / <kbd>Ctrl</kbd>+<kbd>Y</kbd>) |
| **Download ▾**            | Plain text, JSON, Markdown and Word                                                                |
| **Download PDF**          | The PDF you see in the preview                                                                     |
| Gear                      | Choose another data folder                                                                         |

## The sidebar

| Tab               | For                                                                                                     |
| ----------------- | ------------------------------------------------------------------------------------------------------- |
| **Content**       | Your name, headline, contact links and sections. Drag the grip to reorder anything                      |
| **Style**         | Template, accent color, font, text size, margins and markers ([Templates & style](./templates))         |
| **Cover letters** | Letters for this resume ([Cover letters](./cover-letters))                                              |
| **Contact**       | Your email, phone and mailing address, for filling in applications. Never printed                       |
| **Supervisors**   | A supervisor per job (name, position, contact, may we contact?). Never printed                          |
| **Versions**      | Named snapshots of the resume and its letters ([Versions](./files#versions))                            |

### Sections

**Add a section** at the bottom of the Content tab offers three kinds:

- **Entries**: jobs, schools, volunteer roles. Each entry has a title, dates, a subtitle, a location and bullets.
  **+ Subheading** adds a small italic label between entries, such as "Non-degree education:".
- **Paragraphs**: a profile or summary.
- **Columns**: a skills grid under one heading, or columns that each act as their own section, side by side.

The eye button on a section, entry, subheading or column hides it from the page without deleting it. It stays in the
editor, dimmed, so you can bring it back for the next application.

## The preview

| Tab                     | Shows                                                                                    |
| ----------------------- | ---------------------------------------------------------------------------------------- |
| **Page**                | The real PDF pages, re-rendered as you type. Click any text to edit it                   |
| **What machines read**  | The PDF's text layer, as applicant tracking systems see it                               |
| **Markdown**            | The Markdown export, rendered or as source                                               |
| **Checks**              | Readability findings, with a badge counting the ones to look at                          |

The page count sits at the right of the preview tabs.

## Next

- [Editing](./editing): click-to-edit, formatting, icons, undo.
- [Readability checks](./checks): what each check means.
