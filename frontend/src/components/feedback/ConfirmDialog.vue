<!-- ConfirmDialog.vue — Hộp thoại xác nhận hành động
     Sử dụng BaseModal bên trong
     Có nút Xác nhận + Hủy, hỗ trợ variant danger/primary -->

<script setup lang="ts">
import BaseModal from './BaseModal.vue'
import BaseButton from '../ui/BaseButton.vue'

// --- Props ---
withDefaults(defineProps<{
  show: boolean
  title?: string
  message?: string
  confirmText?: string
  cancelText?: string
  variant?: 'danger' | 'primary'
}>(), {
  title: 'Xác nhận',
  message: '',
  confirmText: 'Xác nhận',
  cancelText: 'Hủy',
  variant: 'primary',
})

const emit = defineEmits<{
  'update:show': [value: boolean]
  confirm: []
  cancel: []
}>()

// Xử lý xác nhận
function onConfirm() {
  emit('confirm')
  emit('update:show', false)
}

// Xử lý hủy
function onCancel() {
  emit('cancel')
  emit('update:show', false)
}
</script>

<template>
  <BaseModal
    :show="show"
    :title="title"
    size="sm"
    @update:show="emit('update:show', $event)"
    @close="onCancel"
  >
    <!-- Nội dung thông báo -->
    <p class="confirm-dialog__message">{{ message }}</p>

    <!-- Footer: các nút hành động -->
    <template #footer>
      <BaseButton variant="ghost" size="md" @click="onCancel">
        {{ cancelText }}
      </BaseButton>
      <BaseButton :variant="variant === 'danger' ? 'danger' : 'primary'" size="md" @click="onConfirm">
        {{ confirmText }}
      </BaseButton>
    </template>
  </BaseModal>
</template>

<style scoped>
.confirm-dialog__message {
  margin: 0;
  font-family: var(--font-family);
  font-size: var(--font-size-base);
  color: var(--text-secondary);
  line-height: 1.5;
}
</style>
