<script setup lang="ts">
import { withBase } from 'vitepress'

const rows = [
  {
    shot: 'click-to-edit',
    eyebrow: 'Edit on the page',
    title: 'The preview is the editor',
    text: 'Click a bullet, a date or a heading on the rendered page and a small editor opens beside it. Enter adds the next bullet, Tab moves on, and the sidebar scrolls to the same field. Bold, italic and links are a keystroke away.',
    link: '/guide/editing',
  },
  {
    shot: 'machines',
    eyebrow: 'What machines read',
    title: 'See your resume the way a parser does',
    text: 'Applicant tracking systems read the PDF’s text layer, not the page. This tab shows that text, extracted with pdftotext, in reading order or as laid out, so you know nothing got scrambled.',
    link: '/guide/checks#what-machines-read',
  },
  {
    shot: 'checks',
    eyebrow: 'Checks',
    title: 'Readable by people, parsers and screen readers',
    text: 'Every render is checked: typed symbols, titles split from their dates, content missing from the text layer, PDF/UA-1 tags, accent contrast, links that don’t open, and dates out of order.',
    link: '/guide/checks',
  },
  {
    shot: 'julia',
    eyebrow: 'Templates',
    title: 'One resume, more than one look',
    text: 'Switch between Classic and Julia without retyping anything. Julia adds entry icons, a color-coded legend, compact sections and brackets. Pick your accent, font, size and margins.',
    link: '/guide/templates',
  },
  {
    shot: 'cover-letter',
    eyebrow: 'Cover letters',
    title: 'Letters that match the resume',
    text: 'Write a letter per job beside the resume it goes with. It takes the resume’s name, colors and header, so restyling one restyles both, and it’s checked and exported the same way.',
    link: '/guide/cover-letters',
  },
]
</script>

<template>
  <div class="showcase">
    <section v-for="(r, i) in rows" :key="r.shot" class="row" :class="{ flip: i % 2 === 1 }">
      <div class="copy">
        <p class="eyebrow">{{ r.eyebrow }}</p>
        <h2>{{ r.title }}</h2>
        <p class="text">{{ r.text }}</p>
        <a class="more" :href="withBase(r.link)">Learn more →</a>
      </div>
      <img class="shot" :src="withBase(`/screenshots/${r.shot}.webp`)" :alt="r.title" loading="lazy" />
    </section>
  </div>
</template>

<style scoped>
.showcase {
  display: flex;
  flex-direction: column;
  gap: 72px;
  margin-top: 88px;
}

.row {
  display: grid;
  grid-template-columns: 1fr;
  gap: 24px;
  align-items: center;
}

@media (min-width: 960px) {
  .row {
    grid-template-columns: 5fr 7fr;
    gap: 48px;
  }

  .row.flip .copy {
    order: 2;
  }
}

.showcase .eyebrow {
  margin: 0;
  font-size: 13px;
  font-weight: 600;
  letter-spacing: 0.04em;
  text-transform: uppercase;
  color: var(--vp-c-brand-1);
}

.showcase h2 {
  margin: 6px 0 12px;
  padding-top: 0;
  border-top: none;
  font-size: 26px;
  font-weight: 700;
  line-height: 1.25;
  letter-spacing: -0.01em;
}

.showcase .text {
  margin: 0 0 14px;
  font-size: 16px;
  line-height: 1.6;
  color: var(--vp-c-text-2);
}

.showcase .more {
  font-weight: 600;
  color: var(--vp-c-brand-1);
  text-decoration: none;
}

.showcase .more:hover {
  text-decoration: underline;
}

.shot {
  margin: 0;
}
</style>
