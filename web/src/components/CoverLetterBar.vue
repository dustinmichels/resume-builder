<script setup lang="ts">
import { shallowRef, watch } from 'vue'
import type { DocSummary } from '../types'
import PromptDialog, { type PromptRequest } from './PromptDialog.vue'

// Picks, creates, renames and deletes the open resume's cover letters.
const props = defineProps<{
  resumeId: string
  /** Most recently modified first. */
  letters: DocSummary[]
  /** Whether `letters` has loaded for this resume. */
  loaded: boolean
  /** The open letter. */
  current: string | null
  /** Each saves pending edits first. */
  open: (id: string) => Promise<void>
  create: (name: string) => Promise<void>
  duplicate: (name: string) => Promise<void>
  rename: (name: string) => Promise<void>
  remove: () => Promise<void>
}>()

const prompt = shallowRef<PromptRequest | null>(null)
const error = shallowRef<string | null>(null)

function run(action: Promise<void>) {
  error.value = null
  action.catch((e) => (error.value = e instanceof Error ? e.message : String(e)))
}

// Show a letter whenever the resume has one: the most recent, to start with.
watch(
  () => props.loaded && !props.current && props.letters[0]?.id,
  (first) => {
    if (first) run(props.open(first))
  },
  { immediate: true },
)

function promptCreate() {
  prompt.value = {
    title: 'New cover letter',
    message: 'Name it after the job, e.g. “MPO Data Analyst”. It is saved in this resume’s cover-letters folder.',
    value: 'Cover letter',
    placeholder: 'Name',
    confirmLabel: 'Create',
    run: props.create,
  }
}

function promptDuplicate() {
  prompt.value = {
    title: `Duplicate “${props.current}”`,
    value: `${props.current} copy`,
    placeholder: 'Name',
    confirmLabel: 'Duplicate',
    run: props.duplicate,
  }
}

function promptRename() {
  prompt.value = {
    title: `Rename “${props.current}”`,
    value: props.current ?? '',
    placeholder: 'Name',
    confirmLabel: 'Rename',
    run: props.rename,
  }
}

function promptDelete() {
  prompt.value = {
    title: `Delete “${props.current}”?`,
    message: 'The letter is moved to the .trash folder inside your data folder, where you can recover it.',
    confirmLabel: 'Delete',
    danger: true,
    run: props.remove,
  }
}
</script>

<template>
  <div class="letter-bar">
    <p v-if="!loaded" class="empty">Loading cover letters…</p>
    <template v-else-if="!letters.length">
      <p class="empty">
        No cover letters for “{{ resumeId }}” yet. A cover letter shares the resume’s name, accent color and heading
        font, and is saved beside it, in its cover-letters folder.
      </p>
      <button class="btn btn-primary" type="button" @click="promptCreate">New cover letter</button>
    </template>
    <div v-else class="picker">
      <label class="field grow">
        <span class="field-label">Cover letter</span>
        <select class="input" :value="current ?? ''" @change="run(open(($event.target as HTMLSelectElement).value))">
          <option v-if="!current" value="" disabled>Opening…</option>
          <option v-for="l in letters" :key="l.id" :value="l.id">{{ l.id }}</option>
        </select>
      </label>
      <button class="btn" type="button" @click="promptCreate">New</button>
      <button class="btn btn-quiet" type="button" :disabled="!current" @click="promptDuplicate">Duplicate</button>
      <button class="btn btn-quiet" type="button" :disabled="!current" @click="promptRename">Rename</button>
      <button
        class="icon-btn danger"
        type="button"
        title="Delete cover letter"
        aria-label="Delete cover letter"
        :disabled="!current"
        @click="promptDelete"
      >
        ×
      </button>
    </div>
    <p v-if="error" class="error" role="alert">{{ error }}</p>
    <PromptDialog v-model="prompt" />
  </div>
</template>

<style scoped>
.letter-bar {
  display: grid;
  gap: 10px;
  margin-bottom: 22px;
  padding-bottom: 16px;
  border-bottom: 1px solid var(--rule);
}

.letter-bar > .btn {
  justify-self: start;
}

.picker {
  display: flex;
  align-items: end;
  gap: 4px;
}

.grow {
  flex: 1;
  min-width: 0;
  margin-right: 4px;
}

.picker .btn {
  padding: 5px 9px;
  font-size: 13px;
}

.btn:disabled {
  opacity: 0.6;
  cursor: default;
}

.empty {
  margin: 0;
  color: var(--muted);
}
</style>
