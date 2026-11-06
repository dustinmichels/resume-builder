<script setup lang="ts">
import { Eye, EyeOff } from '@lucide/vue'

/** Bound to a section's, item's or column's `hidden` flag; absent means shown. */
const hidden = defineModel<boolean | undefined>()
const { name } = defineProps<{ /** What is toggled, for the tooltip, e.g. "entry". */ name: string }>()
</script>

<template>
  <button
    class="icon-btn"
    :class="{ off: hidden }"
    type="button"
    :title="hidden ? `Hidden from the PDF: show ${name}` : `Hide ${name} from the PDF`"
    :aria-label="`Show ${name} on the PDF`"
    :aria-pressed="!hidden"
    @click="hidden = hidden ? undefined : true"
  >
    <component :is="hidden ? EyeOff : Eye" :size="15" aria-hidden="true" />
  </button>
</template>

<style scoped>
.off {
  color: var(--ink);
}
</style>
