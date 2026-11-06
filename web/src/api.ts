import type { Preview, ResumeDoc, ResumeSummary, Settings, Version } from './types'

/** Formats the server renders; each is also the file extension. */
export type ExportFormat = 'pdf' | 'docx' | 'md'

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

const send = (method: string, body: string, signal?: AbortSignal): RequestInit => ({
  method,
  headers: { 'Content-Type': 'application/json' },
  body,
  signal,
})

const resumePath = (id: string) => `/api/resumes/${encodeURIComponent(id)}`
const versionPath = (id: string, version: string) => `${resumePath(id)}/versions/${encodeURIComponent(version)}`

export const api = {
  async getSettings(): Promise<Settings> {
    return (await request('/api/settings')).json()
  },
  /** Creates the folder if needed and seeds it with a resume when it has none. */
  async setDataDir(dataDir: string): Promise<Settings> {
    return (await request('/api/settings', send('PUT', JSON.stringify({ dataDir })))).json()
  },

  /** Most recently modified first. */
  async listResumes(): Promise<ResumeSummary[]> {
    return (await request('/api/resumes')).json()
  },
  /**
   * Creates a resume: blank, a copy of `from`, or a copy of one of its
   * versions. The server makes the name unique; the returned id is final.
   */
  async createResume(name: string, from?: { resume: string; version?: string }): Promise<string> {
    const body = { name, from: from?.resume, version: from?.version }
    return (await (await request('/api/resumes', send('POST', JSON.stringify(body)))).json()).id
  },
  async getResume(id: string): Promise<ResumeDoc> {
    return (await request(resumePath(id))).json()
  },
  async saveResume(id: string, json: string): Promise<void> {
    await request(resumePath(id), send('PUT', json))
  },
  /** Returns the new id. */
  async renameResume(id: string, name: string): Promise<string> {
    return (await (await request(resumePath(id), send('PATCH', JSON.stringify({ name })))).json()).id
  },
  /** Moves the resume folder into the data folder's .trash. */
  async deleteResume(id: string): Promise<void> {
    await request(resumePath(id), { method: 'DELETE' })
  },

  /** Newest first. */
  async listVersions(id: string): Promise<Version[]> {
    return (await request(`${resumePath(id)}/versions`)).json()
  },
  /** Snapshots the resume as last saved; save pending edits first. */
  async saveVersion(id: string, label: string): Promise<Version> {
    return (await request(`${resumePath(id)}/versions`, send('POST', JSON.stringify({ label })))).json()
  },
  async getVersion(id: string, version: string): Promise<ResumeDoc> {
    return (await request(versionPath(id, version))).json()
  },
  /** Snapshots the current resume, then replaces it with the version. Returns the restored document. */
  async restoreVersion(id: string, version: string): Promise<ResumeDoc> {
    return (await request(`${versionPath(id, version)}/restore`, { method: 'POST' })).json()
  },
  async deleteVersion(id: string, version: string): Promise<void> {
    await request(versionPath(id, version), { method: 'DELETE' })
  },

  async preview(json: string, signal: AbortSignal): Promise<Preview> {
    return (await request('/api/preview', send('POST', json, signal))).json()
  },
  async exportAs(format: ExportFormat, json: string): Promise<Blob> {
    return (await request(`/api/${format}`, send('POST', json))).blob()
  },
}

export function download(data: Blob, filename: string) {
  const url = URL.createObjectURL(data)
  const a = document.createElement('a')
  a.href = url
  a.download = filename
  a.click()
  URL.revokeObjectURL(url)
}
