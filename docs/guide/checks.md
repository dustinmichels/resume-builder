# Readability checks

A resume is read twice: by a person looking at the page, and first by software reading the PDF's text. Resume Builder
shows you both and checks the second on every render.

## What machines read

Applicant tracking systems and parsers don't see the page; they extract the PDF's text layer. The **What machines read**
tab shows that text, extracted with `pdftotext` (from Poppler), in **Reading order** or as **Layout**.

![The What machines read tab: plain text in reading order](/screenshots/machines.webp)

If your job title and its date end up on different lines here, or a bullet character turns into `?`, a parser sees the
same thing.

::: tip Needs Poppler
This tab and the screen-reader and PDF-link checks need Poppler: `brew install poppler` on macOS, `poppler-utils` on
Linux. See [Install](./install).
:::

## Readable by people and machines

Every decoration is drawn by the template as a shape, never typed as a character, so the text layer holds only your
words:

| On the page                               | In the PDF text layer              |
| ----------------------------------------- | ---------------------------------- |
| `SENIOR PLANNER ·········· Mar 2023`      | `SENIOR PLANNER Mar 2023`          |
| `❖ Redesigned 14 bus routes…`             | `Redesigned 14 bus routes…`        |
| `maya@… ● mayaokafor.example ● github…`   | `maya@… mayaokafor.example github…` |

The PDF is also **tagged** (headings, lists) and validated as **PDF/UA-1**, the accessible-PDF standard, so screen
readers can navigate it. It carries your content as an attached `resume.json`, and its document properties come from
the resume:

| Property | Source                                                  |
| -------- | ------------------------------------------------------- |
| Title    | `<name> – Resume`                                       |
| Author   | Your name                                               |
| Subject  | Your headline, joined with `·`                          |
| Keywords | The lines of your skills grid                           |

## The Checks tab

![The Checks tab, grouped by reader, with one warning about a link](/screenshots/checks.webp)

Findings are grouped by who reads the resume. The badge on the tab counts the ones worth a look.

### Parsers

What applicant tracking systems get from the text layer.

- Typed dot leaders (`.....`) instead of drawn ones.
- Symbol characters such as ❖ ● or emoji, which parsers mangle.
- Entry titles split from their dates.
- Dates without a year.
- Any of your content missing from the text layer.
- Whether your email address reads cleanly.

### Screen readers

What assistive technology gets from the PDF's tags.

- PDF/UA-1 violations. A template that breaks them still renders, just without the standard.
- Content hidden from screen readers, or read out of order.
- Section titles not tagged as headings.
- Accent color contrast below WCAG AA (4.5:1).

### Links

- Links without `https://` (or `mailto:`, `tel:`).
- Link text naming a different address than where the link goes. On paper, people see the text.
- Links that aren't clickable in the PDF.
- Web links that don't open. Each is requested at most once every 10 minutes; sites that refuse automated requests
  (LinkedIn) show as unconfirmed, not broken.

### Dates

- Date ranges that end before they start.
- Entries not listed newest first. "Present" counts as this month, and a subheading (like "Non-degree education:")
  starts a new list.
