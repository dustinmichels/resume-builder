// Field paths address one editable value in the previewed document, a
// ResumeDoc or a LetterDoc, dot-separated with indexes into the unfiltered
// arrays: "basics.name", "sections.2.items.0.bullets.3", "basics.contacts.1"
// (a whole contact), "letter.blocks.3.text", "letter.blocks.4.items.0",
// "sections.1.items.0.icons.1" (an icon).
// The server reports them with each rendered field (Preview.fields).
import { newLetterBlock } from './factory'
import { setEntryIcon, setIcon } from './lineIcons'
import type { Contact, EntriesSection, EntryItem, IconLines, LetterBlock, LineIcon, PageDoc } from './types'

export function getAt(doc: PageDoc, path: string): unknown {
  let cur: unknown = doc
  for (const seg of path.split('.')) {
    if (cur === null || typeof cur !== 'object' || !Object.hasOwn(cur, seg)) return undefined
    cur = (cur as Record<string, unknown>)[seg]
  }
  return cur
}

export function getString(doc: PageDoc, path: string): string | undefined {
  const value = getAt(doc, path)
  return typeof value === 'string' ? value : undefined
}

/** Writes a string field in place. Returns false if the path doesn't address one. */
export function setString(doc: PageDoc, path: string, value: string): boolean {
  const dot = path.lastIndexOf('.')
  const parent = getAt(doc, path.slice(0, dot))
  const key = path.slice(dot + 1)
  if (parent === null || typeof parent !== 'object' || typeof (parent as Record<string, unknown>)[key] !== 'string') return false
  ;(parent as Record<string, unknown>)[key] = value
  return true
}

const CONTACT = /^basics\.contacts\.\d+$/

/** The contact a "basics.contacts.<i>" path addresses; undefined for any other path. */
export function getContact(doc: PageDoc, path: string): Contact | undefined {
  if (!CONTACT.test(path)) return undefined
  const value = getAt(doc, path) as Contact | undefined
  return typeof value?.text === 'string' ? value : undefined
}

const LINE_ICON = /^(sections\.\d+\.(?:columns\.\d+|legend))\.icons\.(\d+)$/
const ENTRY_ICON = /^(sections\.\d+\.items\.\d+)\.icons\.(\d+)$/

/**
 * An icon slot: beside a column or legend line ("sections.4.columns.1.icons.2",
 * "sections.1.legend.icons.0"), where it may hold no icon, or one of an
 * entry's icons ("sections.1.items.0.icons.1"), where the slot one past the
 * last adds an icon.
 */
export interface IconSlot {
  icon: LineIcon | null
  /** Sets the slot's icon; null removes it. */
  set: (icon: LineIcon | null) => void
}

/** The icon slot an icons path addresses, if its line or entry exists; null for any other path. */
export function iconSlotOf(doc: PageDoc, path: string): IconSlot | null {
  let m = LINE_ICON.exec(path)
  if (m) {
    const owner = getAt(doc, m[1]!) as IconLines | undefined
    const index = Number(m[2])
    if (typeof owner?.lines?.[index] !== 'string') return null
    return { icon: owner.icons?.[index] ?? null, set: (icon) => setIcon(owner, index, icon) }
  }
  m = ENTRY_ICON.exec(path)
  if (m) {
    const entry = getAt(doc, m[1]!) as EntryItem | undefined
    const index = Number(m[2])
    if (entry?.type !== 'entry' || index > (entry.icons?.length ?? 0)) return null
    return { icon: entry.icons?.[index] ?? null, set: (icon) => setEntryIcon(entry, index, icon) }
  }
  return null
}

/** Whether the path still addresses an editable value. */
export function resolves(doc: PageDoc, path: string): boolean {
  if (CONTACT.test(path)) return getContact(doc, path) !== undefined
  if (LINE_ICON.test(path) || ENTRY_ICON.test(path)) return iconSlotOf(doc, path) !== null
  return getString(doc, path) !== undefined
}

const ENTRY_TITLE = /^(sections\.\d+\.items\.\d+)\.title$/

/** For an entry's title: the path of the slot that adds an icon to the entry; null for any other path. */
export function addEntryIconPath(doc: PageDoc, path: string): string | null {
  const m = ENTRY_TITLE.exec(path)
  const entry = m && (getAt(doc, m[1]!) as EntryItem | undefined)
  return entry?.type === 'entry' ? `${m![1]}.icons.${entry.icons?.length ?? 0}` : null
}

