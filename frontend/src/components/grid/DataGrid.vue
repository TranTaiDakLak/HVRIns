<script setup lang="ts">
// DataGrid.vue — Component grid chính với virtual scroll, sort, multi-select
// Đây là component phức tạp nhất, nền tảng cho Accounts grid

import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import { ChevronUp, ChevronDown } from 'lucide-vue-next'
import type { ColumnDef } from '../../composables/useColumnVisibility'

export interface DataGridProps {
  // Dữ liệu đã sorted/filtered (từ useDataGrid)
  rows: { item: Record<string, unknown>; index: number }[]
  // Cột hiển thị
  columns: ColumnDef[]
  // Tổng chiều cao nội dung (cho virtual scroll spacer)
  totalHeight: number
  // Offset phía trên (cho virtual scroll positioning)
  offsetTop: number
  // Chiều cao mỗi row
  rowHeight?: number
  // Sort state hiện tại
  sortColumn?: string
  sortDirection?: 'asc' | 'desc' | null
  sortDisabled?: boolean
  // Checked (checkbox state)
  selectedIds?: Set<number>
  // Highlighted (bôi đen - row click/drag)
  highlightedIds?: Set<number>
  // Selected cells — key format "rowId:colKey"
  selectedCells?: Set<string>
  // Có hiện checkbox column không
  showCheckbox?: boolean
  // Có hiện STT column không
  showIndex?: boolean
  // Custom row class function
  rowClass?: (item: Record<string, unknown>) => string
}

const props = withDefaults(defineProps<DataGridProps>(), {
  rowHeight: 36,
  sortColumn: '',
  sortDirection: null,
  sortDisabled: false,
  showCheckbox: true,
  showIndex: true,
})

const emit = defineEmits<{
  // Sort column
  sort: [column: string]
  // Scroll event (cho virtual scroll)
  scroll: [event: Event]
  // Click row
  'row-click': [id: number, event: MouseEvent]
  // Double-click row
  'row-dblclick': [id: number]
  // Toggle checkbox header (select all)
  'toggle-all': []
  // Toggle checkbox row
  'toggle-row': [id: number, event: MouseEvent]
  // Context menu
  'contextmenu': [id: number, event: MouseEvent]
  // Drag select
  'row-mousedown': [id: number, event: MouseEvent]
  'row-mouseenter': [id: number, event: MouseEvent]
  // Cell click — cho select từng ô + Ctrl+C copy
  'cell-click': [rowId: number, colKey: string, event: MouseEvent]
  // Cell drag — cho kéo chọn nhiều ô
  'cell-mousedown': [rowId: number, colKey: string, event: MouseEvent]
  'cell-mouseenter': [rowId: number, colKey: string, event: MouseEvent]
}>()

// Refs cho scroll sync
const scrollContainer = ref<HTMLElement>()
const headerContainer = ref<HTMLElement>()

// Theo dõi viewport height cho virtual scroll
const viewportHeight = ref(600)
let resizeObserver: ResizeObserver | null = null

onMounted(() => {
  if (scrollContainer.value) {
    viewportHeight.value = scrollContainer.value.clientHeight
    resizeObserver = new ResizeObserver(entries => {
      viewportHeight.value = entries[0].contentRect.height
    })
    resizeObserver.observe(scrollContainer.value)
  }
})

onUnmounted(() => {
  resizeObserver?.disconnect()
})

// Header checkbox: dựa trên checked (selectedIds), không phải highlighted
const allSelected = computed(() => {
  if (!props.selectedIds || props.rows.length === 0) return false
  return props.rows.every(r => props.selectedIds!.has((r.item as { id: number }).id))
})

const someSelected = computed(() => {
  if (!props.selectedIds || props.rows.length === 0) return false
  const count = props.rows.filter(r => props.selectedIds!.has((r.item as { id: number }).id)).length
  return count > 0 && count < props.rows.length
})

// === Column Resize ===
const columnWidths = ref<Record<string, number>>({})
const resizing = ref<{ column: string; startX: number; startWidth: number } | null>(null)

