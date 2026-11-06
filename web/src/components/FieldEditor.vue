<script setup lang="ts">
import { computed, onMounted, useTemplateRef } from 'vue'
import { describe, getContact, getString, isRich, listOf, setString } from '../fieldPath'
import type { ResumeDoc } from '../types'
import RichText from './RichText.vue'

// Edits one value of the document in place. The parent keys this component by
// the edited path, so every switch mounts fresh and focuses the new value.
const { doc, path } = defineProps<{ doc: ResumeDoc; path: string }>()
const emit = defineEmits<{ close: []; navigate: [path: string]; tab: [direction: 1 | -1] }>()

const label = computed(() => describe(doc, path))
const contact = computed(() => getContact(doc, path))
const rich = computed(() => isRich(path))
const list = computed(() => listOf(doc, path))
const value = computed(() => getString(doc, path) ?? '')

const richEl = useTemplateRef<InstanceType<typeof RichText>>('richEl')
const inputEl = useTemplateRef<HTMLInputElement>('inputEl')

function focus() {
  const input = inputEl.value
  if (input) {
    input.focus({ preventScroll: true })
    input.setSelectionRange(input.value.length, input.value.length)
  } else {
    richEl.value?.focus(true, true)
  }
}
onMounted(focus)
defineExpose({ focus })

function set(text: string) {
  setString(doc, path, text)
}

/** Enter adds an item below in lists and closes the editor for single values. */
function onEnter() {
  const l = list.value
  if (!l) return emit('close')
  l.array.splice(l.index + 1, 0, '')
  emit('navigate', `${l.base}.${l.index + 1}`)
}

function removeItem() {
  const l = list.value
  if (!l) return
  l.array.splice(l.index, 1)
  if (l.index > 0) emit('navigate', `${l.base}.${l.index - 1}`)
  else if (l.array.length) emit('navigate', `${l.base}.0`)
  else emit('close')
}

function onInputKeydown(e: KeyboardEvent) {
  if (e.isComposing) return
  if (e.key === 'Enter') {
    e.preventDefault()
    onEnter()
  } else if (e.key === 'Backspace' && list.value && value.value === '') {
    e.preventDefault()
    removeItem()
  }
}

function onKeydown(e: KeyboardEvent) {
  if (e.isComposing) return
  if (e.key === 'Escape') {
    e.preventDefault()
    emit('close')
  } else if (e.key === 'Tab') {
    const target = e.target as HTMLElement
    // Buttons keep normal focus order; Tab also moves between a contact's two inputs.
    if (target.tagName === 'BUTTON' || target.dataset.contact === (e.shiftKey ? 'url' : 'text')) return
    e.preventDefault()
    emit('tab', e.shiftKey ? -1 : 1)
  }
}
</script>

<template>
  <div class="field-editor" role="dialog" :aria-label="`Edit ${label}`" @keydown="onKeydown">
    <div class="fe-head">
      <span class="fe-label">{{ label }}</span>
      <button class="icon-btn" type="button" title="Close (Esc)" aria-label="Close" @click="emit('close')">×</button>
    </div>

    <div v-if="contact" class="fe-contact">
      <input
        ref="inputEl"
        v-model="contact.text"
        class="input"
        data-contact="text"
        placeholder="Shown text"
        aria-label="Shown text"
        @keydown="onInputKeydown"
      />
      <input
        v-model="contact.url"
        class="input"
        data-contact="url"
        placeholder="Link (optional)"
        aria-label="Link"
        @keydown="onInputKeydown"
      />
    </div>
    <RichText
      v-else-if="rich"
      ref="richEl"
      :model-value="value"
      toolbar
      :aria-label="label"
      @update:model-value="set"
      @enter="onEnter"
      @backspace-empty="removeItem"
    />
    <input
      v-else
      ref="inputEl"
      class="input"
      :value="value"
      :aria-label="label"
      @input="set(($event.target as HTMLInputElement).value)"
      @keydown="onInputKeydown"
    />

    <div v-if="list" class="fe-actions">
      <button class="btn" type="button" @click="onEnter">+ Add below</button>
      <button class="btn fe-delete" type="button" @click="removeItem">Delete</button>
      <span class="fe-keys">Enter adds · Tab moves on</span>
    </div>
  </div>
</template>

<style scoped>
.field-editor {
  display: grid;
  gap: 6px;
  width: min(380px, 100%);
  padding: 6px 10px 10px;
  border: 1px solid var(--rule);
  border-radius: 8px;
  background: var(--panel);
  box-shadow:
    0 2px 4px rgb(0 0 0 / 0.08),
    0 10px 28px rgb(0 0 0 / 0.16);
}

.fe-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-right: -6px;
}

.fe-label {
  font-size: 12px;
  font-weight: 600;
  color: var(--muted);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.fe-contact {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 6px;
}

.fe-actions {
  display: flex;
  align-items: center;
  gap: 6px;
}

.fe-actions .btn {
  padding: 3px 9px;
  font-size: 12px;
}

.fe-delete:hover {
  color: var(--error);
}

.fe-keys {
  margin-left: auto;
  color: var(--faint);
  font-size: 11px;
}
</style>
