import type { Preview, ResumeDoc } from './types'

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

const jsonPost = (body: string, signal?: AbortSignal): RequestInit => ({
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body,
  signal,
})

export const api = {
  async getResume(): Promise<ResumeDoc> {
    return (await request('/api/resume')).json()
  },
  async saveResume(json: string): Promise<void> {
    await request('/api/resume', { ...jsonPost(json), method: 'PUT' })
  },
  async preview(json: string, signal: AbortSignal): Promise<Preview> {
    return (await request('/api/preview', jsonPost(json, signal))).json()
  },
  async pdf(json: string): Promise<Blob> {
    return (await request('/api/pdf', jsonPost(json))).blob()
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
