<script setup lang="ts">
import type { ContactInfo } from '../types'

const contact = defineModel<ContactInfo>({ required: true })

const FIELDS: { key: keyof ContactInfo; label: string; type?: string; rows?: number }[] = [
  { key: 'firstName', label: 'First name' },
  { key: 'lastName', label: 'Last name' },
  { key: 'email', label: 'Email', type: 'email' },
  { key: 'phone', label: 'Phone', type: 'tel' },
  { key: 'address', label: 'Street address', rows: 2 },
  { key: 'city', label: 'City' },
  { key: 'region', label: 'State / region' },
  { key: 'postalCode', label: 'Postal code' },
  { key: 'country', label: 'Country' },
  { key: 'notes', label: 'Notes', rows: 4 },
]
</script>

<template>
  <div class="contact">
    <label v-for="f in FIELDS" :key="f.key" class="field" :class="{ wide: f.rows }">
      <span class="field-label">{{ f.label }}</span>
      <textarea v-if="f.rows" v-model="contact[f.key]" class="input" :rows="f.rows" />
      <input v-else v-model="contact[f.key]" class="input" :type="f.type ?? 'text'" />
    </label>
  </div>
</template>

<style scoped>
.contact {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 10px;
}

.wide {
  grid-column: 1 / -1;
}
</style>
