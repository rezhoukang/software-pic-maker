import { test, expect } from '@playwright/test'
import fs from 'node:fs'
const library=JSON.parse(fs.readFileSync(new URL('../../shared/library.json',import.meta.url),'utf8'))

test('pending JSON survives a real file rename', async ({ page }) => {
 await page.goto('/'); await expect(page.locator('.x6-node').first()).toBeVisible()
 await page.getByRole('button',{name:'JSON',exact:true}).click()
 const editor=page.getByRole('textbox',{name:'图表 JSON'}); const draft=(await editor.inputValue()).replace('概念 ER 图','尚未应用的图')
 await editor.fill(draft)
 await page.locator('.diagram-nav.active').click({button:'right'});await page.locator('.n-dropdown-option:visible').filter({hasText:'重命名'}).click();const rename=page.getByRole('dialog',{name:'重命名',exact:true});await rename.locator('input').fill('新的文件名称');await rename.getByRole('button',{name:'确定',exact:true}).click();await expect(page.locator('.file-breadcrumb')).toContainText('新的文件名称.json')
 await expect(editor).toHaveValue(draft)
 await page.getByRole('button',{name:'保存文件',exact:true}).click()
 await expect(page.getByRole('status')).toContainText('请先应用')
})

test('malformed recovery does not break startup', async ({ page }) => {
 const errors:string[]=[];page.on('pageerror',e=>errors.push(e.message))
 await page.addInitScript(()=>localStorage.setItem('tuxu-session-v1',JSON.stringify({project:{schema_version:'1.0',name:'坏副本',diagrams:[{id:'bad',type:'unsupported'}]}})))
 await page.goto('/');await expect(page.locator('.x6-node').first()).toBeVisible()
 await expect(page.getByRole('textbox',{name:'图表名称',exact:true})).toHaveValue('图书借阅 · 概念 ER 图')
 expect(errors).toEqual([])
})

test('invalid diagram cannot be saved or replace valid recovery',async({page})=>{
 const downloads:string[]=[];page.on('download',d=>downloads.push(d.suggestedFilename()))
 await page.goto('/');await expect(page.locator('.x6-node').first()).toBeVisible()
 await page.getByRole('textbox',{name:'图表名称',exact:true}).fill('有效副本');await page.getByRole('textbox',{name:'图表名称',exact:true}).press('Tab')
 await expect.poll(()=>page.evaluate(()=>JSON.parse(localStorage.getItem('ezruantu-session-v1')||'{}').project?.diagrams?.[0]?.title)).toBe('有效副本')
 await page.locator('.x6-node[data-cell-id="reader"]').click()
 const fields=page.getByRole('textbox',{name:'字段名称',exact:true});await fields.nth(1).fill('读者编号');await fields.nth(1).press('Tab')
 await expect(page.locator('.canvas-error')).toContainText('字段重复')
 await page.getByRole('button',{name:'保存文件',exact:true}).click()
 await expect(page.getByRole('status')).toContainText('字段重复')
 expect(downloads).toEqual([])
 await page.waitForTimeout(1000);await page.reload();await expect(page.locator('.canvas-error')).toHaveCount(0)
 await expect(page.locator('.x6-node').first()).toBeVisible()
})

test('late JSON validation response does not overwrite a newer draft',async({page})=>{
 await page.goto('/');await expect(page.locator('.x6-node').first()).toBeVisible()
 await page.getByRole('button',{name:'JSON',exact:true}).click()
 const editor=page.getByRole('textbox',{name:'图表 JSON'});const d=JSON.parse(await editor.inputValue());d.title='先提交的标题'
 await editor.fill(JSON.stringify(d))
 let release!:()=>void;const gate=new Promise<void>(resolve=>release=resolve)
 let started!:()=>void;const pending=new Promise<void>(resolve=>started=resolve)
 await page.route('**/api/render',async route=>{const response=await route.fetch();started();await gate;await route.fulfill({response})})
 await page.getByRole('button',{name:'校验并应用'}).click();await pending
 d.title='后输入的草稿';const newer=JSON.stringify(d);await editor.fill(newer);release()
 await expect(page.getByRole('status')).toContainText('重新应用')
 await expect(editor).toHaveValue(newer)
 await expect(page.getByRole('textbox',{name:'图表名称',exact:true})).toHaveValue('图书借阅 · 概念 ER 图')
})