// Lấy width hiện tại của column (ưu tiên resize state, fallback props)
function getColumnWidth(col: { key: string; width?: string }): string {
  if (columnWidths.value[col.key]) return columnWidths.value[col.key] + 'px'
  return col.width || 'auto'
}

// Bắt đầu resize khi kéo border phải của header
function startResize(col: { key: string; width?: string }, event: MouseEvent) {
  event.preventDefault()
  event.stopPropagation()
  const th = (event.target as HTMLElement).parentElement!
  const startWidth = th.offsetWidth
  resizing.value = { column: col.key, startX: event.clientX, startWidth }

  function onMouseMove(e: MouseEvent) {
    if (!resizing.value) return
    const diff = e.clientX - resizing.value.startX
    const newWidth = Math.max(50, resizing.value.startWidth + diff)
    columnWidths.value[resizing.value.column] = newWidth
  }

  function onMouseUp() {
    resizing.value = null
    document.removeEventListener('mousemove', onMouseMove)
    document.removeEventListener('mouseup', onMouseUp)
  }

  document.addEventListener('mousemove', onMouseMove)
  document.addEventListener('mouseup', onMouseUp)
}

// Xử lý scroll — sync header scrollLeft với body
function handleScroll(event: Event) {
  emit('scroll', event)
  // Sync header horizontal scroll với body
  if (headerContainer.value && scrollContainer.value) {
    headerContainer.value.scrollLeft = scrollContainer.value.scrollLeft
  }
}

// Sort icon
function getSortIcon(column: string): string {
  if (props.sortColumn !== column) return ''
  return props.sortDirection === 'asc' ? 'asc' : props.sortDirection === 'desc' ? 'desc' : ''
}
</script>

