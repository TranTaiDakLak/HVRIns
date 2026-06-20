<script setup lang="ts">
// AuthSourcePage.vue — Tab top-level "Email/Phone" (Mail / Phone).
//
// Trước đây phần này nằm trong rp-shared-footer của InteractionSetupPage.
// Tách ra tab riêng để giao diện rộng hơn + truy cập nhanh từ sidebar.
//
// Cùng VerifyConfig với InteractionSetupPage — cả 2 đều load/save qua
// getInteractionService(). Backend tự reload realtime cho worker đang chạy.

import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { ArrowLeft, Copy, ClipboardPaste } from 'lucide-vue-next'
import { ROUTE_PATHS } from '@/constants/routes'
import { useAppStore } from '@/stores/app.store'
import { getInteractionService } from '@/services/client'
import type { VerifyConfig } from '@/types/interaction.types'
import { DEFAULT_VERIFY_CONFIG } from '@/types/interaction.types'
import { useAutoSave } from '@/composables/useAutoSave'
import AuthSourcePanel from '@/modules/auth-source/components/AuthSourcePanel.vue'

const appStore = useAppStore()
const router = useRouter()
const form = ref<VerifyConfig>({ ...DEFAULT_VERIFY_CONFIG })

// Fields liên quan đến mail/phone provider — dùng cho export/import
const EXPORT_KEYS: (keyof VerifyConfig)[] = [
  'mailProvider', 'mailList',
  'tempMailDomain', 'tempMailDomains', 'tempMailToken', 'tempMailTokens',
  'tempMailLolApiKey',
  'zeusXApiKey', 'zeusXAccountCode',
  'dvfbApiKey', 'dvfbAccountType',
  'store1sApiKey', 'store1sProductId',
  'mail30sApiKey', 'mail30sProductSlug',
  'muaMailApiKey', 'muaMailProductId',
  'unlimitMailApiKey', 'unlimitMailProductId',
  'sptMailApiKey', 'sptMailServiceCode',
  'emailAPIInfoApiKey', 'emailAPIInfoProductCode',
  'otpCheapApiKey', 'otpCheapServiceId',
  'shopGmail9999ApiKey', 'shopGmail9999Service',
  'rentGmailApiKey', 'rentGmailPlatform',
  'otpCodesSmsApiKey', 'otpCodesSmsServiceId',
  'wmemailApiKey', 'wmemailCommodity',
  'priyoEmailApiKey',
]

onMounted(async () => {
  try {
    const interactionSvc = await getInteractionService()
    const data = await interactionSvc.load()
    if (data) {
      form.value = { ...DEFAULT_VERIFY_CONFIG, ...data }
      if (!form.value.tempMailDomains) form.value.tempMailDomains = {}
      if (form.value.tempMailDomain && !form.value.tempMailDomains[form.value.mailProvider]) {
        form.value.tempMailDomains[form.value.mailProvider] = form.value.tempMailDomain
      }
      if (!form.value.tempMailTokens) form.value.tempMailTokens = {}
      if (form.value.tempMailToken && !form.value.tempMailTokens[form.value.mailProvider]) {
        form.value.tempMailTokens[form.value.mailProvider] = form.value.tempMailToken
      }
    }
  } catch {
    appStore.notify('error', 'Không tải được cấu hình Email/Phone')
  }
})

// Auto-save (debounce 500ms)
const { status: saveStatus, saveNow } = useAutoSave(form, async (value) => {
  const svc = await getInteractionService()
  await svc.save(value)
})

function goBack() {
  router.push(ROUTE_PATHS.INTERACTION_SETUP)
}

async function exportSettings() {
  const snapshot: Partial<VerifyConfig> = {}
  for (const key of EXPORT_KEYS) {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    ;(snapshot as any)[key] = (form.value as any)[key]
  }
  try {
    await navigator.clipboard.writeText(JSON.stringify(snapshot, null, 2))
    appStore.notify('success', 'Đã copy settings vào clipboard')
  } catch {
    appStore.notify('error', 'Không copy được vào clipboard')
  }
}

