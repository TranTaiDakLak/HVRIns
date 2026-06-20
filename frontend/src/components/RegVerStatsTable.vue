<script setup lang="ts">
// RegVerStatsTable.vue - bảng thống kê một nguồn (REG hoặc VERIFY).
import { ref, computed, watch } from 'vue'
import { useAppStore } from '@/stores/app.store'

interface StatRow {
  index?: number
  platform: string
  success: number
  fail: number
  total: number
}
type SortKey = 'platform' | 'success' | 'fail' | 'total' | 'rate'

const props = defineProps<{
  title: string
  accent: 'reg' | 'ver'
  rows: StatRow[]
  exportKind: 'reg' | 'verify'
  platformLabel?: string
}>()

const sortKey = ref<SortKey>('total')
const sortDir = ref<'asc' | 'desc'>('desc')
const selected = ref<Set<string>>(new Set())
const dragMode = ref<'select' | 'deselect' | null>(null)
const lastClicked = ref<string | null>(null)
const appStore = useAppStore()

function rateOf(r: StatRow): number { return r.total > 0 ? r.success / r.total : 0 }
function pct(v: number): string { return (v * 100).toFixed(1) + '%' }
function rateLabel(r: StatRow): string { return r.total > 0 ? pct(rateOf(r)) : '-' }

function setSort(key: SortKey) {
  if (sortKey.value === key) sortDir.value = sortDir.value === 'asc' ? 'desc' : 'asc'
  else { sortKey.value = key; sortDir.value = key === 'platform' ? 'asc' : 'desc' }
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
    if (sortKey.value === 'platform') cmp = a.platform.localeCompare(b.platform, undefined, { numeric: true })
    else if (sortKey.value === 'rate') cmp = rateOf(a) - rateOf(b)
    else cmp = (a[sortKey.value] as number) - (b[sortKey.value] as number)
    if (cmp === 0) cmp = a.platform.localeCompare(b.platform, undefined, { numeric: true })
    return cmp * dir
  })
  return arr
})
const selectedCount = computed(() => selected.value.size)
const totals = computed(() => {
  let s = 0, f = 0
  for (const r of props.rows) { s += r.success; f += r.fail }
  return { success: s, fail: f, total: s + f, rate: (s + f) > 0 ? s / (s + f) : 0 }
})

watch(() => props.rows.map(r => r.platform).join('\n'), () => {
  const available = new Set(props.rows.map(r => r.platform))
  selected.value = new Set([...selected.value].filter(p => available.has(p)))
})

function setSelected(platform: string, on: boolean) {
  const next = new Set(selected.value)
  if (on) next.add(platform)
  else next.delete(platform)
  selected.value = next
}

function selectRange(toPlatform: string, on = true) {
  const from = lastClicked.value
  const fromIdx = from ? sorted.value.findIndex(r => r.platform === from) : -1
  const toIdx = sorted.value.findIndex(r => r.platform === toPlatform)
  if (fromIdx < 0 || toIdx < 0) {
    setSelected(toPlatform, on)
    return
  }
  const [start, end] = fromIdx < toIdx ? [fromIdx, toIdx] : [toIdx, fromIdx]
  const next = new Set(selected.value)
  for (const row of sorted.value.slice(start, end + 1)) {
    if (on) next.add(row.platform)
    else next.delete(row.platform)
  }
  selected.value = next
}

function onRowMouseDown(row: StatRow, e: MouseEvent) {
  if (e.button !== 0) return
  e.preventDefault()
  if (e.shiftKey) {
    selectRange(row.platform, true)
    lastClicked.value = row.platform
    dragMode.value = 'select'
    return
  }
  const shouldSelect = !selected.value.has(row.platform)
  setSelected(row.platform, shouldSelect)
  lastClicked.value = row.platform
  dragMode.value = shouldSelect ? 'select' : 'deselect'
}

function onRowEnter(row: StatRow) {
  if (!dragMode.value) return
  setSelected(row.platform, dragMode.value === 'select')
}

