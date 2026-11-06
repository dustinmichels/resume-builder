import { onScopeDispose, ref, shallowRef, watch, type Ref } from 'vue'
import { api } from '../api'
import type { ResumeDoc } from '../types'

export type SaveStatus = 'loading' | 'saved' | 'unsaved' | 'saving' | 'error'

const AUTOSAVE_MS = 800

const message = (e: unknown) => (e instanceof Error ? e.message : String(e))

/**
 * Loads the resume `id` from the server and autosaves every change to it.
 * Changing `id` saves pending edits to the old resume before the new one loads.
 */
export function useResumeDoc(id: Readonly<Ref<string | null>>) {
  const doc = ref<ResumeDoc | null>(null)
  const status = shallowRef<SaveStatus>('loading')
  const error = shallowRef<string | null>(null)

  // The resume the doc was loaded from; every save goes there.
  let docId: string | null = null
  // The resume loaded or being loaded; guards against stale loads.
  let target: string | null = null
  let savedJson = ''
  let timer: ReturnType<typeof setTimeout> | undefined
  // Saves (and the operations run through `withSaved`) go one at a time, each
  // sending the latest document, so an older request can never land after a
  // newer one or interleave with a rename, restore or delete.
  let queue: Promise<unknown> = Promise.resolve()

  /** Resolves true when the server has the current document. */
  async function writeLatest(): Promise<boolean> {
    if (!doc.value || !docId) return true
    const json = JSON.stringify(doc.value)
    if (json === savedJson) return true
    const to = docId
    status.value = 'saving'
    try {
      await api.saveResume(to, json)
      if (to === docId) savedJson = json
      error.value = null
      status.value = JSON.stringify(doc.value) === json ? 'saved' : 'unsaved'
      return true
    } catch (e) {
      error.value = message(e)
      status.value = 'error'
      return false
    }
  }

  function enqueue<T>(task: () => Promise<T>): Promise<T> {
    const result = queue.then(task)
    queue = result.catch(() => {})
    return result
  }

  /** Saves pending edits now; resolves true when nothing is left unsaved. */
  function save() {
    clearTimeout(timer)
    return enqueue(writeLatest)
  }

  /**
   * Saves pending edits, then runs `task` on the open resume before any later
   * save. Rejects without running `task` if the edits could not be saved.
   */
  function withSaved<T>(task: (id: string) => Promise<T>): Promise<T> {
    clearTimeout(timer)
    return enqueue(async () => {
      if (!(await writeLatest())) throw new Error(`Could not save your changes: ${error.value}`)
      if (!docId) throw new Error('No resume is open.')
      return task(docId)
    })
  }

  /** Shows a document the server already has (e.g. a restored version) without saving it. */
  function replace(next: ResumeDoc) {
    clearTimeout(timer)
    savedJson = JSON.stringify(next)
    doc.value = next
    error.value = null
    status.value = 'saved'
  }

  /** The open resume was renamed: keep the document, save to `next` from now on. */
  function retarget(next: string) {
    docId = next
    target = next
  }

  /** Closes the document without saving (its resume was deleted or the data folder changed). */
  function unload() {
    clearTimeout(timer)
    doc.value = null
    docId = null
    target = null
    savedJson = ''
    error.value = null
    status.value = 'loading'
  }

  function open(next: string | null) {
    target = next
    clearTimeout(timer)
    const loaded = enqueue(async () => {
      await writeLatest()
      if (target !== next) return null
      unload()
      target = next
      return next ? api.getResume(next) : null
    })
    loaded
      .then((value) => {
        if (!value || target !== next) return
        savedJson = JSON.stringify(value)
        docId = next
        doc.value = value
        status.value = 'saved'
      })
      .catch((e) => {
        if (target !== next) return
        error.value = `Could not load the resume: ${message(e)}`
        status.value = 'error'
      })
  }

  /** Opens `id` if it is not the resume already open (e.g. after `unload`). */
  function sync() {
    if (id.value !== target) open(id.value)
  }

  watch(id, sync, { immediate: true })

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

  const warnUnsaved = (event: BeforeUnloadEvent) => {
    if (status.value === 'unsaved' || status.value === 'saving') event.preventDefault()
  }
  window.addEventListener('beforeunload', warnUnsaved)
  onScopeDispose(() => {
    clearTimeout(timer)
    window.removeEventListener('beforeunload', warnUnsaved)
  })

  return { doc, status, error, save, withSaved, replace, retarget, unload, sync }
}
