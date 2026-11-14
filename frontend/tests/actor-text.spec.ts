import { test, expect } from '@playwright/test'
import fs from 'node:fs'

const library = JSON.parse(fs.readFileSync(new URL('../../shared/library.json', import.meta.url), 'utf8'))

test('actor names fit within the SVG viewport with bottom padding', async ({ page, request }) => {
  const d = structuredClone(library[2])
  for (let i = d.model.actors.length; i < 12; i++) d.model.actors.push({ id: `actor${i + 1}`, name: `参与者${i + 1}` })
  d.model.actors[0].name = '参与者5'
  d.model.actors[1].name = 'gyp测试'
  const response = await request.post('http://127.0.0.1:34116/api/render', { data: d })
  const result = await response.json()
  expect(result.validation.valid).toBe(true)
  for (const actor of result.scene.nodes.filter((n: any) => n.shape === 'actor')) {
    await page.setContent(result.node_svgs[actor.id])
    await page.evaluate(() => document.fonts.ready)
    const text = page.locator('text')
    await expect(text).toHaveText(actor.title)
    const b = await text.evaluate(element => {
      const bounds = (element as SVGGraphicsElement).getBBox()
      return { y: bounds.y, height: bounds.height }
    })
    expect(b.y + b.height).toBeLessThanOrEqual(actor.height - 4)
  }
  await page.setContent(result.node_svgs[d.model.actors[0].id])
  await page.locator('svg').screenshot({ path: 'test-results/actor-text/full-name.png' })
})
