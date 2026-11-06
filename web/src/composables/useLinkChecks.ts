import { shallowRef, watch, type Ref } from 'vue'
import { api } from '../api'
import type { Check, ResumeDoc } from '../types'

// Longer than the preview's: requests reach other sites, so wait for a pause.
const DEBOUNCE_MS = 1000

/**
 * Requests the resume's web links whenever it changes. Results stay up until
 * new ones arrive, so edits elsewhere don't flash "Checking…"; they reset when
 * another resume is opened.
 */
export function useLinkChecks(doc: Readonly<Ref<ResumeDoc | null>>) {
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
          checks.value = await api.checkLinks(json, controller.signal)
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
