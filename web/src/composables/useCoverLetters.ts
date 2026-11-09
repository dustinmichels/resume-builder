import { shallowRef, watch, type Ref } from 'vue'
import { api } from '../api'
import { newCoverLetter } from '../factory'
import type { CoverLetter, DocSummary, ResumeDoc } from '../types'

/** What the letters need from the open resume's documents (useResumeDoc). */
export interface OpenResume {
  doc: Readonly<Ref<ResumeDoc | null>>
  letterId: Readonly<Ref<string | null>>
  withSaved<T>(task: (id: string) => Promise<T>): Promise<T>
  setLetter(next: string | null, content: CoverLetter | null): void
  retargetLetter(next: string): void
}

/**
 * The cover letters of the open resume. The open letter itself lives in
 * `resume` (letter, letterId), so its autosave shares the resume's save queue;
 * every action here saves pending edits first, as the resume actions do.
 */
export function useCoverLetters(resumeId: Readonly<Ref<string | null>>, resume: OpenResume) {
  /** Most recently modified first. */
  const letters = shallowRef<DocSummary[]>([])
  /** Whether `letters` has been loaded for the open resume. */
  const loaded = shallowRef(false)

  async function refresh() {
    const id = resumeId.value
    if (!id) return
    const list = await api.listLetters(id)
    if (id !== resumeId.value) return
    letters.value = list
    loaded.value = true
  }

  watch(
    resumeId,
    () => {
      letters.value = []
      loaded.value = false
      refresh().catch(() => {})
    },
    { immediate: true },
  )

  async function open(lid: string) {
    if (lid === resume.letterId.value) return
    await resume.withSaved(async (id) => resume.setLetter(lid, await api.getLetter(id, lid)))
  }

  /** A new letter, laid out like the sample and headed with the resume's details. */
  async function create(name: string) {
    await resume.withSaved(async (id) => {
      if (!resume.doc.value) throw new Error('The resume has not loaded yet.')
      const content = newCoverLetter(resume.doc.value)
      resume.setLetter(await api.createLetter(id, name, { letter: content }), content)
    })
    await refresh()
  }

  async function duplicate(name: string) {
    await resume.withSaved(async (id) => {
      const from = resume.letterId.value
      if (!from) throw new Error('No cover letter is open.')
      const lid = await api.createLetter(id, name, { from })
      resume.setLetter(lid, await api.getLetter(id, lid))
    })
    await refresh()
  }

  async function rename(name: string) {
    await resume.withSaved(async (id) => {
      const from = resume.letterId.value
      if (!from) throw new Error('No cover letter is open.')
      resume.retargetLetter(await api.renameLetter(id, from, name))
    })
    await refresh()
  }

  /** Moves the open letter to the trash and closes it. */
  async function remove() {
    await resume.withSaved(async (id) => {
      const lid = resume.letterId.value
      if (!lid) return
      await api.deleteLetter(id, lid)
      resume.setLetter(null, null)
    })
    await refresh()
  }

  return { letters, loaded, open, create, duplicate, rename, remove }
}
