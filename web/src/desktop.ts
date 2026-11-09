// The desktop app (server/desktop.go) shows the editor in a web view, which
// can't open new windows and mustn't navigate away from the editor: links open
// in their default app through the server instead.
import { api, inApp } from './api'

const OUTSIDE = ['http:', 'https:', 'mailto:', 'tel:']

/** Switches the editor to the app's ways of downloading and opening links. */
export function runInApp() {
  if (inApp.value) return
  inApp.value = true
  document.addEventListener('click', openOutside, true)
}

function openOutside(e: MouseEvent) {
  const a = e.target instanceof Element ? e.target.closest('a[href]') : null
  // Links being edited (rich text) are only for placing the caret.
  if (!(a instanceof HTMLAnchorElement) || a.isContentEditable || e.defaultPrevented) return
  const url = new URL(a.href)
  if (url.origin === location.origin) return
  e.preventDefault()
  if (OUTSIDE.includes(url.protocol)) api.openLink(url.href).catch((err) => console.error('opening a link:', err))
}
