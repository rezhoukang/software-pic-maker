import { test, expect, type Page, type Locator } from '@playwright/test'
import fs from 'node:fs'
import path from 'node:path'
import { inflateRawSync } from 'node:zlib'

const library = JSON.parse(fs.readFileSync(new URL('../../shared/library.json', import.meta.url), 'utf8'))

function unzip(data: Buffer): Map<string, Buffer> {
  const end = data.lastIndexOf(Buffer.from([0x50, 0x4b, 0x05, 0x06]))
  expect(end).toBeGreaterThan(0)
  const result = new Map<string, Buffer>()
  let offset = data.readUInt32LE(end + 16)
  for (let i = 0; i < data.readUInt16LE(end + 10); i++) {
    expect(data.readUInt32LE(offset)).toBe(0x02014b50)
    const method = data.readUInt16LE(offset + 10), size = data.readUInt32LE(offset + 20)
    const nameLength = data.readUInt16LE(offset + 28), extra = data.readUInt16LE(offset + 30), comment = data.readUInt16LE(offset + 32)
    const name = data.subarray(offset + 46, offset + 46 + nameLength).toString('utf8')
    const local = data.readUInt32LE(offset + 42)
    const start = local + 30 + data.readUInt16LE(local + 26) + data.readUInt16LE(local + 28)
    const compressed = data.subarray(start, start + size)
    result.set(name, method === 8 ? inflateRawSync(compressed) : compressed)
    offset += 46 + nameLength + extra + comment
  }
  return result
}
async function setup(page: Page) {
  await page.goto('/')
  await expect(page.locator('.x6-node').first()).toBeVisible()
  await page.evaluate(async diagram => {
    const headers = { 'Content-Type': 'application/json', 'X-EZ-Workspace': localStorage.getItem('ezbitu-preview-workspace')! }
    const call = async (method: string, args: any) => {
      const response = await fetch('/api/workspace/' + method, { method: 'POST', headers, body: JSON.stringify(args) })
      if (!response.ok) throw new Error(await response.text())
      return response.json()
    }
    await call('create-folder', { parent: '', name: '批量项目' })
    await call('create-folder', { parent: '批量项目', name: '子目录' })
    await call('create-file', { parent: '批量项目', name: '同名图', raw: JSON.stringify(diagram) })
    diagram.id = 'second-export'; diagram.title = '第二张图'
    await call('create-file', { parent: '批量项目/子目录', name: '同名图', raw: JSON.stringify(diagram) })
  }, structuredClone(library[3]))
}
async function openDialog(page: Page) {
  await page.getByTestId('export-button').click()
  const dialog = page.getByRole('dialog', { name: '导出', exact: true })
  await expect(dialog).toBeVisible()
  await expect(dialog.locator('.export-summary')).not.toContainText('正在读取')
  return dialog
}
async function choose(page: Page, dialog: Locator, index: number, text: string) {
  await dialog.locator('.n-select').nth(index).click()
  await page.locator('.n-base-select-option:visible').filter({ hasText: text }).filter({ has: page.locator('.n-base-select-option__content') }).first().click()
}
async function chooseFolder(page: Page, dialog: Locator, folder = '批量项目') {
  await choose(page, dialog, 0, '选择文件夹')
  await dialog.getByRole('button', { name: '选择文件夹', exact: true }).click()
  await page.getByRole('dialog', { name: '选择导出文件夹', exact: true }).getByRole('button', { name: '主目录 / ' + folder, exact: true }).click()
}

test('scope and output controls depend on the actual number of diagrams and do not remember choices', async ({ page }) => {
  await setup(page)
  let dialog = await openDialog(page)
  await expect(dialog).toContainText('请选择导出范围')
  await expect(dialog).toContainText('请选择导出形式')
  await choose(page, dialog, 0, '当前图')
  await expect(dialog.locator('.export-summary')).toContainText('共 1 张图')
  await expect(dialog.getByRole('checkbox')).toHaveCount(0)
  await choose(page, dialog, 0, '当前文件夹')
  await expect(dialog.locator('.export-summary')).toContainText('共 8 张图')
  await expect(dialog.getByRole('checkbox', { name: '文件夹', exact: true })).toBeChecked()
  await choose(page, dialog, 0, '全部文件')
  await expect(dialog.locator('.export-summary')).toContainText('共 10 张图')
  await chooseFolder(page, dialog)
  await expect(dialog.locator('.export-summary')).toContainText('共 2 张图')
  await choose(page, dialog, 1, 'SVG 矢量图')
  await dialog.getByRole('checkbox', { name: '文件夹', exact: true }).uncheck()
  await expect(dialog.getByRole('button', { name: '导出', exact: true })).toBeDisabled()
  await expect(dialog).toContainText('请至少选择一种输出方式')
  await dialog.getByRole('checkbox', { name: 'ZIP 压缩包', exact: true }).check()
  await expect(dialog.getByRole('button', { name: '导出', exact: true })).toBeEnabled()
  await dialog.getByRole('button', { name: '取消', exact: true }).click()
  dialog = await openDialog(page)
  await expect(dialog).toContainText('请选择导出范围')
  await expect(dialog).toContainText('请选择导出形式')
  await choose(page, dialog, 0, '当前文件夹')
  await expect(dialog.getByRole('checkbox', { name: '文件夹', exact: true })).toBeChecked()
  await expect(dialog.getByRole('checkbox', { name: 'ZIP 压缩包', exact: true })).not.toBeChecked()
  await dialog.screenshot({ path: 'test-results/export/dialog.png' })
})