<template>
  <div class="data-grid">
    <!-- Header cố định — scroll sync với body -->
    <div ref="headerContainer" class="data-grid__header">
      <table class="data-grid__table">
        <colgroup>
          <col v-if="showCheckbox" style="width:28px;min-width:28px;max-width:28px" />
          <col v-if="showIndex" style="width:40px;min-width:40px;max-width:40px" />
          <col v-for="col in columns" :key="col.key" :style="col.width && col.width !== 'auto' ? { width: getColumnWidth(col), minWidth: getColumnWidth(col) } : {}" />
        </colgroup>
        <thead>
          <tr>
            <!-- Checkbox column -->
            <th v-if="showCheckbox" class="data-grid__cell--checkbox" @click="emit('toggle-all')">
              <input
                type="checkbox"
                :checked="allSelected"
                :indeterminate="someSelected"
                @click.stop="emit('toggle-all')"
              />
            </th>
            <!-- STT column -->
            <th v-if="showIndex" class="data-grid__cell--index">STT</th>
            <!-- Data columns (với resize handle) -->
            <th
              v-for="col in columns"
              :key="col.key"
              :style="{ textAlign: col.align || 'left' }"
              :class="{ 'data-grid__cell--sortable': col.sortable !== false && !sortDisabled }"
              @click="col.sortable !== false && !sortDisabled && emit('sort', col.key)"
            >
              {{ col.label }}
              <span v-if="getSortIcon(col.key)" class="data-grid__sort-icon">
                <ChevronUp v-if="getSortIcon(col.key) === 'asc'" :size="12" />
                <ChevronDown v-else :size="12" />
              </span>
              <!-- Resize handle -->
              <span class="data-grid__resize-handle" @mousedown="startResize(col, $event)" />
            </th>
          </tr>
        </thead>
      </table>
    </div>

    <!-- Body với virtual scroll -->
    <div
      ref="scrollContainer"
      class="data-grid__body"
      @scroll="handleScroll"
    >
      <!-- Spacer tạo chiều cao tổng cho scrollbar -->
      <div :style="{ height: totalHeight + 'px', position: 'relative' }">
        <!-- Container cho visible rows, offset bằng transform -->
        <table
          class="data-grid__table"
          :style="{ transform: `translateY(${offsetTop}px)` }"
        >
          <colgroup>
            <col v-if="showCheckbox" style="width:28px;min-width:28px;max-width:28px" />
            <col v-if="showIndex" style="width:40px;min-width:40px;max-width:40px" />
            <col v-for="col in columns" :key="col.key" :style="col.width && col.width !== 'auto' ? { width: getColumnWidth(col), minWidth: getColumnWidth(col) } : {}" />
          </colgroup>
          <tbody>
            <tr
              v-for="{ item, index: rowIdx } in rows"
              :key="(item as any).id"
              :class="[
                {
                  'data-grid__row--checked': selectedIds?.has((item as any).id),
                  'data-grid__row--highlighted': highlightedIds?.has((item as any).id),
                  'data-grid__row--even': rowIdx % 2 === 0,
                },
                rowClass ? rowClass(item) : '',
              ]"
              :style="{ height: rowHeight + 'px' }"
              @click="emit('row-click', (item as any).id, $event)"
              @dblclick="emit('row-dblclick', (item as any).id)"
              @contextmenu.prevent="emit('contextmenu', (item as any).id, $event)"
              @mousedown.left="emit('row-mousedown', (item as any).id, $event)"
              @mouseenter="emit('row-mouseenter', (item as any).id, $event)"
            >
              <!-- Checkbox -->
              <td v-if="showCheckbox" class="data-grid__cell--checkbox">
                <input
                  type="checkbox"
                  :checked="selectedIds?.has((item as any).id)"
                  @click.stop="emit('toggle-row', (item as any).id, $event)"
                />
              </td>
              <!-- STT -->
              <td v-if="showIndex" class="data-grid__cell--index">
                {{ rowIdx + 1 }}
              </td>
              <!-- Data cells -->
              <td
                v-for="col in columns"
                :key="col.key"
                :style="{ textAlign: col.align || 'left' }"
                :class="[
                  'data-grid__cell',
                  { 'data-grid__cell--selected': selectedCells?.has(`${(item as any).id}:${col.key}`) },
                ]"
                @click.stop="emit('cell-click', (item as any).id, col.key, $event)"
                @mousedown.left.stop="emit('cell-mousedown', (item as any).id, col.key, $event)"
                @mouseenter="emit('cell-mouseenter', (item as any).id, col.key, $event)"
              >
                <!-- Slot cho custom cell render, fallback là text -->
                <slot :name="`cell-${col.key}`" :row="item" :value="item[col.key]">
                  {{ item[col.key] ?? '' }}
                </slot>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </div>
</template>

<style scoped>
.data-grid {
  display: flex;
  flex-direction: column;
  height: 100%;
  overflow: hidden;
  background: var(--surface-sunken);
}

/* === Header === */
.data-grid__header {
  flex-shrink: 0;
  overflow-x: auto;
  overflow-y: hidden;
  scrollbar-width: none; /* Firefox */
}
.data-grid__header::-webkit-scrollbar {
  display: none; /* Chrome/Safari — ẩn scrollbar nhưng vẫn scroll được */
}

.data-grid__header th {
  position: sticky;
  top: 0;
  background: var(--surface-elevated);
  border-bottom: 2px solid var(--border-default);
  padding: var(--grid-padding);
  text-align: left;
  font-weight: 700;
  font-size: var(--font-size-xs);
  color: var(--text-primary);
  text-transform: uppercase;
  letter-spacing: 0.4px;
  white-space: nowrap;
  user-select: none;
}

.data-grid__cell--sortable {
  cursor: pointer;
}

.data-grid__cell--sortable:hover {
  color: var(--text-primary);
}

.data-grid__sort-icon {
  color: var(--brand-primary);
  margin-left: var(--space-1);
  font-size: 10px;
}

/* === Body === */
.data-grid__body {
  flex: 1;
  overflow-y: auto;
  overflow-x: auto;
}

/* === Table === */
.data-grid__table {
  width: 100%;
  border-collapse: collapse;
  font-size: var(--grid-font-size);
  table-layout: fixed;
}

