<script setup lang="ts">
import { ref, watch, onMounted } from 'vue'
import { Zap, FolderOpen } from 'lucide-vue-next'
import { getInteractionService, getFileDialogService } from '@/bridge/client'
import { DEFAULT_VERIFY_CONFIG } from '@/types/interaction.types'
import type { VerifyConfig } from '@/types/interaction.types'
import { useAppStore } from '@/stores/app.store'

const appStore = useAppStore()
const form = ref<VerifyConfig>({ ...DEFAULT_VERIFY_CONFIG })
const saveStatus = ref<'idle' | 'saving' | 'saved' | 'error'>('idle')

onMounted(async () => {
  try {
    const svc = await getInteractionService()
    const data = await svc.load()
    if (data) form.value = { ...DEFAULT_VERIFY_CONFIG, ...data }
  } catch (e) {
    appStore.notify('error', 'Không tải được cấu hình tương tác')
  }
})

async function save() {
  saveStatus.value = 'saving'
  try {
    const svc = await getInteractionService()
    const r = await svc.save(form.value)
    if (r !== 'OK') throw new Error(r)
    saveStatus.value = 'saved'
    setTimeout(() => { saveStatus.value = 'idle' }, 1500)
  } catch {
    saveStatus.value = 'error'
  }
}

watch(form, save, { deep: true })

async function browseAvatarFolder() {
  try {
    const svc = await getFileDialogService()
    const path = await svc.openFolder()
    if (path) form.value.avatarFolderPath = path
  } catch { /* ignore */ }
}
</script>

<template>
  <div class="tt-page">
    <div class="tt-header">
      <div class="tt-header__left">
        <Zap :size="16" class="tt-header__icon" />
        <span class="tt-header__title">Tương tác</span>
        <span class="tt-header__sub">Chạy tự động sau verify Live</span>
      </div>
      <span class="tt-save-status" :data-status="saveStatus">
        <template v-if="saveStatus === 'saving'">Đang lưu...</template>
        <template v-else-if="saveStatus === 'saved'">✓ Đã lưu</template>
        <template v-else-if="saveStatus === 'error'">⚠ Lỗi lưu</template>
        <template v-else>Tự động lưu</template>
      </span>
    </div>

    <div class="tt-body">

      <!-- ── Nhóm: Sau verify ── -->
      <div class="tt-group">
        <div class="tt-group__label">Sau khi verify thành công</div>

        <!-- Upload Avatar -->
        <div class="tt-row">
          <div class="tt-row__info">
            <span class="tt-row__title">Upload Avatar</span>
            <span class="tt-row__desc">Pick ngẫu nhiên ảnh JPG/PNG từ thư mục đã chọn</span>
          </div>
          <div v-if="form.uploadAvatar" class="tt-row__path">
            <input class="tt-input tt-input--path" type="text" v-model="form.avatarFolderPath"
              placeholder="Config/Avatar" readonly />
            <button class="tt-btn-browse" @click="browseAvatarFolder">
              <FolderOpen :size="13" />
            </button>
          </div>
          <label class="tt-toggle">
            <input type="checkbox" v-model="form.uploadAvatar" />
            <span class="tt-toggle__track"><span class="tt-toggle__thumb" /></span>
          </label>
        </div>

        <!-- Bật 2FA -->
        <div class="tt-row">
          <div class="tt-row__info">
            <span class="tt-row__title">Bật 2FA (TOTP)</span>
            <span class="tt-row__desc">Kích hoạt xác thực 2 bước — lưu secret key vào output</span>
          </div>
          <label class="tt-toggle">
            <input type="checkbox" v-model="form.enable2fa" />
            <span class="tt-toggle__track"><span class="tt-toggle__thumb" /></span>
          </label>
        </div>

        <!-- Cập nhật hồ sơ -->
        <div class="tt-row">
          <div class="tt-row__info">
            <span class="tt-row__title">Cập nhật thông tin hồ sơ</span>
            <span class="tt-row__desc">Điền city, trường học, nơi làm việc… từ file Config/AddInfo/</span>
          </div>
          <label class="tt-toggle">
            <input type="checkbox" v-model="form.addInfo" />
            <span class="tt-toggle__track"><span class="tt-toggle__thumb" /></span>
          </label>
        </div>
        <div v-if="form.addInfo" class="tt-sub tt-sub--addinfo">
          <div class="tt-check-grid">
            <label class="tt-check"><input type="checkbox" v-model="form.addInfoCity" /><span>Thành phố</span></label>
            <label class="tt-check"><input type="checkbox" v-model="form.addInfoHometown" /><span>Quê quán</span></label>
            <label class="tt-check"><input type="checkbox" v-model="form.addInfoSchool" /><span>Trường học</span></label>
            <label class="tt-check"><input type="checkbox" v-model="form.addInfoCollege" /><span>Đại học</span></label>
            <label class="tt-check"><input type="checkbox" v-model="form.addInfoWork" /><span>Nơi làm việc</span></label>
            <label class="tt-check"><input type="checkbox" v-model="form.addInfoRelationship" /><span>Độc thân</span></label>
          </div>
        </div>

      </div>

      <!-- Placeholder -->
      <div class="tt-placeholder">Các tính năng khác (like, comment, add friend…) sẽ được thêm vào đây</div>

    </div>
  </div>
