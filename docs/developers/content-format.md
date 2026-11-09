# Content format

A resume is one `resume.json`. The schema lives in `web/src/types.ts` and is mirrored by the templates, the checks
(`server/checks.go`) and the exports (`server/docx.go`, `server/markdown.go`).

```json
{
  "basics": {
    "name": "Maya Okafor",
    "headline": ["Transit data analyst", "Portland, OR"],
    "contacts": [{ "id": "c-1", "text": "maya@example.com", "url": "mailto:maya@example.com" }]
  },
  "sections": [
    { "id": "s-1", "kind": "text", "title": "Profile", "paragraphs": ["Planner who turns **ridership data** into better service."] },
    {
      "id": "s-2", "kind": "entries", "title": "Experience", "uppercaseTitles": true,
      "items": [
        {
          "id": "e-1", "type": "entry",
          "title": "Senior Service Planner", "date": "Mar 2023 – Present",
          "subtitle": "*Rose City Transit*", "location": "Portland, OR",
          "bullets": ["Redesigned 14 bus routes around a frequent grid"]
        },
        { "id": "l-1", "type": "label", "text": "Earlier:" }
      ]
    },
    {
      "id": "s-3", "kind": "columns", "title": "Skills", "variant": "table",
      "columns": [{ "id": "k-1", "title": "Data", "lines": ["Python, SQL"] }]
    }
  ],
  "style": {
    "template": "", "accent": "#7a1a4a", "font": "Helvetica", "fontSize": 10, "margin": 0.6,
    "leader": "dotted", "bullet": "diamonds", "paragraphMarker": "chevron"
  }
}
```

## Sections

| `kind`    | Fields                                                                                                   |
| --------- | -------------------------------------------------------------------------------------------------------- |
| `text`    | `paragraphs: string[]`                                                                                   |
| `entries` | `items` of entries (`title`, `date`, `subtitle`, `location`, `bullets`, `icons?`) and labels (`text`); `uppercaseTitles`; Julia's `compact?`, `brackets?`, `legend?` |
| `columns` | `variant`: `table` (one heading) or `sections` (side by side, each its own heading); `columns` of `{title, lines, icons?, width?}` |

`hidden: true` on a section, entry, label or column leaves it in the file and the editor but off the page and out of
every export.

## Inline formatting

Paragraphs, subtitles, bullets, labels and column lines (and a letter's paragraphs, bullets and header lines) accept:

| Markup                 | Renders as       |
| ---------------------- | ---------------- |
| `**bold**`             | **bold**         |
| `*italic*`             | *italic*         |
| `***bold italic***`    | ***bold italic*** |
| `[label](https://…)`   | a link           |

Everything else is literal text, so `#`, `$` and `@` are safe. Other fields (names, dates, locations) are plain.

## Icons

An icon is `{ "name", "svg", "color"? }`: the SVG markup itself, so a template draws it without needing an icon set.
Column lines carry `icons`, parallel to `lines` (`null` for none); entries carry `icons` in order; a Julia legend is
`legend: { lines, icons, fill?, border? }`. `color` is a legend label's color, from the template's icon set.

## Cover letters

`cover-letters/<name>.json`:

```json
{
  "title": "Cover letter",
  "header": ["Portland, OR"],
  "greeting": "Dear Hiring Committee,",
  "blocks": [
    { "id": "b-1", "kind": "paragraph", "text": "I am writing to apply…" },
    { "id": "b-2", "kind": "heading", "text": "What I would bring" },
    { "id": "b-3", "kind": "bullets", "items": ["Community engagement in four languages"] }
  ],
  "closing": "Sincerely,",
  "style": { "font": "Libertinus Serif", "fontSize": 11 }
}
```

Letter templates read `data.json` as `{ basics, style, letter }`: the resume's `basics` and `style` plus the letter.
