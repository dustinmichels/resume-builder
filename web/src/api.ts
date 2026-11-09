import { shallowRef } from 'vue'
import { newContactInfo, newSupervisor } from './factory'
import type {
  Check,
  ContactInfo,
  CoverLetter,
  DocSummary,
  Preview,
  ResumeDoc,
  SavedVersion,
  Settings,
  SupervisorsDoc,
  SyncResult,
  TemplateInfo,
  Version,
} from './types'

/** Formats the server renders; each is also the file extension. */
export type ExportFormat = 'pdf' | 'docx' | 'md'

/** What a rendering call gets: a ResumeDoc, or a LetterDoc (letter + its resume's basics and style). */
export type DocKind = 'resume' | 'letter'
const renderPath = (kind: DocKind, what: string) => (kind === 'letter' ? `/api/letter/${what}` : `/api/${what}`)

async function request(path: string, init?: RequestInit): Promise<Response> {
  const res = await fetch(path, init)
  if (!res.ok) {
    let message = `${res.status} ${res.statusText}`
    try {
      message = (await res.json()).error ?? message
    } catch {
      // non-JSON error body; keep the status line
    }
    throw new Error(message)
  }
  return res
}

async function fetchJSON(path: string, init?: RequestInit) {
  return (await request(path, init)).json()
}

const send = (method: string, body: string, signal?: AbortSignal): RequestInit => ({
  method,
  headers: { 'Content-Type': 'application/json' },
  body,
  signal,
})

const resumePath = (id: string) => `/api/resumes/${encodeURIComponent(id)}`
const versionPath = (id: string, version: string) => `${resumePath(id)}/versions/${encodeURIComponent(version)}`
const letterPath = (id: string, letter: string) => `${resumePath(id)}/letters/${encodeURIComponent(letter)}`

