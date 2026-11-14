import { test, expect } from '@playwright/test'

test('focus refresh keeps the current tree visible without a loading message', async ({ page }) => {
  let requests = 0
  let finishRefresh: (() => void) | undefined
  await page.route('**/api/workspace/list', async route => {
    requests++
    const entries = [{ name: '已有文件夹', path: '已有文件夹', kind: 'folder', children: [] }]
    if (requests > 1) {
      await new Promise<void>(resolve => { finishRefresh = resolve })
      entries.push({ name: '新增文件夹', path: '新增文件夹', kind: 'folder', children: [] })
    }
    await route.fulfill({ json: { root: 'test-results/file-workspaces', entries } })
  })
  await page.goto('/')
  const existing = page.getByRole('treeitem', { name: '已有文件夹', exact: true })
  await expect(existing).toBeVisible()
  const refreshing = page.waitForRequest(request => request.url().endsWith('/api/workspace/list'))
  await page.setViewportSize({ width: 1200, height: 800 })
  await page.evaluate(() => {
    window.dispatchEvent(new Event('focus'))
    window.dispatchEvent(new Event('focus'))
    window.dispatchEvent(new Event('focus'))
  })
  await refreshing
  await expect.poll(() => Boolean(finishRefresh)).toBe(true)
  await expect(existing).toBeVisible()
  await expect(page.getByText('正在读取目录…', { exact: true })).toHaveCount(0)
  finishRefresh!()
  await expect(page.getByRole('treeitem', { name: '新增文件夹', exact: true })).toBeVisible()
  expect(requests).toBe(2)
})
