<script setup lang="ts">
import { computed, shallowRef } from 'vue'
import type { Preview } from '../types'

const { preview, pageUrls, pending, error } = defineProps<{
  preview: Preview | null
  pageUrls: string[]
  pending: boolean
  error: string | null
}>()

type Tab = 'page' | 'text' | 'checks'
const tab = shallowRef<Tab>('page')
const textMode = shallowRef<'raw' | 'layout'>('raw')

const issues = computed(() => preview?.checks.filter((c) => c.level === 'warn' || c.level === 'error') ?? [])

// Checks first by severity, then the rest.
const SEVERITY = { error: 0, warn: 1, info: 2, ok: 3 } as const
const sortedChecks = computed(() =>
  [...(preview?.checks ?? [])].sort((a, b) => SEVERITY[a.level] - SEVERITY[b.level]),
)

const textPages = computed(() => {
  const text = textMode.value === 'raw' ? preview?.text : preview?.layoutText
  return (text ?? '').split('\f').filter((p) => p.trim() !== '')
})

const ICONS = { ok: '✓', info: 'i', warn: '!', error: '✕' } as const
</script>

<template>
  <div class="pane">
    <div class="bar">
      <div class="tabs" role="tablist">
        <button role="tab" :aria-selected="tab === 'page'" class="tab" @click="tab = 'page'">Page</button>
        <button role="tab" :aria-selected="tab === 'text'" class="tab" @click="tab = 'text'">What machines read</button>
        <button role="tab" :aria-selected="tab === 'checks'" class="tab" @click="tab = 'checks'">
          Checks
          <span v-if="preview" class="badge" :class="issues.length ? 'bad' : 'good'">
            {{ issues.length ? issues.length : '✓' }}
          </span>
        </button>
      </div>
      <span class="status" role="status">
        <template v-if="pending">Rendering…</template>
        <template v-else-if="preview">{{ preview.pages.length }} {{ preview.pages.length === 1 ? 'page' : 'pages' }}</template>
      </span>
    </div>

    <div v-if="error" class="render-error" role="alert">
      <strong>The resume didn’t render.</strong> The preview shows the last good version.
      <pre>{{ error }}</pre>
    </div>
    <div v-if="preview?.warnings.length" class="render-warnings">
      <strong>Typst warnings</strong>
      <pre>{{ preview.warnings.join('\n') }}</pre>
    </div>

    <div class="scroll" :class="{ dim: pending }">
      <div v-if="!preview && !error" class="empty">Rendering the first preview…</div>

      <div v-else-if="tab === 'page'" class="pages">
        <img
          v-for="(url, i) in pageUrls"
          :key="url"
          :src="url"
          class="page"
          :alt="`Resume page ${i + 1}`"
        />
      </div>

      <div v-else-if="tab === 'text'" class="text-view">
        <div class="text-head">
          <p class="explain">
            The PDF’s text layer, extracted with <code>pdftotext</code>. This is what applicant tracking systems and
            other parsers start from.
          </p>
          <div class="seg" role="radiogroup" aria-label="Extraction mode">
            <button role="radio" :aria-checked="textMode === 'raw'" @click="textMode = 'raw'">Reading order</button>
            <button role="radio" :aria-checked="textMode === 'layout'" @click="textMode = 'layout'">Layout</button>
          </div>
        </div>
        <template v-for="(page, i) in textPages" :key="i">
          <div class="page-break">Page {{ i + 1 }}</div>
          <pre class="text" :class="textMode">{{ page }}</pre>
        </template>
      </div>

      <ul v-else class="checks">
        <li v-for="check in sortedChecks" :key="check.title" class="check" :class="check.level">
          <span class="check-icon" aria-hidden="true">{{ ICONS[check.level] }}</span>
          <div>
            <div class="check-title">{{ check.title }}</div>
            <pre v-if="check.detail" class="check-detail">{{ check.detail }}</pre>
          </div>
        </li>
      </ul>
    </div>
  </div>
</template>

<style scoped>
.pane {
  display: flex;
  flex-direction: column;
  height: 100%;
  min-height: 0;
  background: var(--pasteboard);
}

.bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 16px;
  border-bottom: 1px solid #c9cfd7;
  background: #e9ecef;
}

.tabs {
  display: flex;
  gap: 2px;
}

.tab {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 5px 12px;
  border: 0;
  border-radius: var(--radius);
  background: transparent;
  color: var(--muted);
  cursor: pointer;
}

.tab:hover {
  color: var(--ink);
}

.tab[aria-selected='true'] {
  background: var(--panel);
  color: var(--ink);
  box-shadow: 0 1px 2px rgb(0 0 0 / 0.08);
}

.badge {
  min-width: 18px;
  padding: 0 5px;
  border-radius: 9px;
  font-size: 11px;
  font-weight: 700;
  line-height: 18px;
  text-align: center;
  color: #fff;
}

.badge.good {
  background: var(--ok);
}

.badge.bad {
  background: var(--warn);
}

.status {
  font-size: 12px;
  color: var(--muted);
}

.render-error,
.render-warnings {
  margin: 12px 16px 0;
  padding: 10px 12px;
  border-radius: var(--radius);
  font-size: 13px;
}

.render-error {
  border: 1px solid color-mix(in srgb, var(--error) 40%, transparent);
  background: #fdf1f0;
  color: var(--error);
}

.render-warnings {
  border: 1px solid color-mix(in srgb, var(--warn) 35%, transparent);
  background: #fdf6ea;
  color: var(--warn);
}

.render-error pre,
.render-warnings pre {
  margin: 6px 0 0;
  white-space: pre-wrap;
  font-size: 12px;
}

.scroll {
  flex: 1;
  min-height: 0;
  overflow: auto;
  padding: 24px;
  transition: opacity 0.2s;
}

.scroll.dim {
  opacity: 0.85;
}

.empty {
  color: var(--muted);
  text-align: center;
  padding-top: 80px;
}

.pages {
  display: grid;
  justify-items: center;
  gap: 24px;
}

.page {
  display: block;
  width: min(100%, 816px);
  aspect-ratio: 8.5 / 11;
  background: #fff;
  box-shadow:
    0 1px 2px rgb(0 0 0 / 0.12),
    0 8px 24px rgb(0 0 0 / 0.1);
}

.text-view {
  max-width: 816px;
  margin: 0 auto;
}

.text-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 12px;
}

.explain {
  margin: 0;
  color: var(--muted);
  font-size: 13px;
}

.seg {
  display: flex;
  flex: none;
  padding: 2px;
  border-radius: var(--radius);
  background: #cfd5dc;
}

.seg button {
  padding: 3px 10px;
  border: 0;
  border-radius: 4px;
  background: transparent;
  font-size: 12px;
  cursor: pointer;
}

.seg button[aria-checked='true'] {
  background: var(--panel);
}

.page-break {
  margin: 16px 0 6px;
  color: var(--muted);
  font-size: 12px;
}

.text {
  margin: 0;
  padding: 16px 20px;
  overflow-x: auto;
  border-radius: var(--radius);
  background: var(--panel);
  font: 12px/1.55 ui-monospace, 'SF Mono', Menlo, monospace;
  white-space: pre;
}

.text.raw {
  white-space: pre-wrap;
}

.checks {
  max-width: 720px;
  margin: 0 auto;
  padding: 0;
  list-style: none;
  display: grid;
  gap: 8px;
}

.check {
  display: flex;
  gap: 12px;
  padding: 12px 14px;
  border-radius: var(--radius);
  background: var(--panel);
}

.check-icon {
  flex: none;
  display: grid;
  place-items: center;
  width: 22px;
  height: 22px;
  border-radius: 50%;
  font-size: 12px;
  font-weight: 700;
  color: #fff;
}

.check.ok .check-icon {
  background: var(--ok);
}

.check.info .check-icon {
  background: var(--faint);
}

.check.warn .check-icon {
  background: var(--warn);
}

.check.error .check-icon {
  background: var(--error);
}

.check-title {
  font-weight: 600;
  padding-top: 1px;
}

.check-detail {
  margin: 6px 0 0;
  color: var(--muted);
  font: 12px/1.5 ui-monospace, 'SF Mono', Menlo, monospace;
  white-space: pre-wrap;
}
</style>
