<script setup lang="ts">
// LegacyImportWizard.vue — Wizard 4 bước import cấu hình từ tool cũ (WeBM)
// Bước 1: Nhập JSON  | Bước 2: Xem report  | Bước 3: Xác nhận  | Bước 4: Xong

import { ref, computed } from 'vue'
import { useRouter } from 'vue-router'
import { getLegacyImportService } from '@/bridge/client'
import type { LegacyMappedField, LegacyMappingReport } from '@/bridge/contracts'
import { ROUTE_NAMES } from '@/constants/routes'

const router = useRouter()

// ── State ─────────────────────────────────────────────────────────────────────
type Step = 1 | 2 | 3 | 4

const step = ref<Step>(1)
const generalJSON = ref('')
const interactionJSON = ref('')
const loading = ref(false)
const error = ref('')
const report = ref<LegacyMappingReport | null>(null)
const confirmed = ref(false)
const importDone = ref(false)
const importError = ref('')

// ── Bước 1 → 2: Parse ─────────────────────────────────────────────────────────
async function handleParse() {
  if (!generalJSON.value.trim() && !interactionJSON.value.trim()) {
    error.value = 'Vui lòng dán ít nhất một trong hai file JSON.'
    return
  }
  error.value = ''
  loading.value = true
  try {
    const svc = await getLegacyImportService()
    const result = await svc.parse(generalJSON.value, interactionJSON.value)
    if (result.error) {
      error.value = result.error
      return
    }
    report.value = result.report
    step.value = 2
  } catch (e: unknown) {
    error.value = String(e)
  } finally {
    loading.value = false
  }
}

// ── Bước 2 → 3 ────────────────────────────────────────────────────────────────
function goConfirm() {
  confirmed.value = false
  step.value = 3
}

// ── Bước 3 → 4: Apply ─────────────────────────────────────────────────────────
async function handleImport() {
  if (!confirmed.value) return
  importError.value = ''
  loading.value = true
  try {
    const svc = await getLegacyImportService()
    const res = await svc.apply(generalJSON.value, interactionJSON.value)
    if (res !== 'OK') {
      importError.value = res
    } else {
      importDone.value = true
    }
    step.value = 4
  } catch (e: unknown) {
    importError.value = String(e)
    step.value = 4
  } finally {
    loading.value = false
  }
}

function goBack() {
  if (step.value === 2) step.value = 1
  else if (step.value === 3) step.value = 2
}

function goSettings() {
  router.push({ name: ROUTE_NAMES.GENERAL_SETTINGS })
}

// ── Summary counts ─────────────────────────────────────────────────────────────
const totalFields = computed(() => {
  if (!report.value) return 0
  const r = report.value
  return r.mappedOk.length + r.needsConfirm.length + r.sensitive.length + r.unsupported.length
})

function statusLabel(status: LegacyMappedField['status']): string {
  switch (status) {
    case 'ok':          return 'OK'
    case 'confirm':     return 'Xác nhận'
    case 'sensitive':   return 'Nhạy cảm'
    case 'unsupported': return 'Không hỗ trợ'
  }
}
function statusClass(status: LegacyMappedField['status']): string {
  switch (status) {
    case 'ok':          return 'badge--ok'
    case 'confirm':     return 'badge--confirm'
    case 'sensitive':   return 'badge--sensitive'
    case 'unsupported': return 'badge--unsupported'
  }
}
</script>

