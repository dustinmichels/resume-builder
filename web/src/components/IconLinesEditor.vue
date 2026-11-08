<script setup lang="ts">
import { moveIcon, setIcon, spliceIcons } from '../lineIcons'
import type { IconLines } from '../types'
import LineIconButton from './LineIconButton.vue'
import StringList from './StringList.vue'

/** A column's or legend's lines, each with its icon button; icons stay beside their lines. */
const owner = defineModel<IconLines>({ required: true })
const {
  path,
  addLabel,
  sortable = false,
} = defineProps<{
  /** Field path of the column or legend, e.g. "sections.2.legend". */
  path: string
  addLabel: string
  /** Show StringList's A–Z button on each line. */
  sortable?: boolean
}>()
const emit = defineEmits<{ /** The last line was removed. */ empty: [] }>()

function splice(start: number, deleteCount: number, insertCount: number) {
  spliceIcons(owner.value, start, deleteCount, insertCount)
  if (!owner.value.lines.length) emit('empty')
}
</script>

<template>
  <StringList
    v-model="owner.lines"
    :path="`${path}.lines`"
    :add-label="addLabel"
    rich
    :sortable="sortable"
    @splice="splice"
    @move="(from, to) => moveIcon(owner, from, to)"
  >
    <template #lead="{ index }">
      <LineIconButton
        :icon="owner.icons?.[index]"
        :data-path="`${path}.icons.${index}`"
        @pick="(icon) => setIcon(owner, index, icon)"
      />
    </template>
  </StringList>
</template>
