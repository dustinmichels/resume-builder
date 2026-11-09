<script setup lang="ts">
import { moveIcon, setIcon, spliceIcons } from '../lineIcons'
import type { IconLines } from '../types'
import LineIconButton from './LineIconButton.vue'
import StringList from './StringList.vue'

/** A column's or legend's lines, each with its icon button; icons stay beside their lines. */
const owner = defineModel<IconLines>({ required: true })
const { path, addLabel, sortable, labelColors } = defineProps<{
  /** Field path of the column or legend, e.g. "sections.2.legend". */
  path: string
  addLabel: string
  /** Show StringList's A–Z button on each line. */
  sortable?: boolean
  /** A legend: a color swatch beside each line with an icon, editing the icon's label color. */
  labelColors?: boolean
}>()
const emit = defineEmits<{ /** The last line was removed. */ empty: [] }>()

/** The template's ink, which draws a label whose icon has no color (Lucide's). */
const INK = '#231f20'

function splice(start: number, deleteCount: number, insertCount: number) {
  spliceIcons(owner.value, start, deleteCount, insertCount)
  if (!owner.value.lines.length) emit('empty')
}

/** Picking another icon resets the label to that icon's color. */
function setLabelColor(index: number, color: string) {
  const icon = owner.value.icons?.[index]
  if (icon) icon.color = color
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
      <input
        v-if="labelColors && owner.icons?.[index]"
        type="color"
        class="swatch"
        :value="owner.icons[index].color ?? INK"
        title="Label color"
        :aria-label="`Line ${index + 1} label color`"
        @input="setLabelColor(index, ($event.target as HTMLInputElement).value)"
      />
    </template>
  </StringList>
</template>

<style scoped>
.swatch {
  flex: none;
  width: 26px;
  height: 26px;
  padding: 1px;
  border: 1px solid var(--rule);
  border-radius: var(--radius);
  background: var(--panel);
  cursor: pointer;
}
</style>
