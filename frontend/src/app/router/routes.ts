// routes.ts — Định nghĩa routes cho ứng dụng

import type { RouteRecordRaw } from 'vue-router'
import { ROUTE_NAMES, ROUTE_PATHS } from '../../constants/routes'

export const routes: RouteRecordRaw[] = [
  {
    path: '/',
    redirect: ROUTE_PATHS.ACCOUNTS,
  },
  {
    path: ROUTE_PATHS.ACCOUNTS,
    name: ROUTE_NAMES.ACCOUNTS,
    component: () => import('../../pages/AccountsPage.vue'),
  },

  {
    path: ROUTE_PATHS.PROXY_SETTINGS,
    name: ROUTE_NAMES.PROXY_SETTINGS,
    component: () => import('../../pages/ProxySettingsPage.vue'),
  },
  {
    path: ROUTE_PATHS.VIEW_SETTINGS,
    name: ROUTE_NAMES.VIEW_SETTINGS,
    component: () => import('../../pages/ViewSettingsPage.vue'),
  },
  {
    path: ROUTE_PATHS.GENERAL_SETTINGS,
    name: ROUTE_NAMES.GENERAL_SETTINGS,
    component: () => import('../../pages/GeneralSettingsPage.vue'),
  },
  {
    path: ROUTE_PATHS.INTERACTION_SETUP,
    name: ROUTE_NAMES.INTERACTION_SETUP,
    component: () => import('../../pages/InteractionSetupPage.vue'),
  },
  {
    path: ROUTE_PATHS.AUTH_SOURCE,
    name: ROUTE_NAMES.AUTH_SOURCE,
    component: () => import('../../pages/AuthSourcePage.vue'),
  },
  {
    path: ROUTE_PATHS.TUONG_TAC,
    name: ROUTE_NAMES.TUONG_TAC,
    component: () => import('../../pages/TuongTacPage.vue'),
  },
  {
    path: ROUTE_PATHS.UPLOAD_SITE,
    name: ROUTE_NAMES.UPLOAD_SITE,
    component: () => import('../../pages/UploadSitePage.vue'),
  },
  {
    path: ROUTE_PATHS.REG_STATS,
    name: ROUTE_NAMES.REG_STATS,
    component: () => import('../../pages/RegStatsPage.vue'),
  },
  {
    path: ROUTE_PATHS.LEGACY_IMPORT,
    name: ROUTE_NAMES.LEGACY_IMPORT,
    component: () => import('../../pages/LegacyImportWizard.vue'),
  },
]
