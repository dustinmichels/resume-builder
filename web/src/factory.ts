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

export const newId = (prefix: string) => `${prefix}-${crypto.randomUUID().slice(0, 8)}`

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

/** Deep copy with fresh ids, for "Duplicate". */
export function cloneEntry(entry: Entry): Entry {
  return { ...structuredClone(entry), id: newId('e') }
}

export function newLetterBlock(kind: LetterBlockKind): LetterBlock {
  const id = newId('b')
  return kind === 'bullets' ? { id, kind, items: [''] } : { id, kind, text: '' }
}

const LOREM =
  'Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat.'

const paragraph = (text: string): LetterBlock => ({ id: newId('b'), kind: 'paragraph', text })
const heading = (text: string): LetterBlock => ({ id: newId('b'), kind: 'heading', text })

/**
 * A new letter for the resume, headed by its last headline part (often the
 * location), laid out like samples/cover-letter with placeholder text to replace.
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
    style: { font: 'Libertinus Serif', fontSize: 12 },
  }
}
