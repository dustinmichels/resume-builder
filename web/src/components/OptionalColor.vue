<script setup lang="ts">
import { computed, shallowRef } from 'vue'

/**
 * An optional color: a swatch and a Show toggle. Off is stored as no key (undefined); the color is kept while off so
 * Show restores it.
 */
const color = defineModel<string | undefined>()
const { label, initial } = defineProps<{ label: string; /** The color Show first turns on. */ initial: string }>()

const kept = shallowRef(color.value || initial)
const swatch = computed({
  get: () => color.value || kept.value,
  set: (c: string) => {
    kept.value = c
    color.value = c
  },
})
const on = computed({
  get: () => !!color.value,
  set: (show: boolean) => (color.value = show ? kept.value : undefined),
})
</script>

<template>
  <span class="optional-color">
    <span class="label">{{ label }}</span>
    <input v-model="swatch" type="color" class="swatch" :aria-label="`${label} color`" :disabled="!on" />
    <label class="check">
      <input v-model="on" type="checkbox" />
      Show
    </label>
  </span>
</template>

<style scoped>
.optional-color {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
}

.swatch {
  width: 26px;
  height: 26px;
  padding: 1px;
  border: 1px solid var(--rule);
  border-radius: var(--radius);
  background: var(--panel);
  cursor: pointer;
}

.swatch:disabled {
  opacity: 0.4;
  cursor: default;
}

.check {
  display: flex;
  align-items: center;
  gap: 4px;
}
</style>
