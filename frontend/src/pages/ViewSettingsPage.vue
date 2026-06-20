<script setup lang="ts">
import { computed } from 'vue'
import { usePreferencesStore, type DensityMode } from '@/stores/preferences.store'
import { ACCOUNT_COLUMNS } from '@/constants/columns'

const prefs = usePreferencesStore()

const densityOptions: { value: DensityMode; label: string; desc: string; icon: string }[] = [
  { value: 'compact',     label: 'Compact',     desc: 'Row 28px, font 11px', icon: '▤' },
  { value: 'default',     label: 'Default',     desc: 'Row 36px, font 13px', icon: '▦' },
  { value: 'comfortable', label: 'Comfortable', desc: 'Row 44px, font 14px', icon: '▧' },
]

const columnGroups = [
  {
    label: 'Tài khoản',
    keys: ['uid', 'fullData', 'password', 'twofa', 'email', 'passMail', 'mailRecovery', 'cookie', 'token'],
  },
  {
    label: 'Trạng thái',
    keys: ['status', 'checkpoint', 'statusAds', 'bm', 'tkqc', 'chatSupport'],
  },
  {
    label: 'Chạy & Khác',
    keys: ['avatar', 'cover', 'phone', 'proxy', 'userAgent', 'note', 'noteRun', 'importTime', 'category', 'lastRun', 'runProxy', 'activity'],
  },
]

function getGroupColumns(keys: string[]) {
  return ACCOUNT_COLUMNS.filter(c => keys.includes(c.key))
}

function isGroupAllChecked(keys: string[]) {
  return keys.every(k => prefs.isColumnVisible(k))
}

function toggleGroup(keys: string[], checked: boolean) {
  keys.forEach(k => {
    const visible = prefs.isColumnVisible(k)
    if (checked && !visible) prefs.toggleColumn(k)
    if (!checked && visible) prefs.toggleColumn(k)
  })
}

const totalVisible = computed(() => ACCOUNT_COLUMNS.filter(c => prefs.isColumnVisible(c.key)).length)
</script>

<template>
  <div class="vs-page">
    <!-- Header -->
    <div class="vs-header">
      <h2>Hiển thị</h2>
    </div>

    <div class="vs-body">
      <!-- Row 1: Density + Masking -->
      <div class="vs-row-top">
        <!-- Density -->
        <section class="vs-card">
          <div class="vs-card__title">Mật độ bảng</div>
          <div class="vs-density">
            <label
              v-for="opt in densityOptions"
              :key="opt.value"
              class="vs-density__item"
              :class="{ 'vs-density__item--active': prefs.density === opt.value }"
            >
              <input type="radio" name="density" :value="opt.value" :checked="prefs.density === opt.value" @change="prefs.setDensity(opt.value)" />
              <span class="vs-density__icon">{{ opt.icon }}</span>
              <div>
                <div class="vs-density__label">{{ opt.label }}</div>
                <div class="vs-density__desc">{{ opt.desc }}</div>
              </div>
            </label>
          </div>
        </section>

        <!-- Masking -->
        <section class="vs-card vs-card--sm">
          <div class="vs-card__title">Bảo mật dữ liệu</div>
          <label class="vs-masking">
            <div class="vs-masking__toggle" :class="{ 'vs-masking__toggle--on': prefs.dataMasking }" @click="prefs.toggleMasking()">
              <span class="vs-masking__knob" />
            </div>
            <div>
              <div class="vs-masking__label">Ẩn dữ liệu nhạy cảm</div>
              <div class="vs-masking__desc">Password, Cookie, Token hiện ••••••••</div>
            </div>
          </label>
        </section>
      </div>

      <!-- Row 2: Column visibility -->
      <section class="vs-card">
        <div class="vs-col-header">
          <div class="vs-card__title">Cột hiển thị</div>
          <span class="vs-col-count">{{ totalVisible }} / {{ ACCOUNT_COLUMNS.length }} cột</span>
          <button class="vs-reset-btn" @click="prefs.resetColumns()">Đặt lại mặc định</button>
        </div>

        <div class="vs-groups">
          <div v-for="group in columnGroups" :key="group.label" class="vs-group">
            <!-- Group header -->
            <div class="vs-group__header">
              <label class="vs-group__check">
                <input
                  type="checkbox"
                  :checked="isGroupAllChecked(group.keys)"
                  @change="toggleGroup(group.keys, ($event.target as HTMLInputElement).checked)"
                />
              </label>
              <span class="vs-group__label">{{ group.label }}</span>
            </div>
            <!-- Columns -->
            <div class="vs-group__cols">
              <label
                v-for="col in getGroupColumns(group.keys)"
                :key="col.key"
                class="vs-col-item"
                :class="{ 'vs-col-item--on': prefs.isColumnVisible(col.key) }"
              >
                <input
                  type="checkbox"
                  :checked="prefs.isColumnVisible(col.key)"
                  @change="prefs.toggleColumn(col.key)"
                />
                <span>{{ col.label }}</span>
              </label>
            </div>
          </div>
        </div>
      </section>
    </div>
  </div>
