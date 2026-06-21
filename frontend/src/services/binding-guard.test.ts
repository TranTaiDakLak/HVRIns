// binding-guard.test.ts
// REGRESSION GUARD cho bug "FE chạy mock mode": sau khi đổi Go package main→app,
// Wails inject bindings vào window.go.app.* (KHÔNG còn window.go.main.*).
// Mọi tham chiếu `go.main` trong source là BUG (làm isWails()=false → toàn app chạy mock,
// các nút gọi backend trực tiếp đều no-op). Test này quét source chặn `go.main` quay lại.

import { describe, it, expect, beforeEach, afterEach } from 'vitest'
import { readdirSync, readFileSync, statSync } from 'node:fs'
import { dirname, resolve, join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { getBridgeMode } from './client'

const SRC_ROOT = resolve(dirname(fileURLToPath(import.meta.url)), '..')

function walk(dir: string): string[] {
  const out: string[] = []
  for (const name of readdirSync(dir)) {
    const p = join(dir, name)
    const st = statSync(p)
    if (st.isDirectory()) {
      if (name === 'node_modules' || name === 'dist') continue
      out.push(...walk(p))
    } else if (/\.(vue|ts)$/.test(name) && !name.endsWith('.test.ts')) {
      out.push(p)
    }
  }
  return out
}

// Bắt: go.main / go?.main / go['main'] / go["main"]
const STALE = /go\s*\??\s*(\.\s*main\b|\[\s*['"]main['"]\s*\])/

describe('Wails binding namespace guard', () => {
  it('KHÔNG còn tham chiếu go.main trong source (phải là go.app sau rename package)', () => {
    const offenders: string[] = []
    for (const file of walk(SRC_ROOT)) {
      const text = readFileSync(file, 'utf8')
      text.split('\n').forEach((line, i) => {
        if (STALE.test(line)) offenders.push(`${file.replace(SRC_ROOT, 'src')}:${i + 1}  ${line.trim()}`)
      })
    }
    expect(offenders, `Tìm thấy tham chiếu go.main (phải đổi sang go.app):\n${offenders.join('\n')}`).toEqual([])
  })
})

describe('getBridgeMode() detect đúng namespace go.app', () => {
  const w = globalThis as any
  beforeEach(() => { delete w.go })
  afterEach(() => { delete w.go })

  it('trả "wails" khi window.go.app tồn tại', () => {
    w.go = { app: { App: {} } }
    expect(getBridgeMode()).toBe('wails')
  })

  it('trả "mock" khi không có window.go', () => {
    expect(getBridgeMode()).toBe('mock')
  })

  it('trả "mock" khi chỉ có go.main cũ (chứng minh KHÔNG còn phụ thuộc go.main)', () => {
    w.go = { main: { App: {} } }
    expect(getBridgeMode()).toBe('mock')
  })
})
