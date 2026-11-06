<script setup lang="ts">
import { VueDraggable } from 'vue-draggable-plus'
import { newSection } from '../factory'
import type { Section, SectionKind } from '../types'
import SectionCard from './SectionCard.vue'

const sections = defineModel<Section[]>({ required: true })

const ADD_OPTIONS: { kind: SectionKind; label: string; hint: string }[] = [
  { kind: 'entries', label: 'Entries', hint: 'Jobs, schools, volunteer roles' },
  { kind: 'text', label: 'Paragraphs', hint: 'Profile, summary' },
  { kind: 'columns', label: 'Columns', hint: 'Skills table, languages' },
]

function remove(index: number) {
  const name = sections.value[index]!.title || 'this section'
  if (confirm(`Delete ${name} and everything in it?`)) sections.value.splice(index, 1)
}
</script>

<template>
  <div class="sections">
    <VueDraggable v-model="sections" handle=".h-section" :animation="150" class="list">
      <!-- SectionCard edits fields in place; it never replaces the section object. -->
      <SectionCard
        v-for="(section, i) in sections"
        :key="section.id"
        :model-value="section"
        @remove="remove(i)"
      />
    </VueDraggable>

    <div class="add">
      <span class="add-label">Add a section</span>
      <div class="add-options">
        <button
          v-for="opt in ADD_OPTIONS"
          :key="opt.kind"
          class="btn btn-add add-option"
          type="button"
          @click="sections.push(newSection(opt.kind))"
        >
          <strong>{{ opt.label }}</strong>
          <span>{{ opt.hint }}</span>
        </button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.list {
  display: grid;
  gap: 8px;
}

.add {
  margin-top: 16px;
}

.add-label {
  font-size: 12px;
  color: var(--muted);
}

.add-options {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 6px;
  margin-top: 4px;
}

.add-option {
  display: grid;
  justify-items: start;
  gap: 0;
  padding: 7px 10px;
  white-space: normal;
  text-align: left;
}

.add-option strong {
  color: var(--ink);
  font-weight: 600;
}

.add-option span {
  font-size: 12px;
}
</style>
