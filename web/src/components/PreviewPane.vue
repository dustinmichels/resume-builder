<script setup lang="ts">
import markdownit from 'markdown-it'
import { computed, onBeforeUnmount, onMounted, shallowRef, useTemplateRef, watch } from 'vue'
import { describe, resolves } from '../fieldPath'
import type { FieldBox, Preview, ResumeDoc } from '../types'
import FieldEditor from './FieldEditor.vue'

// Defaults keep rendering safe for v-html: raw HTML is escaped and
// javascript:/vbscript:/file:/data: links are dropped.
const md = markdownit()
const renderLinkOpen = md.renderer.rules.link_open ?? ((tokens, idx, options, _env, self) => self.renderToken(tokens, idx, options))
md.renderer.rules.link_open = (tokens, idx, options, env, self) => {
  // Open resume links in a new tab instead of navigating away from the editor.
  tokens[idx].attrSet('target', '_blank')
  tokens[idx].attrSet('rel', 'noopener noreferrer')
  return renderLinkOpen(tokens, idx, options, env, self)
}

const { preview, pageUrls, pending, error, doc } = defineProps<{
  preview: Preview | null
  pageUrls: string[]
  pending: boolean
  error: string | null
  /** Edited in place by clicking values on the Page tab. */
  doc: ResumeDoc | null
}>()

type Tab = 'page' | 'text' | 'markdown' | 'checks'
const tab = shallowRef<Tab>('page')
const textMode = shallowRef<'raw' | 'layout'>('raw')
const markdownMode = shallowRef<'rendered' | 'source'>('rendered')

const markdownHtml = computed(() => md.render(preview?.markdown ?? ''))

const issues = computed(() => preview?.checks.filter((c) => c.level === 'warn' || c.level === 'error') ?? [])

// Checks first by severity, then the rest.
const SEVERITY = { error: 0, warn: 1, info: 2, ok: 3 } as const
const sortedChecks = computed(() =>
  [...(preview?.checks ?? [])].sort((a, b) => SEVERITY[a.level] - SEVERITY[b.level]),
)

const textPages = computed(() => {
  const text = textMode.value === 'raw' ? preview?.text : preview?.layoutText
  return (text ?? '').split('\f').filter((p) => p.trim() !== '')
})

const ICONS = { ok: '✓', info: 'i', warn: '!', error: '✕' } as const

// ── Click-to-edit on the Page tab ───────────────────────────────────────────
// Each rendered value gets a transparent target over the page image; clicking
// one opens a FieldEditor card that writes straight into `doc`.

/** US Letter in points; field boxes are measured from the page's top-left. */
const PAGE_W = 612
const PAGE_H = 792
const CARD_GAP = 6

const fieldsByPage = computed(() => {
  const byPage = new Map<number, FieldBox[]>()
  for (const field of preview?.fields ?? []) {
    const list = byPage.get(field.page)
    if (list) list.push(field)
    else byPage.set(field.page, [field])
  }
  return byPage
})

/** Field paths in page order, for Tab and Shift+Tab. */
const fieldOrder = computed(() => [...new Set((preview?.fields ?? []).map((f) => f.path))])

function boxStyle(f: FieldBox) {
  return {
    left: `${(f.x / PAGE_W) * 100}%`,
    top: `${(f.y / PAGE_H) * 100}%`,
    width: `${(f.w / PAGE_W) * 100}%`,
    height: `${(f.h / PAGE_H) * 100}%`,
  }
}

const activePath = shallowRef<string | null>(null)
/** Bumped on every switch so the editor remounts and focuses the new value. */
const editorKey = shallowRef(0)
/** The active field's last known box, kept while a new empty item has none yet. */
const anchor = shallowRef<FieldBox | null>(null)
const cardPos = shallowRef<{ left: number; top: number; above: boolean } | null>(null)

const scrollEl = useTemplateRef<HTMLElement>('scrollEl')
const pagesEl = useTemplateRef<HTMLElement>('pagesEl')
const cardEl = useTemplateRef<HTMLElement>('cardEl')
const editor = useTemplateRef<InstanceType<typeof FieldEditor>>('editor')

/** The union of a path's boxes on the first page it appears on. */
function boxOf(path: string): FieldBox | null {
  const boxes = preview?.fields.filter((f) => f.path === path) ?? []
  const page = boxes[0]?.page
  if (page === undefined) return null
  const same = boxes.filter((b) => b.page === page)
  const x = Math.min(...same.map((b) => b.x))
  const y = Math.min(...same.map((b) => b.y))
  const w = Math.max(...same.map((b) => b.x + b.w)) - x
  const h = Math.max(...same.map((b) => b.y + b.h)) - y
  return { path, page, x, y, w, h }
}

