<script setup lang="ts">
// AppLayout.vue — Layout chính: Sidebar + Header + Content + StatusBar
// Desktop-first, full viewport

import AppTitleBar from '@/components/shell/AppTitleBar.vue'
import AppSidebar from '@/components/shell/AppSidebar.vue'
import AppHeader from '@/components/shell/AppHeader.vue'
import AppStatusBar from '@/components/shell/AppStatusBar.vue'
import { useAppStore } from '@/stores/app.store'

const appStore = useAppStore()
</script>

<template>
  <div class="app-layout" :class="{ 'app-layout--sidebar-collapsed': appStore.sidebarCollapsed }">
    <AppTitleBar />
    <div class="app-layout__body">
      <!-- Sidebar full height -->
      <AppSidebar />

      <!-- Cột phải: Header + Content + StatusBar -->
      <div class="app-layout__right">
        <AppHeader />
        <main class="app-layout__content">
          <router-view v-slot="{ Component }">
            <keep-alive include="AccountsPage">
              <component :is="Component" />
            </keep-alive>
          </router-view>
        </main>
      </div>
    </div>
    <div class="app-layout__status-shell">
      <AppStatusBar />
    </div>
  </div>
</template>

<style scoped>
.app-layout {
  display: flex;
  flex-direction: column;
  height: 100vh;
  width: 100vw;
  overflow: hidden;
}

.app-layout__body {
  display: flex;
  flex: 1;
  overflow: hidden;
}

.app-layout__right {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.app-layout__content {
  flex: 1;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.app-layout__status-shell {
  flex-shrink: 0;
  margin-left: var(--sidebar-width);
  transition: margin-left var(--transition-normal);
}

.app-layout--sidebar-collapsed .app-layout__status-shell {
  margin-left: var(--sidebar-collapsed-width);
}

@media (max-width: 1280px) {
  .app-layout__status-shell {
    margin-left: 0;
  }
}

</style>
