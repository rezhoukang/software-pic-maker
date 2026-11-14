import { test, expect } from '@playwright/test'
import fs from 'node:fs'

const library = JSON.parse(fs.readFileSync(new URL('../../shared/library.json', import.meta.url), 'utf8'))

test('use-case arrows stay straight and labels follow their own line in X6', async ({ page }) => {
  const diagram = structuredClone(library[2])
  diagram.model.actors = [{ id: 'actor', name: '参与者' }]
  diagram.model.use_cases = ['a', 'b', 'c', 'shared'].map(id => ({ id, name: id === 'shared' ? '共同校验' : `操作 ${id}` }))
  diagram.model.relations = ['a', 'b', 'c'].map(source => ({ id: `${source}s`, source, target: 'shared', kind: 'include' }))
  diagram.view.positions = { a: { x: 100, y: 100 }, b: { x: 100, y: 250 }, c: { x: 100, y: 400 }, shared: { x: 500, y: 250 }, actor: { x: 0, y: 520 } }
  const response = await page.request.post('http://127.0.0.1:34116/api/render', { data: diagram })
  const result = await response.json()
  expect(result.validation.valid).toBe(true)
  await page.goto('/')
  await page.locator('.x6-node').first().waitFor()
  await page.getByRole('button', { name: 'JSON', exact: true }).click()
  await page.getByRole('textbox', { name: '图表 JSON' }).fill(JSON.stringify(diagram))
  await page.getByRole('button', { name: '校验并应用', exact: true }).click()
  await expect(page.getByRole('status')).toContainText('JSON 已应用')
  await page.getByRole('button', { name: '画布', exact: true }).click()
  await expect(page.getByTestId('diagram-canvas')).toHaveAttribute('data-rendered-type', 'usecase')
  await expect(page.locator('.render-progress')).toHaveCount(0)
  for (const edge of result.scene.edges) {
    expect(edge.points).toHaveLength(2)
    const cell = page.locator(`.x6-edge[data-cell-id="${edge.id}"]`)
    const geometry = await cell.locator('path').nth(1).evaluate(el => {
      const path = el as SVGPathElement
      const length = path.getTotalLength(), a = path.getPointAtLength(0), b = path.getPointAtLength(length)
      return { length, distance: Math.hypot(b.x - a.x, b.y - a.y) }
    })
    expect(Math.abs(geometry.length - geometry.distance)).toBeLessThan(.01)
    const label = cell.locator('.x6-edge-label').first()
    await expect(label).toBeVisible()
    const transform = await label.evaluate(el => {
      const m = (el as SVGGElement).transform.baseVal.consolidate()!.matrix
      return { angle: Math.atan2(m.b, m.a) * 180 / Math.PI, x: m.e, y: m.f }
    })
    expect(Math.abs(transform.angle - (edge.label_angle ?? 0))).toBeLessThan(.01)
    expect(Math.abs(transform.angle)).toBeLessThanOrEqual(85)
    expect(Math.hypot(transform.x - edge.label_position.x, transform.y - edge.label_position.y)).toBeLessThan(2)
  }
})
