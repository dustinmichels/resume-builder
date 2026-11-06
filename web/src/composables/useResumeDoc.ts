import { onScopeDispose, ref, shallowRef, watch } from 'vue'
import { api } from '../api'
import type { ResumeDoc } from '../types'

export type SaveStatus = 'loading' | 'saved' | 'unsaved' | 'saving' | 'error'

const AUTOSAVE_MS = 800

/** Loads the resume from the server and autosaves every change. */
export function useResumeDoc() {
  const doc = ref<ResumeDoc | null>(null)
  const status = shallowRef<SaveStatus>('loading')
  const error = shallowRef<string | null>(null)

  let savedJson = ''
  let timer: ReturnType<typeof setTimeout> | undefined
  // Saves run one at a time, each sending the latest document, so an older
  // request can never land after a newer one.
  let queue = Promise.resolve()

  async function writeLatest() {
    if (!doc.value) return
    const json = JSON.stringify(doc.value)
    if (json === savedJson) return
    status.value = 'saving'
    try {
      await api.saveResume(json)
      savedJson = json
      error.value = null
      status.value = JSON.stringify(doc.value) === json ? 'saved' : 'unsaved'
    } catch (e) {
      error.value = e instanceof Error ? e.message : String(e)
      status.value = 'error'
    }
  }

  function save() {
    clearTimeout(timer)
    queue = queue.then(writeLatest)
    return queue
  }

  watch(
    doc,
    (value) => {
      if (!value || status.value === 'loading') return
      if (JSON.stringify(value) === savedJson) return
      status.value = 'unsaved'
      clearTimeout(timer)
      timer = setTimeout(save, AUTOSAVE_MS)
    },
    { deep: true },
  )

  api
    .getResume()
    .then((loaded) => {
      savedJson = JSON.stringify(loaded)
      doc.value = loaded
      status.value = 'saved'
    })
    .catch((e) => {
      error.value = `Could not load the resume: ${e instanceof Error ? e.message : e}`
      status.value = 'error'
    })

  const warnUnsaved = (event: BeforeUnloadEvent) => {
    if (status.value === 'unsaved' || status.value === 'saving') event.preventDefault()
  }
  window.addEventListener('beforeunload', warnUnsaved)
  onScopeDispose(() => {
    clearTimeout(timer)
    window.removeEventListener('beforeunload', warnUnsaved)
  })

  return { doc, status, error, save }
}
