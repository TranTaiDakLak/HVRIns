<script setup lang="ts">
// FlowSettingsPage.vue — Khung cơ bản cho Flow Settings
// Flow list bên trái, form chỉnh flow bên phải

import { ref, onMounted } from 'vue'
import { getFlowService } from '@/services/client'
import type { Flow } from '@/services/contracts'

const flows = ref<Flow[]>([])
const selectedFlow = ref<Flow | null>(null)
const loading = ref(false)

onMounted(async () => {
  loading.value = true
  try {
    const service = await getFlowService()
    flows.value = await service.list()
    if (flows.value.length > 0) {
      selectedFlow.value = flows.value[0]
    }
  } finally {
    loading.value = false
  }
})

function selectFlow(flow: Flow) {
  selectedFlow.value = flow
}
</script>

<template>
  <div class="flow-page">
    <!-- Header -->
    <div class="flow-page__header">
      <h2>Flow Settings</h2>
    </div>

    <div class="flow-page__body">
      <!-- Flow list bên trái -->
      <div class="flow-page__list">
        <div class="flow-page__list-header">
          <span>Flows ({{ flows.length }})</span>
          <button class="flow-page__add-btn" disabled title="Phase 2">+ Thêm</button>
        </div>
        <div
          v-for="flow in flows"
          :key="flow.id"
          class="flow-page__list-item"
          :class="{ 'flow-page__list-item--active': selectedFlow?.id === flow.id }"
          @click="selectFlow(flow)"
        >
          <div class="flow-page__list-item-name">{{ flow.name }}</div>
          <div class="flow-page__list-item-desc">{{ flow.description }}</div>
        </div>
        <div v-if="flows.length === 0 && !loading" class="flow-page__empty">
          Chưa có flow nào
        </div>
      </div>

      <!-- Flow detail bên phải -->
      <div class="flow-page__detail" v-if="selectedFlow">
        <div class="flow-page__detail-header">
          <h3>{{ selectedFlow.name }}</h3>
          <span class="flow-page__engine-type">{{ selectedFlow.engineType }}</span>
        </div>
        <p class="flow-page__description">{{ selectedFlow.description }}</p>

        <!-- Steps table -->
        <div class="flow-page__steps-header">Steps ({{ selectedFlow.steps.length }})</div>
        <table class="flow-page__steps-table">
          <thead>
            <tr>
              <th>#</th>
              <th>Action</th>
              <th>Input</th>
              <th>Timeout</th>
              <th>Retry</th>
              <th>On</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="step in selectedFlow.steps" :key="step.stepNo">
              <td>{{ step.stepNo }}</td>
              <td>{{ step.actionKey }}</td>
              <td class="flow-page__step-input">{{ step.inputText }}</td>
              <td>{{ step.timeout }}s</td>
              <td>{{ step.retry }}</td>
              <td>
                <span :class="step.enabled ? 'flow-page__enabled' : 'flow-page__disabled'">
                  {{ step.enabled ? '✓' : '✕' }}
                </span>
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <div v-else class="flow-page__no-selection">
        Chọn một flow để xem chi tiết
      </div>
    </div>
  </div>
</template>

<style scoped>
.flow-page {
  display: flex;
  flex-direction: column;
  height: 100%;
}

.flow-page__header {
  height: var(--toolbar-height);
  background: var(--surface-elevated);
  border-bottom: 1px solid var(--border-default);
  display: flex;
  align-items: center;
  padding: 0 var(--space-4);
}

.flow-page__header h2 {
  font-size: var(--font-size-lg);
  font-weight: 700;
}

.flow-page__body {
  flex: 1;
  display: flex;
  overflow: hidden;
}

/* List bên trái */
.flow-page__list {
  width: 280px;
  border-right: 1px solid var(--border-default);
  background: var(--surface-elevated);
  overflow-y: auto;
  flex-shrink: 0;
}

.flow-page__list-header {
  padding: var(--space-3) var(--space-4);
  font-size: var(--font-size-sm);
  color: var(--text-muted);
  display: flex;
  justify-content: space-between;
  align-items: center;
  border-bottom: 1px solid var(--border-subtle);
}

.flow-page__add-btn {
  font-size: var(--font-size-xs);
  color: var(--brand-primary);
  padding: 2px 8px;
  border-radius: var(--radius-sm);
}

.flow-page__add-btn:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

.flow-page__list-item {
  padding: var(--space-3) var(--space-4);
  cursor: pointer;
  border-bottom: 1px solid var(--border-subtle);
  transition: background var(--transition-fast);
}

.flow-page__list-item:hover {
  background: var(--surface-hover-subtle);
}

.flow-page__list-item--active {
  background: var(--brand-primary-bg);
  border-left: 3px solid var(--brand-primary);
}

.flow-page__list-item-name {
  font-weight: 600;
  font-size: var(--font-size-base);
  color: var(--text-primary);
}

.flow-page__list-item-desc {
  font-size: var(--font-size-xs);
  color: var(--text-muted);
  margin-top: 2px;
}

/* Detail bên phải */
.flow-page__detail {
  flex: 1;
  padding: var(--space-4);
  overflow-y: auto;
}

.flow-page__detail-header {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  margin-bottom: var(--space-2);
}

.flow-page__detail-header h3 {
  font-size: var(--font-size-lg);
  font-weight: 700;
}

.flow-page__engine-type {
  font-size: var(--font-size-xs);
  background: var(--info-bg);
  color: var(--info-text);
  padding: 2px 8px;
  border-radius: var(--radius-full);
}

.flow-page__description {
  color: var(--text-secondary);
  font-size: var(--font-size-sm);
  margin-bottom: var(--space-4);
}

.flow-page__steps-header {
  font-size: var(--font-size-sm);
  font-weight: 600;
  color: var(--text-secondary);
  margin-bottom: var(--space-2);
}

.flow-page__steps-table {
  width: 100%;
  border-collapse: collapse;
  font-size: var(--font-size-sm);
}

.flow-page__steps-table th {
  background: var(--surface-elevated);
  padding: var(--space-2) var(--space-3);
  text-align: left;
  font-weight: 600;
  color: var(--text-muted);
  font-size: var(--font-size-xs);
  text-transform: uppercase;
  border-bottom: 1px solid var(--border-default);
}

.flow-page__steps-table td {
  padding: var(--space-2) var(--space-3);
  border-bottom: 1px solid var(--border-subtle);
  color: var(--text-primary);
}

.flow-page__step-input {
  font-family: var(--font-mono);
  font-size: var(--font-size-xs);
  max-width: 200px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.flow-page__enabled { color: var(--success-text); }
.flow-page__disabled { color: var(--danger-text); }

.flow-page__empty,
.flow-page__no-selection {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--text-muted);
}
</style>
