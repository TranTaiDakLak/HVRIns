<!-- BaseTabs.vue — Thanh tab ngang
     Hiệu ứng indicator trượt khi chuyển tab
     Hỗ trợ v-model qua key -->

<script setup lang="ts">
import { ref, watch, nextTick, onMounted } from 'vue'

// --- Types ---
interface Tab {
  key: string
  label: string
}

// --- Props ---
const props = defineProps<{
  modelValue: string
  tabs: Tab[]
}>()

const emit = defineEmits<{
  'update:modelValue': [value: string]
}>()

// Ref cho các nút tab để tính vị trí indicator
const tabRefs = ref<HTMLElement[]>([])

// Vị trí và kích thước của thanh indicator
const indicatorLeft = ref(0)
const indicatorWidth = ref(0)

// Cập nhật vị trí indicator dựa vào tab đang active
function updateIndicator() {
  const activeIndex = props.tabs.findIndex(t => t.key === props.modelValue)
  if (activeIndex >= 0 && tabRefs.value[activeIndex]) {
    const el = tabRefs.value[activeIndex]
    indicatorLeft.value = el.offsetLeft
    indicatorWidth.value = el.offsetWidth
  }
}

// Chọn tab mới
function selectTab(key: string) {
  emit('update:modelValue', key)
}

// Theo dõi thay đổi modelValue để cập nhật indicator
watch(() => props.modelValue, () => {
  nextTick(updateIndicator)
})

onMounted(() => {
  nextTick(updateIndicator)
})
</script>

<template>
  <div class="base-tabs">
    <div class="base-tabs__list">
      <button
        v-for="(tab, i) in tabs"
        :key="tab.key"
        :ref="(el) => { if (el) tabRefs[i] = el as HTMLElement }"
        class="base-tabs__tab"
        :class="{ 'base-tabs__tab--active': modelValue === tab.key }"
        @click="selectTab(tab.key)"
      >
        {{ tab.label }}
      </button>

      <!-- Thanh indicator trượt bên dưới tab active -->
      <span
        class="base-tabs__indicator"
        :style="{
          left: indicatorLeft + 'px',
          width: indicatorWidth + 'px',
        }"
      />
    </div>
  </div>
</template>

<style scoped>
/* === Container === */
.base-tabs {
  width: 100%;
}

/* === Danh sách tab === */
.base-tabs__list {
  position: relative;
  display: flex;
  gap: var(--space-1);
  border-bottom: 1px solid var(--border-default);
}

/* === Mỗi tab === */
.base-tabs__tab {
  padding: var(--space-2) var(--space-4);
  background: none;
  border: none;
  font-family: var(--font-family);
  font-size: var(--font-size-base);
  color: var(--text-secondary);
  cursor: pointer;
  transition: color var(--transition-fast);
  white-space: nowrap;
  outline: none;
}

.base-tabs__tab:hover {
  color: var(--text-primary);
}

.base-tabs__tab--active {
  color: var(--brand-primary);
}

.base-tabs__tab:focus-visible {
  color: var(--text-primary);
  background: var(--brand-primary-bg);
  border-radius: var(--radius-sm) var(--radius-sm) 0 0;
}

/* === Thanh indicator trượt === */
.base-tabs__indicator {
  position: absolute;
  bottom: -1px;
  height: 2px;
  background: var(--brand-primary);
  border-radius: 1px;
  transition: left var(--transition-normal), width var(--transition-normal);
}
</style>
