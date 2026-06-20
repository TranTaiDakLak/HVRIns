// legacy-import.mock.ts — Mock cho legacy import (dev/preview mode)

import type { ILegacyImportService, LegacyParseResult } from '@/services/contracts'

const SAMPLE_RESULT: LegacyParseResult = {
  error: '',
  report: {
    mappedOk: [
      { legacyKey: 'threadRequest', newPath: 'profile.runtime.threadRequest', displayValue: '5', status: 'ok', note: '' },
      { legacyKey: 'delayRequest', newPath: 'profile.runtime.delayRequest', displayValue: '1000 ms', status: 'ok', note: '' },
      { legacyKey: 'threadCheckInfo', newPath: 'profile.runtime.threadCheckInfo', displayValue: '3', status: 'ok', note: '' },
      { legacyKey: 'loginPlatform', newPath: 'global.loginPlatform', displayValue: 'facebook', status: 'ok', note: '' },
      { legacyKey: 'loginMethod', newPath: 'global.loginMethod', displayValue: '0', status: 'ok', note: '' },
      { legacyKey: 'ipProvider', newPath: 'profile.proxy.provider', displayValue: 'tinsoft', status: 'ok', note: '' },
      { legacyKey: 'proxyList', newPath: 'profile.proxy.proxyList', displayValue: '12 proxy entries', status: 'ok', note: '' },
      { legacyKey: 'verifyEnabled', newPath: 'profile.verify.enabled', displayValue: 'true', status: 'ok', note: '' },
      { legacyKey: 'mailProvider', newPath: 'profile.mail.provider', displayValue: 'store1s', status: 'ok', note: '' },
      { legacyKey: 'createEnabled', newPath: 'profile.register.enabled', displayValue: 'false', status: 'ok', note: '' },
      { legacyKey: 'saveRunColumn', newPath: 'global.saveRunColumn', displayValue: 'true', status: 'ok', note: '' },
      { legacyKey: 'backupDB', newPath: 'global.backupDB', displayValue: 'false', status: 'ok', note: '' },
    ],
    needsConfirm: [
      {
        legacyKey: 'accountSourcePath',
        newPath: 'profile.account.folderPath',
        displayValue: 'C:\\Users\\Admin\\accounts\\fb',
        status: 'confirm',
        note: 'Kiểm tra thư mục còn tồn tại trên máy hiện tại',
      },
      {
        legacyKey: 'outputPath',
        newPath: 'profile.output.verifyPath',
        displayValue: 'D:\\output\\live',
        status: 'confirm',
        note: 'Kiểm tra thư mục lưu kết quả còn tồn tại',
      },
    ],
    sensitive: [
      { legacyKey: 'cloneHvPassword', newPath: 'profile.account.cloneHv.password', displayValue: '***', status: 'sensitive', note: 'Mật khẩu CloneHV — được import, nên đổi sau' },
      { legacyKey: 'store1sApiKey', newPath: 'profile.mail.providers.store1s.apiKey', displayValue: '***', status: 'sensitive', note: 'API key Store1s' },
    ],
    unsupported: [],
    parseErrors: [],
  },
}

export const legacyImportMock: ILegacyImportService = {
  async parse(_generalJSON: string, _interactionJSON: string): Promise<LegacyParseResult> {
    await new Promise(r => setTimeout(r, 600))
    return SAMPLE_RESULT
  },

  async apply(_generalJSON: string, _interactionJSON: string): Promise<string> {
    await new Promise(r => setTimeout(r, 400))
    return 'OK'
  },
}
