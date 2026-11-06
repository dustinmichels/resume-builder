// Revealing a value in the sidebar: the cards enclosing its field path expand,
// then App scrolls to the input tagged data-path="<path>" (see fieldPath.ts).
import { inject, provide, shallowRef, watch, type InjectionKey, type ShallowRef } from 'vue'

const KEY: InjectionKey<ShallowRef<string | null>> = Symbol('reveal')

/**
 * Provides the path being revealed. Set it, wait a tick for the cards to
 * expand, then clear it so cards mounted later don't expand for it.
 */
export function provideReveal(): ShallowRef<string | null> {
  const target = shallowRef<string | null>(null)
  provide(KEY, target)
  return target
}

/**
 * Calls `expand` when a reveal targets a value inside the card at `path`,
 * including a reveal already in flight when the card mounts.
 */
export function useExpandOnReveal(path: () => string, expand: () => void) {
  const target = inject(KEY, shallowRef(null))
  watch(
    target,
    (t) => {
      if (t?.startsWith(`${path()}.`)) expand()
    },
    { immediate: true },
  )
}
