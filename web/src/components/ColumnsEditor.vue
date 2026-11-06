<script setup lang="ts">
import { VueDraggable } from 'vue-draggable-plus'
import { newColumn } from '../factory'
import type { Column } from '../types'
import StringList from './StringList.vue'

const columns = defineModel<Column[]>({ required: true })

function remove(index: number) {
  const col = columns.value[index]!
  if (col.title && !confirm(`Delete column “${col.title}”?`)) return
  columns.value.splice(index, 1)
}
</script>

<template>
  <div>
    <VueDraggable v-model="columns" handle=".h-item" :animation="150" class="list">
      <div v-for="col in columns" :key="col.id" class="column">
        <div class="column-head">
          <span class="handle h-item" title="Drag to reorder" aria-hidden="true">⠿</span>
          <input v-model="col.title" class="input column-title" placeholder="Column heading" />
          <button class="icon-btn danger" type="button" title="Delete column" @click="remove(columns.indexOf(col))">
            ×
          </button>
        </div>
        <div class="column-lines">
          <StringList v-model="col.lines" add-label="Line" rich />
        </div>
      </div>
    </VueDraggable>
    <button class="btn btn-add add" type="button" @click="columns.push(newColumn())">+ Column</button>
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

.add {
  margin-top: 8px;
}
</style>
