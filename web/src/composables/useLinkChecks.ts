import { shallowRef, watch, type Ref } from 'vue'
import { api, type DocKind } from '../api'
import type { Check, PageDoc } from '../types'

// Longer than the preview's: requests reach other sites, so wait for a pause.
const DEBOUNCE_MS = 1000

/**
 * Requests the document's web links whenever it changes. Results stay up until
 * new ones arrive, so edits elsewhere don't flash "Checking…"; they reset when
 * another document is opened.
 */
export function useLinkChecks(kind: DocKind, doc: Readonly<Ref<PageDoc | null>>) {
  const checks = shallowRef<Check[] | null>(null)
  const error = shallowRef<string | null>(null)

  watch(
    () => doc.value,
    () => {
      checks.value = null
      error.value = null
    },
  )

  watch(
    () => (doc.value ? JSON.stringify(doc.value) : null),
    (json, _old, onCleanup) => {
      if (!json) return
      const controller = new AbortController()
      const timer = setTimeout(async () => {
        try {
          checks.value = await api.checkLinks(kind, json, controller.signal)
          error.value = null
        } catch (e) {
          if (controller.signal.aborted) return
          error.value = e instanceof Error ? e.message : String(e)
        }
      }, DEBOUNCE_MS)
      onCleanup(() => {
        clearTimeout(timer)
        controller.abort()
      })
    },
    { immediate: true },
  )

  return { checks, error }
}
