import { test, expect } from '@playwright/test'
import fs from 'node:fs'

const library = JSON.parse(fs.readFileSync(new URL('../../shared/library.json', import.meta.url), 'utf8'))

test('long box and database titles wrap inside their borders and stay vertically centered', async ({ page, request }) => {
  const titles = [
    '关系数据库' + '1'.repeat(24) + '2'.repeat(28),
    '系统业务数据与用户权限管理'.repeat(6),
    'W'.repeat(120),
    'Account authorization and business database service '.repeat(2),
    '第一行\n第二行\n第三行',
    '关系数据库',
  ]
  let example: any
  for (const [index, title] of titles.entries()) {
    const d = structuredClone(library[6])
    d.view.positions = {}
    d.view.direction = index % 2 ? 'LR' : 'TB'
    for (const id of ['auth', 'db']) d.model.nodes.find((n: any) => n.id === id).name = title
    const response = await request.post('http://127.0.0.1:34116/api/render', { data: d })
    expect(response.ok()).toBe(true)
    const result = await response.json()
    expect(result.validation.valid, JSON.stringify(result.validation)).toBe(true)
    await page.setContent(result.svg)
    await page.evaluate(() => document.fonts.ready)
    expect(await page.locator('parsererror').count()).toBe(0)
    for (const id of ['auth', 'db']) {
      const node = result.scene.nodes.find((n: any) => n.id === id)
      const label = page.locator(`g[transform="translate(${node.x.toFixed(2)},${node.y.toFixed(2)})"] > .node-title`)
      const bounds = await label.evaluate(element => {
        const b = (element as SVGGraphicsElement).getBBox()
        return { x: b.x, y: b.y, width: b.width, height: b.height }
      })
      const top = id === 'db' ? 30 : 0
      const bottom = node.height - (id === 'db' ? 16 : 0)
      expect(bounds.x).toBeGreaterThanOrEqual(17)
      expect(bounds.x + bounds.width).toBeLessThanOrEqual(node.width - 17)
      expect(bounds.y).toBeGreaterThanOrEqual(top)
      expect(bounds.y + bounds.height).toBeLessThanOrEqual(bottom)
      expect(Math.abs(bounds.y + bounds.height / 2 - (top + bottom) / 2)).toBeLessThan(2)
      const text = (await label.locator('text').allTextContents()).join('')
      expect(text.replace(/\s/g, '')).toBe(title.replace(/\s/g, ''))
      if (index < 5) expect(await label.locator('text').count()).toBeGreaterThan(1)
    }
    if (index === 0) example = d
  }

  await page.goto('/')
  await expect(page.locator('.x6-node').first()).toBeVisible()
  await page.getByRole('button', { name: 'JSON', exact: true }).click()
  await page.getByRole('textbox', { name: '图表 JSON' }).fill(JSON.stringify(example))
  await page.getByRole('button', { name: '校验并应用', exact: true }).click()
  await expect(page.getByRole('status')).toContainText('JSON 已应用')
  await page.getByRole('button', { name: '画布', exact: true }).click()
  await page.getByRole('button', { name: '适应画布', exact: true }).click()
  await expect(page.locator('.x6-node[data-cell-id="db"]')).toBeInViewport()
  const image = page.locator('.x6-node[data-cell-id="db"] image')
  await expect.poll(async () => decodeURIComponent((await image.getAttribute('xlink:href')) || '')).toContain('node-title')
  await page.locator('.canvas-area').screenshot({ path: 'test-results/text-fit/long-titles.png' })
})
