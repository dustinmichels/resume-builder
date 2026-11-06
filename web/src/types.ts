// Resume document schema. Mirrored by template/resume.typ (rendering) and
// server/checks.go (machine-readability checks).
//
// Inline text fields accept **bold**, *italic* and [label](url).

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
  text: string
}

export type EntryItem = Entry | Label

export interface TextSection {
  id: string
  kind: 'text'
  title: string
  paragraphs: string[]
}

export interface EntriesSection {
  id: string
  kind: 'entries'
  title: string
  uppercaseTitles: boolean
  items: EntryItem[]
}

export interface Column {
  id: string
  title: string
  lines: string[]
}

export interface ColumnsSection {
  id: string
  kind: 'columns'
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

export interface Check {
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
}
