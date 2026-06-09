<script setup lang="ts" generic="T extends string | number">
// SearchableSelect — dropdown có ô search lọc options theo label (case-insensitive).
// Panel teleport tới <body> + position fixed → tránh bị clip bởi overflow của parent.
// Auto-flip lên trên khi không đủ chỗ bên dưới.
import { ref, computed, watch, nextTick, onMounted, onBeforeUnmount } from 'vue'
import { ChevronDown, Search, X } from 'lucide-vue-next'

interface Option<V> {
  value: V
  label: string
}

interface Props {
  modelValue: T
  options: Option<T>[]
  placeholder?: string
  searchPlaceholder?: string
  maxVisible?: number
  disabled?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  placeholder: '— chọn —',
  searchPlaceholder: 'Tìm kiếm...',
  maxVisible: 10,
  disabled: false,
})

const emit = defineEmits<{
  'update:modelValue': [value: T]
}>()

const isOpen = ref(false)
const searchQuery = ref('')
const rootRef = ref<HTMLElement | null>(null)
const triggerRef = ref<HTMLButtonElement | null>(null)
const panelRef = ref<HTMLElement | null>(null)
const searchInputRef = ref<HTMLInputElement | null>(null)
const highlightIndex = ref(0)

// Panel fixed position — tính lại mỗi lần mở + khi scroll/resize.
const panelStyle = ref<Record<string, string>>({})
const panelMaxHeight = 340

function updatePanelPosition() {
  if (!triggerRef.value) return
  const rect = triggerRef.value.getBoundingClientRect()
  const vh = window.innerHeight
  const spaceBelow = vh - rect.bottom
  const spaceAbove = rect.top
  const gap = 4

  // Ưu tiên mở xuống nếu đủ chỗ; ngược lại flip lên.
  const flipUp = spaceBelow < 200 && spaceAbove > spaceBelow
  const maxH = Math.min(panelMaxHeight, flipUp ? spaceAbove - gap - 8 : spaceBelow - gap - 8)

  panelStyle.value = {
    position: 'fixed',
    left: rect.left + 'px',
    width: rect.width + 'px',
    maxHeight: Math.max(maxH, 180) + 'px',
    ...(flipUp
      ? { bottom: (vh - rect.top + gap) + 'px' }
      : { top: (rect.bottom + gap) + 'px' }),
  }
}

const selectedLabel = computed(() => {
  const found = props.options.find(o => o.value === props.modelValue)
  return found?.label ?? ''
})

const filteredOptions = computed(() => {
  const q = searchQuery.value.trim().toLowerCase()
  if (!q) return props.options
  return props.options.filter(o =>
    o.label.toLowerCase().includes(q) ||
    String(o.value).toLowerCase().includes(q)
  )
})

function toggle() {
  if (props.disabled) return
  isOpen.value = !isOpen.value
  if (isOpen.value) {
    searchQuery.value = ''
    highlightIndex.value = 0
    nextTick(() => {
      updatePanelPosition()
      searchInputRef.value?.focus()
    })
  }
}

function selectOption(opt: Option<T>) {
  emit('update:modelValue', opt.value)
  isOpen.value = false
}

function clearSearch() {
  searchQuery.value = ''
  searchInputRef.value?.focus()
}

function onKeydown(e: KeyboardEvent) {
  if (!isOpen.value) return
  if (e.key === 'Escape') {
    isOpen.value = false
    e.preventDefault()
    return
  }
  const opts = filteredOptions.value
  if (e.key === 'ArrowDown') {
    highlightIndex.value = Math.min(highlightIndex.value + 1, opts.length - 1)
    e.preventDefault()
  } else if (e.key === 'ArrowUp') {
    highlightIndex.value = Math.max(highlightIndex.value - 1, 0)
    e.preventDefault()
  } else if (e.key === 'Enter') {
    if (opts[highlightIndex.value]) {
      selectOption(opts[highlightIndex.value])
      e.preventDefault()
    }
  }
}

function onClickOutside(e: MouseEvent) {
  if (!isOpen.value) return
  const target = e.target as Node
  // Click phải nằm ngoài CẢ trigger lẫn panel (vì panel teleport ra body, không còn trong rootRef).
  const inTrigger = rootRef.value?.contains(target) ?? false
  const inPanel = panelRef.value?.contains(target) ?? false
  if (!inTrigger && !inPanel) {
    isOpen.value = false
  }
}

function onScrollOrResize() {
  if (isOpen.value) updatePanelPosition()
}

onMounted(() => {
  document.addEventListener('click', onClickOutside, true)
  window.addEventListener('scroll', onScrollOrResize, true)
  window.addEventListener('resize', onScrollOrResize)
})
onBeforeUnmount(() => {
  document.removeEventListener('click', onClickOutside, true)
  window.removeEventListener('scroll', onScrollOrResize, true)
  window.removeEventListener('resize', onScrollOrResize)
})

watch(filteredOptions, () => {
  highlightIndex.value = 0
})
</script>

