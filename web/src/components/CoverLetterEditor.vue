<script setup lang="ts">
import { computed, nextTick } from 'vue'
import { VueDraggable } from 'vue-draggable-plus'
import { newLetterBlock } from '../factory'
import { fontOptions as optionsFor } from '../fonts'
import type { CoverLetter, LetterBlockKind } from '../types'
import RichText from './RichText.vue'
import StringList from './StringList.vue'

// Edits the open letter in place. Inputs carry data-path (see fieldPath.ts) so
// a value clicked on the page preview can be revealed here.
const letter = defineModel<CoverLetter>({ required: true })

const ADD_OPTIONS: { kind: LetterBlockKind; label: string }[] = [
  { kind: 'paragraph', label: 'Paragraph' },
  { kind: 'heading', label: 'Subheading' },
  { kind: 'bullets', label: 'Bullet list' },
]
const KIND_LABELS = { paragraph: 'Paragraph', heading: 'Subheading', bullets: 'Bullets' } as const

const fontOptions = computed(() => optionsFor(letter.value.style.font))

// Paragraph editors by block id, for moving the caret between paragraphs.
const editors = new Map<string, InstanceType<typeof RichText>>()
function setEditor(id: string, editor: unknown) {
  if (editor) editors.set(id, editor as InstanceType<typeof RichText>)
  else editors.delete(id)
}

async function add(kind: LetterBlockKind, at = letter.value.blocks.length) {
  const block = newLetterBlock(kind)
  letter.value.blocks.splice(at, 0, block)
  await nextTick()
  editors.get(block.id)?.focus()
}

/** Backspace in an empty paragraph removes it and moves to the paragraph before. */
async function removeEmpty(index: number) {
  const blocks = letter.value.blocks
  blocks.splice(index, 1)
  await nextTick()
  const prev = blocks[index - 1]
  if (prev) editors.get(prev.id)?.focus()
}
</script>

<template>
  <div class="letter-editor">
    <section class="group">
      <h2 class="group-title">Header</h2>
      <p class="hint">The name, accent color and heading font come from the resume.</p>
      <div class="field">
        <span class="field-label">Lines under the name</span>
        <StringList v-model="letter.header" path="letter.header" add-label="Line" rich placeholder="e.g. Somerville, MA" />
      </div>
      <label class="field">
        <span class="field-label">Heading</span>
        <input v-model="letter.title" data-path="letter.title" class="input" placeholder="Cover letter" />
      </label>
    </section>

    <section class="group">
      <h2 class="group-title">Letter</h2>
      <label class="field">
        <span class="field-label">Greeting</span>
        <input v-model="letter.greeting" data-path="letter.greeting" class="input" placeholder="Dear Hiring Manager," />
      </label>

      <VueDraggable v-model="letter.blocks" handle=".h-block" :animation="150" class="blocks">
        <div v-for="(block, i) in letter.blocks" :key="block.id" class="block" :class="block.kind">
          <div class="block-head">
            <span class="handle h-block" title="Drag to reorder" aria-hidden="true">⠿</span>
            <span class="kind">{{ KIND_LABELS[block.kind] }}</span>
            <button
              class="icon-btn danger"
              type="button"
              :title="`Remove ${KIND_LABELS[block.kind].toLowerCase()}`"
              @click="letter.blocks.splice(i, 1)"
            >
              ×
            </button>
          </div>
          <RichText
            v-if="block.kind === 'paragraph'"
            :ref="(c) => setEditor(block.id, c)"
            v-model="block.text"
            :data-path="`letter.blocks.${i}.text`"
            placeholder="Write a paragraph. Enter starts the next one."
            @enter="add('paragraph', i + 1)"
            @backspace-empty="removeEmpty(i)"
          />
          <input
            v-else-if="block.kind === 'heading'"
            v-model="block.text"
            :data-path="`letter.blocks.${i}.text`"
            class="input"
            placeholder="e.g. My experience"
          />
          <StringList v-else v-model="block.items" :path="`letter.blocks.${i}.items`" add-label="Bullet" rich />
        </div>
      </VueDraggable>

      <div class="add">
        <span class="field-label">Add</span>
        <button v-for="opt in ADD_OPTIONS" :key="opt.kind" class="btn btn-add" type="button" @click="add(opt.kind)">
          + {{ opt.label }}
        </button>
      </div>

      <label class="field">
        <span class="field-label">Closing (your name goes below it; leave empty for none)</span>
        <input v-model="letter.closing" data-path="letter.closing" class="input" placeholder="Sincerely," />
      </label>
    </section>

    <section class="group">
      <h2 class="group-title">Body text</h2>
      <div class="style-grid">
        <label class="field">
          <span class="field-label">Font</span>
          <select v-model="letter.style.font" class="input">
            <option v-for="f in fontOptions" :key="f" :value="f">{{ f }}</option>
          </select>
        </label>
        <label class="field">
          <span class="field-label">Text size (pt)</span>
          <input v-model.number="letter.style.fontSize" class="input" type="number" min="8" max="16" step="0.5" />
        </label>
      </div>
    </section>
  </div>
</template>

<style scoped>
.group + .group {
  margin-top: 26px;
}

.group-title {
  margin: 0 0 10px;
  font-size: 13px;
  font-weight: 650;
  color: var(--muted);
}

.group > .field + .field,
.group > .field + .blocks,
.group > .blocks + .add,
.group > .add + .field {
  margin-top: 12px;
}

.hint {
  margin: -4px 0 10px;
  color: var(--muted);
  font-size: 12px;
}

.blocks {
  display: grid;
  gap: 8px;
}

.block {
  display: grid;
  gap: 4px;
  padding: 6px 8px 8px;
  border: 1px solid var(--rule);
  border-radius: 8px;
  background: var(--panel);
}

.block.heading {
  background: var(--sunken);
}

.block-head {
  display: flex;
  align-items: center;
  gap: 6px;
}

.kind {
  flex: 1;
  color: var(--muted);
  font-size: 12px;
  font-weight: 600;
}

.add {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 6px;
}

.add .btn {
  padding: 3px 9px;
  font-size: 13px;
}
</style>