export const api = {
  async getSettings(): Promise<Settings> {
    return fetchJSON('/api/settings')
  },
  /** Creates the folder if needed and seeds it with a resume when it has none. */
  async setDataDir(dataDir: string): Promise<Settings> {
    return fetchJSON('/api/settings', send('PUT', JSON.stringify({ dataDir })))
  },
  /** Asks for a folder in a native dialog (app only); '' if cancelled. */
  async chooseFolder(): Promise<string> {
    return (await fetchJSON('/api/choose-folder', { method: 'POST' })).path
  },
  /** Opens a web, mail or phone link in its default app (app only). */
  async openLink(url: string): Promise<void> {
    await request('/api/open', send('POST', JSON.stringify({ url })))
  },

  /** Classic first. */
  async getTemplates(): Promise<TemplateInfo[]> {
    return fetchJSON('/api/templates')
  },

  /** Most recently modified first. */
  async listResumes(): Promise<DocSummary[]> {
    return fetchJSON('/api/resumes')
  },
  /**
   * Creates a resume: blank, a copy of `from`, or a copy of one of its
   * versions. The server makes the name unique; the returned id is final.
   */
  async createResume(name: string, from?: { resume: string; version?: string }): Promise<string> {
    const body = { name, from: from?.resume, version: from?.version }
    return (await fetchJSON('/api/resumes', send('POST', JSON.stringify(body)))).id
  },
  async getResume(id: string): Promise<ResumeDoc> {
    return fetchJSON(resumePath(id))
  },
  async saveResume(id: string, json: string): Promise<void> {
    await request(resumePath(id), send('PUT', json))
  },
  /** Returns the new id. */
  async renameResume(id: string, name: string): Promise<string> {
    return (await fetchJSON(resumePath(id), send('PATCH', JSON.stringify({ name })))).id
  },
  /** Moves the resume folder into the data folder's .trash. */
  async deleteResume(id: string): Promise<void> {
    await request(resumePath(id), { method: 'DELETE' })
  },

  /** Files the resume never saved read as defaults. */
  async getContact(id: string): Promise<ContactInfo> {
    return { ...newContactInfo(), ...(await fetchJSON(`${resumePath(id)}/contact`)) }
  },
  async saveContact(id: string, json: string): Promise<void> {
    await request(`${resumePath(id)}/contact`, send('PUT', json))
  },
  async getSupervisors(id: string): Promise<SupervisorsDoc> {
    const { supervisors = [], ...rest }: Partial<SupervisorsDoc> = await fetchJSON(`${resumePath(id)}/supervisors`)
    return { ...rest, supervisors: supervisors.map((s) => ({ ...newSupervisor(s.entryId, s.job), ...s })) }
  },
  async saveSupervisors(id: string, json: string): Promise<void> {
    await request(`${resumePath(id)}/supervisors`, send('PUT', json))
  },

  /** Newest first. */
  async listVersions(id: string): Promise<Version[]> {
    return fetchJSON(`${resumePath(id)}/versions`)
  },
  /** Snapshots the resume, its side files and cover letters as last saved, with their copies; save pending edits first. */
  async saveVersion(id: string, label: string): Promise<SavedVersion> {
    return fetchJSON(`${resumePath(id)}/versions`, send('POST', JSON.stringify({ label })))
  },
  /** Snapshots the current resume, then replaces it with the version. Returns the restored document. */
  async restoreVersion(id: string, version: string): Promise<ResumeDoc> {
    return fetchJSON(`${versionPath(id, version)}/restore`, { method: 'POST' })
  },
  async deleteVersion(id: string, version: string): Promise<void> {
    await request(versionPath(id, version), { method: 'DELETE' })
  },
  /** Writes a Markdown, Word and PDF copy beside resume.json, each saved version and each cover letter, skipping up-to-date ones. */
  async syncFiles(id: string): Promise<SyncResult> {
    return fetchJSON(`${resumePath(id)}/sync`, { method: 'POST' })
  },
  /** Opens the resume's folder in Finder (or the system file manager). */
  async revealResume(id: string): Promise<void> {
    await request(`${resumePath(id)}/reveal`, { method: 'POST' })
  },

  /** The resume's cover letters, most recently modified first. */
  async listLetters(id: string): Promise<DocSummary[]> {
    return fetchJSON(`${resumePath(id)}/letters`)
  },
  /** Saves a new letter, or a copy of letter `from`. The server makes the name unique; the returned id is final. */
  async createLetter(id: string, name: string, source: { letter: CoverLetter } | { from: string }): Promise<string> {
    return (await fetchJSON(`${resumePath(id)}/letters`, send('POST', JSON.stringify({ name, ...source })))).id
  },
  async getLetter(id: string, letter: string): Promise<CoverLetter> {
    return fetchJSON(letterPath(id, letter))
  },
  async saveLetter(id: string, letter: string, json: string): Promise<void> {
    await request(letterPath(id, letter), send('PUT', json))
  },
  /** Returns the new id. */
  async renameLetter(id: string, letter: string, name: string): Promise<string> {
    return (await fetchJSON(letterPath(id, letter), send('PATCH', JSON.stringify({ name })))).id
  },
  /** Moves the letter into the data folder's .trash. */
  async deleteLetter(id: string, letter: string): Promise<void> {
    await request(letterPath(id, letter), { method: 'DELETE' })
  },

  async preview(kind: DocKind, json: string, signal: AbortSignal): Promise<Preview> {
    return fetchJSON(renderPath(kind, 'preview'), send('POST', json, signal))
  },
  /** Requests the document's web links; the server remembers answers for a few minutes. */
  async checkLinks(kind: DocKind, json: string, signal: AbortSignal): Promise<Check[]> {
    return fetchJSON(renderPath(kind, 'links'), send('POST', json, signal))
  },
  async exportAs(kind: DocKind, format: ExportFormat, json: string): Promise<Blob> {
    return (await request(renderPath(kind, format), send('POST', json))).blob()
  },
}

/** Set from GET /api/settings when the editor runs in the desktop app's web view rather than a browser. */
export const inApp = shallowRef(false)

/**
 * Saves data as filename: a browser download, or in the app (whose web view
 * doesn't download) a native Save dialog. Resolves false if that was cancelled.
 */
export async function download(data: Blob, filename: string): Promise<boolean> {
  if (inApp.value) {
    return (await fetchJSON(`/api/save?name=${encodeURIComponent(filename)}`, { method: 'POST', body: data })).path !== ''
  }
  const url = URL.createObjectURL(data)
  const a = document.createElement('a')
  a.href = url
  a.download = filename
  a.click()
  URL.revokeObjectURL(url)
  return true
}
