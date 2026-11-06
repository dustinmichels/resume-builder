<script setup lang="ts">
import { VueDraggable } from 'vue-draggable-plus'
import { cloneEntry, newEntry, newLabel } from '../factory'
import type { Entry, EntryItem } from '../types'
import EntryCard from './EntryCard.vue'
import RichText from './RichText.vue'
import VisibilityToggle from './VisibilityToggle.vue'

const items = defineModel<EntryItem[]>({ required: true })
const { path } = defineProps<{ /** Field path of the items array. */ path: string }>()

function remove(index: number) {
  const item = items.value[index]!
  const name = item.type === 'entry' ? item.title : item.text
  if (name && !confirm(`Delete “${name}”?`)) return
  items.value.splice(index, 1)
}

function duplicate(index: number) {
  items.value.splice(index + 1, 0, cloneEntry(items.value[index] as Entry))
}
</script>

<template>
  <div class="entries">
    <VueDraggable v-model="items" handle=".h-item" :animation="150" class="list">
      <template v-for="(item, i) in items" :key="item.id">
        <!-- EntryCard edits fields in place; it never replaces the entry object. -->
        <EntryCard
          v-if="item.type === 'entry'"
          :model-value="item"
          :path="`${path}.${i}`"
          @remove="remove(i)"
          @duplicate="duplicate(i)"
        />
        <div v-else class="label-row" :class="{ 'is-hidden': item.hidden }">
          <span class="handle h-item" title="Drag to reorder" aria-hidden="true">⠿</span>
          <RichText
            v-model="item.text"
            :data-path="`${path}.${i}.text`"
            placeholder="Subheading, e.g. Non-degree education:"
            aria-label="Subheading"
          />
          <VisibilityToggle v-model="item.hidden" name="subheading" />
          <button class="icon-btn danger" type="button" title="Delete subheading" @click="remove(i)">×</button>
        </div>
      </template>
    </VueDraggable>
    <div class="adds">
      <button class="btn btn-add" type="button" @click="items.push(newEntry())">+ Entry</button>
      <button class="btn btn-add" type="button" @click="items.push(newLabel())">+ Subheading</button>
    </div>
  </div>
</template>

<style scoped>
.list {
  display: grid;
  gap: 6px;
}

.label-row {
  display: flex;
  align-items: center;
  gap: 4px;
  padding-left: 8px;
}

.is-hidden > .rich-text {
  opacity: 0.45;
}

.adds {
  display: flex;
  gap: 6px;
  margin-top: 8px;
}
</style>
