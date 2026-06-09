<script setup lang="ts">
// RegStatsPage.vue — Thống kê REG / VERIFY / Mail Domain.
// Dữ liệu từ backend, tự refresh mỗi 10s.

import { ref, onMounted, onBeforeUnmount } from 'vue'
import RegVerStatsTable from '../components/RegVerStatsTable.vue'
import MailDomainStatsTable from '../components/MailDomainStatsTable.vue'

interface StatRow { index: number; platform: string; success: number; fail: number; total: number }
interface MailDomainRow { index: number; domain: string; veri: number; live: number; die: number; rate: number }

type TabId = 'reg-ver' | 'mail' | 'build-ua'
const activeTab = ref<TabId>('reg-ver')

const regRows = ref<StatRow[]>([])
const verRows = ref<StatRow[]>([])
const mailRows = ref<MailDomainRow[]>([])
const buildUARegRows = ref<StatRow[]>([])
const buildUAVerRows = ref<StatRow[]>([])
const lastUpdate = ref('')
const loading = ref(false)
const available = ref(true)
const regRunning = ref<boolean | null>(null)
const verRunning = ref<boolean | null>(null)
let timer: number | null = null

async function fetchStats() {
  const api = (window as any)?.go?.main?.App
  if (!api) { available.value = false; return }
  available.value = true
  loading.value = true
  try {
    if (typeof api.GetRegStats === 'function') {
      const d = await api.GetRegStats()
      regRows.value = Array.isArray(d) ? d : []
    }
    if (typeof api.GetVerifyStats === 'function') {
      const d = await api.GetVerifyStats()
      verRows.value = Array.isArray(d) ? d : []
    }
    if (typeof api.GetMailDomainStats === 'function') {
      const d = await api.GetMailDomainStats()
      mailRows.value = Array.isArray(d) ? d : []
    }
    if (typeof api.GetBuildUARegStats === 'function') {
      const d = await api.GetBuildUARegStats()
      buildUARegRows.value = Array.isArray(d) ? d : []
    }
    if (typeof api.GetBuildUAVerStats === 'function') {
      const d = await api.GetBuildUAVerStats()
      buildUAVerRows.value = Array.isArray(d) ? d : []
    }
    if (typeof api.IsRegisterRunning === 'function') { try { regRunning.value = await api.IsRegisterRunning() } catch { /* */ } }
    if (typeof api.IsVerifyRunning === 'function') { try { verRunning.value = await api.IsVerifyRunning() } catch { /* */ } }
    lastUpdate.value = new Date().toLocaleTimeString()
  } catch { /* giữ data cũ */ }
  finally { loading.value = false }
}

onMounted(() => {
  void fetchStats()
  timer = window.setInterval(() => { void fetchStats() }, 10_000)
})
onBeforeUnmount(() => {
  if (timer !== null) { clearInterval(timer); timer = null }
})
</script>