watch(
  () => activePath.value && boxOf(activePath.value),
  (box) => {
    if (box) anchor.value = box
  },
)

function navigate(path: string) {
  activePath.value = path
  editorKey.value++
}

function open(path: string) {
  if (path === activePath.value) editor.value?.focus()
  else navigate(path)
}

function close() {
  activePath.value = null
  anchor.value = null
  cardPos.value = null
}

function step(direction: 1 | -1) {
  const order = fieldOrder.value
  if (!order.length) return
  let next = order.indexOf(activePath.value ?? '')
  if (next >= 0) {
    next += direction
  } else {
    // A new empty item has no box yet; it sits right after the anchor's field.
    const at = anchor.value ? order.indexOf(anchor.value.path) : -1
    next = at < 0 ? (direction > 0 ? 0 : -1) : direction > 0 ? at + 1 : at
  }
  navigate(order[(next + order.length) % order.length]!)
}

// Another resume, another tab, or a sidebar change that removed the value.
watch(() => doc, close)
watch(tab, close)
watch(
  () => activePath.value !== null && !(doc && resolves(doc, activePath.value)),
  (stale) => {
    if (stale) close()
  },
)

/**
 * Positions the card just below the anchor box, or above it when it would
 * run past the bottom of the scroll viewport. `reposition` (a new field)
 * also scrolls the field into view and re-decides above/below; otherwise the
 * card keeps its side so it doesn't jump while the page re-renders.
 */
function place(reposition: boolean) {
  const pages = pagesEl.value
  const scroller = scrollEl.value
  const box = anchor.value
  if (!pages || !scroller || !box) return
  const page = pages.querySelectorAll<HTMLElement>(':scope > .page')[box.page - 1]
  if (!page) return
  const scale = page.offsetWidth / PAGE_W
  const left = page.offsetLeft + box.x * scale
  const top = page.offsetTop + box.y * (page.offsetHeight / PAGE_H)
  const bottom = top + box.h * (page.offsetHeight / PAGE_H)
  const card = cardEl.value
  let above = cardPos.value?.above ?? false
  if (reposition) {
    const view = scroller.getBoundingClientRect()
    let viewTop = pages.getBoundingClientRect().top + top
    if (viewTop < view.top || viewTop + bottom - top > view.bottom) {
      scroller.scrollTop += viewTop - view.top - view.height / 3
      viewTop = pages.getBoundingClientRect().top + top
    }
    const cardHeight = card?.offsetHeight || 160
    const viewBottom = viewTop + bottom - top
    above = viewBottom + CARD_GAP + cardHeight > view.bottom && viewTop - CARD_GAP - cardHeight >= view.top
  }
  const maxLeft = pages.clientWidth - (card?.offsetWidth || 380)
  cardPos.value = { left: Math.max(0, Math.min(left, maxLeft)), top: above ? top - CARD_GAP : bottom + CARD_GAP, above }
}

watch([anchor, editorKey], ([, key], [, oldKey]) => place(key !== oldKey), { flush: 'post' })

watch(pagesEl, (el, _old, onCleanup) => {
  if (!el) return
  const observer = new ResizeObserver(() => place(false))
  observer.observe(el)
  onCleanup(() => observer.disconnect())
})

const cardStyle = computed(() => {
  const pos = cardPos.value
  if (!pos) return undefined
  return { left: `${pos.left}px`, top: `${pos.top}px`, transform: pos.above ? 'translateY(-100%)' : undefined }
})

function onPointerDown(e: PointerEvent) {
  if (!activePath.value) return
  const target = e.target as Element
  if (cardEl.value?.contains(target) || target.closest?.('.hit')) return
  close()
}
onMounted(() => document.addEventListener('pointerdown', onPointerDown, true))
onBeforeUnmount(() => document.removeEventListener('pointerdown', onPointerDown, true))
</script>

