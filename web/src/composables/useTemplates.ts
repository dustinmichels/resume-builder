import { computed, inject, provide, shallowRef, type ComputedRef, type InjectionKey, type Ref } from 'vue'
import { api } from '../api'
import type { Style, TemplateFeature, TemplateInfo } from '../types'

/** What the editor offers until the list loads, or if it can't. */
const CLASSIC: TemplateInfo = { id: '', name: 'Classic', features: ['markers', 'uppercaseTitles'], icons: [] }

// Loaded once per session; the server lists the templates it can render.
const templates = shallowRef<TemplateInfo[]>([CLASSIC])
let requested = false

function load() {
  if (requested) return
  requested = true
  api.getTemplates().then(
    (list) => {
      if (list.length) templates.value = list
    },
    (e) => console.error('Loading templates failed; only Classic is offered.', e),
  )
}

/** The resume templates, classic first. */
export function useTemplates(): Readonly<Ref<TemplateInfo[]>> {
  load()
  return templates
}

/** The template a style names; classic when it names none, or one the server doesn't list. */
export function templateOf(style: Style | undefined): TemplateInfo {
  const id = style?.template ?? ''
  return templates.value.find((t) => t.id === id) ?? templates.value.find((t) => t.id === '') ?? CLASSIC
}

export const hasFeature = (template: TemplateInfo, feature: TemplateFeature) => template.features.includes(feature)

const KEY: InjectionKey<ComputedRef<TemplateInfo>> = Symbol('template')

/** Provides the open resume's template to the editors below. */
export function provideTemplate(style: () => Style | undefined) {
  load()
  provide(KEY, computed(() => templateOf(style())))
}

/** The open resume's template; classic outside provideTemplate. */
export function useTemplate(): ComputedRef<TemplateInfo> {
  return inject(KEY, () => computed(() => CLASSIC), true)
}
