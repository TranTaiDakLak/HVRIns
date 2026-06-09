<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { useAppStore } from '../stores/app.store'

interface MailDomainRow {
  index?: number
  domain: string
  veri: number
  live: number
  die: number
  rate: number // 0–1
}
type SortKey = 'domain' | 'veri' | 'live' | 'die' | 'rate'

const props = defineProps<{ rows: MailDomainRow[] }>()

const sortKey = ref<SortKey>('live')
const sortDir = ref<'asc' | 'desc'>('desc')
const selected = ref<Set<string>>(new Set())
const dragMode = ref<'select' | 'deselect' | null>(null)
const lastClicked = ref<string | null>(null)
const appStore = useAppStore()

function pct(v: number): string { return (v * 100).toFixed(1) + '%' }

function setSort(key: SortKey) {
  if (sortKey.value === key) sortDir.value = sortDir.value === 'asc' ? 'desc' : 'asc'
  else { sortKey.value = key; sortDir.value = key === 'domain' ? 'asc' : 'desc' }
}
function arrow(key: SortKey): string {
  if (sortKey.value !== key) return '↕'
  return sortDir.value === 'asc' ? '▲' : '▼'
}
function ariaSort(key: SortKey) {
  if (sortKey.value !== key) return undefined
  return sortDir.value === 'asc' ? 'ascending' : 'descending'
}

const sorted = computed(() => {
  const arr = [...props.rows]
  const dir = sortDir.value === 'asc' ? 1 : -1
  arr.sort((a, b) => {
    let cmp: number
    if (sortKey.value === 'domain') cmp = a.domain.localeCompare(b.domain)
    else cmp = (a[sortKey.value] as number) - (b[sortKey.value] as number)
    if (cmp === 0) cmp = a.domain.localeCompare(b.domain)
    return cmp * dir
  })
  return arr
})

const totals = computed(() => {
  let veri = 0, live = 0, die = 0
  for (const r of props.rows) { veri += r.veri; live += r.live; die += r.die }
  return { veri, live, die, rate: veri > 0 ? live / veri : 0 }
})

const selectedCount = computed(() => selected.value.size)

watch(() => props.rows.map(r => r.domain).join('\n'), () => {
  const available = new Set(props.rows.map(r => r.domain))
  selected.value = new Set([...selected.value].filter(d => available.has(d)))
})

function setSelected(domain: string, on: boolean) {
  const next = new Set(selected.value)
  if (on) next.add(domain)
  else next.delete(domain)
  selected.value = next
}

function selectRange(toDomain: string, on = true) {
  const from = lastClicked.value
  const fromIdx = from ? sorted.value.findIndex(r => r.domain === from) : -1
  const toIdx = sorted.value.findIndex(r => r.domain === toDomain)
  if (fromIdx < 0 || toIdx < 0) { setSelected(toDomain, on); return }
  const [start, end] = fromIdx < toIdx ? [fromIdx, toIdx] : [toIdx, fromIdx]
  const next = new Set(selected.value)
  for (const row of sorted.value.slice(start, end + 1)) {
    if (on) next.add(row.domain)
    else next.delete(row.domain)
  }
  selected.value = next
}

function onRowMouseDown(row: MailDomainRow, e: MouseEvent) {
  if (e.button !== 0) return
  e.preventDefault()
  if (e.shiftKey) { selectRange(row.domain, true); lastClicked.value = row.domain; dragMode.value = 'select'; return }
  const shouldSelect = !selected.value.has(row.domain)
  setSelected(row.domain, shouldSelect)
  lastClicked.value = row.domain
  dragMode.value = shouldSelect ? 'select' : 'deselect'
}
function onRowEnter(row: MailDomainRow) {
  if (!dragMode.value) return
  setSelected(row.domain, dragMode.value === 'select')
}
function stopDrag() { dragMode.value = null }
function clearSelection() { selected.value = new Set() }
function selectAllVisible() { selected.value = new Set(sorted.value.map(r => r.domain)) }

async function exportSelection() {
  const list = sorted.value.filter(r => selected.value.has(r.domain)).map(r => r.domain.replace(/^@/, ''))
  if (list.length === 0) return
  try {
    await navigator.clipboard.writeText(list.join(', '))
    appStore.notify('success', `Đã xuất ${list.length} domain vào clipboard`)
  } catch {
    appStore.notify('error', 'Không ghi được vào clipboard')
  }
}
</script>

