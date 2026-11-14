import {test,expect,type Page} from '@playwright/test'
import fs from 'node:fs'
import path from 'node:path'
const version=JSON.parse(fs.readFileSync(new URL('../package.json',import.meta.url),'utf8')).version

async function setup(page:Page){
 await page.goto('/');await expect(page.locator('.x6-node').first()).toBeVisible()
 const tree=await page.evaluate(async()=>{
  const response=await fetch('/api/workspace/list',{method:'POST',headers:{'Content-Type':'application/json','X-EZ-Workspace':localStorage.getItem('ezbitu-preview-workspace')!},body:'{}'})
  return response.json()
 })
 expect(tree.root).toContain('file-workspaces')
 const current=await page.locator('.diagram-nav.active').getAttribute('data-entry-path');expect(current).toBeTruthy()
 return {root:tree.root as string,current:current!,title:page.getByRole('textbox',{name:'图表名称',exact:true})}
}

test('plain wheel zooms canvas and removed footer details stay absent',async({page})=>{
 await setup(page)
 await expect(page.locator('.statusbar-details,.interaction-hint,.save-state')).toHaveCount(0)
 const zoom=()=>page.locator('.zoom-value').innerText().then(text=>Number(text.replace('%','')))
 const before=await zoom(),box=await page.getByTestId('diagram-canvas').boundingBox()
 await page.mouse.move(box!.x+box!.width/2,box!.y+box!.height/2)
 await page.mouse.wheel(0,-500);await expect.poll(zoom).toBeGreaterThan(before)
 const enlarged=await zoom();await page.mouse.wheel(0,500);await expect.poll(zoom).toBeLessThan(enlarged)
})

test('save and switch waits for real disk save and locks competing actions',async({page})=>{
 const {root,current,title}=await setup(page)
 await title.fill('保存后切换的标题');await title.press('Tab')
 await page.locator('.diagram-nav[data-type="table"]').first().click()
 const dialog=page.getByRole('dialog',{name:'切换当前文件？',exact:true});await expect(dialog).toBeVisible();await expect(dialog).toHaveCSS('transform','none')
 const output=path.resolve('..','test-results',`${version}-previews`);fs.mkdirSync(output,{recursive:true})
 await page.screenshot({path:path.join(output,'save-and-switch.png')})
 let release!:()=>void,started!:()=>void
 const gate=new Promise<void>(resolve=>release=resolve),pending=new Promise<void>(resolve=>started=resolve)
 await page.route('**/api/workspace/write',async route=>{started();await gate;await route.continue()})
 await dialog.getByRole('button',{name:'保存并切换',exact:true}).click();await pending
 await expect(dialog.getByRole('button',{name:'取消',exact:true})).toBeDisabled()
 await expect(dialog.getByRole('button',{name:'放弃并切换',exact:true})).toBeDisabled()
 expect(await page.locator('.diagram-nav.active').getAttribute('data-entry-path')).toBe(current)
 release();await expect(dialog).toBeHidden()
 await expect(page.locator('.diagram-nav.active')).toHaveAttribute('data-type','table')
 expect(JSON.parse(fs.readFileSync(path.join(root,current),'utf8')).title).toBe('保存后切换的标题')
})

test('save conflict keeps current edits and the switch dialog open',async({page})=>{
 const {root,current,title}=await setup(page)
 await title.fill('仍未保存的本地编辑');await title.press('Tab')
 const file=path.join(root,current),external=JSON.parse(fs.readFileSync(file,'utf8'));external.title='外部的新版本';fs.writeFileSync(file,JSON.stringify(external))
 await page.locator('.diagram-nav[data-type="table"]').first().click()
 const dialog=page.getByRole('dialog',{name:'切换当前文件？',exact:true})
 await dialog.getByRole('button',{name:'保存并切换',exact:true}).click()
 await expect(dialog.getByRole('alert')).toContainText('其他程序修改')
 await expect(dialog).toBeVisible()
 expect(await page.locator('.diagram-nav.active').getAttribute('data-entry-path')).toBe(current)
 await expect(title).toHaveValue('仍未保存的本地编辑')
 expect(JSON.parse(fs.readFileSync(file,'utf8')).title).toBe('外部的新版本')
})

test('save and switch validates JSON drafts before writing or navigating',async({page})=>{
 const {root,current}=await setup(page),file=path.join(root,current),original=fs.readFileSync(file,'utf8')
 await page.getByRole('button',{name:'JSON',exact:true}).click()
 const editor=page.getByRole('textbox',{name:'图表 JSON'}),draft=JSON.parse(await editor.inputValue())
 await editor.fill('{broken')
 await page.locator('.diagram-nav[data-type="table"]').first().click()
 const dialog=page.getByRole('dialog',{name:'切换当前文件？',exact:true})
 await dialog.getByRole('button',{name:'保存并切换',exact:true}).click()
 await expect(dialog.getByRole('alert')).toBeVisible();await expect(dialog).toBeVisible()
 expect(fs.readFileSync(file,'utf8')).toBe(original)
 await dialog.getByRole('button',{name:'取消',exact:true}).click()
 await expect(editor).toHaveValue('{broken')
 draft.title='JSON 草稿已保存';await editor.fill(JSON.stringify(draft))
 await page.locator('.diagram-nav[data-type="table"]').first().click()
 await dialog.getByRole('button',{name:'保存并切换',exact:true}).click()
 await expect(dialog).toBeHidden();await expect(page.locator('.diagram-nav.active')).toHaveAttribute('data-type','table')
 expect(JSON.parse(fs.readFileSync(file,'utf8')).title).toBe('JSON 草稿已保存')
})
