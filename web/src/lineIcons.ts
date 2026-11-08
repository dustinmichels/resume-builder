// A column's or legend's icons sit in an array parallel to its lines
// (IconLines.icons). These keep each icon beside its line as lines are
// inserted, removed and reordered.
import type { Entry, IconLines, LineIcon } from './types'

/** Mirrors `lines.splice(start, deleteCount, ...insertCount blank lines)`. */
export function spliceIcons(owner: IconLines, start: number, deleteCount: number, insertCount = 0) {
  if (!owner.icons || start >= owner.icons.length) return
  owner.icons.splice(start, deleteCount, ...Array<null>(insertCount).fill(null))
  trim(owner)
}

/** Mirrors moving the line at `from` to `to`. */
export function moveIcon(owner: IconLines, from: number, to: number) {
  if (!owner.icons || from === to) return
  pad(owner.icons, Math.max(from, to))
  const [icon] = owner.icons.splice(from, 1)
  owner.icons.splice(to, 0, icon ?? null)
  trim(owner)
}

export function setIcon(owner: IconLines, index: number, icon: LineIcon | null) {
  const icons = (owner.icons ??= [])
  pad(icons, index)
  icons[index] = icon
  trim(owner)
}

function pad(icons: (LineIcon | null)[], index: number) {
  while (icons.length <= index) icons.push(null)
}

/** Drops trailing empty slots, and the array once it holds no icon. */
function trim(owner: IconLines) {
  const icons = owner.icons!
  while (icons.length && !icons.at(-1)) icons.pop()
  if (!icons.length) delete owner.icons
}

/**
 * Sets the entry's icon at `index`: index === length appends, null removes
 * it. The array is dropped once empty.
 */
export function setEntryIcon(entry: Entry, index: number, icon: LineIcon | null) {
  const icons = (entry.icons ??= [])
  if (icon) icons[Math.min(index, icons.length)] = icon
  else icons.splice(index, 1)
  if (!icons.length) delete entry.icons
}

const iconUrl = (icon: LineIcon) => `url("data:image/svg+xml,${encodeURIComponent(icon.svg)}")`

/**
 * Inline style drawing the icon on an element sized for it; its markup never
 * runs. Icons drawn in currentColor (Lucide's) are a mask showing the
 * element's background color; others, like template icon sets, have fixed
 * colors and are a background image.
 */
export const iconStyle = (icon: LineIcon) =>
  icon.svg.includes('currentColor')
    ? { maskImage: iconUrl(icon) }
    : { background: `${iconUrl(icon)} center / contain no-repeat` }

// Attributes that draw the icon; the rest (class, size, aria, Vue's scoped-style
// data attributes) are the editor's. The serializer adds xmlns itself.
const DRAWING: Record<string, true> = {
  viewBox: true,
  fill: true,
  stroke: true,
  'stroke-width': true,
  'stroke-linecap': true,
  'stroke-linejoin': true,
}

/** The icon a Lucide component rendered as `svg`. */
export function readIcon(svg: SVGSVGElement): LineIcon {
  // Lucide tags its root with "lucide lucide-<name>".
  const name = [...svg.classList].find((c) => c.startsWith('lucide-'))?.slice('lucide-'.length) ?? 'icon'
  const copy = svg.cloneNode(true) as SVGSVGElement
  for (const { name: attr } of [...copy.attributes]) if (!DRAWING[attr]) copy.removeAttribute(attr)
  return { name, svg: new XMLSerializer().serializeToString(copy) }
}