</template>

<style scoped>
.vs-page {
  display: flex;
  flex-direction: column;
  height: 100%;
}

.vs-header {
  height: var(--toolbar-height);
  background: var(--surface-elevated);
  border-bottom: 1px solid var(--border-default);
  display: flex;
  align-items: center;
  padding: 0 var(--space-4);
  flex-shrink: 0;
}

.vs-header h2 {
  font-size: var(--font-size-lg);
  font-weight: 700;
}

.vs-body {
  flex: 1;
  overflow-y: auto;
  padding: var(--space-5);
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

/* Cards */
.vs-card {
  background: var(--surface-elevated);
  border: 1px solid var(--border-default);
  border-radius: var(--radius-lg);
  padding: var(--space-4);
}

.vs-card__title {
  font-size: var(--font-size-sm);
  font-weight: 700;
  color: var(--text-secondary);
  text-transform: uppercase;
  letter-spacing: 0.06em;
  margin-bottom: var(--space-3);
}

/* Top row */
.vs-row-top {
  display: flex;
  gap: var(--space-4);
}

.vs-row-top .vs-card {
  flex: 1;
}

.vs-row-top .vs-card--sm {
  flex: 0 0 260px;
}

/* Density */
.vs-density {
  display: flex;
  gap: var(--space-2);
}

.vs-density__item {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: var(--space-2);
  padding: var(--space-3);
  border: 1px solid var(--border-default);
  border-radius: var(--radius-md);
  cursor: pointer;
  transition: border-color var(--transition-fast), background var(--transition-fast);
  text-align: center;
}

.vs-density__item input { display: none; }

.vs-density__item:hover {
  border-color: var(--border-strong);
  background: var(--surface-hover-subtle);
}

.vs-density__item--active {
  border-color: var(--brand-primary);
  background: var(--brand-primary-bg);
}

.vs-density__icon {
  font-size: 20px;
  color: var(--text-secondary);
}

.vs-density__item--active .vs-density__icon {
  color: var(--brand-primary);
}

.vs-density__label {
  font-size: var(--font-size-sm);
  font-weight: 600;
  color: var(--text-primary);
}

.vs-density__desc {
  font-size: var(--font-size-xs);
  color: var(--text-muted);
}

/* Masking toggle */
.vs-masking {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  cursor: pointer;
}

.vs-masking__toggle {
  flex-shrink: 0;
  width: 40px;
  height: 22px;
  border-radius: 11px;
  background: var(--border-strong);
  position: relative;
  transition: background var(--transition-fast);
}

.vs-masking__toggle--on {
  background: var(--brand-primary);
}

.vs-masking__knob {
  position: absolute;
  top: 3px;
  left: 3px;
  width: 16px;
  height: 16px;
  border-radius: 50%;
  background: white;
  transition: transform var(--transition-fast);
}

.vs-masking__toggle--on .vs-masking__knob {
  transform: translateX(18px);
}

.vs-masking__label {
  font-size: var(--font-size-sm);
  font-weight: 600;
  color: var(--text-primary);
}

.vs-masking__desc {
  font-size: var(--font-size-xs);
  color: var(--text-muted);
  margin-top: 2px;
}

/* Column visibility header */
.vs-col-header {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  margin-bottom: var(--space-4);
}

.vs-col-count {
  font-size: var(--font-size-xs);
  color: var(--brand-primary);
  background: var(--brand-primary-bg);
  padding: 2px 8px;
  border-radius: var(--radius-sm);
  font-weight: 600;
}

.vs-reset-btn {
  margin-left: auto;
  font-size: var(--font-size-xs);
  color: var(--text-link);
  padding: var(--space-1) var(--space-2);
  border-radius: var(--radius-sm);
  transition: background var(--transition-fast);
}

.vs-reset-btn:hover {
  background: var(--surface-hover);
}

/* Groups */
.vs-groups {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
}

.vs-group {
  border: 1px solid var(--border-default);
  border-radius: var(--radius-md);
  overflow: hidden;
}

.vs-group__header {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  padding: var(--space-2) var(--space-3);
  background: var(--surface-hover-subtle);
  border-bottom: 1px solid var(--border-default);
}

.vs-group__check input {
  accent-color: var(--brand-primary);
}

.vs-group__label {
  font-size: var(--font-size-xs);
  font-weight: 700;
  color: var(--text-secondary);
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

/* Columns inside group */
.vs-group__cols {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(130px, 1fr));
  padding: var(--space-2) var(--space-2);
  gap: 2px;
}

.vs-col-item {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  padding: 5px var(--space-2);
  border-radius: var(--radius-sm);
  font-size: var(--font-size-sm);
  cursor: pointer;
  color: var(--text-muted);
  transition: background var(--transition-fast), color var(--transition-fast);
}

.vs-col-item:hover {
  background: var(--surface-hover-subtle);
  color: var(--text-primary);
}

.vs-col-item--on {
  color: var(--text-primary);
}

.vs-col-item input {
  accent-color: var(--brand-primary);
  flex-shrink: 0;
}
</style>