<template>
  <div class="wizard-page">
    <!-- Header -->
    <div class="wizard-header">
      <h1 class="wizard-title">Import cấu hình từ WeBM</h1>
      <p class="wizard-subtitle">Chuyển đổi general.json + interaction.json sang cấu hình mới một cách an toàn.</p>

      <!-- Step indicator -->
      <div class="wizard-steps">
        <div v-for="n in [1,2,3,4]" :key="n" class="wizard-step" :class="{ active: step === n, done: step > n }">
          <span class="wizard-step__num">{{ step > n ? '✓' : n }}</span>
          <span class="wizard-step__label">
            {{ ['Nhập JSON', 'Xem report', 'Xác nhận', 'Hoàn tất'][n - 1] }}
          </span>
        </div>
      </div>
    </div>

    <!-- ── Bước 1: Nhập JSON ──────────────────────────────────────────────── -->
    <div v-if="step === 1" class="wizard-body">
      <p class="wizard-hint">
        Mở thư mục cài đặt tool cũ, copy nội dung <code>general.json</code> và <code>interaction.json</code> vào bên dưới.
        Bạn có thể để trống một trong hai nếu chỉ cần import một phần.
      </p>

      <div class="input-pair">
        <div class="input-block">
          <label class="input-label">general.json</label>
          <textarea
            v-model="generalJSON"
            class="json-textarea"
            placeholder='{ "General": { "ThreadRequest": 5, ... } }'
            spellcheck="false"
          />
        </div>
        <div class="input-block">
          <label class="input-label">interaction.json</label>
          <textarea
            v-model="interactionJSON"
            class="json-textarea"
            placeholder='{ "VerifyEnabled": true, "MailProvider": "store1s", ... }'
            spellcheck="false"
          />
        </div>
      </div>

      <div v-if="error" class="wizard-error">{{ error }}</div>

      <div class="wizard-actions">
        <button class="btn btn--ghost" @click="router.back()">Hủy</button>
        <button class="btn btn--primary" :disabled="loading" @click="handleParse">
          {{ loading ? 'Đang phân tích...' : 'Phân tích →' }}
        </button>
      </div>
    </div>

    <!-- ── Bước 2: Xem report ─────────────────────────────────────────────── -->
    <div v-else-if="step === 2 && report" class="wizard-body">
      <div class="report-summary">
        <div class="summary-chip summary-chip--ok">{{ report.mappedOk.length }} OK</div>
        <div class="summary-chip summary-chip--confirm">{{ report.needsConfirm.length }} Xác nhận</div>
        <div class="summary-chip summary-chip--sensitive">{{ report.sensitive.length }} Nhạy cảm</div>
        <div class="summary-chip summary-chip--unsupported">{{ report.unsupported.length }} Không hỗ trợ</div>
        <div class="summary-chip summary-chip--total">{{ totalFields }} tổng</div>
      </div>

      <!-- Parse errors -->
      <div v-if="report.parseErrors.length" class="section-block section-block--error">
        <h3 class="section-title">Lỗi parse</h3>
        <ul class="field-list">
          <li v-for="e in report.parseErrors" :key="e" class="field-row field-row--error">{{ e }}</li>
        </ul>
      </div>

      <!-- Fields OK -->
      <div v-if="report.mappedOk.length" class="section-block">
        <h3 class="section-title">Fields được map tự động ({{ report.mappedOk.length }})</h3>
        <div class="field-table">
          <div class="field-table__head">
            <span>Tên cũ</span><span>Path mới</span><span>Giá trị</span><span>Trạng thái</span>
          </div>
          <div v-for="f in report.mappedOk" :key="f.legacyKey" class="field-table__row">
            <span class="field-key">{{ f.legacyKey }}</span>
            <span class="field-path">{{ f.newPath }}</span>
            <span class="field-value">{{ f.displayValue }}</span>
            <span :class="['badge', statusClass(f.status)]">{{ statusLabel(f.status) }}</span>
          </div>
        </div>
      </div>

      <!-- Fields cần xác nhận -->
      <div v-if="report.needsConfirm.length" class="section-block section-block--warn">
        <h3 class="section-title">Cần xác nhận ({{ report.needsConfirm.length }})</h3>
        <div class="field-table">
          <div class="field-table__head">
            <span>Tên cũ</span><span>Path mới</span><span>Giá trị</span><span>Ghi chú</span>
          </div>
          <div v-for="f in report.needsConfirm" :key="f.legacyKey" class="field-table__row">
            <span class="field-key">{{ f.legacyKey }}</span>
            <span class="field-path">{{ f.newPath }}</span>
            <span class="field-value">{{ f.displayValue }}</span>
            <span class="field-note">{{ f.note }}</span>
          </div>
        </div>
      </div>

      <!-- Sensitive fields -->
      <div v-if="report.sensitive.length" class="section-block section-block--sensitive">
        <h3 class="section-title">Fields nhạy cảm — sẽ được import ({{ report.sensitive.length }})</h3>
        <p class="section-desc">Giá trị thực sẽ được ghi vào cấu hình nhưng không hiển thị ở đây. Nên đổi mật khẩu và API key sau khi import.</p>
        <div class="field-table">
          <div class="field-table__head">
            <span>Tên cũ</span><span>Path mới</span><span>Ghi chú</span>
          </div>
          <div v-for="f in report.sensitive" :key="f.legacyKey" class="field-table__row">
            <span class="field-key">{{ f.legacyKey }}</span>
            <span class="field-path">{{ f.newPath }}</span>
            <span class="field-note">{{ f.note }}</span>
          </div>
        </div>
      </div>

      <!-- Unsupported -->
      <div v-if="report.unsupported.length" class="section-block section-block--unsupported">
        <h3 class="section-title">Không hỗ trợ — sẽ bỏ qua ({{ report.unsupported.length }})</h3>
        <div class="field-table">
          <div class="field-table__head">
            <span>Tên cũ</span><span>Giá trị</span><span>Ghi chú</span>
          </div>
          <div v-for="f in report.unsupported" :key="f.legacyKey" class="field-table__row">
            <span class="field-key">{{ f.legacyKey }}</span>
            <span class="field-value">{{ f.displayValue }}</span>
            <span class="field-note">{{ f.note }}</span>
          </div>
        </div>
      </div>

      <div class="wizard-actions">
        <button class="btn btn--ghost" @click="goBack">← Quay lại</button>
        <button class="btn btn--primary" @click="goConfirm">Tiếp theo →</button>
      </div>
    </div>

    <!-- ── Bước 3: Xác nhận ───────────────────────────────────────────────── -->
    <div v-else-if="step === 3" class="wizard-body">
      <div class="confirm-card">
        <h2 class="confirm-title">Xác nhận import</h2>
        <p class="confirm-desc">
          Hành động này sẽ ghi đè cấu hình hiện tại bằng dữ liệu từ file cũ.
          Cấu hình cũ sẽ bị mất. Nếu bạn muốn giữ bản backup, hãy export trước.
        </p>

        <ul class="confirm-list" v-if="report">
          <li><strong>{{ report.mappedOk.length }}</strong> fields được import tự động</li>
          <li v-if="report.needsConfirm.length"><strong>{{ report.needsConfirm.length }}</strong> fields cần kiểm tra lại sau</li>
          <li v-if="report.sensitive.length"><strong>{{ report.sensitive.length }}</strong> fields nhạy cảm được import (khuyến nghị đổi mật khẩu/API key sau)</li>
          <li v-if="report.unsupported.length"><strong>{{ report.unsupported.length }}</strong> fields không hỗ trợ sẽ bị bỏ qua</li>
        </ul>

        <label class="confirm-check">
          <input type="checkbox" v-model="confirmed" />
          <span>Tôi đã đọc report và xác nhận muốn import</span>
        </label>

        <div v-if="importError" class="wizard-error">{{ importError }}</div>
      </div>

      <div class="wizard-actions">
        <button class="btn btn--ghost" @click="goBack">← Xem lại</button>
        <button class="btn btn--danger" :disabled="!confirmed || loading" @click="handleImport">
          {{ loading ? 'Đang import...' : 'Xác nhận import' }}
        </button>
      </div>
    </div>

    <!-- ── Bước 4: Hoàn tất ───────────────────────────────────────────────── -->
    <div v-else-if="step === 4" class="wizard-body wizard-body--done">
      <div v-if="importDone" class="done-card done-card--success">
        <div class="done-icon">✓</div>
        <h2 class="done-title">Import thành công</h2>
        <p class="done-desc">Cấu hình đã được cập nhật. Các fields cần xác nhận (đường dẫn, v.v.) hãy kiểm tra lại trong Cài đặt chung.</p>
        <button class="btn btn--primary" @click="goSettings">Đến Cài đặt chung</button>
      </div>
      <div v-else class="done-card done-card--error">
        <div class="done-icon">✗</div>
        <h2 class="done-title">Import thất bại</h2>
        <p class="done-desc">{{ importError || 'Lỗi không xác định.' }}</p>
        <button class="btn btn--ghost" @click="step = 1">Thử lại</button>
      </div>
    </div>
  </div>
