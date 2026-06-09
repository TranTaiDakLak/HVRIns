// client.ts — Bridge factory
// Luôn dùng Wails Go bindings khi available, fallback mock
// Fix: retry detect window.go vì nó inject muộn hơn Vue mount

import type { IAccountService, IFlowService, IProxyService, ISettingsService, IInteractionService, IFileDialogService, ICloneHVService, ILegacyImportService, IResourceUsageService, IVerifyRunnerService, IEventBusService, IProfileService, IAppInfoService, IUploadSiteService } from './contracts'

let _flows: IFlowService | null = null
let _proxies: IProxyService | null = null

function isWails(): boolean {
  try {
    return typeof window !== 'undefined' &&
      (window as any)['go'] !== undefined &&
      (window as any)['go']['main'] !== undefined
  } catch {
    return false
  }
}

// Retry detect window.go tối đa 10 lần (mỗi lần 200ms = 2s tổng)
async function waitForWails(): Promise<boolean> {
  for (let i = 0; i < 10; i++) {
    if (isWails()) return true
    await new Promise(r => setTimeout(r, 200))
  }
  return false
}

export async function getAccountService(): Promise<IAccountService> {
  const wailsReady = await waitForWails()
  if (wailsReady) {
    const { accountsWails } = await import('./wails/accounts.wails')
    return accountsWails
  }
  const { accountsMock } = await import('./mock/accounts.mock')
  return accountsMock
}

export async function getFlowService(): Promise<IFlowService> {
  if (!_flows) {
    const { flowsMock } = await import('./mock/flows.mock')
    _flows = flowsMock
  }
  return _flows
}

export async function getProxyService(): Promise<IProxyService> {
  if (!_proxies) {
    const { proxiesMock } = await import('./mock/proxies.mock')
    _proxies = proxiesMock
  }
  return _proxies
}

export async function getSettingsService(): Promise<ISettingsService> {
  const wailsReady = await waitForWails()
  if (wailsReady) {
    const { settingsWails } = await import('./wails/settings.wails')
    return settingsWails
  }
  const { settingsMock } = await import('./mock/settings.mock')
  return settingsMock
}

export async function getInteractionService(): Promise<IInteractionService> {
  const wailsReady = await waitForWails()
  if (wailsReady) {
    const { interactionWails } = await import('./wails/interaction.wails')
    return interactionWails
  }
  const { interactionMock } = await import('./mock/interaction.mock')
  return interactionMock
}

export async function getFileDialogService(): Promise<IFileDialogService> {
  const wailsReady = await waitForWails()
  if (wailsReady) {
    const { fileDialogWails } = await import('./wails/file-dialog.wails')
    return fileDialogWails
  }
  const { fileDialogMock } = await import('./mock/file-dialog.mock')
  return fileDialogMock
}

export async function getCloneHVService(): Promise<ICloneHVService> {
  const wailsReady = await waitForWails()
  if (wailsReady) {
    const { cloneHVWails } = await import('./wails/file-dialog.wails')
    return cloneHVWails
  }
  const { cloneHVMock } = await import('./mock/file-dialog.mock')
  return cloneHVMock
}

export async function getLegacyImportService(): Promise<ILegacyImportService> {
  const wailsReady = await waitForWails()
  if (wailsReady) {
    const { legacyImportWails } = await import('./wails/legacy-import.wails')
    return legacyImportWails
  }
  const { legacyImportMock } = await import('./mock/legacy-import.mock')
  return legacyImportMock
}

export async function getProfileService(): Promise<IProfileService> {
  const wailsReady = await waitForWails()
  if (wailsReady) {
    const { profileWails } = await import('./wails/profile.wails')
    return profileWails
  }
  const { profileMock } = await import('./mock/profile.mock')
  return profileMock
}

export async function getResourceUsageService(): Promise<IResourceUsageService> {
  const wailsReady = await waitForWails()
  if (wailsReady) {
    const { resourceUsageWails } = await import('./wails/resource-usage.wails')
    return resourceUsageWails
  }
  const { resourceUsageMock } = await import('./mock/resource-usage.mock')
  return resourceUsageMock
}

export async function getVerifyRunnerService(): Promise<IVerifyRunnerService> {
  const wailsReady = await waitForWails()
  if (wailsReady) {
    const { verifyRunnerWails } = await import('./wails/verify-runner.wails')
    return verifyRunnerWails
  }
  const { verifyRunnerMock } = await import('./mock/verify-runner.mock')
  return verifyRunnerMock
}

export async function getEventBusService(): Promise<IEventBusService> {
  const wailsReady = await waitForWails()
  if (wailsReady) {
    const { eventBusWails } = await import('./wails/event-bus.wails')
    return eventBusWails
  }
  const { eventBusMock } = await import('./mock/event-bus.mock')
  return eventBusMock
}

export async function getAppInfoService(): Promise<IAppInfoService> {
  const wailsReady = await waitForWails()
  if (wailsReady) {
    const { appInfoWails } = await import('./wails/app-info.wails')
    return appInfoWails
  }
  const { appInfoMock } = await import('./mock/app-info.mock')
  return appInfoMock
}

export async function getUploadSiteService(): Promise<IUploadSiteService> {
  const wailsReady = await waitForWails()
  if (wailsReady) {
    const { uploadSiteWails } = await import('./wails/upload-site.wails')
    return uploadSiteWails
  }
  const { uploadSiteMock } = await import('./mock/upload-site.mock')
  return uploadSiteMock
}

export function getBridgeMode(): 'wails' | 'mock' {
  return isWails() ? 'wails' : 'mock'
}
