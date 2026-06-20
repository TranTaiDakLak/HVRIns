<script setup lang="ts">
// AppSidebar.vue — Sidebar navigation chính
// Collapsible, chạy full-height không có header riêng

import { computed } from 'vue'
import { useRoute } from 'vue-router'
import { Users, Globe, Wrench, Settings, Eye, Zap, ArrowUpToLine, Mail, BarChart3 } from 'lucide-vue-next'
import { useAppStore } from '@/stores/app.store'
import { useAccountsStore } from '@/features/accounts/store/useAccountsStore'
import { ROUTE_PATHS } from '@/constants/routes'

const route = useRoute()
const appStore = useAppStore()
const accountsStore = useAccountsStore()

const collapsed = computed(() => appStore.sidebarCollapsed)
const accountCount = computed(() => accountsStore.total || null)

function isActive(path: string): boolean {
  return route.path === path
}
</script>

<template>
  <aside class="sidebar" :class="{ 'sidebar--collapsed': collapsed }">

    <!-- Brand -->
    <div class="sidebar__header">
      <svg class="sidebar__logo" width="30" height="30" viewBox="0 0 24 24" fill="none">
        <!-- Hexagon shape -->
        <path d="M12 2L21 7V17L12 22L3 17V7L12 2Z" fill="var(--brand-primary)" opacity="0.18" stroke="var(--brand-primary)" stroke-width="1.5" stroke-linejoin="round"/>
        <!-- Chữ HV -->
        <path d="M7.5 8V16 M11 8V16 M7.5 12H11" stroke="var(--brand-primary)" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round"/>
        <path d="M12.5 8L14.5 16L16.5 8" stroke="var(--brand-primary)" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round"/>
      </svg>
      <template v-if="!collapsed">
        <span class="sidebar__brand-name">Hạ Vũ</span>
        <span class="sidebar__brand-suffix">Clone</span>
      </template>
    </div>

    <!-- Nav items -->
    <div class="sidebar__content">
      <router-link :to="ROUTE_PATHS.ACCOUNTS" class="sidebar__item" :class="{ 'sidebar__item--active': isActive(ROUTE_PATHS.ACCOUNTS) }">
        <span class="sidebar__icon"><Users :size="17" /></span>
        <span v-if="!collapsed" class="sidebar__label">Accounts</span>
        <span v-if="!collapsed && accountCount" class="sidebar__badge">{{ accountCount }}</span>
      </router-link>

      <router-link :to="ROUTE_PATHS.PROXY_SETTINGS" class="sidebar__item" :class="{ 'sidebar__item--active': isActive(ROUTE_PATHS.PROXY_SETTINGS) }">
        <span class="sidebar__icon"><Globe :size="17" /></span>
        <span v-if="!collapsed" class="sidebar__label">Proxy Settings</span>
      </router-link>

      <router-link :to="ROUTE_PATHS.INTERACTION_SETUP" class="sidebar__item" :class="{ 'sidebar__item--active': isActive(ROUTE_PATHS.INTERACTION_SETUP) }">
        <span class="sidebar__icon"><Wrench :size="17" /></span>
        <span v-if="!collapsed" class="sidebar__label">Thiết lập chạy</span>
      </router-link>

      <router-link :to="ROUTE_PATHS.AUTH_SOURCE" class="sidebar__item" :class="{ 'sidebar__item--active': isActive(ROUTE_PATHS.AUTH_SOURCE) }">
        <span class="sidebar__icon"><Mail :size="17" /></span>
        <span v-if="!collapsed" class="sidebar__label">Email/Phone</span>
      </router-link>

      <router-link :to="ROUTE_PATHS.TUONG_TAC" class="sidebar__item" :class="{ 'sidebar__item--active': isActive(ROUTE_PATHS.TUONG_TAC) }">
        <span class="sidebar__icon"><Zap :size="17" /></span>
        <span v-if="!collapsed" class="sidebar__label">Tương tác</span>
      </router-link>

      <router-link :to="ROUTE_PATHS.UPLOAD_SITE" class="sidebar__item" :class="{ 'sidebar__item--active': isActive(ROUTE_PATHS.UPLOAD_SITE) }">
        <span class="sidebar__icon"><ArrowUpToLine :size="17" /></span>
        <span v-if="!collapsed" class="sidebar__label">Đẩy tài khoản</span>
      </router-link>

      <router-link :to="ROUTE_PATHS.REG_STATS" class="sidebar__item" :class="{ 'sidebar__item--active': isActive(ROUTE_PATHS.REG_STATS) }">
        <span class="sidebar__icon"><BarChart3 :size="17" /></span>
        <span v-if="!collapsed" class="sidebar__label">Thống kê</span>
      </router-link>

      <div class="sidebar__divider"></div>

      <router-link :to="ROUTE_PATHS.GENERAL_SETTINGS" class="sidebar__item" :class="{ 'sidebar__item--active': isActive(ROUTE_PATHS.GENERAL_SETTINGS) }">
        <span class="sidebar__icon"><Settings :size="17" /></span>
        <span v-if="!collapsed" class="sidebar__label">Cài đặt chung</span>
      </router-link>

      <router-link :to="ROUTE_PATHS.VIEW_SETTINGS" class="sidebar__item" :class="{ 'sidebar__item--active': isActive(ROUTE_PATHS.VIEW_SETTINGS) }">
        <span class="sidebar__icon"><Eye :size="17" /></span>
        <span v-if="!collapsed" class="sidebar__label">Hiển thị</span>
      </router-link>
    </div>

  </aside>
