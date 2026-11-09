<script setup lang="ts">
import { computed, shallowRef } from 'vue'
import { hasFeature, useTemplate } from '../composables/useTemplates'
import { useOpenOnReveal } from '../reveal'
import type { EntriesSection, Section } from '../types'
import ColumnsEditor from './ColumnsEditor.vue'
import EntriesEditor from './EntriesEditor.vue'
import IconLinesEditor from './IconLinesEditor.vue'
import OptionalColor from './OptionalColor.vue'
import StringList from './StringList.vue'
import VisibilityToggle from './VisibilityToggle.vue'

const section = defineModel<Section>({ required: true })
const { path } = defineProps<{ /** Field path of the section, e.g. "sections.2". */ path: string }>()
const emit = defineEmits<{ remove: [] }>()

const template = useTemplate()

// Entries sections only; a legend without lines is dropped.
function setCompact(s: EntriesSection, on: boolean) {
  s.compact = on || undefined
}
function addLegend(s: EntriesSection) {
  s.legend = { lines: [''] }
}
function dropLegend(s: EntriesSection) {
  delete s.legend
}

const open = shallowRef(false)
useOpenOnReveal(() => path, open)

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
  <section class="section" :class="{ open, 'is-hidden': section.hidden }">
    <header class="section-head">
      <span class="handle h-section" title="Drag to reorder" aria-hidden="true">⠿</span>
      <button class="toggle" type="button" :aria-expanded="open" @click="open = !open">
        <svg class="caret" viewBox="0 0 10 10" aria-hidden="true"><path d="M3.5 2 7 5 3.5 8" /></svg>
        <span class="title">{{ heading }}</span>
        <span class="meta">{{ KIND_LABELS[section.kind] }} · {{ count }}</span>
      </button>
      <VisibilityToggle v-model="section.hidden" name="section" />
      <button class="icon-btn danger" type="button" title="Delete section" @click="emit('remove')">×</button>
    </header>

    <div v-if="open" class="section-body">
      <div class="options">
        <label class="field grow">
          <span class="field-label">Section heading</span>
          <input
            v-model="section.title"
            :data-path="`${path}.title`"
            class="input"
            :placeholder="section.kind === 'columns' && section.variant === 'sections' ? 'None (each column has its own)' : 'Heading'"
          />
        </label>
        <label v-if="section.kind === 'entries' && hasFeature(template, 'uppercaseTitles')" class="check">
          <input v-model="section.uppercaseTitles" type="checkbox" />
          ALL-CAPS titles
        </label>
        <label v-if="section.kind === 'entries' && hasFeature(template, 'compactEntries')" class="check">
          <input
            type="checkbox"
            :checked="section.compact"
            @change="setCompact(section, ($event.target as HTMLInputElement).checked)"
          />
          Compact: one line per entry
        </label>
        <label v-if="section.kind === 'columns'" class="field">
          <span class="field-label">Layout</span>
          <select v-model="section.variant" class="input">
            <option value="table">Table under one heading</option>
            <option value="sections">Side-by-side sections</option>
          </select>
        </label>
      </div>

      <div v-if="section.kind === 'entries' && hasFeature(template, 'legend')" class="field">
        <span class="field-label">Legend</span>
        <template v-if="section.legend">
          <IconLinesEditor
            :model-value="section.legend"
            :path="`${path}.legend`"
            add-label="Legend line"
            label-colors
            @empty="dropLegend(section)"
          />
          <div class="legend-panel">
            <OptionalColor v-model="section.legend.fill" label="Background" initial="#eeeeee" />
            <OptionalColor v-model="section.legend.border" label="Border" initial="#999999" />
          </div>
        </template>
        <button v-else class="btn btn-add add-legend" type="button" @click="addLegend(section)">+ Legend line</button>
      </div>

      <StringList v-if="section.kind === 'text'" v-model="section.paragraphs" :path="`${path}.paragraphs`" add-label="Paragraph" rich />
      <EntriesEditor v-else-if="section.kind === 'entries'" v-model="section.items" :path="`${path}.items`" />
      <ColumnsEditor v-else v-model="section.columns" :path="`${path}.columns`" />
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

.is-hidden > .section-head .toggle {
  opacity: 0.45;
}

.meta {
  margin-left: auto;
  color: var(--faint);
  font-size: 12px;
  white-space: nowrap;
}

.section-body {
  display: grid;
  grid-template-columns: minmax(0, 1fr);
  gap: 12px;
  padding: 2px 12px 14px;
}

.options {
  display: flex;
  flex-wrap: wrap;
  align-items: end;
  gap: 12px;
}

.grow {
  flex: 1 1 12em;
}

.check {
  display: flex;
  align-items: center;
  gap: 6px;
  padding-bottom: 6px;
  white-space: nowrap;
  font-size: 13px;
}

.add-legend {
  justify-self: start;
  padding: 3px 9px;
  font-size: 13px;
}

.legend-panel {
  display: flex;
  flex-wrap: wrap;
  gap: 6px 18px;
}
</style>