<template>
  <div class="rs-page">
    <div class="rs-header">
      <div class="rs-header__left">
        <h2>Thống kê</h2>
        <span v-if="regRunning === true" class="rs-badge rs-badge--reg">● REG đang chạy</span>
        <span v-if="verRunning === true" class="rs-badge rs-badge--ver">● VERIFY đang chạy</span>
        <span v-if="regRunning === false && verRunning === false" class="rs-badge rs-badge--idle">■ Đã dừng</span>
      </div>
      <div class="rs-header__right">
        <nav class="rs-tabs">
          <button
            class="rs-tab"
            :class="{ 'rs-tab--active': activeTab === 'reg-ver' }"
            @click="activeTab = 'reg-ver'"
          >REG / VERIFY</button>
          <button
            class="rs-tab"
            :class="{ 'rs-tab--active': activeTab === 'mail' }"
            @click="activeTab = 'mail'"
          >Mail Domain</button>
          <button
            class="rs-tab"
            :class="{ 'rs-tab--active': activeTab === 'build-ua' }"
            @click="activeTab = 'build-ua'"
          >Build UA</button>
        </nav>
        <span v-if="lastUpdate" class="rs-header__time">Cập nhật {{ lastUpdate }} · tự làm mới 10s</span>
        <button class="rs-refresh" :disabled="loading" @click="fetchStats">↻ Làm mới</button>
      </div>
    </div>

    <div v-if="!available" class="rs-empty">Không kết nối được backend — mở app qua Wails để xem thống kê.</div>

    <template v-else>
      <div v-show="activeTab === 'reg-ver'" class="rs-grid">
        <RegVerStatsTable title="Đăng ký (REG)" accent="reg" export-kind="reg" :rows="regRows" />
        <RegVerStatsTable title="Xác thực (VERIFY)" accent="ver" export-kind="verify" :rows="verRows" />
      </div>
      <div v-show="activeTab === 'mail'" class="rs-grid rs-grid--single">
        <MailDomainStatsTable :rows="mailRows" />
      </div>
      <div v-show="activeTab === 'build-ua'" class="rs-grid">
        <RegVerStatsTable title="Reg – FBAV version" accent="reg" export-kind="reg" platform-label="FBAV" :rows="buildUARegRows" />
        <RegVerStatsTable title="Veri – FBAV version" accent="ver" export-kind="verify" platform-label="FBAV" :rows="buildUAVerRows" />
      </div>
    </template>
  </div>
</template>

<style scoped>
.rs-page {
  height: 100%;
  min-height: 0;
  padding: 20px 24px;
  display: flex;
  flex-direction: column;
}
.rs-header { display: flex; align-items: center; justify-content: space-between; gap: 16px; margin-bottom: 14px; flex-wrap: wrap; }
.rs-header__left { display: flex; align-items: center; gap: 10px; flex-wrap: wrap; }
.rs-header h2 { margin: 0; font-size: 17px; font-weight: 700; color: var(--text-primary); }
.rs-header__right { display: flex; align-items: center; gap: 12px; }
.rs-header__time { font-size: 11.5px; color: var(--text-muted); }

.rs-tabs { display: flex; gap: 2px; background: var(--surface-sunken); border: 1px solid var(--border-default); border-radius: var(--radius-sm); padding: 2px; }
.rs-tab {
  padding: 4px 14px; font-size: 12px; font-weight: 500; border-radius: calc(var(--radius-sm) - 1px);
  border: none; background: transparent; color: var(--text-secondary); cursor: pointer; transition: all .12s;
}
.rs-tab:hover { color: var(--text-primary); background: var(--surface-hover); }
.rs-tab--active { background: var(--surface-elevated); color: var(--text-primary); box-shadow: 0 1px 3px rgba(0,0,0,0.15); }
.rs-badge { font-size: 11px; font-weight: 600; padding: 2px 8px; border-radius: 20px; }
.rs-badge--reg { color: #f97316; background: rgba(249,115,22,0.14); }
.rs-badge--ver { color: #3b82f6; background: rgba(59,130,246,0.14); }
.rs-badge--idle { color: var(--text-muted); background: var(--surface-sunken); }
.rs-refresh {
  padding: 4px 11px; font-size: 12px; border-radius: var(--radius-sm);
  border: 1px solid var(--border-default); background: var(--surface-sunken); color: var(--text-secondary);
  cursor: pointer; transition: all .12s;
}
.rs-refresh:hover:not(:disabled) { background: var(--surface-hover); color: var(--text-primary); }
.rs-refresh:disabled { opacity: .5; cursor: default; }
.rs-empty {
  padding: 32px; text-align: center; color: var(--text-muted); font-size: 13px;
  border: 1px dashed var(--border-default); border-radius: var(--radius-md);
}
.rs-grid {
  flex: 1 1 auto;
  min-height: 0;
  display: flex;
  gap: 16px;
  align-items: stretch;
  flex-wrap: nowrap;
}
.rs-grid--single { justify-content: center; }
.rs-grid--single > * { max-width: 680px; }
@media (max-width: 980px) {
  .rs-grid { flex-wrap: wrap; }
}
</style>
