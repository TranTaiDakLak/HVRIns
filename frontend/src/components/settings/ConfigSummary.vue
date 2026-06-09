<script setup lang="ts">
// ConfigSummary.vue — Card tóm tắt cấu hình chạy hiện tại
// 6 domain: Chế độ · Nguồn TK · Verify · Register · IP/Proxy · Kết quả
// Đọc trong 3 giây: chips + toggle expand.

import { ref, computed } from 'vue'
import { FolderOpen, Key, ChevronUp, ChevronDown, AlertTriangle } from 'lucide-vue-next'
import type { VerifyConfig } from '../../types/interaction.types'
import { VERIFY_MAIL_PROVIDERS } from '../../types/interaction.types'
import { IP_PROVIDERS } from '../../types/settings.types'

const props = defineProps<{
  form: VerifyConfig
  // Tùy chọn — được truyền từ InteractionSetupPage (accountForm)
  accountSource?: 'folder' | 'api'
  accountSourcePath?: string
  cloneHvUsername?: string
  ipProvider?: string
  checkIpBeforeRun?: boolean
  // Khi dùng trong drawer, luôn mở rộng và ẩn nút toggle
  alwaysExpanded?: boolean
}>()

const collapsed = ref(true)

// ─── Helpers ─────────────────────────────────────────────────────────────────

/** Rút gọn đường dẫn: chỉ hiện 2 segment cuối */
function shortPath(p: string): string {
  if (!p) return ''
  const parts = p.replace(/\\/g, '/').split('/').filter(Boolean)
  if (parts.length <= 2) return p
  return '.../' + parts.slice(-2).join('/')
}

// ─── Computed ─────────────────────────────────────────────────────────────────

const mailProviderLabel = computed(() => {
  const p = VERIFY_MAIL_PROVIDERS.find(m => m.value === props.form.mailProvider)
  return p?.label ?? (props.form.mailProvider || '— tự nhập —')
})

const ipLabel = computed(() => {
  if (!props.ipProvider || props.ipProvider === 'none') return null
  const p = IP_PROVIDERS.find(i => i.value === props.ipProvider)
  return p?.label ?? props.ipProvider
})

const mode = computed(() => {
  if (props.form.createEnabled && props.form.verifyEnabled) return 'Register + Verify'
  if (props.form.createEnabled) return 'Register'
  if (props.form.verifyEnabled) return 'Verify'
  return 'Tắt'
})

const modeColorClass = computed(() => {
  if (props.form.createEnabled && props.form.verifyEnabled) return 'chip--both'
  if (props.form.createEnabled) return 'chip--register'
  if (props.form.verifyEnabled) return 'chip--verify'
  return 'chip--off'
})

const sourceChipText = computed(() => {
  if (!props.accountSource) return null
  if (props.accountSource === 'api') return `API:${props.cloneHvUsername || '—'}`
  if (props.accountSourcePath) return shortPath(props.accountSourcePath)
  return 'chưa chọn'
})

const sourceRowText = computed(() => {
  if (!props.accountSource) return null
  if (props.accountSource === 'api') return `CloneHV API — ${props.cloneHvUsername || '(chưa nhập username)'}`
  return props.accountSourcePath ? shortPath(props.accountSourcePath) : '(chưa chọn thư mục)'
})
</script>

