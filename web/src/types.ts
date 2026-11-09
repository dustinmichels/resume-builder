// Resume document schema. Mirrored by the resume templates in template/
// (rendering), server/checks.go (machine-readability checks) and
// server/docx.go + server/markdown.go (Word and Markdown exports).
//
// Inline text fields accept **bold**, *italic*, ***bold italic*** and [label](url).
//
// `hidden: true` on a section, entry, subheading or column keeps it in the
// resume but leaves it off the page and out of every export.

export interface Contact {
  id: string
  text: string
  url: string
}

export interface Basics {
  name: string
  headline: string[]
  contacts: Contact[]
}

export interface Entry {
  id: string
  type: 'entry'
  hidden?: boolean
  title: string
  date: string
  subtitle: string
  location: string
  bullets: string[]
  /** Icons beside the entry's title, in order (templates with the entryIcons feature). Missing when empty. */
  icons?: LineIcon[]
}

/** A small italic heading between entries, e.g. "Non-degree education:". */
export interface Label {
  id: string
  type: 'label'
  hidden?: boolean
  text: string
}

export type EntryItem = Entry | Label

export interface TextSection {
  id: string
  kind: 'text'
  hidden?: boolean
  title: string
  paragraphs: string[]
}

export interface EntriesSection {
  id: string
  kind: 'entries'
  hidden?: boolean
  title: string
  uppercaseTitles: boolean
  items: EntryItem[]
  /** compactEntries templates: each entry runs together on one line, details included. */
  compact?: boolean
  /** entryBrackets templates: one thin bracket in the gutter beside the section's entries. */
  brackets?: boolean
  /** legend templates: labelled icons printed after the section title, explaining the entries' icons. Missing when it has no lines. */
  legend?: Legend
}

/** An icon drawn beside text. `svg` is its markup, so the template draws it without an icon set. */
export interface LineIcon {
  name: string
  svg: string
  /** From a template's icon set: the color of a legend label beside it, editable per legend line. Lucide icons have none (drawn in the accent color, the label in ink). */
  color?: string
}

/** Lines with an optional icon beside each: a column's lines, a section's legend. */
export interface IconLines {
  lines: string[]
  /** The icon beside each line, by line index; null or missing for none. */
  icons?: (LineIcon | null)[]
}

/** A section's legend: its lines, and the panel and border around them. */
export interface Legend extends IconLines {
  /** The panel behind the legend, '#rrggbb'; missing for none. */
  fill?: string
  /** The rounded border around the legend, '#rrggbb'; missing for none. */
  border?: string
}

export interface Column extends IconLines {
  id: string
  hidden?: boolean
  title: string
  /**
   * Side-by-side sections only: the column's share of the line width (0–1),
   * set by dragging the divider after it. Missing: it hugs its content. The
   * last visible column always takes the remaining width.
   */
  width?: number
}

export interface ColumnsSection {
  id: string
  kind: 'columns'
  hidden?: boolean
  title: string
  /** table: columns under one heading; sections: each column is its own section. */
  variant: 'table' | 'sections'
  columns: Column[]
}

export type Section = TextSection | EntriesSection | ColumnsSection
export type SectionKind = Section['kind']

export interface Style {
  /** Resume template id; missing or '' is the classic template (template/resume.typ). */
  template?: string
  accent: string
  font: string
  fontSize: number
  margin: number
  /** Leader, bullet and paragraphMarker: templates with the markers feature only. */
  leader: 'dotted' | 'dashed' | 'solid' | 'none'
  bullet: 'diamonds' | 'diamond' | 'square' | 'disc' | 'none'
  paragraphMarker: 'chevron' | 'none'
}

export interface ResumeDoc {
  basics: Basics
  sections: Section[]
  style: Style
}

/** A paragraph of the letter body; consecutive paragraphs run on, first lines indented. */
export interface LetterParagraph {
  id: string
  kind: 'paragraph'
  text: string
}

/** A small heading within the letter body, e.g. "My experience". */
export interface LetterHeading {
  id: string
  kind: 'heading'
  text: string
}

export interface LetterBullets {
  id: string
  kind: 'bullets'
  items: string[]
}

export type LetterBlock = LetterParagraph | LetterHeading | LetterBullets
export type LetterBlockKind = LetterBlock['kind']