const LETTER_BLOCK = /^letter\.blocks\.(\d+)\.text$/

/** The letter block whose text a "letter.blocks.<i>.text" path addresses. */
function letterBlockOf(doc: PageDoc, path: string): { block: LetterBlock; index: number } | null {
  const m = LETTER_BLOCK.exec(path)
  if (!m) return null
  const block = getAt(doc, `letter.blocks.${m[1]}`) as LetterBlock | undefined
  return block ? { block, index: Number(m[1]) } : null
}

const RICH =
  /^(?:sections\.\d+\.(?:paragraphs\.\d+|items\.\d+\.(?:subtitle|text|bullets\.\d+)|(?:columns\.\d+|legend)\.lines\.\d+)|letter\.(?:header\.\d+|blocks\.\d+\.items\.\d+))$/

/** Fields the templates render with inline markup; all others are literal text. */
export const isRich = (doc: PageDoc, path: string) =>
  RICH.test(path) || letterBlockOf(doc, path)?.block.kind === 'paragraph'

const LIST_ITEM =
  /^(basics\.headline|sections\.\d+\.(?:paragraphs|items\.\d+\.bullets|(?:columns\.\d+|legend)\.lines)|letter\.(?:header|blocks\.\d+\.items))\.(\d+)$/
const LEGEND_LINES = /^(sections\.\d+)\.legend\.lines$/

/** An item in a list the user can add to and remove from. */
export interface ListItem {
  array: unknown[]
  index: number
  /** Path of the array itself. */
  base: string
  /** Appended to "<base>.<index>" to address an item's text ("" for lists of strings). */
  suffix: string
  /** A new, empty item. */
  create: () => unknown
  /** The column or legend, when the list is its lines (which keep icons beside them). */
  iconLines?: IconLines
  /** Call after removing the last item: drops a legend left without lines. */
  onEmpty?: () => void
}

export function listOf(doc: PageDoc, path: string): ListItem | null {
  // A letter's paragraphs are blocks: Enter adds a paragraph after this one.
  const para = letterBlockOf(doc, path)
  if (para?.block.kind === 'paragraph') {
    const array = getAt(doc, 'letter.blocks') as LetterBlock[]
    return { array, index: para.index, base: 'letter.blocks', suffix: '.text', create: () => newLetterBlock('paragraph') }
  }
  const m = LIST_ITEM.exec(path)
  if (!m) return null
  const base = m[1]!
  const array = getAt(doc, base)
  if (!Array.isArray(array)) return null
  const item: ListItem = { array, index: Number(m[2]), base, suffix: '', create: () => '' }
  if (base.endsWith('.lines')) item.iconLines = getAt(doc, base.slice(0, -'.lines'.length)) as IconLines
  const legend = LEGEND_LINES.exec(base)
  if (legend) {
    const section = getAt(doc, legend[1]!) as EntriesSection
    item.onEmpty = () => delete section.legend
  }
  return item
}

const ENTRY_FIELDS: Record<string, string> = {
  date: 'Date',
  subtitle: 'Subtitle',
  location: 'Location',
  bullets: 'Bullet',
  icons: 'Icon',
}

const LETTER_FIELDS: Record<string, string> = {
  title: 'Letter heading',
  header: 'Line under the name',
  greeting: 'Greeting',
  closing: 'Closing',
}

const LETTER_BLOCKS = { paragraph: 'Paragraph', heading: 'Subheading', bullets: 'Bullet' } as const

/** A short human label for the value at a path, e.g. "Bullet in Tufts University". */
export function describe(doc: PageDoc, path: string): string {
  const [root, a, field, b, sub] = path.split('.')
  if (root === 'basics') {
    return a === 'name' ? 'Name' : a === 'headline' ? 'Headline part' : 'Contact link'
  }
  if (root === 'letter') {
    if (a !== 'blocks') return LETTER_FIELDS[a!] ?? 'Text'
    const block = getAt(doc, `letter.blocks.${field}`) as LetterBlock | undefined
    return block ? LETTER_BLOCKS[block.kind] : 'Text'
  }
  if (!('sections' in doc)) return 'Text'
  const section = doc.sections[Number(a)]
  if (!section || field === 'title') return 'Section heading'
  const inName = (name: string | undefined) => (name ? ` in ${name}` : '')
  if (field === 'paragraphs') return `Paragraph${inName(section.title)}`
  if (field === 'legend') return `Legend ${b === 'icons' ? 'icon' : 'line'}${inName(section.title)}`
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
