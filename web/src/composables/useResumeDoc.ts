import { onScopeDispose, ref, shallowRef, watch, type Ref } from 'vue'
import { api } from '../api'
import { jobLabel } from '../format'
import type { ContactInfo, CoverLetter, ResumeDoc, SupervisorsDoc } from '../types'

export type SaveStatus = 'loading' | 'saved' | 'unsaved' | 'saving' | 'error'

const AUTOSAVE_MS = 800

const message = (e: unknown) => (e instanceof Error ? e.message : String(e))

/**
 * Loads the resume `id` and its side documents (contact info, supervisors)
 * from the server and autosaves every change to them, and to the cover letter
 * opened with `setLetter`.
 * Changing `id` saves pending edits to the old resume before the new one loads.
 */
export function useResumeDoc(id: Readonly<Ref<string | null>>) {
  const doc = ref<ResumeDoc | null>(null)
  const contact = ref<ContactInfo | null>(null)
  const supervisors = ref<SupervisorsDoc | null>(null)
  const letter = ref<CoverLetter | null>(null)
  /** The open letter's name (its file name, without .json). */
  const letterId = shallowRef<string | null>(null)
  const status = shallowRef<SaveStatus>('loading')
  const error = shallowRef<string | null>(null)

  // The resume the documents were loaded from; every save goes there.
  let docId: string | null = null
  // The resume loaded or being loaded; guards against stale loads.
  let target: string | null = null
  let timer: ReturnType<typeof setTimeout> | undefined
  // Saves (and the operations run through `withSaved`) go one at a time, each
  // sending the latest documents, so an older request can never land after a
  // newer one or interleave with a rename, restore or delete.
  let queue: Promise<unknown> = Promise.resolve()

  // The files of the open resume: each document, how it is written, and its JSON as last saved.
  const files: { doc: Readonly<Ref<unknown>>; write: (id: string, json: string) => Promise<void>; saved: string }[] = [
    { doc, write: api.saveResume, saved: '' },
    { doc: contact, write: api.saveContact, saved: '' },
    { doc: supervisors, write: api.saveSupervisors, saved: '' },
    // Letters are switched inside the queue (see setLetter), so the id is the one the content was loaded from.
    { doc: letter, write: (id, json) => api.saveLetter(id, letterId.value!, json), saved: '' },
  ]
  const letterFile = files[3]!
  const unsaved = () => files.filter((f) => f.doc.value && JSON.stringify(f.doc.value) !== f.saved)

  function scheduleSave() {
    status.value = 'unsaved'
    clearTimeout(timer)
    timer = setTimeout(save, AUTOSAVE_MS)
  }

  /** Resolves true when the server has the current documents. */
  async function writeLatest(): Promise<boolean> {
    if (!docId) return true
    const pending = unsaved().map((f) => ({ f, json: JSON.stringify(f.doc.value) }))
    if (!pending.length) return true
    const to = docId
    status.value = 'saving'
    try {
      for (const { f, json } of pending) {
        await f.write(to, json)
        if (to === docId) f.saved = json
      }
      error.value = null
      // Edits made (or undone) while saving may not have scheduled a save of their own.
      if (unsaved().length) scheduleSave()
      else status.value = 'saved'
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

  /** Shows a resume the server already has (e.g. a restored version) without saving it. */
  function replace(next: ResumeDoc) {
    clearTimeout(timer)
    files[0]!.saved = JSON.stringify(next)
    doc.value = next
    error.value = null
    status.value = 'saved'
  }

  /** The open resume was renamed: keep the documents, save to `next` from now on. */
  function retarget(next: string) {
    docId = next
    target = next
  }

  /**
   * Opens the letter `next` of the open resume, with its content as the
   * server has it, or closes the open letter (null). Run it inside `withSaved`,
   * so the edits to the letter it replaces are saved first.
   */
  function setLetter(next: string | null, content: CoverLetter | null) {
    letterId.value = next
    letterFile.saved = content ? JSON.stringify(content) : ''
    letter.value = content
  }

  /** The open letter was renamed: keep it, save to `next` from now on. */
  function retargetLetter(next: string) {
    letterId.value = next
  }

  /** Closes the documents without saving (their resume was deleted or the data folder changed). */
  function unload() {
    clearTimeout(timer)
    doc.value = contact.value = supervisors.value = null
    letter.value = null
    letterId.value = null
    for (const f of files) f.saved = ''
    docId = null
    target = null
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
      return next ? Promise.all([api.getResume(next), api.getContact(next), api.getSupervisors(next)]) : null
    })
    loaded
      .then((values) => {
        if (!values || target !== next) return
        docId = next
        ;[doc.value, contact.value, supervisors.value] = values
        for (const f of files) f.saved = JSON.stringify(f.doc.value)
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
    [doc, contact, supervisors, letter],
    () => {
      if (!docId || status.value === 'loading') return
      if (unsaved().length) scheduleSave()
      else if (status.value === 'unsaved') {
        // The edits were reverted (e.g. undone): nothing is left to save.
        clearTimeout(timer)
        status.value = 'saved'
      }
    },
    { deep: true },
  )

  // Keep each supervisor's job label (see Supervisor.job) in step with its entry.
  watch(
    [doc, supervisors],
    () => {
      if (!doc.value || !supervisors.value) return
      const labels = new Map<string, string>()
      for (const section of doc.value.sections) {
        if (section.kind !== 'entries') continue
        for (const item of section.items) if (item.type === 'entry') labels.set(item.id, jobLabel(item))
      }
      for (const s of supervisors.value.supervisors) {
        const label = labels.get(s.entryId)
        if (label !== undefined && label !== s.job) s.job = label
      }
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

  return {
    doc,
    contact,
    supervisors,
    letter,
    letterId,
    status,
    error,
    save,
    withSaved,
    replace,
    retarget,
    setLetter,
    retargetLetter,
    unload,
    sync,
  }
}
