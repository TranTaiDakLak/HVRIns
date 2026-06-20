<script setup lang="ts">
// ContextMenu.vue — Context menu với nested submenu
// Teleport to body, click outside dismiss, Escape close

import { ref, onMounted, onUnmounted, watch } from 'vue'
import type { MenuItemDef } from '@/composables/useContextMenu'

const props = defineProps<{
  visible: boolean
  x: number
  y: number
  items: MenuItemDef[]
}>()

const emit = defineEmits<{
  close: []
}>()

const openSubmenu = ref<string | null>(null)
const openSubSubmenu = ref<string | null>(null)

function handleItemClick(item: MenuItemDef) {
  if (item.disabled || item.separator) return
  if (item.children && item.children.length > 0) return
  item.action?.()
  emit('close')
}

function handleKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape') emit('close')
}

onMounted(() => {
  document.addEventListener('keydown', handleKeydown)
})

onUnmounted(() => {
  document.removeEventListener('keydown', handleKeydown)
})

watch(() => props.visible, (v) => {
  if (!v) { openSubmenu.value = null; openSubSubmenu.value = null }
})
</script>

<template>
  <Teleport to="body">
    <div v-if="visible" class="ctx-backdrop" @mousedown.self="emit('close')">
      <div class="ctx-menu" :style="{ left: x + 'px', top: y + 'px' }">
        <template v-for="item in items" :key="item.key">
          <!-- Separator -->
          <div v-if="item.separator" class="ctx-sep" />

          <!-- Item with children (submenu) -->
          <div
            v-else-if="item.children && item.children.length"
            class="ctx-item ctx-item--has-children"
            :class="{ 'ctx-item--disabled': item.disabled }"
            @mouseenter="openSubmenu = item.key"
            @mouseleave="openSubmenu = null"
          >
            <span v-if="item.icon" class="ctx-icon">{{ item.icon }}</span>
            <span class="ctx-label">{{ item.label }}</span>
            <span class="ctx-arrow">&#x25B6;</span>

            <!-- Submenu (level 2) -->
            <div v-if="openSubmenu === item.key" class="ctx-submenu">
              <template v-for="child in item.children" :key="child.key">
                <div v-if="child.separator" class="ctx-sep" />

                <!-- Child with sub-submenu (level 3) -->
                <div
                  v-else-if="child.children && child.children.length"
                  class="ctx-item ctx-item--has-children"
                  :class="{ 'ctx-item--disabled': child.disabled }"
                  @mouseenter="openSubSubmenu = child.key"
                  @mouseleave="openSubSubmenu = null"
                >
                  <span v-if="child.icon" class="ctx-icon">{{ child.icon }}</span>
                  <span class="ctx-label">{{ child.label }}</span>
                  <span class="ctx-arrow">&#x25B6;</span>

                  <!-- Sub-submenu (level 3) -->
                  <div v-if="openSubSubmenu === child.key" class="ctx-submenu">
                    <template v-for="grandchild in child.children" :key="grandchild.key">
                      <div v-if="grandchild.separator" class="ctx-sep" />
                      <div
                        v-else
                        class="ctx-item"
                        :class="{ 'ctx-item--disabled': grandchild.disabled }"
                        @click.stop="handleItemClick(grandchild)"
                      >
                        <span v-if="grandchild.icon" class="ctx-icon">{{ grandchild.icon }}</span>
                        <span class="ctx-label">{{ grandchild.label }}</span>
                      </div>
                    </template>
                  </div>
                </div>

                <!-- Leaf child -->
                <div
                  v-else
                  class="ctx-item"
                  :class="{ 'ctx-item--disabled': child.disabled }"
                  @click.stop="handleItemClick(child)"
                >
                  <span v-if="child.icon" class="ctx-icon">{{ child.icon }}</span>
                  <span class="ctx-label">{{ child.label }}</span>
                </div>
              </template>
            </div>
          </div>

          <!-- Leaf item -->
          <div
            v-else
            class="ctx-item"
            :class="{ 'ctx-item--disabled': item.disabled }"
            @click.stop="handleItemClick(item)"
          >
            <span v-if="item.icon" class="ctx-icon">{{ item.icon }}</span>
            <span class="ctx-label">{{ item.label }}</span>
          </div>
        </template>
      </div>
    </div>
  </Teleport>
</template>

<style scoped>
.ctx-backdrop {
  position: fixed;
  inset: 0;
  z-index: 200;
}

.ctx-menu {
  position: fixed;
  min-width: 220px;
  background: var(--surface-elevated);
  border: 1px solid var(--border-default);
  border-radius: var(--radius-md);
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.4);
  padding: var(--space-1) 0;
  z-index: 201;
}

.ctx-item {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  padding: 6px 12px;
  font-size: var(--font-size-sm);
  color: var(--text-primary);
  cursor: pointer;
  position: relative;
  white-space: nowrap;
}

.ctx-item:hover {
  background: var(--brand-primary-bg);
  color: var(--brand-primary);
}

.ctx-item--disabled {
  color: var(--text-disabled);
  cursor: not-allowed;
}
.ctx-item--disabled:hover {
  background: transparent;
  color: var(--text-disabled);
}

.ctx-item--has-children {
  padding-right: 28px;
}

.ctx-icon {
  width: 18px;
  text-align: center;
  font-size: 14px;
  flex-shrink: 0;
}

.ctx-label {
  flex: 1;
}

.ctx-arrow {
  position: absolute;
  right: 8px;
  font-size: 8px;
  color: var(--text-muted);
}

.ctx-sep {
  height: 1px;
  background: var(--border-subtle);
  margin: var(--space-1) 0;
}

.ctx-submenu {
  position: absolute;
  left: 100%;
  top: -4px;
  min-width: 200px;
  background: var(--surface-elevated);
  border: 1px solid var(--border-default);
  border-radius: var(--radius-md);
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.4);
  padding: var(--space-1) 0;
  z-index: 202;
}
</style>
