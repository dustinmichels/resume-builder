<script setup lang="ts">
import { computed, nextTick, onScopeDispose, shallowRef, useTemplateRef } from 'vue'
import { api, download, type ExportFormat } from './api'
import BasicsEditor from './components/BasicsEditor.vue'
import ContactEditor from './components/ContactEditor.vue'
import DownloadMenu, { type DownloadItem } from './components/DownloadMenu.vue'
import PreviewPane from './components/PreviewPane.vue'
import PromptDialog, { type PromptRequest } from './components/PromptDialog.vue'
import ResumeMenu from './components/ResumeMenu.vue'
import SectionsEditor from './components/SectionsEditor.vue'
import SetupScreen from './components/SetupScreen.vue'
import StyleEditor from './components/StyleEditor.vue'
import SupervisorsEditor from './components/SupervisorsEditor.vue'
import VersionsPanel from './components/VersionsPanel.vue'
import { useLibrary } from './composables/useLibrary'
import { useLinkChecks } from './composables/useLinkChecks'
import { usePreview } from './composables/usePreview'
import { useUndoHistory } from './composables/useUndoHistory'
import { provideReveal } from './reveal'

const library = useLibrary()
const { settings, resumes, currentId, error: libraryError } = library
const { doc, contact, supervisors, status, error: saveError, save } = library.resume
const { preview, pageUrls, pending, error: renderError } = usePreview(doc)
const { checks: linkChecks, error: linkError } = useLinkChecks(doc)
const { canUndo, canRedo, undo, redo } = useUndoHistory(doc)

const panel = shallowRef<'content' | 'style' | 'contact' | 'supervisors' | 'versions'>('content')
const exportError = shallowRef<string | null>(null)
const exporting = shallowRef<ExportFormat | null>(null)
const actionError = shallowRef<string | null>(null)
const prompt = shallowRef<PromptRequest | null>(null)
const sidebar = useTemplateRef<HTMLElement>('sidebar')
const revealing = provideReveal()

const STATUS_LABELS = {
  loading: 'Loading…',
  saved: 'Saved',
  unsaved: 'Unsaved changes',
  saving: 'Saving…',
  error: 'Not saved',
} as const

const accentStyle = computed(() => (doc.value ? { '--accent': doc.value.style.accent } : {}))
const fileBase = computed(() => `${doc.value?.basics.name || 'Resume'} - Resume`)

const IS_MAC = /Mac|iPhone|iPad/.test(navigator.platform)
const UNDO_KEYS = IS_MAC ? '⌘Z' : 'Ctrl+Z'
const REDO_KEYS = IS_MAC ? '⇧⌘Z' : 'Ctrl+Y'
const REVEAL_LABEL = IS_MAC ? 'Open in Finder' : 'Open folder'

/**
 * Undo/redo keys act on the resume, also inside its text fields. Fields
 * outside the resume editors ([data-resume-undo]) and dialogs keep the
 * browser's own undo.
 */
function onUndoKey(e: KeyboardEvent) {
  if (e.isComposing || e.altKey || !(IS_MAC ? e.metaKey : e.ctrlKey)) return
  const key = e.key.toLowerCase()
  const isRedo = (key === 'z' && e.shiftKey) || (key === 'y' && !IS_MAC && !e.shiftKey)
  if (key !== 'z' && !isRedo) return
  const target = e.target instanceof Element ? e.target : null
  if (target?.closest('dialog')) return
  if (target?.closest('input, textarea, select, [contenteditable]') && !target.closest('[data-resume-undo]')) return
  e.preventDefault()
  if (isRedo) redo()
  else undo()
}
window.addEventListener('keydown', onUndoKey)
onScopeDispose(() => window.removeEventListener('keydown', onUndoKey))

const EXPORT_LABELS = { pdf: 'PDF', docx: 'Word', md: 'Markdown' } as const

async function downloadExport(format: ExportFormat) {
  if (!doc.value) return
  exporting.value = format
  exportError.value = null
  try {
    download(await api.exportAs(format, JSON.stringify(doc.value)), `${fileBase.value}.${format}`)
  } catch (e) {
    exportError.value = `${EXPORT_LABELS[format]} export failed: ${e instanceof Error ? e.message : e}`
  } finally {
    exporting.value = null
  }
}

function downloadText() {
  if (preview.value) download(new Blob([preview.value.text], { type: 'text/plain' }), `${fileBase.value}.txt`)
}

