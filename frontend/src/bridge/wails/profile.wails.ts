// profile.wails.ts — Wails bridge cho profile management API
import type { IProfileService, ProfileInfo } from '../contracts'
import {
  ListProfiles,
  GetActiveProfileID,
  SetActiveProfile,
  CreateProfile,
  CloneProfile,
  RenameProfile,
  DeleteProfile,
} from '../../../wailsjs/go/main/App'

export const profileWails: IProfileService = {
  async list(): Promise<ProfileInfo[]> {
    const profiles = await ListProfiles()
    return profiles.map(p => ({ id: p.id, name: p.name }))
  },
  async getActiveId(): Promise<string> {
    return GetActiveProfileID()
  },
  async setActive(id: string): Promise<string> {
    return SetActiveProfile(id)
  },
  async create(name: string): Promise<string> {
    return CreateProfile(name)
  },
  async clone(name: string): Promise<string> {
    return CloneProfile(name)
  },
  async rename(id: string, name: string): Promise<string> {
    return RenameProfile(id, name)
  },
  async delete(id: string): Promise<string> {
    return DeleteProfile(id)
  },
}
