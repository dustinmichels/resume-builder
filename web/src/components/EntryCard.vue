<script setup lang="ts">
import { computed, shallowRef } from 'vue'
import { hasFeature, useTemplate } from '../composables/useTemplates'
import { setEntryIcon } from '../lineIcons'
import { useOpenOnReveal } from '../reveal'
import { richToText } from '../rich'
import type { Entry } from '../types'
import LineIconButton from './LineIconButton.vue'
import RichText from './RichText.vue'
import StringList from './StringList.vue'
import VisibilityToggle from './VisibilityToggle.vue'

const entry = defineModel<Entry>({ required: true })
const { path } = defineProps<{ /** Field path of the entry, e.g. "sections.1.items.3". */ path: string }>()
const emit = defineEmits<{ remove: []; duplicate: [] }>()

// New (blank) entries open ready for typing; existing ones start collapsed.
const open = shallowRef(!entry.value.title && !entry.value.subtitle)
useOpenOnReveal(() => path, open)
// An entry without a title (an organization line only) goes by its subtitle.
const heading = computed(() => entry.value.title || richToText(entry.value.subtitle) || 'Untitled entry')
const template = useTemplate()
/** Index of the slot that adds an icon. */
const iconCount = computed(() => entry.value.icons?.length ?? 0)
</script>

<template>
  <div class="entry" :class="{ open, 'is-hidden': entry.hidden }">
    <div class="entry-head">
      <span class="handle h-item" title="Drag to reorder" aria-hidden="true">⠿</span>
      <button class="toggle" type="button" :aria-expanded="open" @click="open = !open">
        <span class="title">{{ heading }}</span>
        <span class="date">{{ entry.date }}</span>
      </button>
      <VisibilityToggle v-model="entry.hidden" name="entry" />
      <button class="icon-btn" type="button" title="Duplicate entry" @click="emit('duplicate')">⧉</button>
      <button class="icon-btn danger" type="button" title="Delete entry" @click="emit('remove')">×</button>
    </div>
    <div v-if="open" class="entry-body">
      <label class="field">
        <span class="field-label">Title</span>
        <input v-model="entry.title" :data-path="`${path}.title`" class="input" placeholder="Job title or school" />
      </label>
      <label class="field">
        <span class="field-label">Dates</span>
        <input v-model="entry.date" :data-path="`${path}.date`" class="input" placeholder="JAN 2021 – PRESENT" />
      </label>
      <div v-if="hasFeature(template, 'entryIcons')" class="field wide">
        <span class="field-label">Icons beside the title</span>
        <div class="icons">
          <LineIconButton
            v-for="(icon, k) in entry.icons"
            :key="k"
            :icon="icon"
            :data-path="`${path}.icons.${k}`"
            @pick="(next) => setEntryIcon(entry, k, next)"
          />
          <LineIconButton :data-path="`${path}.icons.${iconCount}`" @pick="(next) => setEntryIcon(entry, iconCount, next)" />
        </div>
      </div>
      <div class="field">
        <span class="field-label">Subtitle</span>
        <RichText v-model="entry.subtitle" :data-path="`${path}.subtitle`" placeholder="Organization or degree" aria-label="Subtitle" />
      </div>
      <label class="field">
        <span class="field-label">Location</span>
        <input v-model="entry.location" :data-path="`${path}.location`" class="input" placeholder="City, ST" />
      </label>
      <div class="field wide">
        <span class="field-label">Bullets</span>
        <StringList v-model="entry.bullets" :path="`${path}.bullets`" add-label="Bullet" rich />
      </div>
    </div>
  </div>
</template>

<style scoped>
.entry {
  border: 1px solid var(--rule);
  border-radius: var(--radius);
  background: var(--panel);
}

.entry.open {
  border-color: #bcc3cd;
}

.entry-head {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 3px 4px 3px 8px;
}

.toggle {
  flex: 1;
  min-width: 0;
  display: flex;
  align-items: baseline;
  gap: 10px;
  padding: 4px 2px;
  border: 0;
  background: none;
  text-align: left;
  cursor: pointer;
}

.title {
  font-weight: 600;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.date {
  margin-left: auto;
  color: var(--muted);
  font-size: 12px;
  white-space: nowrap;
}

.is-hidden .toggle {
  opacity: 0.45;
}

.entry-body {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 8px 10px;
  padding: 4px 12px 12px 30px;
}

.wide {
  grid-column: 1 / -1;
}

.icons {
  display: flex;
  flex-wrap: wrap;
  gap: 2px;
}
</style>
