<script setup lang="ts">
import { useTemplateRef } from 'vue'
import { useMenu } from '../composables/useMenu'

export interface DownloadItem {
  label: string
  disabled: boolean
  action: () => void
}

defineProps<{
  items: DownloadItem[]
  /** Replaces "Download" on the trigger while one of the items is working. */
  busyLabel?: string | null
}>()

const root = useTemplateRef<HTMLElement>('root')
const open = useMenu(root)

function choose(item: DownloadItem) {
  open.value = false
  item.action()
}
</script>

<template>
  <div ref="root" class="download-menu">
    <button class="btn btn-quiet" type="button" aria-haspopup="menu" :aria-expanded="open" @click="open = !open">
      {{ busyLabel ?? 'Download' }}
      <svg class="caret" viewBox="0 0 10 10" aria-hidden="true"><path d="M2 3.5 5 6.5 8 3.5" /></svg>
    </button>

    <div v-if="open" class="menu" role="menu">
      <button
        v-for="item in items"
        :key="item.label"
        class="item"
        type="button"
        role="menuitem"
        :disabled="item.disabled"
        @click="choose(item)"
      >
        {{ item.label }}
      </button>
    </div>
  </div>
</template>

<style scoped>
.download-menu {
  position: relative;
}

.btn[aria-expanded='true'] {
  background: var(--sunken);
  color: var(--ink);
}

.caret {
  width: 10px;
  height: 10px;
  fill: none;
  stroke: currentColor;
  stroke-width: 1.6;
}

.menu {
  position: absolute;
  top: calc(100% + 4px);
  right: 0;
  z-index: 20;
  min-width: 180px;
  padding: 4px;
  border: 1px solid var(--rule);
  border-radius: 8px;
  background: var(--panel);
  box-shadow: 0 8px 28px rgb(0 0 0 / 0.14);
}

.item {
  display: block;
  width: 100%;
  padding: 6px 8px;
  border: 0;
  border-radius: 4px;
  background: none;
  text-align: left;
  white-space: nowrap;
  cursor: pointer;
}

.item:hover:not(:disabled) {
  background: var(--sunken);
}

.item:disabled {
  color: var(--faint);
  cursor: default;
}
</style>
