<script setup lang="ts">
import { computed } from 'vue'
import { newSupervisor } from '../factory'
import { jobLabel } from '../format'
import { richToText } from '../rich'
import type { Section, Supervisor, SupervisorsDoc } from '../types'

const { sections } = defineProps<{ sections: Section[] }>()
const doc = defineModel<SupervisorsDoc>({ required: true })

interface Row {
  entryId: string
  title: string
  detail: string
  date: string
  job: string
  supervisor?: Supervisor
}

const FIELDS: { key: Exclude<keyof Supervisor, 'entryId' | 'job' | 'canContact'>; label: string; type?: string; rows?: number }[] = [
  { key: 'name', label: 'Name' },
  { key: 'position', label: 'Position' },
  { key: 'email', label: 'Email', type: 'email' },
  { key: 'phone', label: 'Phone', type: 'tel' },
  { key: 'address', label: 'Street address', rows: 2 },
  { key: 'city', label: 'City' },
  { key: 'region', label: 'State / region' },
  { key: 'postalCode', label: 'Postal code' },
  { key: 'country', label: 'Country' },
]

// One group per entries section, then the supervisors whose entry was deleted.
const groups = computed(() => {
  const byEntry = new Map(doc.value.supervisors.map((s) => [s.entryId, s]))
  const out: { key: string; title: string; rows: Row[] }[] = []
  for (const section of sections) {
    if (section.kind !== 'entries') continue
    const rows: Row[] = []
    for (const item of section.items) {
      if (item.type !== 'entry') continue
      rows.push({
        entryId: item.id,
        title: richToText(item.title) || 'Untitled entry',
        detail: richToText(item.subtitle),
        date: item.date,
        job: jobLabel(item),
        supervisor: byEntry.get(item.id),
      })
      byEntry.delete(item.id)
    }
    out.push({ key: section.id, title: section.title || 'Untitled section', rows })
  }
  if (byEntry.size) {
    const rows = [...byEntry.values()].map((s) => ({ entryId: s.entryId, title: s.job || 'Deleted entry', detail: '', date: '', job: s.job, supervisor: s }))
    out.push({ key: 'removed', title: 'No longer in the resume', rows })
  }
  return out
})

function remove(supervisor: Supervisor) {
  doc.value.supervisors.splice(doc.value.supervisors.indexOf(supervisor), 1)
}
</script>

<template>
  <p v-if="!groups.length" class="empty">Add an entries section (such as work experience) to the resume first.</p>
  <section v-for="group in groups" :key="group.key" class="group">
    <h3 class="group-title">{{ group.title }}</h3>
    <ul class="jobs">
      <li v-for="row in group.rows" :key="row.entryId" class="job">
        <div class="job-head">
          <div class="info">
            <span class="title">{{ row.title }}</span>
            <span v-if="row.detail" class="detail">{{ row.detail }}</span>
          </div>
          <span class="date">{{ row.date }}</span>
          <button v-if="row.supervisor" class="icon-btn danger" type="button" title="Remove supervisor" @click="remove(row.supervisor)">
            ×
          </button>
          <button v-else class="btn btn-add" type="button" @click="doc.supervisors.push(newSupervisor(row.entryId, row.job))">
            + Supervisor
          </button>
        </div>
        <div v-if="row.supervisor" class="fields">
          <label v-for="f in FIELDS" :key="f.key" class="field" :class="{ wide: f.rows }">
            <span class="field-label">{{ f.label }}</span>
            <textarea v-if="f.rows" v-model="row.supervisor[f.key]" class="input" :rows="f.rows" />
            <input v-else v-model="row.supervisor[f.key]" class="input" :type="f.type ?? 'text'" />
          </label>
          <label class="check">
            <input v-model="row.supervisor.canContact" type="checkbox" />
            Can contact
          </label>
        </div>
      </li>
    </ul>
  </section>
</template>

<style scoped>
.empty {
  margin: 0;
  color: var(--muted);
}

.group + .group {
  margin-top: 18px;
}

.group-title {
  margin: 0 0 6px;
  font-size: 12px;
  font-weight: 600;
  color: var(--muted);
}

.jobs {
  display: grid;
  margin: 0;
  padding: 0;
  list-style: none;
  border: 1px solid var(--rule);
  border-radius: var(--radius);
}

.job {
  padding: 6px 6px 6px 12px;
}

.job + .job {
  border-top: 1px solid var(--rule);
}

.job-head {
  display: flex;
  align-items: center;
  gap: 8px;
}

.info {
  flex: 1;
  min-width: 0;
  display: grid;
}

.title {
  font-weight: 600;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.detail {
  color: var(--muted);
  font-size: 12px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.date {
  color: var(--muted);
  font-size: 12px;
  white-space: nowrap;
}

.btn-add {
  font-size: 13px;
  padding: 3px 9px;
}

.fields {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 8px 10px;
  padding: 6px 6px 8px 0;
}

.wide,
.check {
  grid-column: 1 / -1;
}

.check {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
}
</style>
