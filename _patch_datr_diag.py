"""Patch all register.go to add explicit datr=NONE diagnostic when pool is empty.

Targets pattern A (inside `if profile.MachineID == "" && SharedPool != nil`):
    if poolDatr := SharedPool.GetNext(slotIdx); poolDatr != "" {
        profile.MachineID = poolDatr
        s, f, u, used := SharedPool.GetStats(poolDatr)
        notify(fmt.Sprintf("[TAG] New initial ...", ...))
    }
→ add else branch with explicit "Pool EMPTY" log.

Targets pattern B (POST log datr info):
    datrInfo := ""
→ change to:
    datrInfo := " | datr=NONE"
"""
import os
import re
import sys

ROOT = r"E:/WEMAKE/NullCoreSummer/internal/facebook/register"

# Match the GetNext block. Capture indent + tag.
# Handles both multi-line and single-line notify forms.
PAT_GETNEXT = re.compile(
    r'(\t+if poolDatr := SharedPool\.GetNext\(slotIdx\); poolDatr != "" \{\n'
    r'\t+profile\.MachineID = poolDatr\n'
    r'\t+s, f, u, used := SharedPool\.GetStats\(poolDatr\)\n'
    r'\t+notify\(fmt\.Sprintf\("(\[[^\]]+\]) New initial[^"]*",[^)]*\)\)\n'
    r'(\t+)\}\n)'
)

# Match the datrInfo := "" line.
PAT_DATRINFO = re.compile(r'(\t+)datrInfo := ""\n')

def patch_file(path):
    with open(path, 'r', encoding='utf-8') as f:
        src = f.read()
    original = src
    # Apply Pattern A: add else branch
    def repl_getnext(m):
        block = m.group(1)
        tag = m.group(2)  # e.g., [S415]
        indent = m.group(3)  # indent of the closing }
        # Build else branch
        else_branch = (
            f'{indent}}} else {{\n'
            f'{indent}\tnotify(fmt.Sprintf("{tag} ⚠️ Pool EMPTY (slot=%d) — reg KHÔNG có datr!", slotIdx))\n'
            f'{indent}}}\n'
        )
        # Replace the trailing "}\n" of the if-block with our else branch
        # The block ends with "\t+}\n" — replace that final } with } else {...}
        new_block = block.rstrip()
        # Strip trailing }
        assert new_block.endswith('}'), f"Block does not end with closing brace:\n{block!r}"
        new_block = new_block[:-1].rstrip() + '\n'
        return new_block + else_branch

    src = PAT_GETNEXT.sub(repl_getnext, src, count=1)

    # Apply Pattern B: change datrInfo := "" → datrInfo := " | datr=NONE"
    src = PAT_DATRINFO.sub(r'\1datrInfo := " | datr=NONE"\n', src, count=1)

    if src != original:
        with open(path, 'w', encoding='utf-8', newline='\n') as f:
            f.write(src)
        return True
    return False

def main():
    patched = 0
    skipped = 0
    failed = 0
    for entry in sorted(os.listdir(ROOT)):
        d = os.path.join(ROOT, entry)
        if not os.path.isdir(d):
            continue
        if entry in ('chrome', 'android', 'web', 'webandroid'):
            continue
        rf = os.path.join(d, 'register.go')
        if not os.path.exists(rf):
            continue
        try:
            if patch_file(rf):
                patched += 1
                print(f"OK   {entry}")
            else:
                skipped += 1
                print(f"SKIP {entry} (no change)")
        except Exception as e:
            failed += 1
            print(f"FAIL {entry}: {e}")
    print(f"\nTotal: patched={patched} skipped={skipped} failed={failed}")

if __name__ == '__main__':
    main()
