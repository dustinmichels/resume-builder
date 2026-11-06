import { onScopeDispose, shallowRef, watch, type Ref } from 'vue'

/** Open state for a popup menu that closes on Escape or a pointer press outside `root`. */
export function useMenu(root: Readonly<Ref<HTMLElement | null>>) {
  const open = shallowRef(false)

  function onPointerDown(event: PointerEvent) {
    if (!root.value?.contains(event.target as Node)) open.value = false
  }

  function onKeyDown(event: KeyboardEvent) {
    if (event.key === 'Escape') open.value = false
  }

  function stopListening() {
    document.removeEventListener('pointerdown', onPointerDown)
    document.removeEventListener('keydown', onKeyDown)
  }

  watch(open, (isOpen) => {
    if (!isOpen) return stopListening()
    document.addEventListener('pointerdown', onPointerDown)
    document.addEventListener('keydown', onKeyDown)
  })

  onScopeDispose(stopListening)

  return open
}
