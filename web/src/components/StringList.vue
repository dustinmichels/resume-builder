<script setup lang="ts">
import { nextTick, useTemplateRef } from 'vue'
import { VueDraggable } from 'vue-draggable-plus'

const items = defineModel<string[]>({ required: true })
const {
  addLabel = 'Add',
  multiline = true,
  placeholder = '',
} = defineProps<{ addLabel?: string; multiline?: boolean; placeholder?: string }>()

const rows = useTemplateRef<HTMLElement>('rows')

async function add() {
  items.value.push('')
  await nextTick()
  rows.value?.querySelector<HTMLElement>('.row:last-child .input')?.focus()
}

function remove(index: number) {
  items.value.splice(index, 1)
}
</script>

<template>
  <div class="string-list">
    <div ref="rows">
      <VueDraggable v-model="items" handle=".h-row" :animation="150" class="rows">
        <!-- Strings have no identity of their own; index keys are fine because each
             row is only an input bound to items[i]. -->
        <div v-for="(_, i) in items" :key="i" class="row">
          <span class="handle h-row" title="Drag to reorder" aria-hidden="true">⠿</span>
          <textarea v-if="multiline" v-model="items[i]" class="input" rows="1" :placeholder="placeholder" />
          <input v-else v-model="items[i]" class="input" :placeholder="placeholder" />
          <button class="icon-btn danger" type="button" title="Remove" :aria-label="`Remove item ${i + 1}`" @click="remove(i)">
            ×
          </button>
        </div>
      </VueDraggable>
    </div>
    <button class="btn btn-add add" type="button" @click="add">+ {{ addLabel }}</button>
  </div>
</template>

<style scoped>
.rows {
  display: grid;
  gap: 5px;
}

.row {
  display: flex;
  align-items: flex-start;
  gap: 4px;
}

.row .handle {
  padding-top: 5px;
  width: 14px;
  text-align: center;
}

.add {
  margin-top: 6px;
  font-size: 13px;
  padding: 3px 9px;
}
</style>
