#!/usr/bin/env python3
"""Author-side static checks for the D2 P0 acceptance PowerShell package.

NOT a PowerShell parser and NOT a substitute for Test-D2Package.ps1 on Windows PowerShell 5.1.
It catches a few defect classes without PowerShell being available:

  * exact expected file set (no missing or extra script files);
  * ASCII-only source (Windows PowerShell 5.1 reads BOM-less files as ANSI);
  * structural balance of (), [], {} outside strings, comments and here-strings;
  * "$name:" inside double-quoted strings (PowerShell reads it as a drive/scope-qualified variable);
  * variables in one function whose names differ only by case (PowerShell names are
    case-insensitive, so they are the same variable);
  * a logical/comparison operator directly after an unparenthesized command on the same
    statement (the Patch F "Test-Path ... -or" defect), as a heuristic.

Usage: python3 -I tests/static_check.py [package_dir]
Exit 0 only when every check passes.
"""
import os
import re
import sys

EXPECTED = {'D2Evidence.psm1', 'D2Host.psm1', 'Test-D2Package.ps1', os.path.join('tests', 'D2Evidence.Tests.ps1')}
SCOPES = {'env', 'script', 'global', 'local', 'private', 'using', 'function', 'variable'}
OPS = r'(?:c|i)?(?:or|and|xor|eq|ne|gt|ge|lt|le|like|notlike|match|notmatch|contains|notcontains|in|notin)'


def strip(src):
    """Return src with strings/comments/here-strings replaced by spaces (newlines kept), and the
    list of double-quoted string bodies with their line numbers."""
    out, dq, i, n, line = [], [], 0, len(src), 1
    while i < n:
        ch = src[i]
        two = src[i:i + 2]
        if two in ("@'", '@"') and src[i + 2:i + 3] in ('\n', '\r'):
            q = two[1]
            end = src.find('\n' + q + '@', i)
            if end < 0:
                raise ValueError(f'unterminated here-string at line {line}')
            body = src[i:end + 3]
            if q == '"':
                dq.append((line, body))
            out.append(re.sub(r'[^\n]', ' ', body))
            line += body.count('\n')
            i = end + 3
            continue
        if two == '<#':
            end = src.find('#>', i)
            if end < 0:
                raise ValueError(f'unterminated block comment at line {line}')
            body = src[i:end + 2]
            out.append(re.sub(r'[^\n]', ' ', body))
            line += body.count('\n')
            i = end + 2
            continue
        if ch == '#':
            end = src.find('\n', i)
            end = n if end < 0 else end
            out.append(' ' * (end - i))
            i = end
            continue
        if ch == "'":
            j = i + 1
            while True:
                if j >= n:
                    raise ValueError(f'unterminated single-quoted string at line {line}')
                if src[j] == "'":
                    if src[j + 1:j + 2] == "'":
                        j += 2
                        continue
                    break
                j += 1
            body = src[i:j + 1]
            out.append(re.sub(r'[^\n]', ' ', body))
            line += body.count('\n')
            i = j + 1
            continue
        if ch == '"':
            j, depth = i + 1, 0
            while True:
                if j >= n:
                    raise ValueError(f'unterminated double-quoted string at line {line}')
                c = src[j]
                if c == '`':
                    j += 2
                    continue
                if src[j:j + 2] == '$(':
                    depth += 1
                    j += 2
                    continue
                if c == ')' and depth:
                    depth -= 1
                elif c == '"' and not depth:
                    if src[j + 1:j + 2] == '"':
                        j += 2
                        continue
                    break
                j += 1
            body = src[i:j + 1]
            dq.append((line, body))
            out.append(re.sub(r'[^\n]', ' ', body))
            line += body.count('\n')
            i = j + 1
            continue
        if ch == '`':
            out.append('  ')
            i += 2
            continue
        if ch == '\n':
            line += 1
        out.append(ch)
        i += 1
    return ''.join(out), dq


def balance(code):
    pairs = {')': '(', ']': '[', '}': '{'}
    stack, line = [], 1
    for ch in code:
        if ch == '\n':
            line += 1
        elif ch in '([{':
            stack.append((ch, line))
        elif ch in pairs:
            if not stack or stack[-1][0] != pairs[ch]:
                return f'unbalanced {ch!r} at line {line}'
            stack.pop()
    return f'unclosed {stack[-1][0]!r} from line {stack[-1][1]}' if stack else None