function stopDrag() {
  dragMode.value = null
}

function clearSelection() {
  selected.value = new Set()
}

async function exportSelection() {
  const list = sorted.value.map(r => r.platform).filter(p => selected.value.has(p))
  if (list.length === 0) return

  // Nếu platformLabel = "FBAV" → xuất ra FILE trong result folder + mở Explorer highlight.
  // Backend: ExportFbVersionPool(kind, fbavList) ghi file + lookup FBBV + mở Explorer.
  if (props.platformLabel === 'FBAV') {
    try {
      const { ExportFbVersionPool } = await import('../../wailsjs/go/main/App')
      // exportKind: "reg" / "verify" — map sang backend "reg" / "ver".
      const kind = props.exportKind === 'verify' ? 'ver' : 'reg'
      const result: string = await ExportFbVersionPool(kind, list)
      if (result.startsWith('ERR|')) {
        appStore.notify('error', result.slice(4))
        return
      }
      // Format: "OK|<num>" hoặc "OK|<num>|missing=<n>"
      const parts = result.split('|')
      const num = parts[1] || '?'
      let msg = `Đã xuất ${num} dòng FBAV|FBBV ra file`
      const missing = parts.find(p => p.startsWith('missing='))
      if (missing) {
        msg += ` (${missing.slice(8)} FBAV không tìm thấy FBBV)`
      }
      appStore.notify('success', msg)
    } catch (e) {
      appStore.notify('error', 'Không xuất được file: ' + String(e))
    }
    return
  }

  // Default — xuất JSON cho non-FBAV table (vd platform list).
  const payload = {
    kind: 'hvr-platform-selection',
    version: 1,
    [props.exportKind]: list,
  }
  try {
    await navigator.clipboard.writeText(JSON.stringify(payload, null, 2))
    appStore.notify('success', `Đã xuất ${list.length} dòng ${props.exportKind.toUpperCase()} vào clipboard`)
  } catch {
    appStore.notify('error', 'Không ghi được vào clipboard')
  }
}

function selectAllVisible() {
  selected.value = new Set(sorted.value.map(r => r.platform))
}
</script>

