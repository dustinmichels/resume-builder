<script setup lang="ts">
import {
  Accessibility,
  FileDown,
  FolderOpen,
  History,
  ListChecks,
  Mail,
  MousePointerClick,
  Palette,
  ScanText,
} from '@lucide/vue'
import { withBase } from 'vitepress'

// Lucide, the icon set the editor itself uses.
const features = [
  {
    icon: MousePointerClick,
    title: 'Edit on the page',
    text: 'Click any line of the live PDF preview and type. The page re-renders as you go, and the sidebar follows.',
    link: '/guide/editing',
  },
  {
    icon: ScanText,
    title: 'See what machines read',
    text: 'The PDF’s text layer, in reading order or layout, exactly as an applicant tracking system extracts it.',
    link: '/guide/checks#what-machines-read',
  },
  {
    icon: ListChecks,
    title: 'Readability checks',
    text: 'Parsers, screen readers, links and dates, checked on every render, with the fix spelled out.',
    link: '/guide/checks',
  },
  {
    icon: Accessibility,
    title: 'Accessible PDFs',
    text: 'Tagged headings and lists, validated as PDF/UA-1. Decorations are drawn, never typed, so they stay out of the text.',
    link: '/guide/checks#readable-by-people-and-machines',
  },
  {
    icon: Palette,
    title: 'Templates & style',
    text: 'Classic and Julia templates, your accent color, fonts, margins, and icons beside lines and entries.',
    link: '/guide/templates',
  },
  {
    icon: Mail,
    title: 'Matching cover letters',
    text: 'Any number of letters per resume, drawn by the template’s letter design so the pair always match.',
    link: '/guide/cover-letters',
  },
  {
    icon: FileDown,
    title: 'PDF, Word, Markdown',
    text: 'Editable, parser-friendly Word files built from native Word features, plus Markdown, text and JSON.',
    link: '/guide/downloads',
  },
  {
    icon: History,
    title: 'Versions & undo',
    text: 'Snapshot what you sent to each employer, restore any version, and undo every edit.',
    link: '/guide/files#versions',
  },
  {
    icon: FolderOpen,
    title: 'Plain files you own',
    text: 'No account, no cloud. Everything is JSON in a folder you choose: back it up, sync it, version it.',
    link: '/guide/files',
  },
]
</script>

<template>
  <div class="feature-grid">
    <a v-for="f in features" :key="f.title" class="feature" :href="withBase(f.link)">
      <span class="icon"><component :is="f.icon" :size="22" :stroke-width="1.75" aria-hidden="true" /></span>
      <h3>{{ f.title }}</h3>
      <p>{{ f.text }}</p>
    </a>
  </div>
</template>

<style scoped>
.feature-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(260px, 1fr));
  gap: 16px;
}

.feature {
  display: block;
  padding: 22px 24px;
  border: 1px solid var(--vp-c-bg-soft);
  border-radius: 12px;
  background: var(--vp-c-bg-soft);
  color: inherit;
  text-decoration: none;
  transition: border-color 0.25s;
}

.feature:hover {
  border-color: var(--vp-c-brand-1);
}

.icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 44px;
  height: 44px;
  border-radius: 10px;
  background: var(--vp-c-brand-soft);
  color: var(--vp-c-brand-1);
}

h3 {
  margin: 16px 0 6px;
  font-size: 16px;
  font-weight: 600;
  line-height: 24px;
}

p {
  margin: 0;
  font-size: 14px;
  line-height: 22px;
  color: var(--vp-c-text-2);
}
</style>