function downloadJson() {
  if (!doc.value) return
  const { basics, sections } = doc.value
  download(new Blob([JSON.stringify({ basics, sections }, null, 2)], { type: 'application/json' }), `${fileBase.value}.json`)
}

const downloadItems = computed<DownloadItem[]>(() => [
  { label: 'Plain text (.txt)', disabled: !preview.value, action: downloadText },
  { label: 'JSON (.json)', disabled: !doc.value, action: downloadJson },
  { label: 'Markdown (.md)', disabled: !doc.value || !!exporting.value, action: () => downloadExport('md') },
  { label: 'Word (.docx)', disabled: !doc.value || !!exporting.value, action: () => downloadExport('docx') },
])

/**
 * Shows a value opened on the page in the Content panel: expands the cards
 * around it, scrolls the panel to it unless it's already in view, and
 * flashes it. Only the panel scrolls, never the window, so the stacked
 * narrow layout stays on the preview.
 */
async function revealInSidebar(path: string) {
  panel.value = 'content'
  revealing.value = path
  await nextTick()
  revealing.value = null
  const field = sidebar.value?.querySelector<HTMLElement>(`[data-path="${path}"]`)
  const scroller = field?.closest<HTMLElement>('.editor-scroll')
  if (!field || !scroller) return
  const view = scroller.getBoundingClientRect()
  const box = field.getBoundingClientRect()
  if (box.top < view.top || box.bottom > view.bottom) {
    const margin = Math.max(0, (view.height - box.height) / 2)
    scroller.scrollTo({ top: scroller.scrollTop + box.top - view.top - margin, behavior: 'smooth' })
  }
  field.classList.remove('revealed')
  void field.offsetWidth // restart the animation
  field.classList.add('revealed')
  field.addEventListener('animationend', () => field.classList.remove('revealed'), { once: true })
}

async function switchResume(id: string) {
  actionError.value = null
  try {
    await library.switchTo(id)
  } catch (e) {
    actionError.value = e instanceof Error ? e.message : String(e)
  }
}

async function revealResume() {
  if (!currentId.value) return
  actionError.value = null
  try {
    await api.revealResume(currentId.value)
  } catch (e) {
    actionError.value = `${REVEAL_LABEL} failed: ${e instanceof Error ? e.message : e}`
  }
}

function promptCreate() {
  prompt.value = {
    title: 'New resume',
    value: 'New resume',
    placeholder: 'Name',
    confirmLabel: 'Create',
    run: library.create,
  }
}

function promptDuplicate() {
  prompt.value = {
    title: `Duplicate “${currentId.value}”`,
    value: `${currentId.value} copy`,
    placeholder: 'Name',
    confirmLabel: 'Duplicate',
    run: library.duplicate,
  }
}

function promptRename() {
  prompt.value = {
    title: `Rename “${currentId.value}”`,
    value: currentId.value ?? '',
    placeholder: 'Name',
    confirmLabel: 'Rename',
    run: library.rename,
  }
}

function promptDelete() {
  prompt.value = {
    title: `Delete “${currentId.value}”?`,
    message: `The resume and its versions are moved to the .trash folder inside your data folder (${settings.value?.dataDir}/.trash), where you can recover them.`,
    confirmLabel: 'Delete',
    danger: true,
    run: library.remove,
  }
}

function promptSettings() {
  prompt.value = {
    title: 'Data folder',
    message: `Resumes and their versions are saved in:\n${settings.value?.dataDir}\n\nTo use another folder, enter its path. It is created if needed; existing files are not moved.`,
    value: settings.value?.dataDir ?? '',
    placeholder: settings.value?.defaultDataDir,
    confirmLabel: 'Switch folder',
    run: library.setDataDir,
  }
}
</script>

