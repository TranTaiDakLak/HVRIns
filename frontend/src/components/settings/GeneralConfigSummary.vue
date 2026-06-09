<script setup lang="ts">
// GeneralConfigSummary.vue — Compact summary card for GeneralSettings page
// Shows key config at a glance: threads, login platform, IP provider, captcha.

import { ref, computed } from 'vue'
import { Check, AlertTriangle } from 'lucide-vue-next'
import type { GeneralConfig, IpConfig } from '../../types/settings.types'
import { IP_PROVIDERS, CAPTCHA_PROVIDERS } from '../../types/settings.types'

const props = defineProps<{
  general: GeneralConfig
  ip: IpConfig
}>()

const collapsed = ref(false)

const ipLabel = computed(() => {
  const p = IP_PROVIDERS.find(i => i.value === props.general.ipProvider)
  return p?.label ?? props.general.ipProvider
})

const captchaLabel = computed(() => {
  const p = CAPTCHA_PROVIDERS.find(c => c.value === props.general.captchaProvider)
  return p?.label ?? props.general.captchaProvider
})

const proxyCount = computed(() =>
  props.general.ipProvider === 'proxy'
    ? props.ip.proxyList.split('\n').filter(l => l.trim()).length
    : 0,
)

const platformLabel = computed(() => {
  switch (props.general.loginPlatform) {
    case 'facebook': return 'Facebook'
    case 'instagram': return 'Instagram'
    case 'bm': return 'Business Manager'
    default: return props.general.loginPlatform
  }
})
</script>

<template>
  <div class="gcs" :class="{ 'gcs--collapsed': collapsed }">
    <button type="button" class="gcs__toggle" @click="collapsed = !collapsed">
      <span class="gcs__title">Tổng quan</span>
      <span class="gcs__chips">
        <span class="chip chip--threads">{{ general.threadRequest }} luồng</span>
        <span class="chip chip--platform">{{ platformLabel }}</span>
        <span class="chip chip--ip">{{ ipLabel }}</span>
        <span v-if="proxyCount > 0" class="chip chip--count">{{ proxyCount }} proxy</span>
        <span class="chip chip--captcha">{{ captchaLabel }}</span>
      </span>
      <span class="gcs__caret">{{ collapsed ? '&#x25BC;' : '&#x25B2;' }}</span>
    </button>

    <div v-if="!collapsed" class="gcs__detail">
      <div class="gcs-row">
        <span class="gcs-row__label">Luồng</span>
        <span class="gcs-row__val">{{ general.threadRequest }} request / {{ general.threadCheckInfo }} check</span>
      </div>
      <div class="gcs-row">
        <span class="gcs-row__label">Delay</span>
        <span class="gcs-row__val">{{ general.delayRequest }}ms</span>
      </div>
      <div class="gcs-row">
        <span class="gcs-row__label">Nguồn TK</span>
        <span class="gcs-row__val">{{ general.accountSource === 'folder' ? 'Thư mục' : 'API (CloneHV)' }}</span>
      </div>
      <div class="gcs-row">
        <span class="gcs-row__label">IP</span>
        <span class="gcs-row__val">{{ ipLabel }}{{ general.checkIpBeforeRun ? ' + check trước' : '' }}</span>
      </div>
      <div class="gcs-row">
        <span class="gcs-row__label">Captcha</span>
        <span class="gcs-row__val">
          {{ captchaLabel }}
          <span v-if="general.captchaKeys[general.captchaProvider]" class="gcs-key-badge"><Check :size="10" /> Có key</span>
          <span v-else class="gcs-key-badge gcs-key-badge--empty"><AlertTriangle :size="10" /> Chưa có key</span>
        </span>
      </div>
      <div class="gcs-row">
        <span class="gcs-row__label">Sau khi chạy</span>
        <span class="gcs-row__val">
          <span v-if="general.saveRunColumn">Lưu cột</span>
          <span v-if="general.backupDB">{{ general.saveRunColumn ? ' + ' : '' }}Backup</span>
          <span v-if="general.closeAfterDone">{{ (general.saveRunColumn || general.backupDB) ? ' + ' : '' }}Đóng app</span>
          <span v-if="!general.saveRunColumn && !general.backupDB && !general.closeAfterDone" class="gcs-muted">Không có</span>
        </span>
      </div>
    </div>
  </div>
</template>

<style scoped>
.gcs {
  background: var(--surface-elevated, #1a1a1a);
  border-bottom: 1px solid var(--border-default, #2a2a2a);
  font-size: 12px;
}

.gcs__toggle {
  width: 100%;
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 6px 16px;
  background: none;
  border: none;
  cursor: pointer;
  color: var(--text-primary, #e0e0e0);
  text-align: left;
}

.gcs__title {
  font-weight: 700;
  font-size: 11px;
  text-transform: uppercase;
  letter-spacing: 0.04em;
  color: var(--text-muted, #888);
  white-space: nowrap;
}

.gcs__chips {
  display: flex;
  gap: 6px;
  flex: 1;
  flex-wrap: wrap;
}

.gcs__caret {
  font-size: 10px;
  color: var(--text-muted, #888);
  flex-shrink: 0;
}

.chip {
  padding: 1px 7px;
  border-radius: 10px;
  font-size: 11px;
  font-weight: 600;
}

.chip--threads  { background: rgba(79,195,247,0.12); color: var(--accent, #4fc3f7); }
.chip--platform { background: rgba(102,187,106,0.12); color: #66bb6a; }
.chip--ip       { background: rgba(255,183,77,0.12); color: #ffb74d; }
.chip--count    { background: rgba(171,71,188,0.12); color: #ab47bc; }
.chip--captcha  { background: rgba(239,83,80,0.12); color: #ef5350; }

.gcs__detail {
  padding: 8px 16px 10px;
  display: flex;
  flex-wrap: wrap;
  gap: 4px 24px;
  border-top: 1px solid var(--border-default, #2a2a2a);
}

.gcs-row {
  display: flex;
  gap: 6px;
  align-items: baseline;
  min-width: 200px;
}

.gcs-row__label {
  color: var(--text-muted, #888);
  font-size: 11px;
  white-space: nowrap;
  min-width: 80px;
}

.gcs-row__val {
  font-size: 12px;
  color: var(--text-primary, #e0e0e0);
}

.gcs-key-badge {
  display: inline-block;
  font-size: 10px;
  padding: 0 4px;
  border-radius: 3px;
  margin-left: 4px;
  background: rgba(102,187,106,0.12);
  color: #66bb6a;
}

.gcs-key-badge--empty {
  background: rgba(255,183,77,0.12);
  color: #ffb74d;
}

.gcs-muted { color: var(--text-muted, #888); }
</style>
