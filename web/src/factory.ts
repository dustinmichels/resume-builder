import type { Column, Contact, ContactInfo, Entry, Label, Section, SectionKind, Supervisor } from './types'

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
  canContact: false,
})

/** Deep copy with fresh ids, for "Duplicate". */
export function cloneEntry(entry: Entry): Entry {
  return { ...structuredClone(entry), id: newId('e') }
}