<template>
  <p v-if="libraryError" class="fatal" role="alert">{{ libraryError }}</p>
  <p v-else-if="!settings" class="fatal">Loading…</p>
  <SetupScreen
    v-else-if="!settings.dataDir"
    :default-data-dir="settings.defaultDataDir"
    :choose="library.setDataDir"
  />
  <div v-else class="app" :style="accentStyle">
    <header class="toolbar">
      <ResumeMenu
        :resumes="resumes"
        :current="currentId"
        @open="library.refresh().catch(() => {})"
        @select="switchResume"
        @create="promptCreate"
        @duplicate="promptDuplicate"
        @rename="promptRename"
        @remove="promptDelete"
      />
      <button
        class="save-status"
        :class="status"
        type="button"
        :title="saveError ?? 'Changes save automatically. Click to save now.'"
        @click="save"
      >
        {{ STATUS_LABELS[status] }}
      </button>
      <div class="history">
        <button class="icon-btn" type="button" :title="`Undo (${UNDO_KEYS})`" aria-label="Undo" :disabled="!canUndo" @click="undo">
          <svg class="icon" viewBox="0 0 24 24" aria-hidden="true">
            <path d="M9 14 4 9l5-5" />
            <path d="M4 9h10.5a5.5 5.5 0 0 1 0 11H11" />
          </svg>
        </button>
        <button class="icon-btn" type="button" :title="`Redo (${REDO_KEYS})`" aria-label="Redo" :disabled="!canRedo" @click="redo">
          <svg class="icon" viewBox="0 0 24 24" aria-hidden="true">
            <path d="m15 14 5-5-5-5" />
            <path d="M20 9H9.5a5.5 5.5 0 0 0 0 11H13" />
          </svg>
        </button>
      </div>
      <span v-if="actionError || saveError || exportError" class="toolbar-error" role="alert">
        {{ actionError ?? exportError ?? saveError }}
      </span>
      <div class="actions">
        <DownloadMenu :items="downloadItems" :busy-label="exporting && exporting !== 'pdf' ? 'Exporting…' : null" />
        <button class="btn btn-primary" type="button" :disabled="!doc || !!exporting" @click="downloadExport('pdf')">
          {{ exporting === 'pdf' ? 'Exporting…' : 'Download PDF' }}
        </button>
      </div>
      <button
        class="icon-btn"
        type="button"
        :title="`${REVEAL_LABEL}: this resume's folder`"
        :aria-label="REVEAL_LABEL"
        :disabled="!currentId"
        @click="revealResume"
      >
        <svg class="icon" viewBox="0 0 24 24" aria-hidden="true">
          <path d="M3 6.5A1.5 1.5 0 0 1 4.5 5h4l2 2.5h9A1.5 1.5 0 0 1 21 9v8.5a1.5 1.5 0 0 1-1.5 1.5h-15A1.5 1.5 0 0 1 3 17.5z" />
        </svg>
      </button>
      <button class="icon-btn" type="button" title="Data folder settings" aria-label="Settings" @click="promptSettings">
        <svg class="icon" viewBox="0 0 24 24" aria-hidden="true">
          <circle cx="12" cy="12" r="3" />
          <path
            d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 1 1-2.83 2.83l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-4 0v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 1 1-2.83-2.83l.06-.06A1.65 1.65 0 0 0 4.68 15a1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1 0-4h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 1 1 2.83-2.83l.06.06A1.65 1.65 0 0 0 9 4.68a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 4 0v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 1 1 2.83 2.83l-.06.06A1.65 1.65 0 0 0 19.4 9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 0 4h-.09a1.65 1.65 0 0 0-1.51 1z"
          />
        </svg>
      </button>
    </header>

    <main class="workspace">
      <aside ref="sidebar" class="editor">
        <div class="editor-tabs" role="tablist">
          <button role="tab" :aria-selected="panel === 'content'" @click="panel = 'content'">Content</button>
          <button role="tab" :aria-selected="panel === 'style'" @click="panel = 'style'">Style</button>
          <button role="tab" :aria-selected="panel === 'contact'" @click="panel = 'contact'">Contact</button>
          <button role="tab" :aria-selected="panel === 'supervisors'" @click="panel = 'supervisors'">Supervisors</button>
          <button role="tab" :aria-selected="panel === 'versions'" @click="panel = 'versions'">Versions</button>
        </div>

        <div v-if="!doc || !contact || !supervisors || !currentId" class="editor-empty">{{ saveError ?? 'Loading resume…' }}</div>

        <div v-else-if="panel === 'content'" class="editor-scroll" data-resume-undo>
          <p class="hint">Select text and press ⌘B, ⌘I or ⌘K to format. Click anything on the page preview to edit it there.</p>
          <section class="group">
            <h2 class="group-title">Header</h2>
            <BasicsEditor v-model="doc.basics" />
          </section>
          <section class="group">
            <h2 class="group-title">Sections</h2>
            <SectionsEditor v-model="doc.sections" />
          </section>
        </div>

        <div v-else-if="panel === 'style'" class="editor-scroll" data-resume-undo>
          <section class="group">
            <h2 class="group-title">Page style</h2>
            <StyleEditor v-model="doc.style" />
          </section>
        </div>

        <div v-else-if="panel === 'contact'" class="editor-scroll">
          <p class="hint">Your details for filling in applications. Saved as additional/contact.json in the resume's folder; not printed or exported.</p>
          <ContactEditor v-model="contact" />
        </div>

        <div v-else-if="panel === 'supervisors'" class="editor-scroll">
          <p class="hint">A supervisor for each job. Saved as additional/supervisors.json in the resume's folder; not printed or exported.</p>
          <SupervisorsEditor v-model="supervisors" :sections="doc.sections" />
        </div>

        <div v-else class="editor-scroll">
          <section class="group">
            <h2 class="group-title">Versions of “{{ currentId }}”</h2>
            <VersionsPanel
              :resume-id="currentId"
              :save-version="library.saveVersion"
              :restore-version="library.restoreVersion"
              :copy-version="library.copyVersion"
              :sync-files="library.syncFiles"
            />
          </section>
        </div>
      </aside>

      <PreviewPane
        class="preview"
        data-resume-undo
        :doc="doc"
        :preview="preview"
        :page-urls="pageUrls"
        :pending="pending"
        :error="renderError"
        :link-checks="linkChecks"
        :link-error="linkError"
        @select="revealInSidebar"
      />
    </main>

    <PromptDialog v-model="prompt" />
  </div>