</template>

<style scoped>
.sidebar {
  width: var(--sidebar-width);
  background: var(--sidebar-bg, var(--surface-elevated));
  border-right: 2px solid transparent;
  border-image: linear-gradient(180deg, #405DE6 0%, #833AB4 30%, #E1306C 60%, #FD1D1D 80%, #FCAF45 100%) 1;
  display: flex;
  flex-direction: column;
  flex-shrink: 0;
  transition: width var(--transition-normal);
  overflow: hidden;
}

.sidebar--collapsed {
  width: var(--sidebar-collapsed-width);
}

/* Header gộp brand + toggle */
.sidebar__header {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 18px 10px 14px;
  flex-shrink: 0;
}

.sidebar__toggle {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  border-radius: 6px;
  color: var(--text-muted);
  flex-shrink: 0;
  transition: background var(--transition-fast), color var(--transition-fast);
}
.sidebar__toggle:hover {
  background: var(--surface-hover);
  color: var(--text-primary);
}

.sidebar__logo { flex-shrink: 0; display: block; }

.sidebar--collapsed .sidebar__header {
  justify-content: center;
  padding: 10px 0;
}

.sidebar__brand-name {
  font-weight: 700;
  font-size: 16px;
  color: var(--brand-primary);
  letter-spacing: -0.3px;
}
.sidebar__brand-suffix {
  font-weight: 400;
  font-size: 16px;
  color: var(--text-muted);
}

.sidebar__content {
  flex: 1;
  padding: 16px 6px;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

/* Divider mỏng phân cách nhóm */
.sidebar__divider {
  height: 1px;
  background: var(--border-subtle);
  margin: 6px 4px;
}

/* Item menu */
.sidebar__item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 10px;
  border-radius: 8px;
  font-size: 13px;
  font-weight: 500;
  color: var(--text-secondary);
  text-decoration: none;
  margin: 1px 0;
  transition: background var(--transition-fast), color var(--transition-fast);
  position: relative;
}

.sidebar__item:hover {
  background: var(--surface-hover);
  color: var(--text-primary);
  text-decoration: none;
}

.sidebar__item--active {
  background: var(--brand-primary-bg);
  color: var(--brand-primary);
  font-weight: 600;
}

/* Icon — luôn căn giữa, kích thước ổn định */
.sidebar__icon {
  width: 20px;
  height: 20px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  opacity: 0.75;
  transition: opacity var(--transition-fast);
}

.sidebar__item--active .sidebar__icon,
.sidebar__item:hover .sidebar__icon {
  opacity: 1;
}

.sidebar__label {
  flex: 1;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  line-height: 1.3;
}

/* Badge số lượng account */
.sidebar__badge {
  font-size: 11px;
  font-weight: 600;
  padding: 1px 7px;
  border-radius: 20px;
  background: var(--surface-hover-strong);
  color: var(--text-muted);
  line-height: 1.6;
}

.sidebar__item--active .sidebar__badge {
  background: var(--brand-primary-bg);
  color: var(--brand-primary);
}

/* Collapsed: căn giữa icon */
.sidebar--collapsed .sidebar__item {
  justify-content: center;
  padding: 8px;
}
</style>