<template>
  <div class="mdt">
    <div class="mdt__head">
      <span class="mdt__title">Mail Domain</span>
      <span v-if="rows.length" class="mdt__sub">
        {{ totals.veri }} lượt veri · Live {{ totals.live }} · Die {{ totals.die }} · {{ pct(totals.rate) }}
      </span>
      <div v-if="rows.length" class="mdt__actions">
        <span class="mdt__selected">{{ selectedCount }} chọn</span>
        <button class="mdt__btn" @click="selectAllVisible">Chọn tất cả</button>
        <button class="mdt__btn" :disabled="selectedCount === 0" @click="exportSelection">Xuất domain</button>
        <button class="mdt__btn" :disabled="selectedCount === 0" @click="clearSelection">Bỏ chọn</button>
      </div>
    </div>
    <div v-if="rows.length" class="mdt__bar" :title="pct(totals.rate)">
      <div class="mdt__bar-ok" :style="{ width: pct(totals.rate) }"></div>
    </div>

    <div v-if="rows.length === 0" class="mdt__empty">Chưa có dữ liệu — bắt đầu chạy Verify để xem.</div>

    <div v-else class="mdt__wrap" @mouseleave="stopDrag" @mouseup="stopDrag">
      <table class="mdt__table">
        <thead>
          <tr>
            <th class="mdt-th mdt-th--pick"></th>
            <th class="mdt-th mdt-th--stt">STT</th>
            <th class="mdt-th mdt-th--sortable mdt-th--domain" :aria-sort="ariaSort('domain')" @click="setSort('domain')">Domain <span class="mdt-th__a">{{ arrow('domain') }}</span></th>
            <th class="mdt-th mdt-th--sortable mdt-th--num" :aria-sort="ariaSort('veri')" @click="setSort('veri')">Veri <span class="mdt-th__a">{{ arrow('veri') }}</span></th>
            <th class="mdt-th mdt-th--sortable mdt-th--num" :aria-sort="ariaSort('live')" @click="setSort('live')">Live <span class="mdt-th__a">{{ arrow('live') }}</span></th>
            <th class="mdt-th mdt-th--sortable mdt-th--num" :aria-sort="ariaSort('die')" @click="setSort('die')">Die <span class="mdt-th__a">{{ arrow('die') }}</span></th>
            <th class="mdt-th mdt-th--sortable mdt-th--rate" :aria-sort="ariaSort('rate')" @click="setSort('rate')">Live Rate <span class="mdt-th__a">{{ arrow('rate') }}</span></th>
          </tr>
        </thead>
        <tbody>
          <tr
            v-for="(r, i) in sorted"
            :key="r.domain"
            :class="{ 'mdt-row--selected': selected.has(r.domain) }"
            @mousedown="onRowMouseDown(r, $event)"
            @mouseenter="onRowEnter(r)"
          >
            <td class="mdt-td mdt-td--pick">
              <input type="checkbox" :checked="selected.has(r.domain)" tabindex="-1" readonly />
            </td>
            <td class="mdt-td mdt-td--stt">{{ i + 1 }}</td>
            <td class="mdt-td mdt-td--domain">{{ r.domain }}</td>
            <td class="mdt-td mdt-td--num">{{ r.veri }}</td>
            <td class="mdt-td mdt-td--num mdt-live">{{ r.live }}</td>
            <td class="mdt-td mdt-td--num mdt-die">{{ r.die }}</td>
            <td class="mdt-td mdt-td--rate">
              <div class="mdt-cellrate">
                <span class="mdt-cellrate__txt">{{ r.veri > 0 ? pct(r.rate) : '-' }}</span>
                <span class="mdt-cellrate__bar"><span class="mdt-cellrate__fill" :style="{ width: r.veri > 0 ? pct(r.rate) : '0%' }"></span></span>
              </div>
            </td>
          </tr>
        </tbody>
        <tfoot v-if="rows.length > 1">
          <tr>
            <td class="mdt-td mdt-td--pick"></td>
            <td class="mdt-td mdt-td--stt"></td>
            <td class="mdt-td mdt-td--domain">Tổng cộng</td>
            <td class="mdt-td mdt-td--num">{{ totals.veri }}</td>
            <td class="mdt-td mdt-td--num mdt-live">{{ totals.live }}</td>
            <td class="mdt-td mdt-td--num mdt-die">{{ totals.die }}</td>
            <td class="mdt-td mdt-td--rate">
              <div class="mdt-cellrate"><span class="mdt-cellrate__txt">{{ pct(totals.rate) }}</span></div>
            </td>
          </tr>
        </tfoot>
      </table>
    </div>
  </div>
</template>

<style scoped>
.mdt {
  flex: 1 1 0; min-width: 340px;
  border: 1px solid var(--border-default); border-radius: var(--radius-md);
  background: var(--surface-elevated); overflow: hidden;
  display: flex; flex-direction: column;
  height: calc(100vh - 190px); min-height: 440px;
  --mdt-accent: #a855f7;
}

