<script setup lang="ts">
// AppHeader.vue — Header bar chính
// Brand, sidebar toggle, dark mode toggle, notifications, profile dropdown

import { ref } from 'vue'
import { useAppStore } from '@/stores/app.store'
import { usePreferencesStore } from '@/stores/preferences.store'

const appStore = useAppStore()
const prefs = usePreferencesStore()
const showProfile = ref(false)

// Đóng dropdown khi click ngoài
function onClickOutside(e: MouseEvent) {
  const target = e.target as HTMLElement
  if (!target.closest('.app-header__profile')) {
    showProfile.value = false
  }
}

// Toggle profile dropdown
function toggleProfile() {
  showProfile.value = !showProfile.value
  if (showProfile.value) {
    setTimeout(() => document.addEventListener('click', onClickOutside, { once: true }), 0)
  }
}
</script>

<template>
  <header class="app-header">
    <!-- Sidebar toggle -->
    <button class="app-header__toggle" @click.stop="appStore.toggleSidebar" title="Thu/mở sidebar">
      <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <line x1="3" y1="6" x2="21" y2="6"/><line x1="3" y1="12" x2="21" y2="12"/><line x1="3" y1="18" x2="21" y2="18"/>
      </svg>
    </button>

    <div class="app-header__spacer" />

    <!-- Dark mode toggle -->
    <button class="app-header__action" @click.stop="prefs.toggleTheme" :title="prefs.theme === 'dark' ? 'Light mode' : 'Dark mode'">
      <svg v-if="prefs.theme === 'dark'" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z"/>
      </svg>
      <svg v-else width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <circle cx="12" cy="12" r="5"/><line x1="12" y1="1" x2="12" y2="3"/><line x1="12" y1="21" x2="12" y2="23"/><line x1="4.22" y1="4.22" x2="5.64" y2="5.64"/><line x1="18.36" y1="18.36" x2="19.78" y2="19.78"/><line x1="1" y1="12" x2="3" y2="12"/><line x1="21" y1="12" x2="23" y2="12"/><line x1="4.22" y1="19.78" x2="5.64" y2="18.36"/><line x1="18.36" y1="5.64" x2="19.78" y2="4.22"/>
      </svg>
    </button>

    <!-- Notification bell -->
    <button class="app-header__action" :class="{ 'app-header__action--has-notif': appStore.hasNotifications }">
      <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <path d="M18 8A6 6 0 0 0 6 8c0 7-3 9-3 9h18s-3-2-3-9"/><path d="M13.73 21a2 2 0 0 1-3.46 0"/>
      </svg>
    </button>

    <!-- Profile dropdown -->
    <div class="app-header__profile" @click="toggleProfile">
      <div class="app-header__avatar">
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"/><circle cx="12" cy="7" r="4"/>
        </svg>
      </div>

      <!-- Dropdown menu -->
      <Transition name="dropdown">
        <div v-if="showProfile" class="app-header__dropdown">
          <div class="app-header__dropdown-header">
            <div class="app-header__dropdown-avatar">T</div>
            <div>
              <div class="app-header__dropdown-name">Tài Trần</div>
              <div class="app-header__dropdown-email">trantai@company.com</div>
            </div>
          </div>
          <div class="app-header__dropdown-divider" />
          <button class="app-header__dropdown-item">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"/><circle cx="12" cy="7" r="4"/></svg>
            Thông tin cá nhân
          </button>
          <button class="app-header__dropdown-item" @click.stop="prefs.toggleTheme">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="5"/><line x1="12" y1="1" x2="12" y2="3"/></svg>
            {{ prefs.theme === 'dark' ? 'Chuyển Light mode' : 'Chuyển Dark mode' }}
          </button>
          <div class="app-header__dropdown-divider" />
          <button class="app-header__dropdown-item app-header__dropdown-item--danger">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4"/><polyline points="16 17 21 12 16 7"/><line x1="21" y1="12" x2="9" y2="12"/></svg>
            Đăng xuất
          </button>
        </div>
      </Transition>
    </div>
  </header>
