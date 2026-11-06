// Field paths address one editable value in a ResumeDoc, dot-separated with
// indexes into the unfiltered arrays: "basics.name",
// "sections.2.items.0.bullets.3", "basics.contacts.1" (a whole contact).
// The server reports them with each rendered field (Preview.fields).
import type { Column, Contact, ResumeDoc } from './types'

export function getAt(doc: ResumeDoc, path: string): unknown {
  let cur: unknown = doc
  for (const seg of path.split('.')) {
    if (cur === null || typeof cur !== 'object' || !Object.hasOwn(cur, seg)) return undefined
    cur = (cur as Record<string, unknown>)[seg]
  }
  return cur
}

export function getString(doc: ResumeDoc, path: string): string | undefined {
  const value = getAt(doc, path)
  return typeof value === 'string' ? value : undefined
}

/** Writes a string field in place. Returns false if the path doesn't address one. */
export function setString(doc: ResumeDoc, path: string, value: string): boolean {
  const dot = path.lastIndexOf('.')
  const parent = getAt(doc, path.slice(0, dot))
  const key = path.slice(dot + 1)
  if (parent === null || typeof parent !== 'object' || typeof (parent as Record<string, unknown>)[key] !== 'string') return false
  ;(parent as Record<string, unknown>)[key] = value
  return true
}

const CONTACT = /^basics\.contacts\.\d+$/

/** The contact a "basics.contacts.<i>" path addresses; undefined for any other path. */
export function getContact(doc: ResumeDoc, path: string): Contact | undefined {
  if (!CONTACT.test(path)) return undefined
  const value = getAt(doc, path) as Contact | undefined
  return typeof value?.text === 'string' ? value : undefined
}

const ICON = /^(sections\.\d+\.columns\.\d+)\.icons\.(\d+)$/

/** A column line's icon slot ("sections.4.columns.1.icons.2"); it may hold no icon. */
export interface IconSlot {
  column: Column
  index: number
}

/** The icon slot an icons path addresses, if its line exists; null for any other path. */
export function iconSlotOf(doc: ResumeDoc, path: string): IconSlot | null {
  const m = ICON.exec(path)
  if (!m) return null
  const column = getAt(doc, m[1]!) as Column | undefined
  const index = Number(m[2])
  return typeof column?.lines?.[index] === 'string' ? { column, index } : null
}

/** Whether the path still addresses an editable value. */
export function resolves(doc: ResumeDoc, path: string): boolean {
  if (CONTACT.test(path)) return getContact(doc, path) !== undefined
  if (ICON.test(path)) return iconSlotOf(doc, path) !== null
  return getString(doc, path) !== undefined
}

const RICH = /^sections\.\d+\.(?:paragraphs\.\d+|items\.\d+\.(?:subtitle|text|bullets\.\d+)|columns\.\d+\.lines\.\d+)$/

/** Fields the template renders with inline markup; all others are literal text. */
export const isRich = (path: string) => RICH.test(path)

const LIST_ITEM = /^(basics\.headline|sections\.\d+\.(?:paragraphs|items\.\d+\.bullets|columns\.\d+\.lines))\.(\d+)$/

/** A string in a list the user can add to and remove from. */
export interface ListItem {
  array: string[]
  index: number
  /** Path of the array itself. */
  base: string
  /** The column, when the list is its lines (which keep icons beside them). */
  column?: Column
}

export function listOf(doc: ResumeDoc, path: string): ListItem | null {
  const m = LIST_ITEM.exec(path)
  if (!m) return null
  const base = m[1]!
  const array = getAt(doc, base)
  if (!Array.isArray(array)) return null
  const column = base.endsWith('.lines') ? (getAt(doc, base.slice(0, -'.lines'.length)) as Column) : undefined
  return { array, index: Number(m[2]), base, column }
}

const ENTRY_FIELDS: Record<string, string> = {
  date: 'Date',
  subtitle: 'Subtitle',
  location: 'Location',
  bullets: 'Bullet',
}

/** A short human label for the value at a path, e.g. "Bullet in Tufts University". */
export function describe(doc: ResumeDoc, path: string): string {
  const [root, a, field, b, sub] = path.split('.')
  if (root === 'basics') {
    return a === 'name' ? 'Name' : a === 'headline' ? 'Headline part' : 'Contact link'
  }
  const section = doc.sections[Number(a)]
  if (!section || field === 'title') return 'Section heading'
  const inName = (name: string | undefined) => (name ? ` in ${name}` : '')
  if (field === 'paragraphs') return `Paragraph${inName(section.title)}`
  if (field === 'items' && section.kind === 'entries') {
    const item = section.items[Number(b)]
    if (item?.type === 'label') return 'Subheading'
    if (sub === 'title') return 'Entry title'
    return `${ENTRY_FIELDS[sub!] ?? 'Entry'}${inName(item?.title)}`
  }
  if (field === 'columns' && section.kind === 'columns') {
    if (sub === 'title') return 'Column heading'
    if (sub === 'icons') return `Icon${inName(section.columns[Number(b)]?.title || section.title)}`
    return `Line${inName(section.columns[Number(b)]?.title || section.title)}`
  }
  return 'Text'
}
