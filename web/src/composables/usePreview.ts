import { onScopeDispose, shallowRef, watch, type Ref } from 'vue'
import { api, type DocKind } from '../api'
import type { PageDoc, Preview } from '../types'

const DEBOUNCE_MS = 250

/**
 * Re-renders the document (pages, text layer, checks) whenever it changes.
 * While `doc` is null the last result stays, so showing the document again
 * is instant until its fresh render arrives.
 */
export function usePreview(kind: DocKind, doc: Readonly<Ref<PageDoc | null>>) {
  const preview = shallowRef<Preview | null>(null)
  const pageUrls = shallowRef<string[]>([])
  const pending = shallowRef(false)
  const error = shallowRef<string | null>(null)

  // Pages are shown as <img> blob URLs rather than inline SVG, so content in
  // the SVG (e.g. link targets) can never run as script in the app.
  function setPages(svgs: string[]) {
    pageUrls.value.forEach(URL.revokeObjectURL)
    pageUrls.value = svgs.map((svg) => URL.createObjectURL(new Blob([svg], { type: 'image/svg+xml' })))
  }

  watch(
    () => (doc.value ? JSON.stringify(doc.value) : null),
    (json, _old, onCleanup) => {
      if (!json) {
        pending.value = false
        return
      }
      // Pending from the edit itself, not after the debounce, so the preview
      // reacts the moment something changes.
      pending.value = true
      const controller = new AbortController()
      const timer = setTimeout(async () => {
        try {
          const result = await api.preview(kind, json, controller.signal)
          setPages(result.pages)
          preview.value = result
          error.value = null
        } catch (e) {
          if (controller.signal.aborted) return
          error.value = e instanceof Error ? e.message : String(e)
        } finally {
          if (!controller.signal.aborted) pending.value = false
        }
      }, DEBOUNCE_MS)
      onCleanup(() => {
        clearTimeout(timer)
        controller.abort()
      })
    },
    { immediate: true },
  )

  onScopeDispose(() => pageUrls.value.forEach(URL.revokeObjectURL))

  return { preview, pageUrls, pending, error }
}
