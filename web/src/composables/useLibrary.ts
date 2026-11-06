import { shallowRef } from 'vue'
import { api } from '../api'
import { formatDateTime } from '../format'
import type { ResumeSummary, Settings, Version } from '../types'
import { useResumeDoc } from './useResumeDoc'

const CURRENT_KEY = 'resume-builder:current'

/**
 * The data folder, the resumes in it and the open one. Every action that
 * touches the open resume saves its pending edits first, so nothing is lost
 * and no save lands in the wrong resume.
 */
export function useLibrary() {
  const settings = shallowRef<Settings | null>(null)
  const resumes = shallowRef<ResumeSummary[]>([])
  const currentId = shallowRef<string | null>(null)
  /** Why the library could not be loaded. */
  const error = shallowRef<string | null>(null)
  const resume = useResumeDoc(currentId)

  function select(id: string) {
    currentId.value = id
    localStorage.setItem(CURRENT_KEY, id)
  }

  async function refresh() {
    resumes.value = await api.listResumes()
  }

  /** Opens the last-used resume, else the most recently modified, else a new one. */
  async function openDefault() {
    try {
      await refresh()
      const stored = localStorage.getItem(CURRENT_KEY)
      let id = resumes.value.find((r) => r.id === stored)?.id ?? resumes.value[0]?.id
      if (!id) {
        id = await api.createResume('Resume')
        await refresh()
      }
      select(id)
      // The new folder may hold a resume with the same name as the old one.
      resume.sync()
    } catch (e) {
      error.value = `Could not open your resumes: ${e instanceof Error ? e.message : e}`
    }
  }

  async function init() {
    try {
      settings.value = await api.getSettings()
    } catch (e) {
      error.value = `Could not reach the server: ${e instanceof Error ? e.message : e}`
      return
    }
    if (settings.value?.dataDir) await openDefault()
  }

  /** Chooses (or switches) the data folder; existing files are not moved. */
  async function setDataDir(path: string) {
    const apply = async () => {
      settings.value = await api.setDataDir(path)
      resume.unload()
    }
    // Saves to the old folder must finish before the server switches folders.
    await (resume.doc.value ? resume.withSaved(apply) : apply())
    await openDefault()
  }

  async function switchTo(id: string) {
    if (id === currentId.value) return
    if (!(await resume.save())) throw new Error(`Could not save your changes: ${resume.error.value}`)
    select(id)
  }

  async function create(name: string) {
    const id = await api.createResume(name)
    await refresh()
    await switchTo(id)
  }

  async function duplicate(name: string) {
    const id = await resume.withSaved((from) => api.createResume(name, { resume: from }))
    await refresh()
    await switchTo(id)
  }

  async function rename(name: string) {
    const id = await resume.withSaved(async (from) => {
      const renamed = await api.renameResume(from, name)
      resume.retarget(renamed)
      return renamed
    })
    select(id)
    await refresh()
  }

  /** Moves the open resume to the trash and opens another. */
  async function remove() {
    await resume.withSaved(async (id) => {
      await api.deleteResume(id)
      resume.unload()
    })
    await openDefault()
  }

  async function saveVersion(label: string) {
    const version = await resume.withSaved((id) => api.saveVersion(id, label))
    await refresh()
    return version
  }

  /** The server snapshots the current state before restoring. */
  async function restoreVersion(version: Version) {
    await resume.withSaved(async (id) => resume.replace(await api.restoreVersion(id, version.id)))
    await refresh()
  }

  /** The copy gets the current side files, so pending edits to them are saved first. */
  async function copyVersion(version: Version) {
    if (!currentId.value) return
    const name = `${currentId.value} – ${version.label || formatDateTime(version.created)}`
    const id = await resume.withSaved((from) => api.createResume(name, { resume: from, version: version.id }))
    await refresh()
    await switchTo(id)
  }

  /** Pending edits are saved first so resume.md and resume.pdf match the editor. */
  function syncFiles() {
    return resume.withSaved((id) => api.syncFiles(id))
  }

  init()

  return {
    settings,
    resumes,
    currentId,
    error,
    resume,
    refresh,
    setDataDir,
    switchTo,
    create,
    duplicate,
    rename,
    remove,
    saveVersion,
    restoreVersion,
    copyVersion,
    syncFiles,
  }
}