<template>
  <div class="cfg-summary" :class="{ 'cfg-summary--collapsed': collapsed }">

    <!-- Toggle row: mode badge + chips -->
    <button
      type="button"
      class="cfg-summary__toggle"
      :class="{ 'cfg-summary__toggle--static': alwaysExpanded }"
      @click="!alwaysExpanded && (collapsed = !collapsed)"
    >
      <span :class="['chip', 'chip--mode', modeColorClass]">{{ mode }}</span>
      <span class="cfg-summary__chips">
        <!-- Nguồn TK chip: chỉ hiện loại nguồn, không hiện path dài -->
        <span v-if="accountSource" class="chip chip--source"
          :title="accountSource === 'folder' ? accountSourcePath : cloneHvUsername">
          <Key v-if="accountSource === 'api'" :size="11" /> <FolderOpen v-else :size="11" />
          {{ accountSource === 'api' ? 'CloneHV' : 'Thư mục' }}
        </span>
        <!-- Mail/verify chip: chỉ khi verify BẬT -->
        <span v-if="form.verifyEnabled" class="chip chip--mail">{{ mailProviderLabel }}</span>
        <!-- IP chip: chỉ khi có provider thật -->
        <span v-if="ipLabel" class="chip chip--ip">
          {{ ipLabel }}{{ checkIpBeforeRun ? ' ✓' : '' }}
        </span>
        <!-- Các chip chi tiết (L/D, register-sm, path) bỏ khỏi compact row
             — xem trong panel mở rộng bên dưới -->
      </span>
      <span v-if="!alwaysExpanded" class="cfg-summary__caret">
        <ChevronDown v-if="collapsed" :size="14" /><ChevronUp v-else :size="14" />
      </span>
    </button>

    <!-- Expanded detail: 6 domain rows -->
    <div v-if="alwaysExpanded || !collapsed" class="cfg-summary__detail">

      <!-- 1. Chế độ chạy -->
      <div class="cfg-row">
        <span class="cfg-row__label">Chế độ</span>
        <span :class="['cfg-row__val', 'chip', 'chip--mode', modeColorClass]">{{ mode }}</span>
      </div>

      <!-- 2. Nguồn tài khoản -->
      <div v-if="accountSource" class="cfg-row">
        <span class="cfg-row__label">Nguồn TK</span>
        <span class="cfg-row__val">
          <span v-if="accountSource === 'folder'"><FolderOpen :size="13" /> {{ sourceRowText }}</span>
          <span v-else><Key :size="13" /> {{ sourceRowText }}</span>
        </span>
      </div>

      <!-- 3. Verify -->
      <div class="cfg-row">
        <span class="cfg-row__label">Verify</span>
        <span v-if="form.verifyEnabled" class="cfg-row__val">
          <span class="val--on">BẬT</span>
          — {{ mailProviderLabel }}
          <span v-if="form.checkLiveDieEnabled"> · L/D {{ form.timeDelayCheck }}s</span>
          <span v-if="form.sendAgainCode"> · gửi lại ✓</span>
        </span>
        <span v-else class="cfg-row__val val--off">TẮT</span>
      </div>

      <!-- 4. Register -->
      <div class="cfg-row">
        <span class="cfg-row__label">Register</span>
        <span v-if="form.createEnabled" class="cfg-row__val">
          <span class="val--on">BẬT</span>
          — {{ form.createType || 'spam' }}
          <span v-if="form.createOutputPath"> · {{ shortPath(form.createOutputPath) }}</span>
        </span>
        <span v-else class="cfg-row__val val--off">TẮT</span>
      </div>

      <!-- 5. IP/Proxy -->
      <div class="cfg-row">
        <span class="cfg-row__label">IP/Proxy</span>
        <span v-if="ipLabel" class="cfg-row__val">
          {{ ipLabel }}<span v-if="checkIpBeforeRun"> · check trước ✓</span>
        </span>
        <span v-else class="cfg-row__val val--off">Không đổi IP</span>
      </div>

      <!-- 6. Kết quả verify -->
      <div class="cfg-row">
        <span class="cfg-row__label">Kết quả</span>
        <span v-if="form.outputPath" class="cfg-row__val cfg-row__val--path" :title="form.outputPath">
          {{ shortPath(form.outputPath) }} → Live.txt / Die.txt
        </span>
        <span v-else class="cfg-row__val val--warn"><AlertTriangle :size="13" /> Chưa chọn thư mục lưu</span>
      </div>

    </div>
  </div>
</template>

<style scoped>
.cfg-summary {
  background: var(--surface-elevated, #1a1a1a);
  border-bottom: 1px solid var(--border-default, #2a2a2a);
  font-size: 12px;
}

.cfg-summary__toggle {
  width: 100%;
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 16px;
  background: none;
  border: none;
  cursor: pointer;
  color: var(--text-primary, #e0e0e0);
  text-align: left;
}

.cfg-summary__chips {
  display: flex;
  gap: 5px;
  flex: 1;
  flex-wrap: wrap;
  align-items: center;
}

.cfg-summary__caret {
  font-size: 10px;
  color: var(--text-muted, #888);
  flex-shrink: 0;
}

.cfg-summary__toggle--static {
  cursor: default;
}

/* ─── Chips ──────────────────────────────────────────────────────────────── */
.chip {
  padding: 1px 7px;
  border-radius: 10px;
  font-size: 11px;
  font-weight: 600;
  white-space: nowrap;
}

/* Mode chips */
.chip--mode   { font-size: 12px; padding: 2px 9px; }
.chip--both   { background: rgba(171,71,188,0.15); color: #ce93d8; }
.chip--verify { background: rgba(79,195,247,0.12); color: var(--accent, #4fc3f7); }
.chip--register { background: rgba(102,187,106,0.12); color: #66bb6a; }
.chip--off    { background: var(--surface-sunken, #111); color: var(--text-muted, #888); font-weight: 400; }

/* Data chips */
.chip--source   { background: var(--surface-hover); color: var(--text-secondary, #bbb); }
.chip--mail     { background: rgba(79,195,247,0.1); color: var(--accent, #4fc3f7); max-width: 140px; overflow: hidden; text-overflow: ellipsis; }
.chip--ip       { background: rgba(255,183,77,0.12); color: #ffb74d; }
.chip--check    { background: rgba(102,187,106,0.12); color: #66bb6a; }
.chip--register-sm { background: rgba(102,187,106,0.1); color: #a5d6a7; }
.chip--path     { background: var(--surface-hover-subtle); color: var(--text-muted, #888); font-family: var(--font-mono, monospace); font-weight: 400; max-width: 160px; overflow: hidden; text-overflow: ellipsis; }

/* ─── Expanded detail ────────────────────────────────────────────────────── */
.cfg-summary__detail {
  padding: 8px 16px 10px;
  display: flex;
  flex-wrap: wrap;
  gap: 4px 28px;
  border-top: 1px solid var(--border-default, #2a2a2a);
}

.cfg-row {
  display: flex;
  gap: 6px;
  align-items: baseline;
  min-width: 220px;
}

.cfg-row__label {
  color: var(--text-muted, #888);
  font-size: 11px;
  white-space: nowrap;
  min-width: 72px;
}

.cfg-row__val {
  font-size: 12px;
  color: var(--text-primary, #e0e0e0);
}

.cfg-row__val--path {
  font-family: var(--font-mono, monospace);
  font-size: 11px;
  color: var(--text-secondary, #bbb);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  max-width: 260px;
}

.val--on   { color: #66bb6a; font-weight: 600; }
.val--off  { color: var(--text-muted, #888); }
.val--warn { color: #ffb74d; }
</style>