test('batch JSON ZIP preserves nested paths and current unsaved content', async ({ page }) => {
  await page.goto('/')
  await expect(page.locator('.x6-node').first()).toBeVisible()
  const title = page.getByRole('textbox', { name: '图表名称', exact: true })
  await title.fill('尚未保存的标题'); await title.press('Tab')
  const dialog = await openDialog(page)
  await choose(page, dialog, 0, '全部文件')
  await choose(page, dialog, 1, 'JSON 设计数据')
  await dialog.getByRole('checkbox', { name: '文件夹', exact: true }).uncheck()
  await dialog.getByRole('checkbox', { name: 'ZIP 压缩包', exact: true }).check()
  const download = page.waitForEvent('download')
  await dialog.getByRole('button', { name: '导出', exact: true }).click()
  const saved = await download
  const target = path.resolve('test-results/export/current.zip'); await saved.saveAs(target)
  const entries = unzip(fs.readFileSync(target))
  expect(entries.size).toBe(8)
  expect([...entries.keys()].every(name => name.startsWith('示例图/'))).toBe(true)
  expect([...entries.values()].map(data => JSON.parse(data.toString()).title)).toContain('尚未保存的标题')
  await expect(page.getByTestId('file-save-status')).toContainText('未保存')
  await expect(dialog).toBeHidden()
})

test('PNG folder and ZIP are both produced and existing exports are preserved', async ({ page }) => {
  await page.addInitScript(() => {
    const files: Record<string, number[]> = { '导出-批量项目/保留.txt': [111, 108, 100] }
    const directories = new Set(['', '导出-批量项目'])
    const handle = (base: string): any => ({
      async getDirectoryHandle(name: string, options?: { create?: boolean }) {
        const full = base ? base + '/' + name : name
        if (files[full]) throw new DOMException('file', 'TypeMismatchError')
        if (!directories.has(full) && !options?.create) throw new DOMException('missing', 'NotFoundError')
        directories.add(full); return handle(full)
      },
      async getFileHandle(name: string, options?: { create?: boolean }) {
        const full = base ? base + '/' + name : name
        if (directories.has(full)) throw new DOMException('directory', 'TypeMismatchError')
        if (!files[full] && !options?.create) throw new DOMException('missing', 'NotFoundError')
        return { async createWritable() { return { async write(data: Uint8Array) { files[full] = Array.from(data) }, async close() {} } } }
      },
    })
    ;(window as any).showDirectoryPicker = async () => handle('')
    ;(window as any).exportedFiles = files
  })
  await setup(page)
  const dialog = await openDialog(page)
  await chooseFolder(page, dialog)
  await choose(page, dialog, 1, 'PNG 图片')
  await dialog.getByRole('checkbox', { name: 'ZIP 压缩包', exact: true }).check()
  await dialog.getByRole('button', { name: '导出', exact: true }).click()
  await expect(dialog).toBeHidden()
  const files = await page.evaluate(() => (window as any).exportedFiles) as Record<string, number[]>
  expect(files['导出-批量项目/保留.txt']).toEqual([111, 108, 100])
  for (const name of ['同名图.png', '子目录/同名图.png']) {
    const data = Buffer.from(files['导出-批量项目 (2)/' + name])
    expect(data.subarray(0, 8).toString('hex')).toBe('89504e470d0a1a0a')
  }
  const archive = unzip(Buffer.from(files['导出-批量项目 (2).zip']))
  expect([...archive.keys()]).toEqual(['子目录/同名图.png', '同名图.png'])
  for (const [name, data] of archive) expect(data).toEqual(Buffer.from(files['导出-批量项目 (2)/' + name]))
})

test('a folder containing one diagram exports a single file without output checkboxes', async ({ page }) => {
  await setup(page)
  const dialog = await openDialog(page)
  await chooseFolder(page, dialog, '批量项目 / 子目录')
  await expect(dialog.locator('.export-summary')).toContainText('共 1 张图')
  await expect(dialog.getByRole('checkbox')).toHaveCount(0)
  await choose(page, dialog, 1, 'SVG 矢量图')
  const download = page.waitForEvent('download')
  await dialog.getByRole('button', { name: '导出', exact: true }).click()
  expect((await download).suggestedFilename()).toBe('第二张图.svg')
})

test('invalid batch content identifies its file and leaves the dialog available to retry', async ({ page }) => {
  await setup(page)
  const dialog = await openDialog(page)
  await chooseFolder(page, dialog)
  await choose(page, dialog, 1, 'JSON 设计数据')
  await dialog.getByRole('checkbox', { name: '文件夹', exact: true }).uncheck()
  await dialog.getByRole('checkbox', { name: 'ZIP 压缩包', exact: true }).check()
  await page.route('**/api/render', route => route.fulfill({ json: { validation: { valid: false, issues: [{ severity: 'error', message: '测试图表引用无效' }] } } }))
  await dialog.getByRole('button', { name: '导出', exact: true }).click()
  await expect(dialog.getByRole('alert')).toContainText('同名图.json')
  await expect(dialog.getByRole('alert')).toContainText('测试图表引用无效')
  await expect(dialog.getByRole('button', { name: '导出', exact: true })).toBeEnabled()
})