<template>
  <div class="pane">
    <div class="bar">
      <div class="tabs" role="tablist">
        <button role="tab" :aria-selected="tab === 'page'" class="tab" @click="tab = 'page'">Page</button>
        <button role="tab" :aria-selected="tab === 'text'" class="tab" @click="tab = 'text'">What machines read</button>
        <button role="tab" :aria-selected="tab === 'markdown'" class="tab" @click="tab = 'markdown'">Markdown</button>
        <button role="tab" :aria-selected="tab === 'checks'" class="tab" @click="tab = 'checks'">
          Checks
          <span v-if="preview" class="badge" :class="issues.length ? 'bad' : 'good'">
            {{ issues.length ? issues.length : '✓' }}
          </span>
        </button>
      </div>
      <span class="status" role="status">
        <template v-if="pending">Rendering…</template>
        <template v-else-if="preview">{{ preview.pages.length }} {{ preview.pages.length === 1 ? 'page' : 'pages' }}</template>
      </span>
    </div>

    <div v-if="error" class="render-error" role="alert">
      <strong>The resume didn’t render.</strong> The preview shows the last good version.
      <pre>{{ error }}</pre>
    </div>
    <div v-if="preview?.warnings.length" class="render-warnings">
      <strong>Typst warnings</strong>
      <pre>{{ preview.warnings.join('\n') }}</pre>
    </div>

    <div ref="scrollEl" class="scroll" :class="{ dim: pending }">
      <div v-if="!preview && !error" class="empty">Rendering the first preview…</div>

      <div v-else-if="tab === 'page'" ref="pagesEl" class="pages">
        <div v-for="(url, i) in pageUrls" :key="i" class="page">
          <img :src="url" class="page-img" :alt="`Resume page ${i + 1}`" />
          <template v-if="doc">
            <div
              v-for="(field, j) in fieldsByPage.get(i + 1)"
              :key="j"
              class="hit"
              :class="{ active: field.path === activePath }"
              :style="boxStyle(field)"
              :title="describe(doc, field.path)"
              @mousedown.prevent
              @click="open(field.path)"
            />
          </template>
        </div>
        <div v-if="doc && activePath" ref="cardEl" class="card" :class="{ placed: cardPos }" :style="cardStyle">
          <FieldEditor
            ref="editor"
            :key="editorKey"
            :doc="doc"
            :path="activePath"
            @close="close"
            @navigate="navigate"
            @tab="step"
          />
        </div>
      </div>

      <div v-else-if="tab === 'text'" class="text-view">
        <div class="text-head">
          <p class="explain">
            The PDF’s text layer, extracted with <code>pdftotext</code>. This is what applicant tracking systems and
            other parsers start from.
          </p>
          <div class="seg" role="radiogroup" aria-label="Extraction mode">
            <button role="radio" :aria-checked="textMode === 'raw'" @click="textMode = 'raw'">Reading order</button>
            <button role="radio" :aria-checked="textMode === 'layout'" @click="textMode = 'layout'">Layout</button>
          </div>
        </div>
        <template v-for="(page, i) in textPages" :key="i">
          <div class="page-break">Page {{ i + 1 }}</div>
          <pre class="text" :class="textMode">{{ page }}</pre>
        </template>
      </div>

      <div v-else-if="tab === 'markdown'" class="text-view">
        <div class="text-head">
          <p class="explain">The Markdown download, as a Markdown viewer shows it.</p>
          <div class="seg" role="radiogroup" aria-label="Markdown view">
            <button role="radio" :aria-checked="markdownMode === 'rendered'" @click="markdownMode = 'rendered'">
              Rendered
            </button>
            <button role="radio" :aria-checked="markdownMode === 'source'" @click="markdownMode = 'source'">Source</button>
          </div>
        </div>
        <article v-if="markdownMode === 'rendered'" class="markdown" v-html="markdownHtml" />
        <pre v-else class="text raw">{{ preview?.markdown }}</pre>
      </div>

      <ul v-else class="checks">
        <li v-for="check in sortedChecks" :key="check.title" class="check" :class="check.level">
          <span class="check-icon" aria-hidden="true">{{ ICONS[check.level] }}</span>
          <div>
            <div class="check-title">{{ check.title }}</div>
            <pre v-if="check.detail" class="check-detail">{{ check.detail }}</pre>
          </div>
        </li>
      </ul>
    </div>
  </div>
</template>

<style scoped>
.pane {
  display: flex;
  flex-direction: column;
  height: 100%;
  min-height: 0;
  background: var(--pasteboard);
}

.bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 16px;
  border-bottom: 1px solid #c9cfd7;
  background: #e9ecef;
}

.tabs {
  display: flex;
  gap: 2px;
}

.tab {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 5px 12px;
  border: 0;
  border-radius: var(--radius);
  background: transparent;
  color: var(--muted);
  cursor: pointer;
}

.tab:hover {
  color: var(--ink);
}

.tab[aria-selected='true'] {
  background: var(--panel);
  color: var(--ink);
  box-shadow: 0 1px 2px rgb(0 0 0 / 0.08);
}

.badge {
  min-width: 18px;
  padding: 0 5px;
  border-radius: 9px;
  font-size: 11px;
  font-weight: 700;
  line-height: 18px;
  text-align: center;
  color: #fff;
}

.badge.good {
  background: var(--ok);
}

.badge.bad {
  background: var(--warn);
}

.status {
  font-size: 12px;
  color: var(--muted);
}

.render-error,
.render-warnings {
  margin: 12px 16px 0;
  padding: 10px 12px;
  border-radius: var(--radius);
  font-size: 13px;
}

