/// <reference types="node" />

import { readdirSync, readFileSync, statSync } from 'fs'
import { dirname, extname, join, resolve } from 'path'
import { fileURLToPath } from 'url'
import { describe, expect, it } from 'vitest'

const srcDir = resolve(dirname(fileURLToPath(import.meta.url)), '..', '..')
const sourceExtensions = new Set(['.ts', '.vue'])
const forbiddenImportPattern = /@arco-design\/web-vue\/es\/icon/

function collectSourceFiles(dir: string): string[] {
  return readdirSync(dir).flatMap((entry: string) => {
    const filepath = join(dir, entry)
    const stats = statSync(filepath)

    if (stats.isDirectory()) {
      return collectSourceFiles(filepath)
    }

    return sourceExtensions.has(extname(filepath)) ? [filepath] : []
  })
}

describe('local icon migration', () => {
  it('does not directly import Arco icon modules in frontend source', () => {
    const files = collectSourceFiles(srcDir)
    const offenders = files.filter((filepath) =>
      forbiddenImportPattern.test(readFileSync(filepath, 'utf8')),
    )

    expect(offenders).toEqual([])
  })
})
