import { test, expect } from '@playwright/test'
import fs from 'node:fs'

const library = JSON.parse(fs.readFileSync(new URL('../../shared/library.json', import.meta.url), 'utf8'))

test('desktop close can be cancelled, confirmed, and retried after a bridge error', async ({ page }) => {
  await page.addInitScript((diagrams) => {
    const w = window as any
    const listeners: Record<string, () => void> = {}
    w.closeAttempts = 0
    w.failQuit = false
    w.requestClose = () => listeners['app:close-requested']?.()
    w.runtime = { EventsOnMultiple: (event: string, callback: () => void) => { listeners[event] = callback; return () => delete listeners[event] } }
    w.go = { main: { App: {
      GetSession: async () => ({ project: { schema_version: '1.0', name: '退出测试项目', diagrams }, path: '', recovered: true }),
      GetEnvironment: async () => ({ version: '0.1.0', mcp_path: '', data_dir: '', mcp_ready: false }),
      RenderDiagram: async (raw: string) => (await fetch('/api/render', { method: 'POST', body: raw })).json(),
      SetDirty: async () => {},
      SaveRecovery: async () => {},
      ConfirmQuit: async () => { w.closeAttempts++; if (w.failQuit) throw new Error('测试退出失败') },
    } } }
  }, library)
  await page.goto('/')
  await expect(page.locator('.x6-node').first()).toBeVisible()
  await expect(page.locator('.brand')).toHaveCount(0); await expect(page.getByRole('button', {name:'MCP 接入',exact:true})).toBeVisible()
  await page.evaluate(() => (window as any).requestClose())
  const dialog = page.getByRole('dialog', { name: '退出 EZ软图？' })
  await expect(dialog).toBeVisible()
  await dialog.getByRole('button', { name: '取消', exact: true }).click()
  await expect(dialog).toBeHidden()
  expect(await page.evaluate(() => (window as any).closeAttempts)).toBe(0)
  await page.evaluate(() => { (window as any).failQuit = true; (window as any).requestClose() })
  await dialog.getByRole('button', { name: '直接退出', exact: true }).click()
  await expect(page.getByRole('status')).toContainText('测试退出失败')
  await expect(dialog.getByRole('button', { name: '直接退出', exact: true })).toBeEnabled()
  await page.evaluate(() => { (window as any).failQuit = false })
  await dialog.getByRole('button', { name: '直接退出', exact: true }).click()
  await expect.poll(() => page.evaluate(() => (window as any).closeAttempts)).toBe(2)
})