test('invalid project import leaves the current project intact',async({page})=>{
 await page.goto('/');await expect(page.locator('.x6-node').first()).toBeVisible()
 const broken=structuredClone(library);broken[0].model.relations![0].target='missing'
 const chooser=page.waitForEvent('filechooser');await page.getByRole('button',{name:'导入项目',exact:true}).click()
 await (await chooser).setFiles({name:'invalid.tuxu.json',mimeType:'application/json',buffer:Buffer.from(JSON.stringify({schema_version:'1.0',name:'不能打开的项目',diagrams:broken}))})
 await expect(page.getByRole('status')).toContainText('不存在的节点')
 await expect(page.getByRole('textbox',{name:'图表名称',exact:true})).toHaveValue('图书借阅 · 概念 ER 图')
 await expect(page.locator('.diagram-nav')).toHaveCount(8)
})

test('empty relations and monochrome survive export and reload',async({page})=>{
 await page.goto('/');await expect(page.locator('.x6-node').first()).toBeVisible()
 await page.locator('.inspector .theme-picker').getByText('论文',{exact:true}).click()
 await page.getByRole('button',{name:'JSON',exact:true}).click();const editor=page.getByRole('textbox',{name:'图表 JSON'});const d=JSON.parse(await editor.inputValue());d.model.relations=[];d.view.theme='mono'
 await editor.fill(JSON.stringify(d));await page.getByRole('button',{name:'校验并应用'}).click();await page.getByRole('button',{name:'画布',exact:true}).click()
 await expect(page.locator('.render-progress')).toHaveCount(0);await expect(page.locator('.canvas-error')).toHaveCount(0)
 await page.getByTestId('export-button').click();const modal=page.getByRole('dialog',{name:'导出',exact:true})
 await modal.locator('.n-select').nth(0).click();await page.locator('.n-base-select-option:visible').filter({hasText:/^当前图$/}).click()
 await modal.locator('.n-select').nth(1).click();await page.locator('.n-base-select-option:visible').filter({hasText:/^SVG/}).click()
 const promise=page.waitForEvent('download');await modal.getByRole('button',{name:'导出',exact:true}).click();const download=await promise
 const p='test-results/monochrome.svg';await download.saveAs(p);const svg=fs.readFileSync(p,'utf8');expect(svg).toContain('#333333');expect(svg).not.toContain('#63758f')
 await expect.poll(()=>page.evaluate(()=>JSON.parse(localStorage.getItem('ezruantu-session-v1')||'{}').project?.diagrams?.[0]?.view?.theme)).toBe('mono')
 await page.reload();await expect(page.locator('.x6-node').first()).toBeVisible();await expect(page.locator('.diagram-stats')).toContainText('0 条关系')
})

test('minimum desktop viewport keeps primary controls visible',async({page})=>{
 await page.setViewportSize({width:1120,height:740});await page.goto('/');await expect(page.locator('.x6-node').first()).toBeVisible()
 for(const name of ['保存文件','导出','添加实体','自动排版','收起属性面板']){
  const box=await page.getByRole('button',{name,exact:true}).boundingBox();expect(box).not.toBeNull();expect(box!.x).toBeGreaterThanOrEqual(0);expect(box!.x+box!.width).toBeLessThanOrEqual(1120)
 }
 expect(await page.evaluate(()=>document.documentElement.scrollWidth)).toBe(1120)
 await page.screenshot({path:'test-results/minimum-window.png'})
 await page.getByRole('button',{name:'新增文件',exact:true}).click();await expect(page.locator('.new-file-modal')).toBeVisible();await page.screenshot({path:'test-results/template-picker.png'})
})