<template>
  <div ref="rootRef" class="searchable-select" :class="{ 'is-open': isOpen, 'is-disabled': disabled }">
    <button ref="triggerRef" type="button" class="searchable-select__trigger" @click="toggle" :disabled="disabled">
      <span class="searchable-select__value" :class="{ 'searchable-select__value--placeholder': !selectedLabel }">
        {{ selectedLabel || placeholder }}
      </span>
      <ChevronDown :size="16" class="searchable-select__caret" />
    </button>

    <Teleport to="body">
      <div
        v-if="isOpen"
        ref="panelRef"
        class="searchable-select__panel"
        :style="panelStyle"
        @keydown="onKeydown"
      >
        <div class="searchable-select__search">
          <Search :size="14" class="searchable-select__search-icon" />
          <input
            ref="searchInputRef"
            v-model="searchQuery"
            type="text"
            class="searchable-select__search-input"
            :placeholder="searchPlaceholder"
            @keydown="onKeydown"
          />
          <button
            v-if="searchQuery"
            type="button"
            class="searchable-select__clear"
            @click="clearSearch"
            title="Xóa tìm kiếm"
          >
            <X :size="12" />
          </button>
        </div>

        <div class="searchable-select__list">
          <div
            v-for="(opt, i) in filteredOptions"
            :key="String(opt.value)"
            :class="[
              'searchable-select__option',
              { 'is-selected': opt.value === modelValue, 'is-highlighted': i === highlightIndex }
            ]"
            @click="selectOption(opt)"
            @mouseenter="highlightIndex = i"
          >
            {{ opt.label }}
          </div>
          <div v-if="filteredOptions.length === 0" class="searchable-select__empty">
            Không tìm thấy
          </div>
        </div>
      </div>
    </Teleport>
  </div>
</template>

<style>
/* Unscoped cho panel vì teleport tới body — scoped sẽ mất style. */
.searchable-select {
  position: relative;
  display: inline-block;
  width: 100%;
  max-width: 360px;
}
.searchable-select.is-disabled {
  opacity: 0.6;
  pointer-events: none;
}

.searchable-select__trigger {
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 100%;
  padding: 6px 10px;
  border: 1px solid var(--border-default);
  border-radius: var(--radius-md);
  background: var(--surface-elevated, #fff);
  color: var(--text-primary);
  font-size: 13px;
  cursor: pointer;
  transition: border-color 0.15s;
}
.searchable-select__trigger:hover {
  border-color: var(--brand-primary);
}
.searchable-select.is-open .searchable-select__trigger {
  border-color: var(--brand-primary);
}

.searchable-select__value {
  text-align: left;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  flex: 1;
}
.searchable-select__value--placeholder {
  color: var(--text-muted);
}
.searchable-select__caret {
  flex-shrink: 0;
  color: var(--text-secondary);
  transition: transform 0.15s;
}
.searchable-select.is-open .searchable-select__caret {
  transform: rotate(180deg);
}

/* Panel teleport tới body — z-index cao, style unscoped */
.searchable-select__panel {
  z-index: 9999;
  background: var(--surface-elevated, #fff);
  border: 1px solid var(--border-default, #e5e7eb);
  border-radius: var(--radius-md, 6px);
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.15);
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.searchable-select__search {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 10px;
  border-bottom: 1px solid var(--border-default, #e5e7eb);
  background: var(--surface-sunken, rgba(0,0,0,0.02));
  flex-shrink: 0;
}
.searchable-select__search-icon {
  color: var(--text-muted, #9ca3af);
  flex-shrink: 0;
}
.searchable-select__search-input {
  flex: 1;
  border: none;
  outline: none;
  background: transparent;
  font-size: 13px;
  color: var(--text-primary, #111);
  min-width: 0;
}
.searchable-select__clear {
  background: none;
  border: none;
  padding: 2px;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  color: var(--text-muted, #9ca3af);
  border-radius: 3px;
}
.searchable-select__clear:hover {
  background: rgba(0,0,0,0.06);
  color: var(--text-primary, #111);
}

.searchable-select__list {
  overflow-y: auto;
  flex: 1 1 auto;
  padding: 4px 0;
  min-height: 0;
}

.searchable-select__option {
  padding: 6px 12px;
  font-size: 13px;
  color: var(--text-primary, #111);
  cursor: pointer;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.searchable-select__option:hover,
.searchable-select__option.is-highlighted {
  background: rgba(34, 197, 94, 0.1);
  color: var(--brand-primary, #22c55e);
}
.searchable-select__option.is-selected {
  background: var(--brand-primary, #22c55e);
  color: #fff;
  font-weight: 500;
}
.searchable-select__option.is-selected.is-highlighted {
  background: var(--brand-primary, #22c55e);
  color: #fff;
}

.searchable-select__empty {
  padding: 12px;
  text-align: center;
  color: var(--text-muted, #9ca3af);
  font-size: 12px;
  font-style: italic;
}
</style>
