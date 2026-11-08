<script setup lang="ts">
import { ImagePlus } from '@lucide/vue'
import { shallowRef, useTemplateRef, watch } from 'vue'
import { useTemplate } from '../composables/useTemplates'
import { iconStyle } from '../lineIcons'
import type { LineIcon } from '../types'
import IconPicker from './IconPicker.vue'

/** An icon in the sidebar; clicking opens the icon picker, with the template's icons, below it. */
const { icon = null } = defineProps<{ icon?: LineIcon | null }>()
const emit = defineEmits<{ pick: [icon: LineIcon | null] }>()

const template = useTemplate()
const open = shallowRef(false)
const root = useTemplateRef<HTMLElement>('root')

function pick(next: LineIcon | null) {
  emit('pick', next)
  open.value = false
}

function onPointerDown(e: PointerEvent) {
  if (!root.value?.contains(e.target as Node)) open.value = false
}
watch(open, (isOpen, _, onCleanup) => {
  if (!isOpen) return
  document.addEventListener('pointerdown', onPointerDown, true)
  onCleanup(() => document.removeEventListener('pointerdown', onPointerDown, true))
})
</script>

<template>
  <div ref="root" class="line-icon" @keydown.esc.stop="open = false">
    <button
      class="icon-btn"
      type="button"
      :title="icon ? `Icon: ${icon.name}` : 'Add an icon'"
      :aria-label="icon ? `Change icon (${icon.name})` : 'Add an icon'"
      :aria-expanded="open"
      @click="open = !open"
    >
      <span v-if="icon" class="glyph" :style="iconStyle(icon)" aria-hidden="true" />
      <ImagePlus v-else class="placeholder" :size="15" aria-hidden="true" />
    </button>
    <div v-if="open" class="popover">
      <IconPicker :current="icon" :icons="template.icons" @pick="pick" />
    </div>
  </div>
</template>

<style scoped>
.line-icon {
  position: relative;
}

.glyph {
  display: block;
  width: 15px;
  height: 15px;
  background: var(--accent);
  mask-size: contain;
  mask-repeat: no-repeat;
}

.placeholder {
  color: var(--faint);
  opacity: 0.6;
}

.icon-btn:hover .placeholder {
  opacity: 1;
}

.popover {
  position: absolute;
  top: calc(100% + 4px);
  left: 0;
  z-index: 20;
  width: 300px;
  padding: 8px;
  border: 1px solid var(--rule);
  border-radius: 8px;
  background: var(--panel);
  box-shadow:
    0 2px 4px rgb(0 0 0 / 0.08),
    0 10px 28px rgb(0 0 0 / 0.16);
}
</style>