<template>
  <div class="svt" :class="`svt--${accent}`">
    <div class="svt__head">
      <span class="svt__title">{{ title }}</span>
      <span v-if="rows.length" class="svt__sub">
        {{ totals.total }} lượt · TC {{ totals.success }} · TB {{ totals.fail }} · {{ pct(totals.rate) }}
      </span>
      <div v-if="rows.length" class="svt__actions">
        <span class="svt__selected">{{ selectedCount }} chọn</span>
        <button class="svt__btn" @click="selectAllVisible">Chọn tất cả</button>
        <button class="svt__btn" :disabled="selectedCount === 0" @click="exportSelection">{{ platformLabel === 'FBAV' ? 'Xuất FBAV|FBBV' : 'Xuất JSON' }}</button>
        <button class="svt__btn" :disabled="selectedCount === 0" @click="clearSelection">Bỏ chọn</button>
      </div>
    </div>
    <div v-if="rows.length" class="svt__bar" :title="pct(totals.rate)">
      <div class="svt__bar-ok" :style="{ width: pct(totals.rate) }"></div>
    </div>

    <div v-if="rows.length === 0" class="svt__empty">Chưa có dữ liệu - bắt đầu chạy để xem.</div>

    <div v-else class="svt__wrap" @mouseleave="stopDrag" @mouseup="stopDrag">
      <table class="svt__table">
        <thead>
          <tr>
            <th class="svt-th svt-th--pick"></th>
            <th class="svt-th svt-th--stt">STT</th>
            <th class="svt-th svt-th--sortable svt-th--api" :aria-sort="ariaSort('platform')" @click="setSort('platform')">{{ platformLabel ?? 'API' }} <span class="svt-th__a">{{ arrow('platform') }}</span></th>
            <th class="svt-th svt-th--sortable svt-th--num" :aria-sort="ariaSort('success')" @click="setSort('success')">Thành công <span class="svt-th__a">{{ arrow('success') }}</span></th>
            <th class="svt-th svt-th--sortable svt-th--num" :aria-sort="ariaSort('fail')" @click="setSort('fail')">Thất bại <span class="svt-th__a">{{ arrow('fail') }}</span></th>
            <th class="svt-th svt-th--sortable svt-th--num" :aria-sort="ariaSort('total')" @click="setSort('total')">Tổng <span class="svt-th__a">{{ arrow('total') }}</span></th>
            <th class="svt-th svt-th--sortable svt-th--rate" :aria-sort="ariaSort('rate')" @click="setSort('rate')">Tỉ lệ <span class="svt-th__a">{{ arrow('rate') }}</span></th>
          </tr>
        </thead>
        <tbody>
          <tr
            v-for="(r, i) in sorted"
            :key="r.platform"
            :class="{ 'svt-row--selected': selected.has(r.platform) }"
            @mousedown="onRowMouseDown(r, $event)"
            @mouseenter="onRowEnter(r)"
          >
            <td class="svt-td svt-td--pick">
              <input type="checkbox" :checked="selected.has(r.platform)" tabindex="-1" readonly />
            </td>
            <td class="svt-td svt-td--stt">{{ i + 1 }}</td>
            <td class="svt-td svt-td--api">{{ r.platform }}</td>
            <td class="svt-td svt-td--num svt-ok">{{ r.success }}</td>
            <td class="svt-td svt-td--num svt-fail">{{ r.fail }}</td>
            <td class="svt-td svt-td--num">{{ r.total }}</td>
            <td class="svt-td svt-td--rate">
              <div class="svt-cellrate">
                <span class="svt-cellrate__txt">{{ rateLabel(r) }}</span>
                <span class="svt-cellrate__bar"><span class="svt-cellrate__fill" :style="{ width: pct(rateOf(r)) }"></span></span>
              </div>
            </td>
          </tr>
        </tbody>
        <tfoot v-if="rows.length > 1">
          <tr>
            <td class="svt-td svt-td--pick"></td>
            <td class="svt-td svt-td--stt"></td>
            <td class="svt-td svt-td--api">Tổng cộng</td>
            <td class="svt-td svt-td--num svt-ok">{{ totals.success }}</td>
            <td class="svt-td svt-td--num svt-fail">{{ totals.fail }}</td>
            <td class="svt-td svt-td--num">{{ totals.total }}</td>
            <td class="svt-td svt-td--rate"><div class="svt-cellrate"><span class="svt-cellrate__txt">{{ pct(totals.rate) }}</span></div></td>
          </tr>
        </tfoot>
      </table>
    </div>
  </div>
</template>

