<script setup lang="ts">
import { onBeforeUnmount, onMounted, reactive, shallowRef, useTemplateRef, watch } from 'vue'
import { escapeHtml, htmlToRich, richToHtml } from '../rich'
import PromptDialog, { type PromptRequest } from './PromptDialog.vue'

defineOptions({ inheritAttrs: false })

const model = defineModel<string>({ required: true })
const { placeholder = '', toolbar = false } = defineProps<{ placeholder?: string; toolbar?: boolean }>()
const emit = defineEmits<{ enter: []; 'backspace-empty': [] }>()

const el = useTemplateRef<HTMLDivElement>('el')
const prompt = shallowRef<PromptRequest | null>(null)

const MOD = /Mac|iPhone|iPad/.test(navigator.platform) ? '⌘' : 'Ctrl+'

// The DOM is rewritten only when the model changes from outside; while the
// user types it already serializes to the model, so the caret never jumps.
function render() {
  const node = el.value
  if (node && htmlToRich(node) !== model.value) node.innerHTML = richToHtml(model.value)
}
onMounted(render)
watch(model, render)

function sync() {
  const node = el.value
  if (!node) return
  const value = htmlToRich(node)
  // Drop a leftover <br> so :empty shows the placeholder.
  if (value === '' && node.textContent === '' && node.firstChild) node.replaceChildren()
  if (value !== model.value) model.value = value
  updateActive()
}

function anchorAt(node: Node | null): HTMLAnchorElement | null {
  const start = node instanceof Element ? node : node?.parentElement
  const a = start?.closest('a')
  return a && el.value?.contains(a) ? a : null
}

function ensureFocus() {
  if (document.activeElement !== el.value) focus()
}

function format(command: 'bold' | 'italic') {
  ensureFocus()
  document.execCommand(command)
  sync()
}

function editLink() {
  ensureFocus()
  const node = el.value
  const sel = getSelection()
  if (!node || !sel?.rangeCount || !node.contains(sel.anchorNode)) return
  const anchor = anchorAt(sel.anchorNode)
  if (anchor && sel.isCollapsed) sel.selectAllChildren(anchor)
  const range = sel.getRangeAt(0).cloneRange()
  prompt.value = {
    title: anchor ? 'Edit link' : 'Add link',
    message: anchor ? 'Leave the address empty to remove the link.' : undefined,
    value: anchor?.getAttribute('href') ?? '',
    placeholder: 'https://…',
    optional: true,
    confirmLabel: anchor ? 'Save' : 'Add link',
    // Applied once the dialog has closed (a macrotask after Vue's updates): the
    // page behind a modal dialog is inert, so it can't take focus or edits.
    run: (url) =>
      setTimeout(() => {
        node.focus({ preventScroll: true })
        sel.removeAllRanges()
        sel.addRange(range)
        if (!url) document.execCommand('unlink')
        else if (range.collapsed) document.execCommand('insertHTML', false, `<a href="${escapeHtml(url)}">${escapeHtml(url)}</a>`)
        else document.execCommand('createLink', false, url)
        sync()
      }),
  }
}

function onKeydown(e: KeyboardEvent) {
  if (e.isComposing) return
  if ((e.metaKey || e.ctrlKey) && !e.altKey && !e.shiftKey) {
    const key = e.key.toLowerCase()
    if (key === 'b' || key === 'i' || key === 'k' || key === 'u') e.preventDefault()
    if (key === 'b') format('bold')
    else if (key === 'i') format('italic')
    else if (key === 'k') editLink()
    // ⌘U (underline) is swallowed: the markup has no underline.
    return
  }
  if (e.key === 'Enter' && !e.shiftKey) {
    // Shift+Enter keeps the browser's line break (<br>).
    e.preventDefault()
    emit('enter')
  } else if (e.key === 'Backspace' && !el.value?.textContent) {
    e.preventDefault()
    emit('backspace-empty')
  }
}

function onPaste(e: ClipboardEvent) {
  e.preventDefault()
  const text = e.clipboardData?.getData('text/plain').replace(/\r\n?/g, '\n')
  if (text) document.execCommand('insertText', false, text)
}

const active = reactive({ bold: false, italic: false, link: false })

function updateActive() {
  if (!toolbar || !el.value || document.activeElement !== el.value) return
  active.bold = document.queryCommandState('bold')
  active.italic = document.queryCommandState('italic')
  active.link = anchorAt(getSelection()?.anchorNode ?? null) !== null
}

onMounted(() => {
  if (toolbar) document.addEventListener('selectionchange', updateActive)
})
onBeforeUnmount(() => document.removeEventListener('selectionchange', updateActive))

function focus(atEnd = true, preventScroll = false) {
  const node = el.value
  if (!node) return
  node.focus({ preventScroll })
  const range = document.createRange()
  range.selectNodeContents(node)
  range.collapse(!atEnd)
  const sel = getSelection()
  sel?.removeAllRanges()
  sel?.addRange(range)
}

defineExpose({ focus })
</script>

<template>
  <div class="rich-text">
    <div v-if="toolbar" class="rt-toolbar" @mousedown.prevent>
      <button type="button" class="rt-btn" :aria-pressed="active.bold" :title="`Bold (${MOD}B)`" @click="format('bold')">
        <b>B</b>
      </button>
      <button type="button" class="rt-btn" :aria-pressed="active.italic" :title="`Italic (${MOD}I)`" @click="format('italic')">
        <i>I</i>
      </button>
      <button type="button" class="rt-btn" :aria-pressed="active.link" :title="`Link (${MOD}K)`" @click="editLink">
        Link
      </button>
      <span class="rt-hint">Shift+Enter for a line break</span>
    </div>
    <div
      ref="el"
      v-bind="$attrs"
      class="input rt-editable"
      contenteditable="true"
      role="textbox"
      aria-multiline="true"
      :data-placeholder="placeholder"
      @input="sync"
      @focus="updateActive"
      @keydown="onKeydown"
      @paste="onPaste"
    />
    <PromptDialog v-model="prompt" />
  </div>
</template>

<style scoped>
.rich-text {
  flex: 1 1 auto;
  min-width: 0;
}

.rt-toolbar {
  display: flex;
  align-items: center;
  gap: 2px;
  margin-bottom: 4px;
}

.rt-btn {
  min-width: 26px;
  height: 24px;
  padding: 0 6px;
  border: 1px solid transparent;
  border-radius: var(--radius);
  background: transparent;
  color: var(--muted);
  font-size: 13px;
  cursor: pointer;
}

.rt-btn:hover {
  background: var(--sunken);
  color: var(--ink);
}

.rt-btn[aria-pressed='true'] {
  border-color: var(--rule);
  background: var(--sunken);
  color: var(--ink);
}

.rt-hint {
  margin-left: auto;
  color: var(--faint);
  font-size: 11px;
}

.rt-editable {
  min-height: calc(1.45em + 12px);
  white-space: pre-wrap;
  overflow-wrap: anywhere;
  cursor: text;
}

.rt-editable:empty::before {
  content: attr(data-placeholder);
  color: var(--faint);
  pointer-events: none;
}

.rt-editable :deep(a) {
  color: var(--accent);
  text-decoration: underline;
}
</style>
