<script setup lang="ts">
import type { Style } from '../types'

const style = defineModel<Style>({ required: true })

// Fonts installed on macOS by default. Any installed font name works;
// unknown names show up as a Typst warning in the preview.
const FONTS = ['Helvetica', 'Helvetica Neue', 'Arial', 'Avenir Next', 'Gill Sans', 'Optima', 'Georgia', 'Palatino', 'Baskerville', 'Charter']
</script>

<template>
  <div class="style-grid">
    <label class="field">
      <span class="field-label">Accent color</span>
      <span class="color">
        <input v-model="style.accent" type="color" class="swatch" aria-label="Accent color" />
        <code>{{ style.accent }}</code>
      </span>
    </label>

    <label class="field">
      <span class="field-label">Font</span>
      <input v-model="style.font" class="input" list="font-options" />
      <datalist id="font-options">
        <option v-for="f in FONTS" :key="f" :value="f" />
      </datalist>
    </label>

    <label class="field">
      <span class="field-label">Text size (pt)</span>
      <input v-model.number="style.fontSize" class="input" type="number" min="7" max="14" step="0.25" />
    </label>

    <label class="field">
      <span class="field-label">Page margin (in)</span>
      <input v-model.number="style.margin" class="input" type="number" min="0.25" max="1.5" step="0.05" />
    </label>

    <label class="field">
      <span class="field-label">Leader between title and date</span>
      <select v-model="style.leader" class="input">
        <option value="dotted">Dotted</option>
        <option value="dashed">Dashed</option>
        <option value="solid">Solid line</option>
        <option value="none">None (blank space)</option>
      </select>
    </label>

    <label class="field">
      <span class="field-label">Bullet</span>
      <select v-model="style.bullet" class="input">
        <option value="diamonds">Four diamonds (❖)</option>
        <option value="diamond">Diamond</option>
        <option value="square">Square</option>
        <option value="disc">Disc</option>
        <option value="none">None</option>
      </select>
    </label>

    <label class="field">
      <span class="field-label">Paragraph marker</span>
      <select v-model="style.paragraphMarker" class="input">
        <option value="chevron">Chevron (›)</option>
        <option value="none">None</option>
      </select>
    </label>

    <p class="note">
      Leaders, bullets and markers are drawn as shapes, so they never appear in the PDF’s text layer.
    </p>
  </div>
</template>

<style scoped>
.style-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 12px 14px;
}

.color {
  display: flex;
  align-items: center;
  gap: 8px;
}

.swatch {
  width: 40px;
  height: 30px;
  padding: 2px;
  border: 1px solid var(--rule);
  border-radius: var(--radius);
  background: var(--panel);
  cursor: pointer;
}

.note {
  grid-column: 1 / -1;
  margin: 4px 0 0;
  color: var(--muted);
  font-size: 13px;
}
</style>
