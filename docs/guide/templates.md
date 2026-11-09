# Templates & style

Each resume picks its template on the **Style** tab. Switching never touches your content: the same resume renders in
either design, and its cover letters follow along.

![The Julia template, with entry icons, a color-coded legend and a bracket beside Education](/screenshots/julia.webp)

## Style settings

| Setting              | Applies to       | Notes                                                                       |
| -------------------- | ---------------- | --------------------------------------------------------------------------- |
| **Template**         | both             | Classic or Julia                                                            |
| **Accent color**     | both             | Name, section titles, links. Checked for contrast against WCAG AA           |
| **Font**             | both             | System fonts plus the bundled Quicksand and Source Sans 3                   |
| **Text size**        | both             | In points                                                                   |
| **Page margin**      | both             | In inches                                                                   |
| **Leader**           | Classic          | Dotted, dashed, solid or no line between an entry's title and its date      |
| **Bullet**           | Classic          | Four diamonds, diamond, square, disc or none                                |
| **Paragraph marker** | Classic          | A chevron before paragraphs, or none                                        |
| **ALL-CAPS titles**  | Classic          | Per entries section, on the Content tab                                     |

Leaders, bullets and markers are drawn as shapes, so they never appear in the PDF's text layer. See
[Readability checks](./checks#readable-by-people-and-machines).

## Classic

Name and headline centered in the accent color, section titles with a square hung in the margin, and entries as a bold
title with a leader running to the date. Column lines can carry [Lucide icons](./editing#icons-beside-lines).

## Julia

The name centered over a gray line of headline parts and contacts, section titles in the accent color, and entries as an
organization line in small capitals, a bold title followed by the entry's icons, and indented bullets. Text after
` - ` in a title is set light, for a lab or department.

Julia adds controls that show only while it's selected:

- **Icons** on each entry, from the sidebar entry card or the image button in the page editor. Julia ships its own
  hand-drawn icon set (research, agriculture, food access, design, and more), listed first in the picker.
- **Legend** on an entries section: labelled icons printed after the section title, each label in its icon's color.
  The swatch beside a legend line changes the color. Tick **Background → Show** or **Border → Show** for a panel or a
  rounded border behind it.
- **Compact** on an entries section: each entry on one line, bullets in smaller type. Good for education.
- **Bracket** on an entries section: a thin gray bracket in the gutter beside the section's entries.

Julia's fonts are free stand-ins for the original design's commercial ones: Quicksand for the body and Source Sans 3
for the name, section titles and bold text. Both are bundled, so they print the same on every computer.

::: info Word and fonts
The Word export names the font but doesn't embed it. Word on a computer without Quicksand or Source Sans 3 substitutes
another, though line heights stay exact so only line wrapping changes.
:::

## Your own template

Templates are [Typst](https://typst.app) files, and the bundled ones are re-read on every render. See
[Writing templates](/developers/templates).
