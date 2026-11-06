<script setup lang="ts">
import { computed, shallowRef } from 'vue'
import { api, download } from './api'
import BasicsEditor from './components/BasicsEditor.vue'
import PreviewPane from './components/PreviewPane.vue'
import SectionsEditor from './components/SectionsEditor.vue'
import StyleEditor from './components/StyleEditor.vue'
import { usePreview } from './composables/usePreview'
import { useResumeDoc } from './composables/useResumeDoc'

const { doc, status, error: saveError, save } = useResumeDoc()
const { preview, pageUrls, pending, error: renderError } = usePreview(doc)

const panel = shallowRef<'content' | 'style'>('content')
const exportError = shallowRef<string | null>(null)
const exporting = shallowRef(false)

const STATUS_LABELS = {
  loading: 'Loading…',
  saved: 'Saved',
  unsaved: 'Unsaved changes',
  saving: 'Saving…',
  error: 'Not saved',
} as const

const accentStyle = computed(() => (doc.value ? { '--accent': doc.value.style.accent } : {}))
const fileBase = computed(() => `${doc.value?.basics.name || 'Resume'} - Resume`)

async function downloadPdf() {
  if (!doc.value) return
  exporting.value = true
  exportError.value = null
  try {
    download(await api.pdf(JSON.stringify(doc.value)), `${fileBase.value}.pdf`)
  } catch (e) {
    exportError.value = `PDF export failed: ${e instanceof Error ? e.message : e}`
  } finally {
    exporting.value = false
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
</script>

<template>
  <div class="app" :style="accentStyle">
    <header class="toolbar">
      <h1 class="app-name">Resume</h1>
      <button
        class="save-status"
        :class="status"
        type="button"
        :title="saveError ?? 'Changes save automatically. Click to save now.'"
        @click="save"
      >
        {{ STATUS_LABELS[status] }}
      </button>
      <span v-if="saveError || exportError" class="toolbar-error" role="alert">{{ exportError ?? saveError }}</span>
      <div class="actions">
        <button class="btn btn-quiet" type="button" :disabled="!preview" @click="downloadText">Download text</button>
        <button class="btn btn-quiet" type="button" :disabled="!doc" @click="downloadJson">Download JSON</button>
        <button class="btn btn-primary" type="button" :disabled="!doc || exporting" @click="downloadPdf">
          {{ exporting ? 'Exporting…' : 'Download PDF' }}
        </button>
      </div>
    </header>

    <main class="workspace">
      <aside class="editor">
        <div class="editor-tabs" role="tablist">
          <button role="tab" :aria-selected="panel === 'content'" @click="panel = 'content'">Content</button>
          <button role="tab" :aria-selected="panel === 'style'" @click="panel = 'style'">Style</button>
        </div>

        <div v-if="!doc" class="editor-empty">{{ saveError ?? 'Loading resume…' }}</div>

        <div v-else-if="panel === 'content'" class="editor-scroll">
          <p class="hint">
            Inline formatting: <code>**bold**</code> <code>*italic*</code> <code>[text](https://link)</code>
          </p>
          <section class="group">
            <h2 class="group-title">Header</h2>
            <BasicsEditor v-model="doc.basics" />
          </section>
          <section class="group">
            <h2 class="group-title">Sections</h2>
            <SectionsEditor v-model="doc.sections" />
          </section>
        </div>

        <div v-else class="editor-scroll">
          <section class="group">
            <h2 class="group-title">Page style</h2>
            <StyleEditor v-model="doc.style" />
          </section>
        </div>
      </aside>

      <PreviewPane class="preview" :preview="preview" :page-urls="pageUrls" :pending="pending" :error="renderError" />
    </main>
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

.app-name {
  margin: 0;
  font-size: 15px;
  font-weight: 700;
  color: var(--accent);
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

.hint code {
  padding: 1px 4px;
  border-radius: 3px;
  background: var(--sunken);
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