def functions(code):
    """Yield (name, start_line, body) for each top-level 'function' definition in stripped code."""
    for m in re.finditer(r'(?im)^\s*function\s+([A-Za-z0-9_-]+)', code):
        start = code.find('{', m.end())
        depth, j = 0, start
        while j < len(code):
            if code[j] == '{':
                depth += 1
            elif code[j] == '}':
                depth -= 1
                if depth == 0:
                    break
            j += 1
        yield m.group(1), code.count('\n', 0, m.start()) + 1, code[start:j + 1]


def main():
    root = sys.argv[1] if len(sys.argv) > 1 else os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
    problems = []
    found = set()
    for d, _, files in os.walk(root):
        for f in files:
            if f.endswith(('.ps1', '.psm1', '.psd1')):
                found.add(os.path.relpath(os.path.join(d, f), root))
    for x in sorted(EXPECTED - found):
        problems.append(f'missing {x}')
    for x in sorted(found - EXPECTED):
        problems.append(f'unexpected {x}')
    if not found:
        problems.append('no script files found')
    for rel in sorted(found & EXPECTED):
        raw = open(os.path.join(root, rel), 'rb').read()
        bad = [i for i, b in enumerate(raw) if b > 0x7E or (b < 0x20 and b not in (9, 10, 13))]
        if bad:
            problems.append(f'{rel}: non-ASCII/control byte at offset {bad[0]}')
        src = raw.decode('ascii', 'replace')
        try:
            code, dq = strip(src)
        except ValueError as e:
            problems.append(f'{rel}: {e}')
            continue
        b = balance(code)
        if b:
            problems.append(f'{rel}: {b}')
        for line, body in dq:
            for m in re.finditer(r'(?<!`)\$([A-Za-z_][A-Za-z0-9_]*):(?![A-Za-z_])', body):
                if m.group(1).lower() not in SCOPES:
                    problems.append(f'{rel} line {line}: "${m.group(1)}:" in a double-quoted string; use ${{{m.group(1)}}}:')
        # command extent scan: a Verb-Noun command at command position owns every token up to its
        # closing ')', '|', ';' or '}' at depth 0; an operator inside that extent binds as a parameter.
        for i, ln in enumerate(code.split('\n'), 1):
            for m in re.finditer(r'(?:^|(?<=[({|;=&,]))\s*([A-Z][a-z]+-[A-Z][A-Za-z0-9]+)\b', ln):
                j, depth = m.end(), 0
                while j < len(ln):
                    ch = ln[j]
                    if ch in '([{':
                        depth += 1
                    elif ch in ')]}':
                        if depth == 0:
                            break
                        depth -= 1
                    elif ch in '|;' and depth == 0:
                        break
                    elif depth == 0:
                        op = re.match(r'\s-' + OPS + r'\b', ln[j:])
                        if op and ln[j:j + 1].isspace():
                            problems.append(f'{rel} line {i}: operator{op.group(0)} is inside the extent of command {m.group(1)}; parenthesize the command')
                            break
                    j += 1
        for name, start, body in functions(code):
            seen = {}
            for m in re.finditer(r'\$((?:script|global|local|private):)?([A-Za-z_][A-Za-z0-9_]*)', body):
                scope, v = (m.group(1) or '').lower(), m.group(2)
                if v.lower() in ('_', 'true', 'false', 'null', 'args', 'input', 'this', 'psscriptroot', 'pscmdlet', 'env'):
                    continue
                seen.setdefault(scope + v.lower(), set()).add(v)
            for k, spell in seen.items():
                if len(spell) > 1:
                    problems.append(f'{rel} function {name} (line {start}): variables {sorted(spell)} are one case-insensitive variable')
    lists = {}
    for rel, pat in (('D2Host.psm1', r'\$script:PkgFiles = @\(([^)]*)\)'), ('Test-D2Package.ps1', r'\$Expected = @\(([^)]*)\)')):
        try:
            m = re.search(pat, open(os.path.join(root, rel)).read())
            lists[rel] = {x.replace('\\\\', '\\').replace('\\', os.sep) for x in re.findall(r"'([^']+)'", m.group(1))} if m else None
        except OSError:
            lists[rel] = None
    for rel, got in lists.items():
        if got != EXPECTED:
            problems.append(f'{rel}: package file list {sorted(got) if got else got} differs from {sorted(EXPECTED)}')
    if problems:
        print(f'STATIC CHECK: FAIL ({len(problems)})')
        for p in problems:
            print('  ' + p)
        return 1
    print(f'STATIC CHECK: PASS ({len(found)} files; not a PowerShell parse)')
    return 0


if __name__ == '__main__':
    sys.exit(main())
