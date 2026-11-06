import { computed, shallowRef, toRaw, watch, type Ref } from 'vue'

/** Edits closer together than this form one undo step (typing a word, dragging a slider). */
const MERGE_MS = 1000
const LIMIT = 200

/**
 * Undo and redo for the document in `source`. A different document object
 * (another resume opened, a version restored) starts a fresh history.
 * Undo and redo rewrite the document in place, so editors bound to it stay bound.
 */
export function useUndoHistory<T extends object>(source: Readonly<Ref<T | null>>) {
  // JSON snapshots: the states before each step (oldest first), and the states undone (latest last).
  const past = shallowRef<readonly string[]>([])
  const future = shallowRef<readonly string[]>([])
  // The document object being tracked and its JSON as last recorded.
  let tracked: T | null = null
  let current = ''
  let lastEdit = 0

  watch(
    source,
    (value) => {
      const raw = value && toRaw(value)
      if (raw !== tracked) {
        tracked = raw
        current = value ? JSON.stringify(value) : ''
        past.value = []
        future.value = []
        lastEdit = 0
        return
      }
      if (!value) return
      const json = JSON.stringify(value)
      if (json === current) return
      const now = Date.now()
      if (now - lastEdit > MERGE_MS) past.value = [...past.value.slice(1 - LIMIT), current]
      lastEdit = now
      future.value = []
      current = json
    },
    { deep: true },
  )

  function apply(json: string) {
    const target = source.value as Record<string, unknown>
    const next = JSON.parse(json) as Record<string, unknown>
    for (const key of Object.keys(target)) if (!(key in next)) delete target[key]
    Object.assign(target, next)
    current = JSON.stringify(target)
    // The next edit starts a new step instead of joining the one before the undo.
    lastEdit = 0
  }

  function undo() {
    const previous = past.value.at(-1)
    if (previous === undefined || !source.value) return
    past.value = past.value.slice(0, -1)
    future.value = [...future.value, current]
    apply(previous)
  }

  function redo() {
    const next = future.value.at(-1)
    if (next === undefined || !source.value) return
    future.value = future.value.slice(0, -1)
    past.value = [...past.value, current]
    apply(next)
  }

  return {
    canUndo: computed(() => past.value.length > 0),
    canRedo: computed(() => future.value.length > 0),
    undo,
    redo,
  }
}