.render-error {
  border: 1px solid color-mix(in srgb, var(--error) 40%, transparent);
  background: #fdf1f0;
  color: var(--error);
}

.render-warnings {
  border: 1px solid color-mix(in srgb, var(--warn) 35%, transparent);
  background: #fdf6ea;
  color: var(--warn);
}

.render-error pre,
.render-warnings pre {
  margin: 6px 0 0;
  white-space: pre-wrap;
  font-size: 12px;
}

.scroll {
  flex: 1;
  min-height: 0;
  overflow: auto;
  padding: 24px;
}

/* Dim the content while rendering, but never the field editor. */
.scroll :is(.empty, .page-img, .text-view, .checks) {
  transition: opacity 0.2s;
}

.scroll.dim :is(.empty, .page-img, .text-view, .checks) {
  opacity: 0.85;
}

.empty {
  color: var(--muted);
  text-align: center;
  padding-top: 80px;
}

.pages {
  position: relative;
  display: grid;
  justify-items: center;
  gap: 24px;
}

.page {
  position: relative;
  width: min(100%, 816px);
  aspect-ratio: 8.5 / 11;
  background: #fff;
  box-shadow:
    0 1px 2px rgb(0 0 0 / 0.12),
    0 8px 24px rgb(0 0 0 / 0.1);
}

.page-img {
  display: block;
  width: 100%;
  height: 100%;
}

/* Click-to-edit targets over each rendered value. */
.hit {
  position: absolute;
  border-radius: 2px;
  cursor: text;
}

.hit:hover {
  outline: 1px solid color-mix(in srgb, var(--accent) 55%, transparent);
  outline-offset: 2px;
  background: color-mix(in srgb, var(--accent) 5%, transparent);
}

.hit.active {
  outline: 2px solid var(--accent);
  outline-offset: 2px;
}

.card {
  position: absolute;
  z-index: 2;
  width: min(380px, 100%);
}

.card:not(.placed) {
  opacity: 0;
}

.text-view {
  max-width: 816px;
  margin: 0 auto;
}

.text-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 12px;
}

.explain {
  margin: 0;
  color: var(--muted);
  font-size: 13px;
}

.seg {
  display: flex;
  flex: none;
  padding: 2px;
  border-radius: var(--radius);
  background: #cfd5dc;
}

.seg button {
  padding: 3px 10px;
  border: 0;
  border-radius: 4px;
  background: transparent;
  font-size: 12px;
  cursor: pointer;
}

.seg button[aria-checked='true'] {
  background: var(--panel);
}

.page-break {
  margin: 16px 0 6px;
  color: var(--muted);
  font-size: 12px;
}

.text {
  margin: 0;
  padding: 16px 20px;
  overflow-x: auto;
  border-radius: var(--radius);
  background: var(--panel);
  font: 12px/1.55 ui-monospace, 'SF Mono', Menlo, monospace;
  white-space: pre;
}

.text.raw {
  white-space: pre-wrap;
}

.markdown {
  padding: 28px 36px;
  border-radius: var(--radius);
  background: var(--panel);
  line-height: 1.55;
}

.markdown :deep(:is(h1, h2, h3)) {
  margin: 1.4em 0 0.5em;
  line-height: 1.25;
}

.markdown :deep(h1) {
  margin-top: 0;
  font-size: 26px;
}

.markdown :deep(h2) {
  padding-bottom: 4px;
  border-bottom: 1px solid var(--rule);
  font-size: 19px;
}

.markdown :deep(h3) {
  font-size: 15px;
}

.markdown :deep(:is(p, ul)) {
  margin: 0 0 0.8em;
}

.markdown :deep(ul) {
  padding-left: 1.4em;
}

.markdown :deep(a) {
  color: var(--accent);
}

.checks {
  max-width: 720px;
  margin: 0 auto;
  padding: 0;
  list-style: none;
  display: grid;
  gap: 8px;
}

.check {
  display: flex;
  gap: 12px;
  padding: 12px 14px;
  border-radius: var(--radius);
  background: var(--panel);
}

.check-icon {
  flex: none;
  display: grid;
  place-items: center;
  width: 22px;
  height: 22px;
  border-radius: 50%;
  font-size: 12px;
  font-weight: 700;
  color: #fff;
}

.check.ok .check-icon {
  background: var(--ok);
}

.check.info .check-icon {
  background: var(--faint);
}

.check.warn .check-icon {
  background: var(--warn);
}

.check.error .check-icon {
  background: var(--error);
}

.check-title {
  font-weight: 600;
  padding-top: 1px;
}

.check-detail {
  margin: 6px 0 0;
  color: var(--muted);
  font: 12px/1.5 ui-monospace, 'SF Mono', Menlo, monospace;
  white-space: pre-wrap;
}
</style>
