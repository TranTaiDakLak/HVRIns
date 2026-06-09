// file-dialog.mock.ts — Mock implementation cho IFileDialogService + ICloneHVService
import type { IFileDialogService, ICloneHVService, ImportResult, CloneHVStockResult } from '../contracts'

export const fileDialogMock: IFileDialogService = {
  async openFolder(): Promise<string> {
    return '/mock/selected/folder'
  },

  async openTextFile(): Promise<string> {
    return 'mock_uid|mock_pass|mock_cookie\nmock_uid2|mock_pass2|mock_cookie2'
  },

  async openFilePath(): Promise<string> {
    return 'C:\\mock\\ua_iphone.txt'
  },

  async readTextFile(_path: string): Promise<string> {
    return 'Mozilla/5.0 (iPhone; CPU iPhone OS 17_2 like Mac OS X) AppleWebKit/605.1.15\nMozilla/5.0 (iPhone; CPU iPhone OS 16_0 like Mac OS X) AppleWebKit/605.1.15'
  },

  async validatePath(path: string): Promise<string> {
    if (!path) return 'Đường dẫn không được trống'
    return ''
  },

  async getAccountSourceFolder(): Promise<string> {
    return ''
  },

  async setAccountSourceFolder(_path: string): Promise<ImportResult> {
    return { imported: 0, errors: [] }
  },

  async refreshAccountSource(): Promise<ImportResult> {
    return { imported: 0, errors: ['Mock: không có account mới'] }
  },

  async openFolderInExplorer(_path: string): Promise<string> {
    return ''
  },
}

export const cloneHVMock: ICloneHVService = {
  async checkStock(_username: string, _password: string, _productId: string): Promise<CloneHVStockResult> {
    return {
      name: 'Mock Facebook Account',
      amount: '999',
      price: 5000,
      error: '',
    }
  },
}
