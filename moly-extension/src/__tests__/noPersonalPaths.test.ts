import { describe, it, expect } from 'vitest';
import { readdirSync, readFileSync, statSync } from 'node:fs';
import { join } from 'node:path';

const walk = (dir: string): string[] =>
  readdirSync(dir).flatMap((f) => {
    const p = join(dir, f);
    if (statSync(p).isDirectory()) return f === '__tests__' ? [] : walk(p);
    return /\.(ts|tsx)$/.test(f) ? [p] : [];
  });

describe('shipped text', () => {
  it('names no folder of the developer', () => {
    const bad = walk(join(__dirname, '..')).filter((p) => /vs_projects|\/home\/[a-z]/.test(readFileSync(p, 'utf8')));
    expect(bad).toEqual([]);
  });
});
