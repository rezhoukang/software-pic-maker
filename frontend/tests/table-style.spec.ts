import { test, expect, type Page } from '@playwright/test'

async function choose(page: Page, label: string, option: string) {
  await page.getByLabel(label, { exact: true }).click()
  await page.locator('.n-base-select-option:visible').filter({ hasText: new RegExp('^' + option + '$') }).first().click()
}

test('table layout aligns headers and global styling lives in diagram settings', async ({ page }) => {
  await page.goto('/')
  await page.locator('.diagram-nav[data-type="table"]').first().click()
  await expect(page.getByTestId('diagram-canvas')).toHaveAttribute('data-rendered-type', 'table')
  await expect(page.locator('.render-progress')).toHaveCount(0)
  const nodes = ['reader', 'loan', 'book'].map(id => page.locator(`.x6-node[data-cell-id="${id}"]`))
  const boxes = await Promise.all(nodes.map(node => node.boundingBox()))
  expect(Math.abs(boxes[0]!.y - boxes[1]!.y)).toBeLessThan(1)
  expect(Math.abs(boxes[1]!.y - boxes[2]!.y)).toBeLessThan(1)
  await expect(page.locator('.inspector .theme-picker')).toBeVisible()
})

test('global style changes output without changing source data or undo history', async ({ page }) => {
  await page.goto('/')
  await expect(page.locator('.x6-node').first()).toBeVisible()
  const title = page.getByRole('textbox', { name: '图表名称', exact: true })
  const original = await title.inputValue()
  await title.fill('导出风格测试'); await title.press('Tab')
  const sourcePath = await page.locator('.diagram-nav.active').getAttribute('data-entry-path')
  await page.locator('.inspector .theme-picker').getByText('论文',{exact:true}).click()
  await page.getByTestId('export-button').click()
  const dialog = page.getByRole('dialog', { name: '导出', exact: true })
  await expect(dialog.locator('.export-summary')).not.toContainText('正在读取')
  await choose(page, '导出范围', '当前图')
  await choose(page, '导出形式', 'SVG 矢量图')
  await expect(dialog.locator('.theme-picker')).toHaveCount(0)
  await page.keyboard.press('Control+z')
  const download = page.waitForEvent('download')
  await dialog.getByRole('button', { name: '导出', exact: true }).click()
  const stream = await (await download).createReadStream()
  const chunks: Buffer[] = []
  for await (const chunk of stream!) chunks.push(Buffer.from(chunk))
  expect(Buffer.concat(chunks).toString()).toContain('#333333')
  await expect(title).toHaveValue('导出风格测试')
  await expect(page.getByTestId('file-save-status')).toContainText('未保存')
  const sourceTheme = await page.evaluate(async path => {
    const response = await fetch('/api/workspace/read', { method: 'POST', headers: { 'Content-Type': 'application/json', 'X-EZ-Workspace': localStorage.getItem('ezbitu-preview-workspace')! }, body: JSON.stringify({ path }) })
    return (await response.json()).diagram.view.theme
  }, sourcePath)
  expect(sourceTheme).toBe('blue')
  await page.getByRole('button', { name: '撤销', exact: true }).click()
  await expect(title).toHaveValue(original)
})

test('batch export applies one style without changing stored files', async ({ page }) => {
  await page.goto('/')
  await expect(page.locator('.x6-node').first()).toBeVisible()
  await page.locator('.inspector .theme-picker').getByText('青绿',{exact:true}).click()
  await page.getByTestId('export-button').click()
  const dialog = page.getByRole('dialog', { name: '导出', exact: true })
  await expect(dialog.locator('.export-summary')).not.toContainText('正在读取')
  await choose(page, '导出范围', '全部文件')
  await choose(page, '导出形式', 'JSON 设计数据')

  await dialog.getByRole('checkbox', { name: '文件夹', exact: true }).uncheck()
  await dialog.getByRole('checkbox', { name: 'ZIP 压缩包', exact: true }).check()
  let exported = false
  await page.route('**/api/export/zip', async route => {
    const files = route.request().postDataJSON().files as { data: string }[]
    expect(files).toHaveLength(8)
    for (const file of files) expect(JSON.parse(Buffer.from(file.data, 'base64').toString()).view.theme).toBe('sage')
    exported = true
    await route.continue()
  })
  const download = page.waitForEvent('download')
  await dialog.getByRole('button', { name: '导出', exact: true }).click()
  await download
  expect(exported).toBe(true)
  const stored = await page.evaluate(async () => {
    const headers = { 'Content-Type': 'application/json', 'X-EZ-Workspace': localStorage.getItem('ezbitu-preview-workspace')! }
    const tree = await (await fetch('/api/workspace/list', { method: 'POST', headers, body: '{}' })).json()
    return Promise.all(tree.entries[0].children.map(async (file: { path: string }) => (await (await fetch('/api/workspace/read', { method: 'POST', headers, body: JSON.stringify({ path: file.path }) })).json()).diagram.view.theme))
  })
  expect(stored).toEqual(Array(8).fill('blue'))
})

test('global style follows file switches, new files and reload without dirtying documents',async({page})=>{
 const themes:string[]=[]
 page.on('request',request=>{if(request.url().endsWith('/api/render'))themes.push(request.postDataJSON().view.theme)})
 await page.goto('/')
 await expect(page.locator('.x6-node').first()).toBeVisible()
 await page.locator('.inspector .theme-picker').getByText('论文',{exact:true}).click()
 await expect.poll(()=>themes.at(-1)).toBe('mono')
 await expect(page.getByTestId('file-save-status')).toContainText('已保存')
 await page.locator('.diagram-nav[data-type="architecture"]').first().click()
 await expect(page.getByTestId('diagram-canvas')).toHaveAttribute('data-rendered-type','architecture')
 await expect(page.getByRole('radio',{name:'论文',exact:true})).toBeChecked()
 expect(themes.at(-1)).toBe('mono')
 await expect(page.getByTestId('file-save-status')).toContainText('已保存')
 await page.getByRole('button',{name:'新增文件',exact:true}).click()
 const dialog=page.getByRole('dialog',{name:'新增文件',exact:true})
 await dialog.getByRole('textbox',{name:'文件名称'}).fill('全局风格新图')
 await dialog.getByRole('button',{name:'创建文件',exact:true}).click()
 await expect(dialog).toBeHidden()
 await expect(page.getByRole('radio',{name:'论文',exact:true})).toBeChecked()
 expect(themes.at(-1)).toBe('mono')
 await page.reload()
 await expect(page.locator('.x6-node').first()).toBeVisible()
 await expect(page.getByRole('radio',{name:'论文',exact:true})).toBeChecked()
 expect(themes.at(-1)).toBe('mono')
 await expect(page.getByTestId('file-save-status')).toContainText('已保存')
 await page.screenshot({path:'test-results/global-theme-settings.png'})
})
