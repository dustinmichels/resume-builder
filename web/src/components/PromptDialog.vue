<script lang="ts">
/** A confirmation, or a question with one text answer when `value` is set. */
export interface PromptRequest {
  title: string
  /** Line breaks are kept. */
  message?: string
  /** Initial answer; omit for a plain confirmation. */
  value?: string
  placeholder?: string
  confirmLabel: string
  danger?: boolean
  /** Runs on confirm; the dialog stays open showing the error if it throws. */
  run: (value: string) => unknown
}
</script>

<script setup lang="ts">
import { nextTick, shallowRef, useTemplateRef, watch } from 'vue'

const request = defineModel<PromptRequest | null>({ required: true })

const dialog = useTemplateRef<HTMLDialogElement>('dialog')
const input = useTemplateRef<HTMLInputElement>('input')
const value = shallowRef('')
const busy = shallowRef(false)
const error = shallowRef<string | null>(null)

watch(request, async (next) => {
  error.value = null
  if (!next) {
    dialog.value?.close()
    return
  }
  value.value = next.value ?? ''
  await nextTick()
  if (!dialog.value?.open) dialog.value?.showModal()
  input.value?.select()
})

async function submit() {
  const current = request.value
  if (!current || busy.value) return
  busy.value = true
  error.value = null
  try {
    await current.run(value.value.trim())
    if (request.value === current) request.value = null
  } catch (e) {
    error.value = e instanceof Error ? e.message : String(e)
  } finally {
    busy.value = false
  }
}

function onCancel(event: Event) {
  if (busy.value) event.preventDefault()
}
</script>

<template>
  <dialog ref="dialog" class="prompt" @cancel="onCancel" @close="request = null">
    <form v-if="request" @submit.prevent="submit">
      <h2 class="title">{{ request.title }}</h2>
      <p v-if="request.message" class="message">{{ request.message }}</p>
      <input
        v-if="request.value !== undefined"
        ref="input"
        v-model="value"
        class="input"
        :placeholder="request.placeholder"
        required
        :aria-label="request.title"
      />
      <p v-if="error" class="error" role="alert">{{ error }}</p>
      <div class="buttons">
        <button class="btn btn-quiet" type="button" :disabled="busy" @click="request = null">Cancel</button>
        <button class="btn" :class="request.danger ? 'btn-danger' : 'btn-primary'" type="submit" :disabled="busy">
          {{ busy ? 'Working…' : request.confirmLabel }}
        </button>
      </div>
    </form>
  </dialog>
</template>

<style scoped>
.prompt {
  width: min(440px, calc(100vw - 32px));
  padding: 18px 20px 16px;
  border: 1px solid var(--rule);
  border-radius: 10px;
  background: var(--panel);
  color: var(--ink);
  box-shadow: 0 12px 40px rgb(0 0 0 / 0.18);
}

.prompt::backdrop {
  background: rgb(29 36 48 / 0.28);
}

form {
  display: grid;
  gap: 12px;
}

.title {
  margin: 0;
  font-size: 15px;
  font-weight: 650;
}

.message {
  margin: 0;
  color: var(--muted);
  white-space: pre-line;
  overflow-wrap: anywhere;
}

.error {
  margin: 0;
  color: var(--error);
  font-size: 13px;
}

.buttons {
  display: flex;
  justify-content: flex-end;
  gap: 6px;
}

.btn:disabled {
  opacity: 0.6;
  cursor: default;
}

.btn-danger {
  border-color: var(--error);
  background: var(--error);
  color: #fff;
}

.btn-danger:hover {
  background: color-mix(in srgb, var(--error) 85%, black);
}
</style>
