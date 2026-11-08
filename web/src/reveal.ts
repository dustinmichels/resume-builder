// Revealing a value in the sidebar: the cards enclosing its field path expand
// and every other card collapses, then App scrolls to the input tagged
// data-path="<path>" (see fieldPath.ts).
import { inject, provide, shallowRef, watch, type InjectionKey, type Ref, type ShallowRef } from 'vue'

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
 * Sets `open` when a reveal targets a value: true if the value lies inside the
 * card at `path`, false otherwise. Includes a reveal already in flight when the
 * card mounts.
 */
export function useOpenOnReveal(path: () => string, open: Ref<boolean>) {
  const target = inject(KEY, shallowRef(null))
  watch(
    target,
    (t) => {
      if (t !== null) open.value = t.startsWith(`${path()}.`)
    },
    { immediate: true },
  )
}
