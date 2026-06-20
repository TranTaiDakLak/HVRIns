#!/usr/bin/env python3
"""One-time migration transform: replace hard-coded blue/cyan colors with the
Instagram accent token across a fixed list of target ``.vue`` files.

Implements rules C1-C3 from NVRINS_BUILD_GUIDE.md section 2.8 / requirements
12.1-12.4. All rules are applied case-insensitively.

- C1: replace ``#4fc3f7`` with ``var(--accent)``, skipping occurrences that are
  already inside an existing ``var(--accent, ...)`` fallback.
- C2: replace ``#3b82f6`` with ``var(--accent)``, skipping occurrences that are
  already inside an existing ``var(--accent-hover, ...)`` fallback.
- C3: replace ``rgba(79, 195, 247, A)`` (any internal whitespace) with
  ``rgba(225, 48, 108, A)``, preserving the alpha value ``A``.

The transform is pure text-in/text-out and idempotent: re-running it on
already-transformed files produces no further changes.

Usage::

    python scripts/recolor.py [ROOT]

``ROOT`` defaults to ``e:\\WEMAKE\\HVRIns``. Target file paths are resolved
relative to ``ROOT``. Files are read and written as UTF-8. Per-file change
counts are printed to stdout.
"""

import os
import re
import sys

DEFAULT_ROOT = r"e:\WEMAKE\HVRIns"

# Target files, relative to ROOT.
FILES = [
    "frontend/src/pages/InteractionSetupPage.vue",
    "frontend/src/pages/GeneralSettingsPage.vue",
    "frontend/src/modules/auth-source/components/AuthSourcePanel.vue",
]

# (pattern, replacement) pairs, applied in order with re.IGNORECASE.
#
# The negative lookbehind/lookahead on C1 and C2 keep the rules idempotent and
# guard against rewriting hex literals that already live inside a
# ``var(--accent, #...)`` / ``var(--accent-hover, #...)`` fallback expression.
# The final pair collapses any pre-existing ``var(--accent, #4fc3f7)`` fallback
# down to a bare ``var(--accent)`` so the output converges to a single form.
REPLACEMENTS = [
    (r"(?<!var\(--accent, )#4fc3f7\b(?!\s*\))", "var(--accent)"),
    (r"(?<!var\(--accent-hover, )#3b82f6\b(?!\s*\))", "var(--accent)"),
    (r"rgba\(79\s*,\s*195\s*,\s*247\s*,", "rgba(225,48,108,"),
    (r"var\(--accent,\s*#4fc3f7\)", "var(--accent)"),
]

# Pre-compile patterns once (case-insensitive).
_COMPILED = [(re.compile(pat, flags=re.IGNORECASE), repl) for pat, repl in REPLACEMENTS]


def transform(text):
    """Apply rules C1-C3 to ``text`` and return ``(new_text, change_count)``.

    Pure function: no I/O, no mutation of inputs. ``change_count`` is the total
    number of substitutions made across all rules.
    """
    total = 0
    for pattern, repl in _COMPILED:
        text, count = pattern.subn(repl, text)
        total += count
    return text, total


def recolor_file(path):
    """Read ``path`` (UTF-8), apply the transform, write it back only if
    changed, and return the number of substitutions made."""
    with open(path, "r", encoding="utf-8") as fh:
        original = fh.read()

    new_text, count = transform(original)

    if new_text != original:
        with open(path, "w", encoding="utf-8") as fh:
            fh.write(new_text)
    return count


def main(argv):
    root = argv[1] if len(argv) > 1 else DEFAULT_ROOT

    total = 0
    for rel in FILES:
        path = os.path.join(root, rel)
        try:
            count = recolor_file(path)
        except OSError as exc:
            print("ERROR: {}: {}".format(rel, exc))
            continue
        total += count
        print("{}: {} change(s)".format(rel, count))

    print("Total: {} change(s) across {} file(s)".format(total, len(FILES)))
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv))
