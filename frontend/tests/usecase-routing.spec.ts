import { test, expect } from '@playwright/test'
import fs from 'node:fs'

const library = JSON.parse(fs.readFileSync(new URL('../../shared/library.json', import.meta.url), 'utf8'))

test('a moved actor uses oblique detours without right-angle corners', async ({ page }) => {
  await page.goto('/')
  await expect(page.locator('.x6-node').first()).toBeVisible()
  const d = structuredClone(library[2])
  d.view.positions = { reader: { x: 55, y: 80 } }
  await page.getByRole('button', { name: 'JSON', exact: true }).click()
  await page.getByRole('textbox', { name: '图表 JSON' }).fill(JSON.stringify(d))
  await page.getByRole('button', { name: '校验并应用', exact: true }).click()
  await expect(page.getByRole('status')).toContainText('JSON 已应用')
  await page.getByRole('button', { name: '画布', exact: true }).click()
  await page.getByRole('button', { name: '适应画布', exact: true }).click()
  for (const edge of await page.locator('.x6-edge').all()) {
    const path = await edge.locator('path').nth(1).getAttribute('d')
    const values = path!.match(/-?\d+(?:\.\d+)?/g)!.map(Number)
    for (let i = 2; i < values.length - 2; i += 2) {
      const dx = values[i] - values[i - 2], dy = values[i + 1] - values[i - 1]
      const ex = values[i + 2] - values[i], ey = values[i + 3] - values[i + 1]
      expect(Math.abs(dx * ex + dy * ey) / (Math.hypot(dx, dy) * Math.hypot(ex, ey))).toBeGreaterThan(.001)
    }
  }
  await page.locator('.canvas-area').screenshot({ path: 'test-results/usecase-oblique.png' })
})