<style scoped>
.svt {
  flex: 1 1 0; min-width: 340px;
  border: 1px solid var(--border-default); border-radius: var(--radius-md);
  background: var(--surface-elevated); overflow: hidden;
  display: flex;
  flex-direction: column;
  height: calc(100vh - 190px);
  min-height: 440px;
  --svt-accent: #f97316;
}
.svt--ver { --svt-accent: #3b82f6; }

.svt__head {
  display: flex; align-items: center; gap: 10px; flex-wrap: wrap;
  padding: 9px 12px; background: var(--surface-sunken); border-bottom: 1px solid var(--border-default);
  border-left: 3px solid var(--svt-accent);
}
.svt__title { font-weight: 700; font-size: 13px; color: var(--text-primary); }
.svt__sub { font-size: 11px; color: var(--text-muted); font-variant-numeric: tabular-nums; }
.svt__actions { margin-left: auto; display: flex; align-items: center; gap: 6px; }
.svt__selected { font-size: 11px; color: var(--text-muted); }
.svt__btn {
  border: 1px solid var(--border-default); background: var(--surface-elevated);
  color: var(--text-secondary); border-radius: var(--radius-sm); font-size: 11px;
  padding: 3px 8px; cursor: pointer;
}
.svt__btn:hover:not(:disabled) { background: var(--surface-hover); color: var(--text-primary); }
.svt__btn:disabled { opacity: .5; cursor: default; }
.svt__bar { height: 5px; background: rgba(248,81,73,0.28); }
.svt__bar-ok { height: 100%; background: #3fb950; transition: width .3s; }

.svt__empty { padding: 24px; text-align: center; color: var(--text-muted); font-size: 12.5px; }

.svt__wrap {
  overflow: auto;
  user-select: none;
  flex: 1 1 auto;
  min-height: 0;
}
.svt__table { width: 100%; min-height: 100%; border-collapse: separate; border-spacing: 0; font-size: 12.5px; }
.svt-th {
  padding: 7px 10px; font-weight: 600; font-size: 11px; text-align: left;
  color: var(--text-secondary); background: var(--surface-sunken);
  border-bottom: 1px solid var(--border-default); border-right: 1px solid var(--border-subtle);
  white-space: nowrap; user-select: none;
  position: sticky;
  top: 0;
  z-index: 1;
}
.svt-th:last-child { border-right: none; }
.svt-th--pick { width: 32px; text-align: center; }
.svt-th--num, .svt-th--rate { text-align: right; }
.svt-th--sortable { cursor: pointer; transition: background .12s, color .12s; }
.svt-th--sortable:hover { background: var(--surface-hover); color: var(--text-primary); }
.svt-th__a { font-size: 9px; opacity: .5; margin-left: 3px; }
.svt-th[aria-sort] { color: var(--svt-accent); }
.svt-th[aria-sort] .svt-th__a { opacity: 1; }

.svt-td {
  padding: 7px 10px; color: var(--text-primary);
  border-bottom: 1px solid var(--border-subtle); border-right: 1px solid var(--border-subtle);
}
.svt-td:last-child { border-right: none; }
.svt__table tbody tr { cursor: pointer; }
.svt__table tbody tr:last-child .svt-td { border-bottom: none; }
.svt__table tbody tr:nth-child(even) .svt-td { background: rgba(127,127,127,0.04); }
.svt__table tbody tr:hover .svt-td { background: var(--surface-hover-subtle, rgba(127,127,127,0.09)); }
.svt__table tbody tr.svt-row--selected .svt-td {
  background: color-mix(in srgb, var(--svt-accent) 14%, transparent);
}

.svt-td--pick { width: 32px; text-align: center; padding-left: 8px; padding-right: 8px; }
.svt-td--pick input { pointer-events: none; accent-color: var(--svt-accent); }
.svt-td--stt { width: 46px; color: var(--text-muted); font-variant-numeric: tabular-nums; }
.svt-td--api { font-family: var(--font-mono, monospace); font-weight: 600; }
.svt-td--num { text-align: right; font-variant-numeric: tabular-nums; width: 92px; }
.svt-td--rate { width: 132px; }
.svt-ok { color: #3fb950; }
.svt-fail { color: #f85149; }

.svt-cellrate { display: flex; align-items: center; justify-content: flex-end; gap: 7px; }
.svt-cellrate__txt { font-variant-numeric: tabular-nums; font-weight: 600; min-width: 40px; text-align: right; }
.svt-cellrate__bar { flex: 0 0 44px; height: 6px; border-radius: 3px; overflow: hidden; background: rgba(248,81,73,0.25); }
.svt-cellrate__fill { display: block; height: 100%; background: #3fb950; }

.svt__table tfoot .svt-td {
  position: sticky;
  bottom: 0;
  z-index: 2;
  font-weight: 700; background: var(--surface-sunken);
  border-top: 2px solid var(--border-default); border-bottom: none;
  box-shadow: 0 -2px 6px rgba(0,0,0,0.06);
}
</style>
