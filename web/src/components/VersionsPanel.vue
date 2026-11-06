<script setup lang="ts">
import { shallowRef, watch } from 'vue'
import { api } from '../api'
import { formatDateTime } from '../format'
import type { Version } from '../types'
import PromptDialog, { type PromptRequest } from './PromptDialog.vue'

const props = defineProps<{
  resumeId: string
  /** Saves pending edits, then snapshots the resume. */
  saveVersion: (label: string) => Promise<Version>
  /** Replaces the open resume with the version (the server snapshots it first). */
  restoreVersion: (version: Version) => Promise<void>
  /** Creates a new resume from the version and opens it. */
  copyVersion: (version: Version) => Promise<void>
}>()

const versions = shallowRef<Version[]>([])
const loading = shallowRef(true)
const label = shallowRef('')
const saving = shallowRef(false)
const error = shallowRef<string | null>(null)
const prompt = shallowRef<PromptRequest | null>(null)

const describe = (v: Version) => v.label || 'Untitled version'

async function load() {
  const id = props.resumeId
  let list: Version[]
  try {
    list = await api.listVersions(id)
  } catch (e) {
    if (id === props.resumeId) error.value = `Could not load versions: ${e instanceof Error ? e.message : e}`
    return
  } finally {
    if (id === props.resumeId) loading.value = false
  }
  if (id !== props.resumeId) return
  versions.value = list
  error.value = null
}

watch(
  () => props.resumeId,
  () => {
    versions.value = []
    loading.value = true
    load()
  },
  { immediate: true },
)

async function save() {
  saving.value = true
  error.value = null
  try {
    await props.saveVersion(label.value.trim())
    label.value = ''
    await load()
  } catch (e) {
    error.value = e instanceof Error ? e.message : String(e)
  } finally {
    saving.value = false
  }
}

function restore(version: Version) {
  prompt.value = {
    title: `Restore “${describe(version)}”?`,
    message: 'The resume is replaced with this version. Its current state is saved as a version first, so you can go back.',
    confirmLabel: 'Restore',
    run: async () => {
      await props.restoreVersion(version)
      await load()
    },
  }
}

function copy(version: Version) {
  // Errors show in the panel; the resume switches on success.
  props.copyVersion(version).catch((e) => {
    error.value = e instanceof Error ? e.message : String(e)
  })
}

function remove(version: Version) {
  prompt.value = {
    title: `Delete “${describe(version)}”?`,
    message: `Saved ${formatDateTime(version.created)}. This cannot be undone.`,
    confirmLabel: 'Delete',
    danger: true,
    run: async () => {
      await api.deleteVersion(props.resumeId, version.id)
      await load()
    },
  }
}
</script>

<template>
  <div class="versions">
    <form class="new-version" @submit.prevent="save">
      <label class="field grow">
        <span class="field-label">Save the resume as it is now</span>
        <input v-model="label" class="input" placeholder="Label, e.g. “Sent to Acme”" :disabled="saving" />
      </label>
      <button class="btn btn-primary" type="submit" :disabled="saving">{{ saving ? 'Saving…' : 'Save version' }}</button>
    </form>

    <p v-if="error" class="error" role="alert">{{ error }}</p>

    <p v-if="loading" class="empty">Loading versions…</p>
    <p v-else-if="!versions.length" class="empty">
      No versions yet. A version is a named snapshot of this resume, e.g. the copy you sent with an application. You can
      restore it later or start a new resume from it.
    </p>
    <ul v-else class="list">
      <li v-for="v in versions" :key="v.id" class="version">
        <div class="info">
          <span class="label" :class="{ untitled: !v.label }">{{ describe(v) }}</span>
          <span class="date">{{ formatDateTime(v.created) }}</span>
        </div>
        <button class="btn btn-quiet" type="button" @click="restore(v)">Restore</button>
        <button class="btn btn-quiet" type="button" @click="copy(v)">Copy to new resume</button>
        <button class="icon-btn danger" type="button" title="Delete version" @click="remove(v)">×</button>
      </li>
    </ul>

    <PromptDialog v-model="prompt" />
  </div>
</template>

<style scoped>
.versions {
  display: grid;
  gap: 14px;
}

.new-version {
  display: flex;
  align-items: end;
  gap: 8px;
}

.grow {
  flex: 1;
}

.btn:disabled {
  opacity: 0.6;
  cursor: default;
}

.error {
  margin: 0;
  color: var(--error);
  font-size: 13px;
}

.empty {
  margin: 0;
  color: var(--muted);
}

.list {
  display: grid;
  margin: 0;
  padding: 0;
  list-style: none;
  border: 1px solid var(--rule);
  border-radius: 8px;
}

.version {
  display: flex;
  align-items: center;
  gap: 2px;
  padding: 6px 6px 6px 12px;
}

.version + .version {
  border-top: 1px solid var(--rule);
}

.info {
  flex: 1;
  min-width: 0;
  display: grid;
}

.label {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-weight: 600;
}

.label.untitled {
  color: var(--muted);
  font-weight: 400;
  font-style: italic;
}

.date {
  color: var(--faint);
  font-size: 12px;
}

.version .btn {
  padding: 3px 7px;
  font-size: 13px;
}
</style>
