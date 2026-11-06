// A column's icons sit in an array parallel to its lines (Column.icons). These
// keep each icon beside its line as lines are inserted, removed and reordered.
import type { Column, LineIcon } from './types'

/** Mirrors `lines.splice(start, deleteCount, ...insertCount blank lines)`. */
export function spliceIcons(col: Column, start: number, deleteCount: number, insertCount = 0) {
  if (!col.icons || start >= col.icons.length) return
  col.icons.splice(start, deleteCount, ...Array<null>(insertCount).fill(null))
  trim(col)
}

/** Mirrors moving the line at `from` to `to`. */
export function moveIcon(col: Column, from: number, to: number) {
  if (!col.icons || from === to) return
  pad(col.icons, Math.max(from, to))
  const [icon] = col.icons.splice(from, 1)
  col.icons.splice(to, 0, icon ?? null)
  trim(col)
}

export function setIcon(col: Column, index: number, icon: LineIcon | null) {
  const icons = (col.icons ??= [])
  pad(icons, index)
  icons[index] = icon
  trim(col)
}

function pad(icons: (LineIcon | null)[], index: number) {
  while (icons.length <= index) icons.push(null)
}

/** Drops trailing empty slots, and the array once it holds no icon. */
function trim(col: Column) {
  const icons = col.icons!
  while (icons.length && !icons.at(-1)) icons.pop()
  if (!icons.length) delete col.icons
}

/** For a CSS mask: the icon shows in currentColor and its markup never runs. */
export const iconUrl = (icon: LineIcon) => `url("data:image/svg+xml,${encodeURIComponent(icon.svg)}")`

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
