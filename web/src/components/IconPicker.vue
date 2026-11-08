<script setup lang="ts">
import { icons as lucideIcons } from '@lucide/vue'
import { computed, onMounted, shallowRef, useTemplateRef } from 'vue'
import { iconStyle, readIcon } from '../lineIcons'
import type { LineIcon } from '../types'

/** Searches the template's icon set, then the Lucide set; picking one emits the icon, Remove emits null. */
const { current = null, icons = [] } = defineProps<{
  current?: LineIcon | null
  /** The template's own icon set, offered first and picked as they are. */
  icons?: LineIcon[]
}>()
const emit = defineEmits<{ pick: [icon: LineIcon | null] }>()

/** Shown at once; more would make the grid slow to render and to scan. */
const LIMIT = 120

const fold = (s: string) => s.toLowerCase().replace(/[^a-z0-9]/g, '')
const LUCIDE = Object.entries(lucideIcons)
  .map(([key, component]) => ({ key, component, folded: fold(key) }))
  .sort((a, b) => a.key.localeCompare(b.key))
const own = computed(() => icons.map((icon) => ({ icon, folded: fold(icon.name) })))

const query = shallowRef('')

function search<T extends { folded: string }>(all: T[]): T[] {
  const q = fold(query.value)
  if (!q) return all
  // Exact names first, then names starting with the query.
  const rank = (folded: string) => (folded === q ? 0 : folded.startsWith(q) ? 1 : 2)
  return all.filter((i) => i.folded.includes(q)).sort((a, b) => rank(a.folded) - rank(b.folded))
}
const ownMatches = computed(() => search(own.value))
const matches = computed(() => search(LUCIDE))

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

/** The template's icons come first, so Enter prefers them. */
function pickFirst() {
  gridEl.value?.querySelector<HTMLElement>('button')?.click()
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
        :placeholder="icons.length ? 'Search icons' : 'Search Lucide icons'"
        aria-label="Search icons"
        @keydown.enter.prevent="pickFirst"
      />
      <button v-if="current" class="btn" type="button" @click="emit('pick', null)">Remove</button>
    </div>
    <div ref="gridEl" class="ip-scroll">
      <template v-if="ownMatches.length">
        <p class="ip-label">Template icons</p>
        <div class="ip-grid">
          <button
            v-for="(m, i) in ownMatches"
            :key="i"
            class="ip-icon"
            type="button"
            :title="m.icon.name"
            :aria-label="m.icon.name"
            @click="emit('pick', { ...m.icon })"
          >
            <span class="ip-glyph" :style="iconStyle(m.icon)" aria-hidden="true" />
          </button>
        </div>
        <p v-if="matches.length" class="ip-label">Lucide</p>
      </template>
      <div class="ip-grid">
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
    </div>
    <p v-if="!matches.length && !ownMatches.length" class="ip-note">No icons match.</p>
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

.ip-scroll {
  max-height: 180px;
  overflow-y: auto;
}

.ip-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(30px, 1fr));
  gap: 2px;
}

.ip-label {
  margin: 2px 0;
  color: var(--faint);
  font-size: 11px;
  font-weight: 600;
}

.ip-glyph {
  width: 18px;
  height: 18px;
  background-color: currentColor;
  mask-size: contain;
  mask-repeat: no-repeat;
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
