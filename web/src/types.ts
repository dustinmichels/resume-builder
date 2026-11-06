// Resume document schema. Mirrored by template/resume.typ (rendering),
// server/checks.go (machine-readability checks) and server/docx.go +
// server/markdown.go (Word and Markdown exports).
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
}

/** A Lucide icon. `svg` is its markup, so the template draws it without an icon set. */
export interface LineIcon {
  name: string
  svg: string
}

export interface Column {
  id: string
  hidden?: boolean
  title: string
  lines: string[]
  /** The icon beside each line, by line index; null or missing for none. */
  icons?: (LineIcon | null)[]
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
  accent: string
  font: string
  fontSize: number
  margin: number
  leader: 'dotted' | 'dashed' | 'solid' | 'none'
  bullet: 'diamonds' | 'diamond' | 'square' | 'disc' | 'none'
  paragraphMarker: 'chevron' | 'none'
}

export interface ResumeDoc {
  basics: Basics
  sections: Section[]
  style: Style
}

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
}

/** A saved resume; `id` is its folder name and display name. */
export interface ResumeSummary {
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

/** What syncing a resume's files did; paths are relative to the resume folder. */
export interface SyncResult {
  written: string[]
  failed: { file: string; error: string }[]
}
