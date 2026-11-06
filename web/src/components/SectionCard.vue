<script setup lang="ts">
import { computed, shallowRef } from 'vue'
import type { Section } from '../types'
import ColumnsEditor from './ColumnsEditor.vue'
import EntriesEditor from './EntriesEditor.vue'
import StringList from './StringList.vue'

const section = defineModel<Section>({ required: true })
const emit = defineEmits<{ remove: [] }>()

const open = shallowRef(false)

const KIND_LABELS = { text: 'Paragraphs', entries: 'Entries', columns: 'Columns' } as const

const heading = computed(() => {
  const s = section.value
  if (s.title) return s.title
  if (s.kind === 'columns' && s.variant === 'sections') {
    return s.columns.map((c) => c.title).filter(Boolean).join(' + ') || 'Untitled columns'
  }
  return 'Untitled section'
})

const count = computed(() => {
  const s = section.value
  if (s.kind === 'text') return s.paragraphs.length
  if (s.kind === 'entries') return s.items.filter((i) => i.type === 'entry').length
  return s.columns.length
})
</script>

<template>
  <section class="section" :class="{ open }">
    <header class="section-head">
      <span class="handle h-section" title="Drag to reorder" aria-hidden="true">⠿</span>
      <button class="toggle" type="button" :aria-expanded="open" @click="open = !open">
        <svg class="caret" viewBox="0 0 10 10" aria-hidden="true"><path d="M3.5 2 7 5 3.5 8" /></svg>
        <span class="title">{{ heading }}</span>
        <span class="meta">{{ KIND_LABELS[section.kind] }} · {{ count }}</span>
      </button>
      <button class="icon-btn danger" type="button" title="Delete section" @click="emit('remove')">×</button>
    </header>

    <div v-if="open" class="section-body">
      <div class="options">
        <label class="field grow">
          <span class="field-label">Section heading</span>
          <input
            v-model="section.title"
            class="input"
            :placeholder="section.kind === 'columns' && section.variant === 'sections' ? 'None (each column has its own)' : 'Heading'"
          />
        </label>
        <label v-if="section.kind === 'entries'" class="check">
          <input v-model="section.uppercaseTitles" type="checkbox" />
          ALL-CAPS titles
        </label>
        <label v-if="section.kind === 'columns'" class="field">
          <span class="field-label">Layout</span>
          <select v-model="section.variant" class="input">
            <option value="table">Table under one heading</option>
            <option value="sections">Side-by-side sections</option>
          </select>
        </label>
      </div>

      <StringList v-if="section.kind === 'text'" v-model="section.paragraphs" add-label="Paragraph" rich />
      <EntriesEditor v-else-if="section.kind === 'entries'" v-model="section.items" />
      <ColumnsEditor v-else v-model="section.columns" />
    </div>
  </section>
</template>

<style scoped>
.section {
  border: 1px solid var(--rule);
  border-radius: 8px;
  background: var(--panel);
}

.section.open {
  box-shadow: 0 1px 0 var(--rule);
}

.section-head {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 6px 6px 6px 10px;
}

.toggle {
  flex: 1;
  min-width: 0;
  display: flex;
  align-items: baseline;
  gap: 8px;
  padding: 2px;
  border: 0;
  background: none;
  text-align: left;
  cursor: pointer;
}

.caret {
  flex: none;
  align-self: center;
  width: 10px;
  height: 10px;
  fill: none;
  stroke: var(--faint);
  stroke-width: 1.6;
  transition: transform 0.15s;
}

.open .caret {
  transform: rotate(90deg);
}

.title {
  font-weight: 650;
  color: var(--accent);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.meta {
  margin-left: auto;
  color: var(--faint);
  font-size: 12px;
  white-space: nowrap;
}

.section-body {
  display: grid;
  gap: 12px;
  padding: 2px 12px 14px;
}

.options {
  display: flex;
  align-items: end;
  gap: 12px;
}

.grow {
  flex: 1;
}

.check {
  display: flex;
  align-items: center;
  gap: 6px;
  padding-bottom: 6px;
  white-space: nowrap;
  font-size: 13px;
}
</style>
