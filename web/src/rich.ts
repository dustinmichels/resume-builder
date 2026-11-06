// Inline rich text: the markup subset stored in the resume (see types.ts) and
// the HTML a contenteditable shows for it.
//
// The pattern must match `rich-pattern` in template/resume.typ and
// `richPattern` in server/rich.go: matched leftmost-first, bold and italic
// contents parse recursively, link labels are literal, and there is no escape.
const RICH = /\*\*\*(.+?)\*\*\*|\*\*(.+?)\*\*|\*(.+?)\*|\[([^\]]+)\]\(([^)\s]+)\)/g

const ESCAPES: Record<string, string> = { '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }

export const escapeHtml = (s: string) => s.replace(/[&<>"']/g, (c) => ESCAPES[c]!)

/** Literal text as HTML: escaped, with line breaks as <br>. */
const textHtml = (s: string) => escapeHtml(s).replace(/\r\n?|\n/g, '<br>')

/** Link targets the editor turns into real links; anything else stays literal text. */
const SAFE_URL = /^(?:https?:|mailto:|tel:)/i

function parseToHtml(src: string): string {
  let out = ''
  let pos = 0
  for (const m of src.matchAll(RICH)) {
    const [whole, boldItalic, bold, italic, label, url] = m
    out += textHtml(src.slice(pos, m.index))
    if (boldItalic !== undefined) out += `<b><i>${parseToHtml(boldItalic)}</i></b>`
    else if (bold !== undefined) out += `<b>${parseToHtml(bold)}</b>`
    else if (italic !== undefined) out += `<i>${parseToHtml(italic)}</i>`
    // Unsafe links keep their source text, so editing around them loses nothing.
    else if (SAFE_URL.test(url!)) out += `<a href="${escapeHtml(url!)}">${textHtml(label!)}</a>`
    else out += textHtml(whole)
    pos = m.index + whole.length
  }
  return out + textHtml(src.slice(pos))
}

/** Renders markup as HTML for a contenteditable. All text and attributes are escaped. */
export function richToHtml(src: string): string {
  const html = parseToHtml(src)
  // A trailing <br> only shows its empty line when followed by another one;
  // htmlToRich drops that placeholder again.
  return html.endsWith('<br>') ? `${html}<br>` : html
}

/** The text markup shows, formatting dropped. */
export function richToText(src: string): string {
  return src.replace(RICH, (_, boldItalic, bold, italic, label) => label ?? richToText(boldItalic ?? bold ?? italic))
}

const collator = new Intl.Collator(undefined, { sensitivity: 'accent' })

/**
 * Sorts a comma-separated list A–Z, ignoring case and formatting. Commas inside
 * parentheses, brackets, bold, italic or links don't split, so
 * "R, Python (*Pandas, Django*)" has two items.
 */
export function sortCommaList(src: string): string {
  // Formatted runs and links are skipped whole: start index → end index.
  const spanEnd = new Map([...src.matchAll(RICH)].map((m) => [m.index, m.index + m[0].length]))
  const items: string[] = []
  let start = 0
  let depth = 0
  for (let i = 0; i < src.length; i++) {
    const end = spanEnd.get(i)
    if (end !== undefined) {
      i = end - 1
      continue
    }
    const c = src[i]
    if (c === '(' || c === '[') depth++
    else if ((c === ')' || c === ']') && depth > 0) depth--
    else if (c === ',' && depth === 0) {
      items.push(src.slice(start, i))
      start = i + 1
    }
  }
  items.push(src.slice(start))
  return items
    .map((s) => s.trim())
    .filter(Boolean)
    .sort((a, b) => collator.compare(richToText(a), richToText(b)))
    .join(', ')
}

interface Format {
  bold: boolean
  italic: boolean
  href: string | null
}

interface Run extends Format {
  text: string
}

const PLAIN: Format = { bold: false, italic: false, href: null }

function formatOf(el: Element, outer: Format): Format {
  let { bold, italic, href } = outer
  const tag = el.tagName
  if (tag === 'B' || tag === 'STRONG') bold = true
  if (tag === 'I' || tag === 'EM') italic = true
  if (tag === 'A') href = el.getAttribute('href') || href
  // Spans from execCommand or pasted HTML carry formatting as inline styles.
  const style = (el as Partial<HTMLElement>).style
  if (style?.fontWeight) bold = style.fontWeight === 'bold' || style.fontWeight === 'bolder' || Number(style.fontWeight) >= 600
  if (style?.fontStyle) italic = style.fontStyle === 'italic' || style.fontStyle.startsWith('oblique')
  return { bold, italic, href }
}

function wrap(core: string, run: Format): string {
  // Markup can't hold whitespace or ")" in a link target, so percent-encode them.
  if (run.href) return `[${core}](${run.href.replace(/\s/g, encodeURIComponent).replace(/\)/g, '%29')})`
  if (run.bold && run.italic) return `***${core}***`
  if (run.bold) return `**${core}**`
  return `*${core}*`
}

/**
 * Serializes a contenteditable's DOM back to markup. Formatting is flattened
 * into runs and emitted without nesting; formatting inside a link label is
 * dropped. <br> and block boundaries become "\n", and no formatted run spans
 * a line break.
 */
export function htmlToRich(root: Node): string {
  const runs: Run[] = []
  let block = false // a block boundary since the last content
  let trailingBr = false // the last content was a <br>

  const add = (text: string, format: Format) => {
    const last = runs.at(-1)
    // Link labels are literal, so formatting inside a link is dropped.
    const fmt = format.href ? { ...PLAIN, href: format.href } : format
    if (last && text !== '\n' && last.text !== '\n' && last.bold === fmt.bold && last.italic === fmt.italic && last.href === fmt.href) {
      last.text += text
    } else {
      runs.push({ text, ...fmt })
    }
  }

  const push = (text: string, format: Format) => {
    if (block && runs.length && runs.at(-1)!.text !== '\n') add('\n', PLAIN)
    block = false
    trailingBr = false
    text.split(/\r\n?|\n/).forEach((line, i) => {
      if (i > 0) add('\n', PLAIN)
      if (line) add(line, format)
    })
  }

  const walk = (node: Node, format: Format) => {
    for (const child of node.childNodes) {
      if (child.nodeType === Node.TEXT_NODE) {
        if (child.nodeValue) push(child.nodeValue, format)
      } else if (child.nodeType === Node.ELEMENT_NODE) {
        const el = child as Element
        if (el.tagName === 'BR') {
          push('\n', PLAIN)
          trailingBr = true
          continue
        }
        const isBlock = el.tagName === 'DIV' || el.tagName === 'P'
        if (isBlock) block = true
        walk(el, formatOf(el, format))
        if (isBlock) block = true
      }
    }
  }
  walk(root, PLAIN)
  // The browser keeps a placeholder <br> after a trailing line break.
  if (trailingBr) runs.pop()

  let out = ''
  for (const run of runs) {
    if (!run.bold && !run.italic && !run.href) {
      out += run.text
      continue
    }
    // Markers hug the text: surrounding whitespace goes outside them.
    const [, lead, core, trail] = /^(\s*)([\s\S]*?)(\s*)$/.exec(run.text)!
    out += core ? lead! + wrap(core, run) + trail! : run.text
  }
  return out
}