</template>

<style scoped>
.app {
  display: flex;
  flex-direction: column;
  height: 100vh;
}

.toolbar {
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 8px 16px;
  border-bottom: 1px solid var(--rule);
}

.fatal {
  margin: 0;
  padding: 40px 24px;
  color: var(--muted);
  text-align: center;
}

.fatal[role='alert'] {
  color: var(--error);
}

.icon {
  width: 17px;
  height: 17px;
  fill: none;
  stroke: currentColor;
  stroke-width: 1.7;
  stroke-linecap: round;
  stroke-linejoin: round;
}

.history {
  display: flex;
  gap: 2px;
}

.history .icon-btn:disabled {
  background: transparent;
  color: var(--faint);
  opacity: 0.4;
  cursor: default;
}

.save-status {
  padding: 2px 8px;
  border: 0;
  border-radius: 10px;
  background: var(--sunken);
  color: var(--muted);
  font-size: 12px;
  cursor: pointer;
}

.save-status.saved {
  color: var(--ok);
}

.save-status.error {
  background: #fdf1f0;
  color: var(--error);
}

.toolbar-error {
  color: var(--error);
  font-size: 13px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.actions {
  display: flex;
  gap: 6px;
  margin-left: auto;
}

.actions .btn:disabled {
  opacity: 0.5;
  cursor: default;
}

.workspace {
  flex: 1;
  min-height: 0;
  display: grid;
  grid-template-columns: minmax(420px, 540px) 1fr;
}

.editor {
  display: flex;
  flex-direction: column;
  min-height: 0;
  border-right: 1px solid var(--rule);
}

.editor-tabs {
  display: flex;
  gap: 18px;
  padding: 0 20px;
  border-bottom: 1px solid var(--rule);
}

.editor-tabs button {
  padding: 10px 0 8px;
  border: 0;
  border-bottom: 2px solid transparent;
  background: none;
  color: var(--muted);
  cursor: pointer;
}

.editor-tabs button[aria-selected='true'] {
  border-bottom-color: var(--accent);
  color: var(--ink);
  font-weight: 600;
}

.editor-scroll {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  padding: 14px 20px 40px;
}

.editor-empty {
  padding: 40px 20px;
  color: var(--muted);
}

.hint {
  margin: 0 0 14px;
  color: var(--muted);
  font-size: 12px;
}

.group + .group {
  margin-top: 26px;
}

.group-title {
  margin: 0 0 10px;
  font-size: 13px;
  font-weight: 650;
  color: var(--muted);
}

.preview {
  min-width: 0;
}

@media (max-width: 900px) {
  .app {
    height: auto;
  }

  .workspace {
    grid-template-columns: 1fr;
  }

  .editor-scroll {
    overflow: visible;
  }
}
</style>
