<script setup lang="ts">
import { shallowRef } from 'vue'

const props = defineProps<{
  defaultDataDir: string
  choose: (path: string) => Promise<void>
}>()

const path = shallowRef(props.defaultDataDir)
const busy = shallowRef(false)
const error = shallowRef<string | null>(null)

async function submit() {
  busy.value = true
  error.value = null
  try {
    await props.choose(path.value.trim())
  } catch (e) {
    error.value = e instanceof Error ? e.message : String(e)
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <main class="setup">
    <form class="card" @submit.prevent="submit">
      <h1 class="title">Choose where to keep your resumes</h1>
      <p class="text">
        Your resumes and their saved versions are kept as plain JSON files in this folder. It is created if it doesn't
        exist yet.
      </p>
      <label class="field">
        <span class="field-label">Data folder</span>
        <input v-model="path" class="input" required spellcheck="false" autocomplete="off" />
      </label>
      <p v-if="error" class="error" role="alert">{{ error }}</p>
      <div class="buttons">
        <button class="btn btn-primary" type="submit" :disabled="busy">
          {{ busy ? 'Creating…' : 'Create folder' }}
        </button>
      </div>
    </form>
  </main>
</template>

<style scoped>
.setup {
  display: grid;
  place-items: center;
  min-height: 100vh;
  padding: 24px;
  background: var(--sunken);
}

.card {
  display: grid;
  gap: 14px;
  width: min(520px, 100%);
  padding: 26px 28px;
  border: 1px solid var(--rule);
  border-radius: 10px;
  background: var(--panel);
}

.title {
  margin: 0;
  font-size: 18px;
  font-weight: 650;
  color: var(--accent);
}

.text {
  margin: 0;
  color: var(--muted);
}

.error {
  margin: 0;
  color: var(--error);
  font-size: 13px;
}

.buttons {
  display: flex;
  justify-content: flex-end;
}

.btn:disabled {
  opacity: 0.6;
  cursor: default;
}
</style>