</template>

<style scoped>
.tt-page {
  display: flex;
  flex-direction: column;
  height: 100%;
  overflow: hidden;
  background: var(--surface-base);
}

/* Header */
.tt-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 10px 16px;
  border-bottom: 1px solid var(--border-default);
  background: var(--surface-elevated);
  flex-shrink: 0;
  gap: 12px;
}
.tt-header__left { display: flex; align-items: center; gap: 8px; }
.tt-header__icon { color: var(--brand-primary); }
.tt-header__title { font-size: 14px; font-weight: 700; color: var(--text-primary); }
.tt-header__sub { font-size: 11px; color: var(--text-muted); }
.tt-save-status { font-size: 11px; color: var(--text-muted); margin-left: auto; }
.tt-save-status[data-status="saved"] { color: var(--color-success, #22c55e); }
.tt-save-status[data-status="error"] { color: var(--color-danger, #ef4444); }

/* Body */
.tt-body {
  flex: 1;
  overflow-y: auto;
  padding: 14px 20px;
  display: flex;
  flex-direction: column;
  gap: 12px;
  align-items: flex-start;
}
.tt-body > * {
  width: 100%;
  max-width: 680px;
}

/* Group */
.tt-group {
  background: var(--surface-elevated);
  border: 1px solid var(--border-default);
  border-radius: 8px;
  overflow: hidden;
}
.tt-group__label {
  font-size: 10px;
  font-weight: 700;
  letter-spacing: 0.06em;
  text-transform: uppercase;
  color: var(--text-muted);
  padding: 8px 14px 6px;
  border-bottom: 1px solid var(--border-default);
  background: var(--surface-hover);
}

/* Row */
.tt-row {
  display: flex;
  align-items: center;
  padding: 8px 14px;
  gap: 10px;
  border-bottom: 1px solid var(--border-default);
}
.tt-row:last-child { border-bottom: none; }
.tt-row__info { display: flex; flex-direction: column; gap: 1px; flex: 1; min-width: 0; }
.tt-row__title { font-size: 12px; font-weight: 600; color: var(--text-primary); }
.tt-row__desc { font-size: 11px; color: var(--text-muted); }
.tt-row__path {
  display: flex;
  align-items: center;
  gap: 4px;
  flex-shrink: 0;
}

/* Sub-panel */
.tt-sub {
  background: var(--surface-hover);
  border-bottom: 1px solid var(--border-default);
}
.tt-sub--addinfo {
  padding: 8px 14px;
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.tt-check-grid {
  display: grid;
  grid-template-columns: repeat(3, auto);
  gap: 5px 20px;
  justify-content: start;
}
.tt-sub__dir {
  display: flex;
  align-items: center;
  gap: 8px;
  padding-top: 6px;
  border-top: 1px solid var(--border-default);
}
.tt-sub__label { font-size: 11px; color: var(--text-muted); white-space: nowrap; }

/* Toggle */
.tt-toggle { display: flex; align-items: center; cursor: pointer; flex-shrink: 0; }
.tt-toggle input { display: none; }
.tt-toggle__track {
  width: 32px; height: 17px; border-radius: 9px;
  background: var(--border-default);
  position: relative; transition: background 0.2s; flex-shrink: 0;
}
.tt-toggle input:checked ~ .tt-toggle__track { background: var(--brand-primary); }
.tt-toggle__thumb {
  position: absolute; top: 2px; left: 2px;
  width: 13px; height: 13px; border-radius: 50%;
  background: white; transition: left 0.2s;
  box-shadow: 0 1px 2px rgba(0,0,0,.25);
}
.tt-toggle input:checked ~ .tt-toggle__track .tt-toggle__thumb { left: 17px; }

/* Checkbox */
.tt-check {
  display: flex; align-items: center; gap: 6px;
  font-size: 11px; color: var(--text-secondary); cursor: pointer;
}
.tt-check input[type="checkbox"] {
  width: 13px; height: 13px; cursor: pointer; accent-color: var(--brand-primary); flex-shrink: 0;
}

/* Input + Browse */
.tt-input {
  height: 26px; padding: 0 8px;
  font-size: 11px; border: 1px solid var(--border-default);
  border-radius: 5px; background: var(--surface-base);
  color: var(--text-primary); outline: none; min-width: 0;
}
.tt-input--path { width: 220px; }
.tt-input--dir  { width: 200px; }
.tt-btn-browse {
  display: flex; align-items: center; justify-content: center;
  width: 26px; height: 26px;
  border-radius: 5px; background: var(--brand-primary);
  color: white; border: none; cursor: pointer;
  transition: opacity 0.15s; flex-shrink: 0;
}
.tt-btn-browse:hover { opacity: 0.85; }

/* Placeholder */
.tt-placeholder {
  text-align: center; font-size: 11px; color: var(--text-muted);
  padding: 18px; border: 1px dashed var(--border-default); border-radius: 8px;
}
</style>
