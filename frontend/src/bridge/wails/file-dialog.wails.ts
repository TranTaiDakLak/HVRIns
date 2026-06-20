// file-dialog.wails.ts — Wails implementation cho IFileDialogService
import type { IFileDialogService, ICloneHVService, ImportResult, CloneHVStockResult } from '@/bridge/contracts'
import {
  OpenFolderDialog,
  OpenFileDialogPath,
  OpenTextFileDialog,
  ReadTextFile,
  ValidatePath,
  GetAccountSourceFolder,
  SetAccountSourceFolder,
  RefreshAccountSource,
  CheckCloneHVStock,
} from '../../../wailsjs/go/main/App'

export const fileDialogWails: IFileDialogService = {
  async openFolder(): Promise<string> {
    return await OpenFolderDialog()
  },

  async openTextFile(): Promise<string> {
    return await OpenTextFileDialog()
  },

  async openFilePath(): Promise<string> {
    return await OpenFileDialogPath()
  },

  async readTextFile(path: string): Promise<string> {
    return await ReadTextFile(path)
  },

  async validatePath(path: string): Promise<string> {
    return await ValidatePath(path)
  },

  async getAccountSourceFolder(): Promise<string> {
    return await GetAccountSourceFolder()
  },

  async setAccountSourceFolder(path: string): Promise<ImportResult> {
    return await SetAccountSourceFolder(path) as ImportResult
  },

  async refreshAccountSource(): Promise<ImportResult> {
    return await RefreshAccountSource() as ImportResult
  },

  async openFolderInExplorer(path: string): Promise<string> {
    const fn = (window as any)?.go?.main?.App?.OpenFolderInExplorer
    if (fn) return await fn(path)
    return ''
  },
}

export const cloneHVWails: ICloneHVService = {
  async checkStock(username: string, password: string, productId: string): Promise<CloneHVStockResult> {
    return await CheckCloneHVStock(username, password, productId) as CloneHVStockResult
  },
}