.mdt__head {
  display: flex; align-items: center; gap: 10px; flex-wrap: wrap;
  padding: 9px 12px; background: var(--surface-sunken); border-bottom: 1px solid var(--border-default);
  border-left: 3px solid var(--mdt-accent);
}
.mdt__title { font-weight: 700; font-size: 13px; color: var(--text-primary); }
.mdt__sub { font-size: 11px; color: var(--text-muted); font-variant-numeric: tabular-nums; }
.mdt__actions { margin-left: auto; display: flex; align-items: center; gap: 6px; }
.mdt__selected { font-size: 11px; color: var(--text-muted); }
.mdt__btn {
  border: 1px solid var(--border-default); background: var(--surface-elevated);
  color: var(--text-secondary); border-radius: var(--radius-sm); font-size: 11px;
  padding: 3px 8px; cursor: pointer;
}
.mdt__btn:hover:not(:disabled) { background: var(--surface-hover); color: var(--text-primary); }
.mdt__btn:disabled { opacity: .5; cursor: default; }
.mdt__bar { height: 5px; background: rgba(248,81,73,0.28); }
.mdt__bar-ok { height: 100%; background: #3fb950; transition: width .3s; }

.mdt__empty { padding: 24px; text-align: center; color: var(--text-muted); font-size: 12.5px; }

.mdt__wrap { overflow: auto; user-select: none; flex: 1 1 auto; min-height: 0; }
.mdt__table { width: 100%; min-height: 100%; border-collapse: separate; border-spacing: 0; font-size: 12.5px; }

.mdt-th {
  padding: 7px 10px; font-weight: 600; font-size: 11px; text-align: left;
  color: var(--text-secondary); background: var(--surface-sunken);
  border-bottom: 1px solid var(--border-default); border-right: 1px solid var(--border-subtle);
  white-space: nowrap; user-select: none;
  position: sticky; top: 0; z-index: 1;
}
.mdt-th:last-child { border-right: none; }
.mdt-th--pick { width: 32px; text-align: center; }
.mdt-th--num, .mdt-th--rate { text-align: right; }
.mdt-th--sortable { cursor: pointer; transition: background .12s, color .12s; }
.mdt-th--sortable:hover { background: var(--surface-hover); color: var(--text-primary); }
.mdt-th__a { font-size: 9px; opacity: .5; margin-left: 3px; }
.mdt-th[aria-sort] { color: var(--mdt-accent); }
.mdt-th[aria-sort] .mdt-th__a { opacity: 1; }

.mdt-td {
  padding: 7px 10px; color: var(--text-primary);
  border-bottom: 1px solid var(--border-subtle); border-right: 1px solid var(--border-subtle);
}
.mdt-td:last-child { border-right: none; }
.mdt__table tbody tr { cursor: pointer; }
.mdt__table tbody tr:last-child .mdt-td { border-bottom: none; }
.mdt__table tbody tr:nth-child(even) .mdt-td { background: rgba(127,127,127,0.04); }
.mdt__table tbody tr:hover .mdt-td { background: var(--surface-hover-subtle, rgba(127,127,127,0.09)); }
.mdt__table tbody tr.mdt-row--selected .mdt-td {
  background: color-mix(in srgb, var(--mdt-accent) 14%, transparent);
}

.mdt-td--pick { width: 32px; text-align: center; padding-left: 8px; padding-right: 8px; }
.mdt-td--pick input { pointer-events: none; accent-color: var(--mdt-accent); }
.mdt-td--stt { width: 46px; color: var(--text-muted); font-variant-numeric: tabular-nums; }
.mdt-td--domain { font-family: var(--font-mono, monospace); font-weight: 600; }
.mdt-td--num { text-align: right; font-variant-numeric: tabular-nums; width: 72px; }
.mdt-td--rate { width: 132px; }
.mdt-live { color: #3fb950; }
.mdt-die { color: #f85149; }

.mdt-cellrate { display: flex; align-items: center; justify-content: flex-end; gap: 7px; }
.mdt-cellrate__txt { font-variant-numeric: tabular-nums; font-weight: 600; min-width: 40px; text-align: right; }
.mdt-cellrate__bar { flex: 0 0 44px; height: 6px; border-radius: 3px; overflow: hidden; background: rgba(248,81,73,0.25); }
.mdt-cellrate__fill { display: block; height: 100%; background: #3fb950; }

.mdt__table tfoot .mdt-td {
  position: sticky; bottom: 0; z-index: 2;
  font-weight: 700; background: var(--surface-sunken);
  border-top: 2px solid var(--border-default); border-bottom: none;
  box-shadow: 0 -2px 6px rgba(0,0,0,0.06);
}
</style>
