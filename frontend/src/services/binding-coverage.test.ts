import { readFileSync, readdirSync, statSync } from 'fs'
import { join, resolve, relative } from 'path'

const ROOT = resolve(__dirname, '../../..')
const BINDING_DTS = join(ROOT, 'frontend/wailsjs/go/app/App.d.ts')
const SRC_DIR = join(ROOT, 'frontend/src')
const WAILS_DIR = join(ROOT, 'frontend/wailsjs/go/app/App')

function parseBindingMethods(dtsPath: string): Set<string> {
  const content = readFileSync(dtsPath, 'utf-8')
  const set = new Set<string>()
  for (const m of content.matchAll(/^export function (\w+)\(/gm)) {
    set.add(m[1])
  }
  return set
}

function walkFiles(dir: string, exts: string[], exclude: RegExp): string[] {
  const results: string[] = []
  for (const entry of readdirSync(dir)) {
    const full = join(dir, entry)
    if (statSync(full).isDirectory()) {
      results.push(...walkFiles(full, exts, exclude))
    } else if (exts.some(e => full.endsWith(e)) && !exclude.test(full)) {
      results.push(full)
    }
  }
  return results
}

function extractGoAppCalls(files: string[]): Map<string, string[]> {
  const calls = new Map<string, string[]>()
  const re = /go(?:\?\.|\.)app(?:\?\.|\.)App(?:\?\.|\.)(\w+)/g
  for (const f of files) {
    const content = readFileSync(f, 'utf-8')
    let m: RegExpExecArray | null
    re.lastIndex = 0
    while ((m = re.exec(content)) !== null) {
      const method = m[1]
      if (!calls.has(method)) calls.set(method, [])
      calls.get(method)!.push(relative(ROOT, f))
    }
  }
  return calls
}

function extractWailsImports(files: string[]): Map<string, string[]> {
  const imports = new Map<string, string[]>()
  const re = /import\s*\{([^}]+)\}\s*from\s*['"][^'"]*wailsjs\/go\/app\/App['"]/g
  for (const f of files) {
    const content = readFileSync(f, 'utf-8')
    let m: RegExpExecArray | null
    re.lastIndex = 0
    while ((m = re.exec(content)) !== null) {
      for (const name of m[1].split(',').map(s => s.trim()).filter(Boolean)) {
        if (!imports.has(name)) imports.set(name, [])
        imports.get(name)!.push(relative(ROOT, f))
      }
    }
  }
  return imports
}

describe('binding-coverage: FE call sites vs App.d.ts', () => {
  let bindingMethods: Set<string>
  let srcFiles: string[]

  beforeAll(() => {
    bindingMethods = parseBindingMethods(BINDING_DTS)
    srcFiles = walkFiles(SRC_DIR, ['.vue', '.ts'], /\.test\.ts$/)
  })

  it('App.d.ts phải có ít nhất 50 exported function', () => {
    expect(bindingMethods.size).toBeGreaterThanOrEqual(50)
  })

  it('mọi go.app.App.<Method> FE gọi phải tồn tại trong binding', () => {
    const calls = extractGoAppCalls(srcFiles)
    const missing: string[] = []
    for (const [method, sites] of calls) {
      if (!bindingMethods.has(method)) {
        missing.push(`${method} (gọi tại: ${sites.join(', ')})`)
      }
    }
    expect(
      missing,
      `NÚT CHẾT: ${missing.length} method FE gọi KHÔNG có trong binding:\n${missing.join('\n')}`
    ).toHaveLength(0)
  })

  it('mọi method FE gọi được liệt kê (báo cáo)', () => {
    const calls = extractGoAppCalls(srcFiles)
    expect(calls.size).toBeGreaterThan(0)
    // Không assert fail, chỉ để biết số lượng
    console.info(`[binding-coverage] FE gọi ${calls.size} method: ${[...calls.keys()].sort().join(', ')}`)
  })

  it('mọi import wails service từ App phải tồn tại trong binding', () => {
    const wailsFiles = walkFiles(SRC_DIR, ['.ts'], /\.test\.ts$/).filter(f =>
      f.includes('services/wails') || f.includes('services\\wails')
    )
    const imports = extractWailsImports(wailsFiles)
    const missing: string[] = []
    for (const [name, sites] of imports) {
      if (!bindingMethods.has(name)) {
        missing.push(`${name} (import tại: ${sites.join(', ')})`)
      }
    }
    expect(
      missing,
      `IMPORT SAI: ${missing.length} import không tồn tại trong binding:\n${missing.join('\n')}`
    ).toHaveLength(0)
  })
})
