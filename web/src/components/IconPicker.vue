<script setup lang="ts">
import { icons } from '@lucide/vue'
import { computed, onMounted, shallowRef, useTemplateRef } from 'vue'
import { readIcon } from '../lineIcons'
import type { LineIcon } from '../types'

/** Searches the Lucide set; picking one emits the icon, Remove emits null. */
const { current = null } = defineProps<{ current?: LineIcon | null }>()
const emit = defineEmits<{ pick: [icon: LineIcon | null] }>()

/** Shown at once; more would make the grid slow to render and to scan. */
const LIMIT = 120

const fold = (s: string) => s.toLowerCase().replace(/[^a-z0-9]/g, '')
const ALL = Object.entries(icons)
  .map(([key, component]) => ({ key, component, folded: fold(key) }))
  .sort((a, b) => a.key.localeCompare(b.key))

const query = shallowRef('')
const matches = computed(() => {
  const q = fold(query.value)
  if (!q) return ALL
  // Exact names first, then names starting with the query.
  const rank = (folded: string) => (folded === q ? 0 : folded.startsWith(q) ? 1 : 2)
  return ALL.filter((i) => i.folded.includes(q)).sort((a, b) => rank(a.folded) - rank(b.folded))
})

const searchEl = useTemplateRef<HTMLInputElement>('searchEl')
const gridEl = useTemplateRef<HTMLElement>('gridEl')

function focus() {
  searchEl.value?.focus({ preventScroll: true })
}
onMounted(focus)
defineExpose({ focus })

function pick(button: HTMLElement) {
  const svg = button.querySelector('svg')
  if (svg) emit('pick', readIcon(svg))
}

function pickFirst() {
  const first = gridEl.value?.querySelector<HTMLElement>('button')
  if (first) pick(first)
}
</script>

<template>
  <div class="icon-picker">
    <div class="ip-bar">
      <input
        ref="searchEl"
        v-model="query"
        class="input"
        type="search"
        placeholder="Search Lucide icons"
        aria-label="Search icons"
        @keydown.enter.prevent="pickFirst"
      />
      <button v-if="current" class="btn" type="button" @click="emit('pick', null)">Remove</button>
    </div>
    <div ref="gridEl" class="ip-grid">
      <button
        v-for="m in matches.slice(0, LIMIT)"
        :key="m.key"
        class="ip-icon"
        type="button"
        :title="m.key"
        :aria-label="m.key"
        @click="pick($event.currentTarget as HTMLElement)"
      >
        <component :is="m.component" :size="18" aria-hidden="true" />
      </button>
    </div>
    <p v-if="!matches.length" class="ip-note">No icons match.</p>
    <p v-else-if="matches.length > LIMIT" class="ip-note">{{ matches.length - LIMIT }} more: narrow the search.</p>
  </div>
</template>

<style scoped>
.icon-picker {
  display: grid;
  gap: 6px;
}

.ip-bar {
  display: flex;
  gap: 6px;
}

.ip-bar .input {
  flex: 1;
}

.ip-bar .btn {
  padding: 3px 9px;
  font-size: 12px;
}

.ip-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(30px, 1fr));
  gap: 2px;
  max-height: 180px;
  overflow-y: auto;
}

.ip-icon {
  display: grid;
  place-items: center;
  height: 30px;
  border: 1px solid transparent;
  border-radius: 6px;
  background: none;
  color: var(--muted);
  cursor: pointer;
}

.ip-icon:hover,
.ip-icon:focus-visible {
  border-color: var(--rule);
  color: var(--accent);
}

.ip-note {
  margin: 0;
  color: var(--faint);
  font-size: 11px;
}
</style>
