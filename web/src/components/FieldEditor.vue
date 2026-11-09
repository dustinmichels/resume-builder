<script setup lang="ts">
import { ImagePlus } from '@lucide/vue'
import { computed, defineAsyncComponent, onMounted, useTemplateRef } from 'vue'
import { hasFeature, useTemplate } from '../composables/useTemplates'
import { addEntryIconPath, describe, getContact, getString, iconSlotOf, isRich, listOf, setString } from '../fieldPath'
import { spliceIcons } from '../lineIcons'
import type { LineIcon, PageDoc } from '../types'
import RichText from './RichText.vue'

// Lazy: the picker pulls in the whole Lucide set (~700 kB), only needed once a picker opens.
const IconPicker = defineAsyncComponent(() => import('./IconPicker.vue'))

// Edits one value of the document in place. The parent keys this component by
// the edited path, so every switch mounts fresh and focuses the new value.
const { doc, path } = defineProps<{ doc: PageDoc; path: string }>()
const emit = defineEmits<{ close: []; navigate: [path: string]; tab: [direction: 1 | -1] }>()

const template = useTemplate()
const label = computed(() => describe(doc, path))
const contact = computed(() => getContact(doc, path))
const rich = computed(() => isRich(doc, path))
const list = computed(() => listOf(doc, path))
const value = computed(() => getString(doc, path) ?? '')
const iconSlot = computed(() => iconSlotOf(doc, path))
/** For a column or legend line: the path of the icon beside it; for an entry's title: the slot adding an icon. */
const iconPath = computed(() => {
  const l = list.value
  if (l?.iconLines) return `${l.base.replace(/lines$/, 'icons')}.${l.index}`
  return hasFeature(template.value, 'entryIcons') ? addEntryIconPath(doc, path) : null
})

const richEl = useTemplateRef<InstanceType<typeof RichText>>('richEl')
const inputEl = useTemplateRef<HTMLInputElement>('inputEl')
const pickerEl = useTemplateRef<InstanceType<typeof IconPicker>>('pickerEl')

function focus() {
  const input = inputEl.value
  if (input) {
    input.focus({ preventScroll: true })
    input.setSelectionRange(input.value.length, input.value.length)
  } else if (pickerEl.value) {
    pickerEl.value.focus()
  } else {
    richEl.value?.focus(true, true)
  }
}
onMounted(focus)
defineExpose({ focus })

function set(text: string) {
  setString(doc, path, text)
}

function pickIcon(icon: LineIcon | null) {
  iconSlot.value?.set(icon)
  emit('close')
}

/** Enter adds an item below in lists and closes the editor for single values. */
function onEnter() {
  const l = list.value
  if (!l) return emit('close')
  l.array.splice(l.index + 1, 0, l.create())
  if (l.iconLines) spliceIcons(l.iconLines, l.index + 1, 0, 1)
  emit('navigate', `${l.base}.${l.index + 1}${l.suffix}`)
}

function removeItem() {
  const l = list.value
  if (!l) return
  l.array.splice(l.index, 1)
  if (l.iconLines) spliceIcons(l.iconLines, l.index, 1)
  if (!l.array.length) l.onEmpty?.()
  if (l.index > 0) emit('navigate', `${l.base}.${l.index - 1}${l.suffix}`)
  else if (l.array.length) emit('navigate', `${l.base}.0${l.suffix}`)
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
      <span class="fe-tools">
        <button
          v-if="iconPath"
          class="icon-btn"
          type="button"
          :title="list?.iconLines ? 'Icon' : 'Add an icon'"
          :aria-label="list?.iconLines ? 'Choose an icon' : 'Add an icon'"
          @click="emit('navigate', iconPath)"
        >
          <ImagePlus :size="15" aria-hidden="true" />
        </button>
        <button class="icon-btn" type="button" title="Close (Esc)" aria-label="Close" @click="emit('close')">×</button>
      </span>
    </div>

    <IconPicker v-if="iconSlot" ref="pickerEl" :current="iconSlot.icon" :icons="template.icons" @pick="pickIcon" />

    <div v-else-if="contact" class="fe-contact">
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
  grid-template-columns: minmax(0, 1fr);
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

.fe-tools {
  display: flex;
  align-items: center;
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
