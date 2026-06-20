// verify-runner.wails.ts — Wails bridge cho RunVerify, StopVerify, RunRegister, StopRegister
import type { IVerifyRunnerService, VerifyRunConfig } from '@/bridge/contracts'
import type { VerifyConfig } from '@/types/interaction.types'
import {
  RunVerify,
  StopVerify,
  IsVerifyRunning,
  RunRegister,
  StopRegister,
  LoadInteractionConfig,
  SimulatePlatformUA,
} from '../../../wailsjs/go/main/App'
import { main } from '../../../wailsjs/go/models'

export const verifyRunnerWails: IVerifyRunnerService = {
  async run(config: VerifyRunConfig): Promise<string> {
    const cfg = new main.VerifyRunConfig({
      accountIds: config.accountIds,
      maxThreads: config.maxThreads,
      verifyConfig: config.verifyConfig,
      outputPath: config.outputPath,
      proxy: (config as any).proxy ?? '',
    })
    return RunVerify(cfg)
  },

  async stop(): Promise<string> {
    return StopVerify()
  },

  async isRunning(): Promise<boolean> {
    return IsVerifyRunning()
  },

  async runRegister(maxThreads: number): Promise<string> {
    return RunRegister(maxThreads)
  },

  async stopRegister(): Promise<string> {
    return StopRegister()
  },

  async loadInteractionConfig(): Promise<VerifyConfig> {
    return LoadInteractionConfig() as unknown as Promise<VerifyConfig>
  },

  async simulatePlatformUA(platform: string, cfg: import('../../types/interaction.types').PlatformUAConfig): Promise<string> {
    return SimulatePlatformUA(platform, cfg as any)
  },
}
