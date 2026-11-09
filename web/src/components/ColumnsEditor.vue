<script setup lang="ts">
import { shallowRef } from 'vue'
import { VueDraggable } from 'vue-draggable-plus'
import { newColumn } from '../factory'
import type { Column } from '../types'
import IconLinesEditor from './IconLinesEditor.vue'
import PromptDialog, { type PromptRequest } from './PromptDialog.vue'
import VisibilityToggle from './VisibilityToggle.vue'

const columns = defineModel<Column[]>({ required: true })
const { path } = defineProps<{ /** Field path of the columns array. */ path: string }>()
const prompt = shallowRef<PromptRequest | null>(null)

function remove(index: number) {
  const col = columns.value[index]!
  const run = () => {
    const i = columns.value.indexOf(col)
    if (i >= 0) columns.value.splice(i, 1)
  }
  if (!col.title) return run()
  prompt.value = {
    title: `Delete column “${col.title}”?`,
    confirmLabel: 'Delete',
    danger: true,
    run,
  }
}
</script>

<template>
  <div>
    <VueDraggable v-model="columns" handle=".h-item" :animation="150" class="list">
      <div v-for="(col, i) in columns" :key="col.id" class="column" :class="{ 'is-hidden': col.hidden }">
        <div class="column-head">
          <span class="handle h-item" title="Drag to reorder" aria-hidden="true">⠿</span>
          <input v-model="col.title" :data-path="`${path}.${i}.title`" class="input column-title" placeholder="Column heading" />
          <VisibilityToggle v-model="col.hidden" name="column" />
          <button class="icon-btn danger" type="button" title="Delete column" @click="remove(i)">
            ×
          </button>
        </div>
        <div class="column-lines">
          <!-- IconLinesEditor edits the column in place; it never replaces it. -->
          <IconLinesEditor :model-value="col" :path="`${path}.${i}`" add-label="Line" sortable />
        </div>
      </div>
    </VueDraggable>
    <button class="btn btn-add add" type="button" @click="columns.push(newColumn())">+ Column</button>
    <PromptDialog v-model="prompt" />
  </div>
</template>

<style scoped>
.list {
  display: grid;
  gap: 6px;
}

.column {
  border: 1px solid var(--rule);
  border-radius: var(--radius);
  padding: 6px 10px 10px 8px;
}

.column-head {
  display: flex;
  align-items: center;
  gap: 4px;
}

.column-title {
  font-weight: 600;
}

.column-lines {
  padding: 6px 0 0 22px;
}

.is-hidden .column-title,
.is-hidden .column-lines {
  opacity: 0.45;
}

.add {
  margin-top: 8px;
}
</style>