</template>

<style scoped>
.app-header {
  height: var(--header-height);
  background: var(--surface-elevated);
  border-bottom: 1px solid var(--border-default);
  display: flex;
  align-items: center;
  padding: 0 var(--space-3);
  gap: var(--space-2);
  flex-shrink: 0;
}

.app-header__toggle {
  padding: var(--space-2);
  border-radius: var(--radius-md);
  color: var(--text-secondary);
  display: flex;
  align-items: center;
}
.app-header__toggle:hover { background: var(--surface-hover); color: var(--text-primary); }

.app-header__brand {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  margin-left: var(--space-1);
}
.app-header__logo { flex-shrink: 0; }
.app-header__brand-name { font-weight: 700; font-size: var(--font-size-md); color: var(--brand-primary); letter-spacing: -0.3px; }
.app-header__brand-suffix { font-weight: 400; font-size: var(--font-size-md); color: var(--text-muted); }

.app-header__spacer { flex: 1; }

.app-header__action {
  padding: var(--space-2);
  border-radius: var(--radius-md);
  color: var(--text-secondary);
  position: relative;
  display: flex;
  align-items: center;
}
.app-header__action:hover { background: var(--surface-hover); color: var(--text-primary); }

.app-header__action--has-notif::after {
  content: '';
  position: absolute;
  top: 6px; right: 6px;
  width: 6px; height: 6px;
  border-radius: 50%;
  background: var(--danger-solid);
}

/* Profile */
.app-header__profile {
  position: relative;
  cursor: pointer;
}

.app-header__avatar {
  width: 30px; height: 30px;
  background: linear-gradient(135deg, var(--brand-primary), #059669);
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  color: white;
  transition: box-shadow var(--transition-fast);
}
.app-header__profile:hover .app-header__avatar {
  box-shadow: 0 0 0 2px var(--surface-elevated), 0 0 0 4px var(--brand-primary);
}

/* Dropdown */
.app-header__dropdown {
  position: absolute;
  top: calc(100% + 8px);
  right: 0;
  width: 260px;
  background: var(--surface-elevated);
  border: 1px solid var(--border-default);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-lg);
  z-index: var(--z-dropdown);
  padding: var(--space-2) 0;
  overflow: hidden;
}

.app-header__dropdown-header {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  padding: var(--space-3) var(--space-4);
}

.app-header__dropdown-avatar {
  width: 36px; height: 36px;
  background: linear-gradient(135deg, var(--brand-primary), #059669);
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-weight: 700;
  font-size: var(--font-size-md);
  color: white;
  flex-shrink: 0;
}

.app-header__dropdown-name { font-weight: 600; font-size: var(--font-size-sm); }
.app-header__dropdown-email { font-size: var(--font-size-xs); color: var(--text-muted); }

.app-header__dropdown-divider {
  height: 1px;
  background: var(--border-default);
  margin: var(--space-1) 0;
}

.app-header__dropdown-item {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  width: 100%;
  padding: var(--space-2) var(--space-4);
  font-size: var(--font-size-sm);
  color: var(--text-secondary);
  text-align: left;
  transition: background var(--transition-fast);
}
.app-header__dropdown-item:hover { background: var(--surface-hover); color: var(--text-primary); }
.app-header__dropdown-item--danger { color: var(--danger-text); }
.app-header__dropdown-item--danger:hover { background: var(--danger-bg); }

/* Dropdown animation */
.dropdown-enter-active { transition: opacity 0.15s ease, transform 0.15s ease; }
.dropdown-leave-active { transition: opacity 0.1s ease, transform 0.1s ease; }
.dropdown-enter-from, .dropdown-leave-to { opacity: 0; transform: translateY(-4px) scale(0.97); }
</style>
