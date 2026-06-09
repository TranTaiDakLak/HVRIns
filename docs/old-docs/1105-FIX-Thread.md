# 1105-FIX Thread

## Problem

When Register runs with many concurrent threads, the UI can display the wrong User-Agent when an account succeeds, fails, or when a slot moves to the next account.

Example: Reg is set to `Fb_509 + Original UA`, but the UA column jumps to:

```txt
FBAV/560.0.0.26.63;FBBV/959741200
FBAV/559.1.0.52.72;FBBV/959738728
```

This does not always mean the first Reg request was sent with the wrong UA. In this code pattern, the common causes are shared config/platform variables across goroutines, or Verify/Token events writing the Verify UA into the Register UA column.

## Symptoms

- One thread may look correct, but 50-100 threads start showing mixed UA versions.
- The account starts with the correct UA, then changes at `success`, `failed`, token fetch, verify, or when the slot starts the next account.
- The wrong UA is often from the Verify API, for example `559` with random devices.
- `Original UA` is enabled, but random devices appear, such as Pixel, Xiaomi, or Vivo.
- The same slot alternates between the correct version and another version.

## Root Causes

### 1. Config/platform variables are shared across goroutines

Buggy pattern:

```go
interactionCfg := a.LoadInteractionConfig()
regPlatform := interactionCfg.ApiRegPlatform

for {
    interactionCfg = a.LoadInteractionConfig()
    if interactionCfg.ApiRegPlatform != "" {
        regPlatform = interactionCfg.ApiRegPlatform
    }

    go func(slotIdx int, prof RegInput) {
        // BUG: this goroutine reads interactionCfg/regPlatform from the outer loop.
        // Multiple goroutines read/write the same variables, causing logic races.
        runRegister(interactionCfg, regPlatform, prof)
    }(slotIdx, profile)
}
```

When many accounts run in parallel, goroutine A may be registering with `s509`, while goroutine B reloads config or the UI changes platform to `s560`. Goroutine A can then read the shared variables and emit/process the wrong UA.

### 2. Keep-session reload does not re-apply per-platform UA config

Buggy pattern:

```go
interactionCfg = a.LoadInteractionConfig()
```

If the project has `RegPlatformUA[platform]`, every reload must re-apply the UA config for the platform currently owned by that account. Otherwise, the next register attempt in the same session can fall back to global/default UA settings.

### 3. Verify/Token events overwrite the Register UA column

Buggy pattern:

```go
runtime.EventsEmit(ctx, "register:status", map[string]any{
    "userAgent": verifyUA,
})
```

The Register table must show the Reg UA. Verify UA should only be used internally for verify requests or shown in the Verify pane.

## Fix

### 1. Snapshot config/platform before spawning the goroutine

Use this pattern:

```go
interactionCfg := a.LoadInteractionConfig()
regPlatform := interactionCfg.ApiRegPlatform
interactionCfg = applyRegPlatformUAConfig(interactionCfg, regPlatform)

for {
    latestCfg := a.LoadInteractionConfig()
    latestPlatform := latestCfg.ApiRegPlatform
    latestCfg = applyRegPlatformUAConfig(latestCfg, latestPlatform)

    accountCfg := latestCfg
    accountRegPlatform := latestPlatform

    go func(slotIdx int, prof RegInput, cfg InteractionConfig, platform string) {
        runRegisterWithSnapshot(slotIdx, prof, cfg, platform)
    }(slotIdx, profile, accountCfg, accountRegPlatform)
}
```

Inside the goroutine, use only the `cfg` and `platform` passed as parameters. Do not read `interactionCfg` or `regPlatform` directly from the outer loop.

### 2. Add a helper to apply Reg platform UA config

```go
func applyRegPlatformUAConfig(cfg InteractionConfig, platform string) InteractionConfig {
    if platform == "" {
        platform = cfg.ApiRegPlatform
    }
    if uaCfg, ok := cfg.RegPlatformUA[platform]; ok {
        cfg.BuildUA = uaCfg.BuildUA
        cfg.AddVirtualSpecAndroid = uaCfg.AddVirtualSpecAndroid
        cfg.UseOriginalUA = uaCfg.UseOriginalUA
        cfg.ReplaceCarrier = uaCfg.ReplaceCarrier
        if uaCfg.UaPoolKey != "" {
            cfg.UaPoolKey = uaCfg.UaPoolKey
        }
    }
    return cfg
}
```

Every config reload inside the goroutine should use:

```go
cfg = applyRegPlatformUAConfig(a.LoadInteractionConfig(), platform)
```

Do not use:

```go
cfg = a.LoadInteractionConfig()
```

### 3. Register table should emit only the Reg UA

When fetching an Android token or running verify:

```go
tokenUA := pickVerifyUA(...)
fetchToken(tokenUA)

runtime.EventsEmit(ctx, "register:status", map[string]any{
    "userAgent": regUA, // prof.UserAgent from the Reg flow
    "msg": "[Token] OK",
})
```

Do not emit:

```go
"userAgent": tokenUA
"userAgent": verifyUA
```

### 4. Keep UA must include platform

If the app has a `Keep UA` feature, do not store only the UA string by slot. Store the platform too:

```go
type regSlotUA struct {
    Platform string
    UA       string
}

regUABySlot.Store(slotIdx, regSlotUA{
    Platform: platform,
    UA:       prof.UserAgent,
})
```

When reusing:

```go
if kept, ok := v.(regSlotUA); ok && kept.Platform == platform && kept.UA != "" {
    prof.UserAgent = kept.UA
} else {
    regUABySlot.Delete(slotIdx)
}
```

Without the platform check, changing from `s503` to `s509` in the same run can reuse the old UA.

### 5. Frontend should ignore late status events after a slot is done

If a status event arrives after `success/failed`, it must not reset the slot back to running or overwrite the UA:

```ts
if (!data.reset && existing && (existing.status === 'success' || existing.status === 'failed')) {
  continue
}
```

Only reset a slot when the backend starts a new account and sends:

```ts
reset: true
```

## Porting Checklist

- Find config/platform variables declared outside register goroutines.
- If a goroutine reads those outer variables directly, pass snapshots as goroutine parameters instead.
- Each account must have its own `accountCfg` and `accountRegPlatform`.
- Config reload inside keep-session must re-apply per-platform UA config for that account's platform.
- `register:status` and `register:account-done` must emit only the Reg UA.
- Token/Verify UA must not overwrite the Register UA column.
- Keep UA cache must include platform.
- Frontend must ignore late status events after a slot is done.
- Test with 100 threads, `Original UA` enabled, and a specific version such as `s509`.
- Watch the Register UA column after success/fail/next slot: the version must stay fixed.

## Quick Checks

```powershell
rg -n "go func\\(|interactionCfg = .*LoadInteractionConfig|regPlatform =|register:status|register:account-done|userAgent" app.go
```

```powershell
go test ./...
```

```powershell
cd frontend
npm run build
```
