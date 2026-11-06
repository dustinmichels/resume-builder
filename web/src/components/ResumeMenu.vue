<script setup lang="ts">
import { onScopeDispose, shallowRef, useTemplateRef, watch } from 'vue'
import { formatRelative } from '../format'
import type { ResumeSummary } from '../types'

defineProps<{
  resumes: ResumeSummary[]
  current: string | null
}>()

const emit = defineEmits<{
  /** The menu opened; refresh the list. */
  open: []
  select: [id: string]
  create: []
  duplicate: []
  rename: []
  remove: []
}>()

const open = shallowRef(false)
const root = useTemplateRef<HTMLElement>('root')

function choose(action: () => void) {
  open.value = false
  action()
}

function onPointerDown(event: PointerEvent) {
  if (!root.value?.contains(event.target as Node)) open.value = false
}

function onKeyDown(event: KeyboardEvent) {
  if (event.key === 'Escape') open.value = false
}

function stopListening() {
  document.removeEventListener('pointerdown', onPointerDown)
  document.removeEventListener('keydown', onKeyDown)
}

watch(open, (isOpen) => {
  if (!isOpen) return stopListening()
  emit('open')
  document.addEventListener('pointerdown', onPointerDown)
  document.addEventListener('keydown', onKeyDown)
})

onScopeDispose(stopListening)
</script>

<template>
  <div ref="root" class="resume-menu">
    <button class="switcher" type="button" aria-haspopup="menu" :aria-expanded="open" @click="open = !open">
      <span class="name">{{ current ?? 'Resume' }}</span>
      <svg class="caret" viewBox="0 0 10 10" aria-hidden="true"><path d="M2 3.5 5 6.5 8 3.5" /></svg>
    </button>

    <div v-if="open" class="menu" role="menu">
      <div class="list">
        <button
          v-for="r in resumes"
          :key="r.id"
          class="item resume"
          type="button"
          role="menuitemradio"
          :aria-checked="r.id === current"
          @click="choose(() => emit('select', r.id))"
        >
          <span class="check" aria-hidden="true">{{ r.id === current ? '✓' : '' }}</span>
          <span class="resume-name">{{ r.id }}</span>
          <span class="modified" :title="new Date(r.modified).toLocaleString()">{{ formatRelative(r.modified) }}</span>
        </button>
      </div>
      <hr class="rule" />
      <button class="item" type="button" role="menuitem" @click="choose(() => emit('create'))">New resume…</button>
      <template v-if="current">
        <button class="item" type="button" role="menuitem" @click="choose(() => emit('duplicate'))">Duplicate…</button>
        <button class="item" type="button" role="menuitem" @click="choose(() => emit('rename'))">Rename…</button>
        <button class="item danger" type="button" role="menuitem" @click="choose(() => emit('remove'))">Delete…</button>
      </template>
    </div>
  </div>
</template>

<style scoped>
.resume-menu {
  position: relative;
  min-width: 0;
}

.switcher {
  display: flex;
  align-items: center;
  gap: 6px;
  max-width: 280px;
  padding: 3px 8px;
  margin-left: -8px;
  border: 0;
  border-radius: var(--radius);
  background: none;
  cursor: pointer;
}

.switcher:hover,
.switcher[aria-expanded='true'] {
  background: var(--sunken);
}

.name {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 15px;
  font-weight: 700;
  color: var(--accent);
}

.caret {
  flex: none;
  width: 10px;
  height: 10px;
  fill: none;
  stroke: var(--faint);
  stroke-width: 1.6;
}

.menu {
  position: absolute;
  top: calc(100% + 4px);
  left: -8px;
  z-index: 20;
  width: 320px;
  padding: 4px;
  border: 1px solid var(--rule);
  border-radius: 8px;
  background: var(--panel);
  box-shadow: 0 8px 28px rgb(0 0 0 / 0.14);
}

.list {
  max-height: 50vh;
  overflow-y: auto;
}

.item {
  display: flex;
  align-items: baseline;
  gap: 8px;
  width: 100%;
  padding: 6px 8px;
  border: 0;
  border-radius: 4px;
  background: none;
  text-align: left;
  cursor: pointer;
}

.item:hover {
  background: var(--sunken);
}

.item.danger {
  color: var(--error);
}

.check {
  flex: none;
  width: 12px;
  color: var(--accent);
}

.resume-name {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.resume[aria-checked='true'] .resume-name {
  font-weight: 600;
}

.modified {
  flex: none;
  color: var(--faint);
  font-size: 12px;
}

.rule {
  margin: 4px 0;
  border: 0;
  border-top: 1px solid var(--rule);
}
</style>
