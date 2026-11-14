import { test, expect } from '@playwright/test'
import fs from 'node:fs'
import path from 'node:path'

test('eight diagrams render without runtime errors', async ({ page }) => {
 const errors: string[]=[]; page.on('pageerror',e=>errors.push(e.message))
 await page.goto('/'); await expect(page.locator('.x6-node').first()).toBeVisible()
 for(let i=0;i<8;i++){
  await page.locator(`.diagram-nav[data-type="${['er','table','usecase','flowchart','sequence','class','architecture','structure'][i]}"]`).first().click()
  await expect(page.getByTestId('diagram-canvas')).toHaveAttribute('data-rendered-type',['er','table','usecase','flowchart','sequence','class','architecture','structure'][i])
  await expect(page.locator('.statusbar .diagram-validity')).toContainText('图表结构有效')
  await expect(page.locator('.render-progress')).toHaveCount(0)
  await expect(page.locator('.x6-node').first()).toBeVisible()
  await page.screenshot({path:`test-results/diagram-${i}.png`})
 }
 expect(errors).toEqual([])
})

test('edit node, drag, undo, JSON validation and recovery', async ({ page }) => {
 await page.goto('/');await expect(page.locator('.x6-node').first()).toBeVisible()
 const node=page.locator('.x6-node[data-cell-id="reader"]');await node.click()
 const name=page.locator('.inspector .form-label').filter({hasText:'名称'}).locator('input')
 await name.fill('测试读者');await name.press('Tab');await expect(page.locator('.property-heading strong')).toHaveText('测试读者')
 await page.getByRole('button',{name:'撤销',exact:true}).click();await node.click();await expect(name).toHaveValue('读者')
 await page.getByRole('button',{name:'重做',exact:true}).click();await node.click();await expect(name).toHaveValue('测试读者')
 await expect(page.locator('.render-progress')).toHaveCount(0);await expect(node).toBeVisible()
 await page.getByRole('button',{name:'手动调整',exact:true}).click()
 const box=await node.boundingBox();if(!box)throw new Error('no node')
 await page.mouse.move(box.x+box.width/2,box.y+box.height/2);await page.mouse.down();await page.mouse.move(box.x+box.width/2+50,box.y+box.height/2+40,{steps:10});await page.mouse.up()
 await page.waitForTimeout(900)
 await page.getByRole('button',{name:'JSON',exact:true}).click()
 const editor=page.getByRole('textbox',{name:'图表 JSON'});const original=JSON.parse(await editor.inputValue());expect(original.view.positions.reader).toBeDefined()
 await editor.fill('{broken');await page.getByRole('button',{name:'校验并应用'}).click();await expect(page.locator('.json-error')).toBeVisible()
 // Clicking the already-active JSON tab must not discard unsaved text.
 await page.getByRole('button',{name:'JSON',exact:true}).click();await expect(editor).toHaveValue('{broken')
 const changed=structuredClone(original);changed.title='测试 JSON 标题';await editor.fill(JSON.stringify(changed));await page.getByRole('button',{name:'校验并应用'}).click();await expect(page.getByRole('textbox',{name:'图表名称',exact:true})).toHaveValue('测试 JSON 标题')
 await page.waitForTimeout(900);await page.reload();await expect(page.getByRole('textbox',{name:'图表名称',exact:true})).toHaveValue('测试 JSON 标题')
 await page.locator('.x6-node[data-cell-id="reader"]').click();await expect(name).toHaveValue('测试读者')
})

test('all four exports and project save/open', async ({ page }) => {
 await page.goto('/');await expect(page.locator('.x6-node').first()).toBeVisible()
 for(const format of ['SVG','PNG','draw.io','JSON']){
  await page.getByTestId('export-button').click();const modal=page.getByRole('dialog',{name:'导出',exact:true})
  await modal.locator('.n-select').nth(0).click();await page.locator('.n-base-select-option:visible').filter({hasText:/^当前图$/}).click()
  await modal.locator('.n-select').nth(1).click();await page.locator('.n-base-select-option:visible').filter({hasText:format==='draw.io'?/^draw\.io$/:new RegExp('^'+format)}).click()
  const promise=page.waitForEvent('download');await modal.getByRole('button',{name:'导出',exact:true}).click();const d=await promise
  const target=path.join('test-results',d.suggestedFilename());await d.saveAs(target);const bytes=fs.readFileSync(target);expect(bytes.length).toBeGreaterThan(100)
  if(format==='PNG')expect(bytes.subarray(0,8).toString('hex')).toBe('89504e470d0a1a0a')
  if(format==='draw.io')expect(bytes.toString()).toContain('<mxfile')
 }
 const p=page.waitForEvent('download');await page.getByRole('button',{name:'导出项目副本',exact:true}).click();const download=await p;expect(download.suggestedFilename()).toMatch(/\.ezruantu\.json$/);const target=path.resolve('test-results/project.ezruantu.json');await download.saveAs(target)
 const content=JSON.parse(fs.readFileSync(target,'utf8'));content.name='打开的测试项目';fs.writeFileSync(target,JSON.stringify(content))
 const chooser=page.waitForEvent('filechooser');await page.getByRole('button',{name:'导入项目',exact:true}).click();await (await chooser).setFiles(target);await expect(page.locator('.file-breadcrumb')).toContainText('打开的测试项目')
})

test('add/remove elements, relation and diagram',async({page})=>{
 await page.goto('/');await expect(page.locator('.x6-node').first()).toBeVisible()
 await page.locator('.diagram-nav[data-type="table"]').first().click();await expect(page.getByRole('textbox',{name:'图表名称',exact:true})).toHaveValue('数据库表关系图')
 await page.getByRole('button',{name:'添加表',exact:true}).click();await expect(page.locator('.property-heading strong')).toHaveText('表4')
 await page.getByRole('button',{name:'删除元素及相关连线'}).click();await expect(page.locator('.diagram-stats')).toContainText('3 个元素')
 await page.getByRole('button',{name:'连线',exact:true}).click();await page.locator('.compact-modal input').fill('新关系');await page.getByRole('button',{name:'添加关系',exact:true}).click();await expect(page.locator('.diagram-stats')).toContainText('3 条关系')
 await page.getByRole('button',{name:'删除关系',exact:true}).click();await expect(page.locator('.diagram-stats')).toContainText('2 条关系')
 await page.getByRole('button',{name:'保存文件',exact:true}).click();await expect(page.getByRole('status')).toContainText('文件已保存');await page.getByRole('button',{name:'新增文件',exact:true}).click();await page.locator('.new-file-modal input').fill('我的 ER 图');await page.getByRole('button',{name:'创建文件',exact:true}).click();await expect(page.getByRole('textbox',{name:'图表名称',exact:true})).toHaveValue('我的 ER 图');await expect(page.locator('.diagram-nav')).toHaveCount(9)
 await page.getByRole('button',{name:'删除当前文件'}).click();await page.getByRole('button',{name:'确认删除',exact:true}).click();await expect(page.locator('.diagram-nav')).toHaveCount(8)
})