/* === Rows === */
.data-grid__table tbody tr {
  border-bottom: 1px solid var(--border-subtle);
  cursor: pointer;
  transition: background var(--transition-fast);
}

.data-grid__table tbody tr:hover {
  background: var(--surface-elevated);
}

/* Checked row (checkbox ticked) — xanh rõ hơn + border trái accent. */
.data-grid__row--checked {
  background: rgba(34, 197, 94, 0.18);
  box-shadow: inset 3px 0 0 0 var(--brand-primary, #22c55e);
}
.data-grid__row--checked:hover {
  background: rgba(34, 197, 94, 0.25);
}

/* Highlighted row (bôi đen - click/drag) — nổi bật hơn */
.data-grid__row--highlighted {
  background: rgba(96, 165, 250, 0.25) !important;
}

/* Row vừa checked vừa highlighted — đậm nhất */
.data-grid__row--checked.data-grid__row--highlighted {
  background: rgba(34, 197, 94, 0.32) !important;
}

/* === Cells === */
.data-grid__cell {
  padding: var(--grid-padding);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 200px;
  color: var(--text-primary);
}

/* Cell đang được select (Excel-style cell selection) — đậm + outline rõ */
.data-grid__cell--selected {
  background: rgba(59, 130, 246, 0.32) !important;
  outline: 2px solid rgba(59, 130, 246, 0.9);
  outline-offset: -2px;
  position: relative;
  z-index: 1;
  font-weight: 500;
}

.data-grid__cell--checkbox {
  width: 28px;
  min-width: 28px;
  max-width: 28px;
  text-align: center;
  padding: 0;
}

.data-grid__cell--checkbox input[type="checkbox"] {
  accent-color: var(--brand-primary);
  cursor: pointer;
}

/* === Resize Handle === */
.data-grid__header th {
  position: relative;
}

.data-grid__resize-handle {
  position: absolute;
  right: 0;
  top: 0;
  bottom: 0;
  width: 4px;
  cursor: col-resize;
  background: transparent;
  transition: background var(--transition-fast);
}

.data-grid__resize-handle:hover,
.data-grid__resize-handle:active {
  background: var(--brand-primary);
}

.data-grid__cell--index {
  width: 40px;
  min-width: 40px;
  max-width: 40px;
  text-align: center;
  color: var(--text-muted);
  font-size: var(--font-size-xs);
  padding: var(--grid-padding);
}

/* === Row status colors === */
.data-grid__row--status-live       { background: rgba(34, 197, 94, 0.13); }
.data-grid__row--status-die        { background: rgba(239, 68, 68, 0.18); }
.data-grid__row--status-checkpoint { background: rgba(251, 191, 36, 0.06); }
.data-grid__row--status-new        { background: rgba(96, 165, 250, 0.06); }
.data-grid__row--status-unknown    { background: transparent; }
/* NVR: reg xong chưa verify — vàng nhạt */
.data-grid__row--status-nvr        { background: rgba(234, 179, 8, 0.20); }
/* AddMail: addmail fail sau retry — cam đậm */
.data-grid__row--status-addmail    { background: rgba(194, 65, 12, 0.28); }
/* VerFail: verify xác định die — xám */
.data-grid__row--status-verfail    { background: rgba(107, 114, 128, 0.12); }

/* Highlighted overrides status color */
.data-grid__row--highlighted.data-grid__row--status-live,
.data-grid__row--highlighted.data-grid__row--status-die,
.data-grid__row--highlighted.data-grid__row--status-checkpoint,
.data-grid__row--highlighted.data-grid__row--status-new,
.data-grid__row--highlighted.data-grid__row--status-unknown,
.data-grid__row--highlighted.data-grid__row--status-nvr,
.data-grid__row--highlighted.data-grid__row--status-addmail,
.data-grid__row--highlighted.data-grid__row--status-verfail {
  background: rgba(96, 165, 250, 0.15) !important;
}
</style>
