// legacy-import.wails.ts — Wails binding cho ParseLegacyConfig / ImportLegacyConfig

import type { ILegacyImportService, LegacyParseResult } from '../contracts'
import { ParseLegacyConfig, ImportLegacyConfig } from '../../../wailsjs/go/main/App'

export const legacyImportWails: ILegacyImportService = {
  async parse(generalJSON: string, interactionJSON: string): Promise<LegacyParseResult> {
    // Cast: Wails generated models use `string` for status; contracts use a string union
    return ParseLegacyConfig(generalJSON, interactionJSON) as unknown as LegacyParseResult
  },

  async apply(generalJSON: string, interactionJSON: string): Promise<string> {
    return ImportLegacyConfig(generalJSON, interactionJSON)
  },
}
