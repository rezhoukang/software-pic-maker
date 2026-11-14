import { test, expect } from '@playwright/test'
import fs from 'node:fs'

const library = JSON.parse(fs.readFileSync(new URL('../../shared/library.json', import.meta.url), 'utf8'))

test('a table accepts fifty fields and new fields have no assumed type', async ({ page }) => {
  await page.goto('/')
  await expect(page.locator('.x6-node').first()).toBeVisible()
  const d = structuredClone(library[1])
  d.model.entities[0].fields = Array.from({ length: 49 }, (_, i) => ({ name: `field_${i + 1}`, type: i === 0 ? 'bigint' : 'text' }))
  d.view.positions = {}
  await page.getByRole('button', { name: 'JSON', exact: true }).click()
  await page.getByRole('textbox', { name: '图表 JSON' }).fill(JSON.stringify(d))
  await page.getByRole('button', { name: '校验并应用', exact: true }).click()
  await expect(page.getByRole('status')).toContainText('JSON 已应用')
  await page.getByRole('button', { name: '画布', exact: true }).click()
  await page.getByRole('button', { name: '适应画布', exact: true }).click()
  await page.locator(`.x6-node[data-cell-id="${d.model.entities[0].id}"]`).click()
  const add = page.locator('.property-section-title').getByRole('button', { name: '添加', exact: true })
  await add.click()
  await expect(page.getByRole('textbox', { name: '字段名称', exact: true })).toHaveCount(50)
  await expect(page.getByRole('textbox', { name: '字段类型', exact: true }).last()).toHaveValue('')
  await expect(add).toBeDisabled()
  await page.getByRole('textbox', { name: '字段类型', exact: true }).last().fill('datetime')
  await page.getByRole('textbox', { name: '字段类型', exact: true }).last().press('Tab')
  await page.getByRole('button', { name: 'JSON', exact: true }).click()
  const saved = JSON.parse(await page.getByRole('textbox', { name: '图表 JSON' }).inputValue())
  expect(saved.model.entities[0].fields).toHaveLength(50)
  expect(saved.model.entities[0].fields[0].type).toBe('bigint')
  expect(saved.model.entities[0].fields[49].type).toBe('datetime')
})
