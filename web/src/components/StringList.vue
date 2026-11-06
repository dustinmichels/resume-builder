<script setup lang="ts">
import { ArrowDownAZ } from '@lucide/vue'
import { nextTick, useTemplateRef } from 'vue'
import { VueDraggable } from 'vue-draggable-plus'
import { sortCommaList } from '../rich'
import RichText from './RichText.vue'

const items = defineModel<string[]>({ required: true })
const {
  path,
  addLabel = 'Add',
  rich = false,
  placeholder = '',
  sortable = false,
} = defineProps<{
  /** Field path of the array; rows are tagged "<path>.<i>". */
  path: string
  addLabel?: string
  rich?: boolean
  placeholder?: string
  /** Show a button on each row that sorts its comma-separated items A–Z. */
  sortable?: boolean
}>()

/** Every change to the rows' order or count, for data kept in step with them. */
const emit = defineEmits<{
  /** Mirrors `items.splice(start, deleteCount, ...insertCount new rows)`. */
  splice: [start: number, deleteCount: number, insertCount: number]
  move: [from: number, to: number]
}>()
defineSlots<{ /** Before each row's editor. */ lead?(props: { index: number }): unknown }>()

const rows = useTemplateRef<HTMLElement>('rows')
// Rich rows by index, for moving the caret between them.
const editors: (InstanceType<typeof RichText> | null)[] = []

function splice(start: number, deleteCount: number, insertCount = 0) {
  items.value.splice(start, deleteCount, ...Array<string>(insertCount).fill(''))
  emit('splice', start, deleteCount, insertCount)
}

async function add() {
  splice(items.value.length, 0, 1)
  await nextTick()
  rows.value?.querySelector<HTMLElement>('.row:last-child .input')?.focus()
}

function remove(index: number) {
  splice(index, 1)
}

async function insertAfter(index: number) {
  splice(index + 1, 0, 1)
  await nextTick()
  editors[index + 1]?.focus()
}

async function removeEmpty(index: number) {
  if (items.value.length === 1) return
  splice(index, 1)
  await nextTick()
  if (index > 0) editors[index - 1]?.focus()
  else editors[0]?.focus(false)
}
</script>

<template>
  <div class="string-list">
    <div ref="rows">
      <VueDraggable
        v-model="items"
        handle=".h-row"
        :animation="150"
        class="rows"
        @update="(e) => emit('move', e.oldIndex!, e.newIndex!)"
      >
        <!-- Strings have no identity of their own; index keys are fine because each
             row is only an editor bound to items[i]. -->
        <div v-for="(_, i) in items" :key="i" class="row">
          <span class="handle h-row" title="Drag to reorder" aria-hidden="true">⠿</span>
          <slot name="lead" :index="i" />
          <RichText
            v-if="rich"
            :ref="(c) => (editors[i] = c as InstanceType<typeof RichText> | null)"
            v-model="items[i]"
            :data-path="`${path}.${i}`"
            :placeholder="placeholder"
            @enter="insertAfter(i)"
            @backspace-empty="removeEmpty(i)"
          />
          <input v-else v-model="items[i]" :data-path="`${path}.${i}`" class="input" :placeholder="placeholder" />
          <button
            v-if="sortable"
            class="icon-btn"
            type="button"
            title="Sort items A–Z"
            :aria-label="`Sort item ${i + 1} alphabetically`"
            @click="items[i] = sortCommaList(items[i]!)"
          >
            <ArrowDownAZ :size="15" aria-hidden="true" />
          </button>
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
