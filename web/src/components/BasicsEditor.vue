<script setup lang="ts">
import { VueDraggable } from 'vue-draggable-plus'
import { newContact } from '../factory'
import type { Basics } from '../types'
import StringList from './StringList.vue'

const basics = defineModel<Basics>({ required: true })

function removeContact(index: number) {
  basics.value.contacts.splice(index, 1)
}
</script>

<template>
  <div class="basics">
    <label class="field">
      <span class="field-label">Name</span>
      <input v-model="basics.name" class="input name" />
    </label>

    <div class="field">
      <span class="field-label">Headline (joined with //)</span>
      <StringList v-model="basics.headline" :multiline="false" add-label="Headline part" />
    </div>

    <div class="field">
      <span class="field-label">Contact links</span>
      <VueDraggable v-model="basics.contacts" handle=".h-item" :animation="150" class="contacts">
        <div v-for="(contact, i) in basics.contacts" :key="contact.id" class="contact">
          <span class="handle h-item" title="Drag to reorder" aria-hidden="true">⠿</span>
          <input v-model="contact.text" class="input" placeholder="Shown text" aria-label="Shown text" />
          <input v-model="contact.url" class="input url" placeholder="Link (optional)" aria-label="Link" />
          <button class="icon-btn danger" type="button" title="Remove contact" @click="removeContact(i)">×</button>
        </div>
      </VueDraggable>
      <button class="btn btn-add add" type="button" @click="basics.contacts.push(newContact())">+ Contact</button>
    </div>
  </div>
</template>

<style scoped>
.basics {
  display: grid;
  gap: 12px;
}

.name {
  font-size: 16px;
  font-weight: 650;
}

.contacts {
  display: grid;
  gap: 5px;
}

.contact {
  display: grid;
  grid-template-columns: 14px 1fr 1fr 26px;
  align-items: center;
  gap: 4px;
}

.url {
  color: var(--muted);
}

.add {
  justify-self: start;
  margin-top: 2px;
  font-size: 13px;
  padding: 3px 9px;
}
</style>