</template>

<style scoped>
/* ── Page layout ─────────────────────────────────────────────────────────── */
.wizard-page {
  max-width: 900px;
  margin: 0 auto;
  padding: 24px 20px 48px;
  color: var(--text-primary, #e0e0e0);
}

.wizard-header {
  margin-bottom: 28px;
}

.wizard-title {
  font-size: 20px;
  font-weight: 700;
  margin: 0 0 4px;
}

.wizard-subtitle {
  font-size: 13px;
  color: var(--text-secondary, #999);
  margin: 0 0 20px;
}

/* ── Step indicator ──────────────────────────────────────────────────────── */
.wizard-steps {
  display: flex;
  gap: 0;
  border-bottom: 1px solid var(--border-color, #333);
  padding-bottom: 16px;
}

.wizard-step {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 4px 16px 4px 0;
  opacity: 0.4;
  font-size: 13px;
}

.wizard-step.active {
  opacity: 1;
  color: var(--accent, #4fc3f7);
}

.wizard-step.done {
  opacity: 0.7;
  color: var(--success-color, #66bb6a);
}

.wizard-step__num {
  width: 22px;
  height: 22px;
  border-radius: 50%;
  border: 1.5px solid currentColor;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 11px;
  font-weight: 700;
}

/* ── Body ────────────────────────────────────────────────────────────────── */
.wizard-body {
  margin-top: 24px;
}

.wizard-hint {
  font-size: 13px;
  color: var(--text-secondary, #999);
  margin-bottom: 16px;
  line-height: 1.6;
}

.wizard-hint code {
  background: var(--surface-hover-strong);
  padding: 1px 5px;
  border-radius: 3px;
  font-family: monospace;
}

/* ── Step 1: JSON inputs ─────────────────────────────────────────────────── */
.input-pair {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 16px;
  margin-bottom: 16px;
}

.input-block {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.input-label {
  font-size: 12px;
  font-weight: 600;
  color: var(--text-secondary, #999);
  font-family: monospace;
}

.json-textarea {
  height: 280px;
  background: var(--bg-secondary, #1e1e1e);
  border: 1px solid var(--border-color, #333);
  border-radius: 6px;
  color: var(--text-primary, #e0e0e0);
  font-family: monospace;
  font-size: 12px;
  padding: 10px;
  resize: vertical;
  outline: none;
  line-height: 1.5;
}

.json-textarea:focus {
  border-color: var(--accent, #4fc3f7);
}

/* ── Report summary chips ────────────────────────────────────────────────── */
.report-summary {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
  margin-bottom: 20px;
}

.summary-chip {
  padding: 4px 10px;
  border-radius: 12px;
  font-size: 12px;
  font-weight: 600;
  border: 1px solid currentColor;
}

.summary-chip--ok          { color: #66bb6a; }
.summary-chip--confirm     { color: #ffa726; }
.summary-chip--sensitive   { color: #ab47bc; }
.summary-chip--unsupported { color: #ef5350; }
.summary-chip--total       { color: var(--text-secondary, #999); }

/* ── Section blocks ──────────────────────────────────────────────────────── */
.section-block {
  margin-bottom: 20px;
  border: 1px solid var(--border-color, #333);
  border-radius: 8px;
  overflow: hidden;
}

.section-block--warn       { border-color: rgba(255,167,38,0.4); }
.section-block--sensitive  { border-color: rgba(171,71,188,0.4); }
.section-block--unsupported{ border-color: rgba(239,83,80,0.4); }
.section-block--error      { border-color: rgba(239,83,80,0.6); }

.section-title {
  font-size: 13px;
  font-weight: 600;
  padding: 10px 14px 8px;
  background: var(--surface-hover-subtle);
  margin: 0;
  border-bottom: 1px solid var(--border-color, #333);
}

.section-desc {
  font-size: 12px;
  color: var(--text-secondary, #999);
  padding: 8px 14px 0;
  margin: 0;
}

/* ── Field table ─────────────────────────────────────────────────────────── */
.field-table {
  font-size: 12px;
}

.field-table__head {
  display: grid;
  grid-template-columns: 160px 220px 1fr auto;
  padding: 6px 14px;
  background: var(--surface-hover-subtle);
  color: var(--text-secondary, #999);
  font-weight: 600;
  border-bottom: 1px solid var(--border-color, #333);
  gap: 8px;
}

.field-table__row {
  display: grid;
  grid-template-columns: 160px 220px 1fr auto;
  padding: 6px 14px;
  gap: 8px;
  border-bottom: 1px solid var(--surface-hover-subtle);
  align-items: center;
}

.field-table__row:last-child {
  border-bottom: none;
}

.field-key   { font-family: monospace; color: var(--accent, #4fc3f7); }
.field-path  { font-family: monospace; color: var(--text-secondary, #999); font-size: 11px; }
.field-value { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.field-note  { font-size: 11px; color: var(--text-secondary, #999); }

/* ── Sensitive table: 3 columns ─────────────────────────────────────────── */
.section-block--sensitive .field-table__head,
.section-block--sensitive .field-table__row,
.section-block--unsupported .field-table__head,
.section-block--unsupported .field-table__row {
  grid-template-columns: 160px 220px 1fr;
}

/* ── Badges ──────────────────────────────────────────────────────────────── */
.badge {
  padding: 2px 7px;
  border-radius: 10px;
  font-size: 11px;
  font-weight: 600;
  white-space: nowrap;
}

.badge--ok          { background: rgba(102,187,106,0.15); color: #66bb6a; }
.badge--confirm     { background: rgba(255,167,38,0.15);  color: #ffa726; }
.badge--sensitive   { background: rgba(171,71,188,0.15);  color: #ab47bc; }
.badge--unsupported { background: rgba(239,83,80,0.15);   color: #ef5350; }

/* ── Confirm step ────────────────────────────────────────────────────────── */
.confirm-card {
  background: var(--bg-secondary, #1e1e1e);
  border: 1px solid var(--border-color, #333);
  border-radius: 10px;
  padding: 24px;
  max-width: 560px;
}

.confirm-title {
  font-size: 16px;
  font-weight: 700;
  margin: 0 0 10px;
}

.confirm-desc {
  font-size: 13px;
  color: var(--text-secondary, #999);
  margin: 0 0 16px;
  line-height: 1.6;
}

.confirm-list {
  font-size: 13px;
  margin: 0 0 20px;
  padding-left: 20px;
  line-height: 2;
  color: var(--text-primary, #e0e0e0);
}

.confirm-check {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
  cursor: pointer;
  user-select: none;
}

/* ── Done step ───────────────────────────────────────────────────────────── */
.wizard-body--done {
  display: flex;
  justify-content: center;
  padding-top: 32px;
}

.done-card {
  text-align: center;
  padding: 40px 48px;
  border-radius: 12px;
  border: 1px solid var(--border-color, #333);
  background: var(--bg-secondary, #1e1e1e);
  max-width: 440px;
}

.done-card--success { border-color: rgba(102,187,106,0.5); }
.done-card--error   { border-color: rgba(239,83,80,0.5); }

.done-icon {
  font-size: 40px;
  margin-bottom: 12px;
}

.done-card--success .done-icon { color: #66bb6a; }
.done-card--error   .done-icon { color: #ef5350; }

.done-title {
  font-size: 18px;
  font-weight: 700;
  margin: 0 0 10px;
}

.done-desc {
  font-size: 13px;
  color: var(--text-secondary, #999);
  margin: 0 0 24px;
  line-height: 1.6;
}

/* ── Common ──────────────────────────────────────────────────────────────── */
.wizard-error {
  background: rgba(239,83,80,0.12);
  border: 1px solid rgba(239,83,80,0.4);
  border-radius: 6px;
  padding: 10px 14px;
  font-size: 13px;
  color: #ef5350;
  margin: 12px 0;
}

.wizard-actions {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
  margin-top: 24px;
  padding-top: 16px;
  border-top: 1px solid var(--border-color, #333);
}

.btn {
  padding: 8px 18px;
  border-radius: 6px;
  border: none;
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
  transition: opacity 0.15s;
}

.btn:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

.btn--primary {
  background: var(--accent, #4fc3f7);
  color: #000;
}

.btn--danger {
  background: #d32f2f;
  color: #fff;
}

.btn--ghost {
  background: transparent;
  border: 1px solid var(--border-color, #444);
  color: var(--text-primary, #e0e0e0);
}
</style>
