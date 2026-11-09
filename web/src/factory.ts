import { toRaw } from 'vue'
import type {
  Column,
  Contact,
  ContactInfo,
  CoverLetter,
  Entry,
  Label,
  LetterBlock,
  LetterBlockKind,
  ResumeDoc,
  Section,
  SectionKind,
  Supervisor,
} from './types'

const newId = (prefix: string) => `${prefix}-${crypto.randomUUID().slice(0, 8)}`

export const newContact = (): Contact => ({ id: newId('c'), text: '', url: '' })

export const newEntry = (): Entry => ({
  id: newId('e'),
  type: 'entry',
  title: '',
  date: '',
  subtitle: '',
  location: '',
  bullets: [''],
})

export const newLabel = (): Label => ({ id: newId('l'), type: 'label', text: '' })

export const newColumn = (): Column => ({ id: newId('k'), title: '', lines: [''] })

export function newSection(kind: SectionKind): Section {
  const id = newId('s')
  switch (kind) {
    case 'text':
      return { id, kind, title: '', paragraphs: [''] }
    case 'entries':
      return { id, kind, title: '', uppercaseTitles: false, items: [newEntry()] }
    case 'columns':
      return { id, kind, title: '', variant: 'table', columns: [newColumn(), newColumn()] }
  }
}

export const newContactInfo = (): ContactInfo => ({
  firstName: '',
  lastName: '',
  email: '',
  phone: '',
  address: '',
  city: '',
  region: '',
  postalCode: '',
  country: '',
  notes: '',
})

export const newSupervisor = (entryId: string, job: string): Supervisor => ({
  entryId,
  job,
  name: '',
  position: '',
  email: '',
  phone: '',
  address: '',
  city: '',
  region: '',
  postalCode: '',
  country: '',
  canContact: false,
})

/** Deep copy with fresh ids, for "Duplicate". `toRaw`: structuredClone rejects reactive proxies. */
export function cloneEntry(entry: Entry): Entry {
  return { ...structuredClone(toRaw(entry)), id: newId('e') }
}

export function newLetterBlock(kind: LetterBlockKind): LetterBlock {
  const id = newId('b')
  return kind === 'bullets' ? { id, kind, items: [''] } : { id, kind, text: '' }
}

type Json = Record<string, unknown>
const isObject = (v: unknown): v is Json => typeof v === 'object' && v !== null && !Array.isArray(v)
/** A text field: a number or true/false as written, anything else ''. */
const text = (v: unknown) => (typeof v === 'string' ? v : typeof v === 'number' || typeof v === 'boolean' ? String(v) : '')
/** A list of text: missing is empty, a lone value a list of one. */
const texts = (v: unknown) => (Array.isArray(v) ? v.map(text) : v == null ? [] : [text(v)])
const idOr = (v: unknown, prefix: string) => (typeof v === 'string' && v ? v : newId(prefix))
function objects(v: unknown, what: string): Json[] {
  if (v == null) return []
  if (!Array.isArray(v) || !v.every(isObject)) throw new Error(`${what} must be a list of objects`)
  return v
}

function importItem(it: Json): Json {
  return it.type === 'label'
    ? { ...it, id: idOr(it.id, 'l'), type: 'label', text: text(it.text) }
    : {
        ...it,
        id: idOr(it.id, 'e'),
        type: 'entry',
        title: text(it.title),
        date: text(it.date),
        subtitle: text(it.subtitle),
        location: text(it.location),
        bullets: texts(it.bullets),
      }
}

function importSection(s: Json, i: number): Json {
  const kind = s.kind ?? ('paragraphs' in s ? 'text' : 'items' in s ? 'entries' : 'columns' in s ? 'columns' : undefined)
  const where = `Section ${i + 1}${s.title ? ` (${text(s.title)})` : ''}`
  const base = { ...s, id: idOr(s.id, 's'), title: text(s.title) }
  switch (kind) {
    case 'text':
      return { ...base, kind, paragraphs: texts(s.paragraphs) }
    case 'entries':
      return { ...base, kind, uppercaseTitles: s.uppercaseTitles === true, items: objects(s.items, `${where}: items`).map(importItem) }
    case 'columns':
      return {
        ...base,
        kind,
        variant: s.variant === 'sections' ? 'sections' : 'table',
        columns: objects(s.columns, `${where}: columns`).map((c) => ({ ...c, id: idOr(c.id, 'k'), title: text(c.title), lines: texts(c.lines) })),
      }
    default:
      throw new Error(`${where} needs a kind: text, entries or columns`)
  }
}

/**
 * Fills in what an imported resume (hand-edited, from an older version or
 * another tool) lacks, so it renders and edits: missing text is '', missing
 * lists are empty, missing ids are new, and a section without a kind gets the
 * one its fields imply. Other fields, such as template-only ones, are kept as
 * they are; the server fills in missing style settings.
 */
export function normalizeImport(raw: unknown): Json {
  if (!isObject(raw) || !isObject(raw.basics) || !Array.isArray(raw.sections)) {
    throw new Error('Not a resume: a resume has "basics" and "sections"')
  }
  const b = raw.basics
  const basics = {
    ...b,
    name: text(b.name),
    headline: texts(b.headline),
    contacts: objects(b.contacts, 'Contacts').map((c) => ({ ...c, id: idOr(c.id, 'c'), text: text(c.text), url: text(c.url) })),
  }
  return { ...raw, basics, sections: objects(raw.sections, 'Sections').map(importSection) }
}

const LOREM =
  'Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat.'

const paragraph = (text: string): LetterBlock => ({ id: newId('b'), kind: 'paragraph', text })
const heading = (text: string): LetterBlock => ({ id: newId('b'), kind: 'heading', text })

/**
 * A new letter for the resume, headed by its last headline part (often the
 * location), laid out like samples/cover-letter with placeholder text to replace.
 * Its body font is Libertinus Serif, as in the sample, except on the Julia
 * template, whose letters start in the resume's font, as on the resume.
 */
export function newCoverLetter(resume: ResumeDoc): CoverLetter {
  const last = resume.basics.headline.filter((h) => h.trim()).at(-1)
  return {
    title: 'Cover letter',
    header: last ? [last] : [],
    greeting: 'Dear Hiring Manager,',
    blocks: [
      paragraph(LOREM),
      heading('My experience'),
      paragraph(LOREM),
      heading('My interest in this role'),
      paragraph(LOREM),
    ],
    closing: 'Sincerely,',
    style:
      resume.style.template === 'julia'
        ? { font: resume.style.font, fontSize: 10.5 }
        : { font: 'Libertinus Serif', fontSize: 12 },
  }
}