async function importSettings() {
  try {
    const text = await navigator.clipboard.readText()
    const parsed = JSON.parse(text)
    if (typeof parsed !== 'object' || parsed === null) throw new Error('invalid')
    for (const key of EXPORT_KEYS) {
      if (key in parsed) {
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        ;(form.value as any)[key] = parsed[key]
      }
    }
    await saveNow()
    appStore.notify('success', 'Import thành công')
  } catch {
    appStore.notify('error', 'Clipboard không chứa settings hợp lệ')
  }
}
</script>

<template>
  <div class="auth-source-page">
    <header class="auth-source-page__header">
      <button class="auth-source-page__back" @click="goBack" title="Quay lại Thiết lập chạy">
        <ArrowLeft :size="14" /> Thiết lập chạy
      </button>
      <h1 class="auth-source-page__title">Email/Phone</h1>
      <span class="auth-source-page__hint">
        Cấu hình mail / phone provider cho verify. Auto-save.
      </span>
      <div class="auth-source-page__actions">
        <button class="auth-source-page__action-btn" @click="exportSettings" title="Export settings ra clipboard">
          <Copy :size="13" /> Export
        </button>
        <button class="auth-source-page__action-btn" @click="importSettings" title="Import settings từ clipboard">
          <ClipboardPaste :size="13" /> Import
        </button>
      </div>
      <span class="auth-source-page__save-status" :data-status="saveStatus">
        <template v-if="saveStatus === 'saving'">&#x25D0; Đang lưu...</template>
        <template v-else-if="saveStatus === 'saved'">&#x2714; Đã lưu</template>
        <template v-else-if="saveStatus === 'error'">&#x26A0; Lỗi lưu</template>
        <template v-else>&#x2022; Tự động lưu</template>
      </span>
    </header>

    <div class="auth-source-page__body">
      <AuthSourcePanel :form="form" :save-now="saveNow" />
    </div>
  </div>
</template>

<style scoped>
.auth-source-page {
  display: flex;
  flex-direction: column;
  height: 100%;
  overflow: hidden;
}

.auth-source-page__header {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 18px;
  border-bottom: 1px solid var(--border-default);
  background: var(--surface-elevated);
}

.auth-source-page__back {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 5px 10px;
  background: var(--surface-base);
  border: 1px solid var(--border-default);
  border-radius: var(--radius-md);
  color: var(--text-secondary);
  font-size: var(--font-size-xs);
  cursor: pointer;
  transition: background var(--transition-fast), color var(--transition-fast);
}
.auth-source-page__back:hover {
  background: var(--surface-hover);
  color: var(--text-primary);
}

.auth-source-page__title {
  font-size: 16px;
  font-weight: 700;
  color: var(--text-primary);
  margin: 0;
}

.auth-source-page__hint {
  font-size: var(--font-size-xs);
  color: var(--text-muted);
  margin-left: 6px;
}

.auth-source-page__actions {
  display: flex;
  gap: 6px;
}

.auth-source-page__action-btn {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 5px 10px;
  background: var(--surface-base);
  border: 1px solid var(--border-default);
  border-radius: var(--radius-md);
  color: var(--text-secondary);
  font-size: var(--font-size-xs);
  cursor: pointer;
  transition: background var(--transition-fast), color var(--transition-fast);
}
.auth-source-page__action-btn:hover {
  background: var(--surface-hover);
  color: var(--text-primary);
}

.auth-source-page__save-status {
  margin-left: auto;
  font-size: var(--font-size-xs);
  color: var(--text-muted);
}
.auth-source-page__save-status[data-status="saved"] { color: #4caf50; }
.auth-source-page__save-status[data-status="saving"] { color: var(--brand-primary); }
.auth-source-page__save-status[data-status="error"] { color: #f44336; }

.auth-source-page__body {
  flex: 1;
  overflow-y: auto;
  padding: 18px;
}
</style>