/**
 * A cover letter, saved as cover-letters/<name>.json in its resume's folder.
 * The name at the top, accent color, heading font and margin come from the
 * resume; its template's letter template renders it (template/cover-letter.typ
 * for Classic, template/julia-letter.typ for Julia). Paragraphs, bullets and
 * header lines accept inline markup; the other fields are plain text.
 */
export interface CoverLetter {
  /** The heading above the greeting, e.g. "Cover letter". */
  title: string
  /** Lines under the name, e.g. the location. */
  header: string[]
  greeting: string
  blocks: LetterBlock[]
  /** e.g. "Sincerely,"; the name is printed below it. Empty leaves both off. */
  closing: string
  style: { font: string; fontSize: number }
}

/** What the letter template renders: the letter with its resume's basics and style. */
export interface LetterDoc {
  basics: Basics
  style: Style
  letter: CoverLetter
}

/** A document the preview renders and edits in place. */
export type PageDoc = ResumeDoc | LetterDoc

// Side documents: kept in the resume's folder, never printed or exported.

/** contact.json: personal details for filling in applications. */
export interface ContactInfo {
  firstName: string
  lastName: string
  email: string
  phone: string
  /** Street lines, one per line. */
  address: string
  city: string
  region: string
  postalCode: string
  country: string
  notes: string
}

/** The supervisor at one job. */
export interface Supervisor {
  /** The resume Entry this belongs to. */
  entryId: string
  /**
   * The entry's title and organization as last seen, so the file reads on its
   * own and a supervisor whose entry was deleted still says which job it was.
   */
  job: string
  name: string
  position: string
  email: string
  phone: string
  /** Street lines, one per line. */
  address: string
  city: string
  region: string
  postalCode: string
  country: string
  /** Whether the employer may contact this supervisor. */
  canContact: boolean
}

/** supervisors.json */
export interface SupervisorsDoc {
  supervisors: Supervisor[]
}

export interface Check {
  group: 'Parsers' | 'Screen readers' | 'Links' | 'Dates'
  level: 'ok' | 'info' | 'warn' | 'error'
  title: string
  detail?: string
}

export interface Preview {
  pages: string[]
  text: string
  layoutText: string
  checks: Check[]
  warnings: string[]
  /** The Markdown export of the same document. */
  markdown: string
  /** Where each editable value sits on the rendered pages. */
  fields: FieldBox[]
  /** Where each column of side-by-side columns sections sits ("sections.4.columns.1"), unpadded. */
  columns: FieldBox[]
}

/**
 * One editable value on a rendered page. `path` addresses the value in the
 * document, dot-separated with array indexes ("sections.2.items.0.bullets.3",
 * "basics.contacts.1"). Geometry is in points from the page's top-left
 * corner; pages are US Letter (612 × 792 pt) and numbered from 1.
 */
export interface FieldBox {
  path: string
  page: number
  x: number
  y: number
  w: number
  h: number
}

/** Where resumes are stored. An empty dataDir means none has been chosen yet. */
export interface Settings {
  dataDir: string
  defaultDataDir: string
  /** The editor runs in the desktop app, not a browser. */
  desktop: boolean
}

/**
 * markers: the leader, bullet and paragraphMarker styles; uppercaseTitles: EntriesSection.uppercaseTitles;
 * entryIcons: Entry.icons; legend: EntriesSection.legend; compactEntries: EntriesSection.compact;
 * entryBrackets: EntriesSection.brackets.
 */
export type TemplateFeature = 'markers' | 'uppercaseTitles' | 'entryIcons' | 'legend' | 'compactEntries' | 'entryBrackets'

/** A resume template; GET /api/templates lists them, classic first. */
export interface TemplateInfo {
  /** '' is classic (template/resume.typ). */
  id: string
  name: string
  /** What the template draws beyond the common schema; the editor shows the matching controls only for these. */
  features: TemplateFeature[]
  /** The template's own icon set; [] for none. */
  icons: LineIcon[]
}

/** A saved resume or cover letter; `id` is its file or folder name and display name. */
export interface DocSummary {
  id: string
  /** RFC 3339 */
  modified: string
}

/** A named snapshot of a resume. */
export interface Version {
  id: string
  label: string
  /** RFC 3339 */
  created: string
}

/** A version just saved, with the copies of its documents that could not be written. */
export interface SavedVersion extends Version {
  failed: SyncResult['failed']
}

/** What syncing a resume's files did; paths are relative to the resume folder. */
export interface SyncResult {
  written: string[]
  failed: { file: string; error: string }[]
}
