import {test,expect,type Page} from '@playwright/test'
import fs from 'node:fs'
import path from 'node:path'

const row=(page:Page,p:string)=>page.locator(`.file-tree-row[data-entry-path="${p}"]`)
async function menu(page:Page,p:string,label:string){await row(page,p).click({button:'right'});await page.locator('.n-dropdown-option:visible').filter({hasText:new RegExp('^'+label+'$')}).click()}
async function folder(page:Page,parent:string,name:string){await menu(page,parent,'新增文件夹');const dialog=page.getByRole('dialog',{name:'新增文件夹',exact:true});await dialog.locator('input').fill(name);await dialog.getByRole('button',{name:'确定',exact:true}).click();await expect(dialog).toBeHidden()}
async function rootPath(page:Page):Promise<string>{const result=await page.evaluate(async()=>{const response=await fetch('/api/workspace/list',{method:'POST',headers:{'Content-Type':'application/json','X-EZ-Workspace':localStorage.getItem('ezbitu-preview-workspace')!},body:'{}'});return response.json()});expect(result.root).toContain('file-workspaces');return result.root}

test('real nested folders, file menus, classification, renames and recoverable deletion',async({page})=>{
 await page.goto('/');await expect(page.locator('.x6-node').first()).toBeVisible()
 await expect(page.locator('.document-heading,.heading-badge,.file-tree-row svg')).toHaveCount(0)
 const root=await rootPath(page)
 await folder(page,'','数据库设计');await folder(page,'数据库设计','用户模块')
 await menu(page,'数据库设计/用户模块','新增文件')
 const create=page.getByRole('dialog',{name:'新增文件',exact:true})
 await create.getByRole('textbox',{name:'文件名称'}).fill('用户表')
 await create.locator('.n-base-selection').click();await page.locator('.n-base-select-option:visible').filter({hasText:/^表关系图$/}).click()
 await create.getByRole('button',{name:'创建文件',exact:true}).click();await expect(create).toBeHidden()
 const old='数据库设计/用户模块/用户表.json'
 await expect(row(page,old)).toHaveClass(/active/)
 expect(JSON.parse(fs.readFileSync(path.join(root,old),'utf8')).type).toBe('table')
 await row(page,old).click({button:'right'})
 await expect(page.locator('.n-dropdown-option:visible')).toHaveCount(3)
 await expect(page.locator('.n-dropdown-option:visible').filter({hasText:'设计表'})).toBeVisible()
 await page.locator('.n-dropdown-option:visible').filter({hasText:'重命名'}).click()
 let rename=page.getByRole('dialog',{name:'重命名',exact:true});await rename.locator('input').fill('账户表');await rename.getByRole('button',{name:'确定',exact:true}).click()
 await expect(row(page,'数据库设计/用户模块/账户表.json')).toHaveClass(/active/)
 expect(fs.existsSync(path.join(root,old))).toBe(false)
 await page.getByRole('button',{name:'JSON',exact:true}).click();const editor=page.getByRole('textbox',{name:'图表 JSON'});await editor.fill('{尚未应用的草稿')
 await menu(page,'数据库设计','重命名');rename=page.getByRole('dialog',{name:'重命名',exact:true});await rename.locator('input').fill('数据模型');await rename.getByRole('button',{name:'确定',exact:true}).click()
 await expect(editor).toHaveValue('{尚未应用的草稿');await expect(page.locator('.file-breadcrumb')).toContainText('数据模型')
 await page.getByRole('combobox',{name:'分类筛选'}).click();await page.locator('.n-base-select-option:visible').filter({hasText:/^表关系图$/}).click()
 await expect(page.locator('.diagram-nav[data-type="er"]')).toHaveCount(0);await expect(page.locator('.diagram-nav[data-type="table"]')).toHaveCount(2)
 await page.getByRole('combobox',{name:'分类筛选'}).click();await page.locator('.n-base-select-option:visible').filter({hasText:/^全部分类$/}).click()
 await menu(page,'数据模型','删除');let remove=page.getByRole('dialog',{name:'删除',exact:true});await expect(remove).toContainText('未保存的编辑');await remove.getByRole('button',{name:'取消',exact:true}).click();await expect(editor).toHaveValue('{尚未应用的草稿')
 await menu(page,'数据模型','删除');remove=page.getByRole('dialog',{name:'删除',exact:true});await remove.getByRole('button',{name:'确认删除',exact:true}).click()
 await expect(row(page,'数据模型')).toHaveCount(0);await expect(page.getByRole('heading',{name:'从主目录选择一个文件'})).toBeVisible()
 expect(fs.existsSync(path.join(root,'数据模型'))).toBe(false)
 const deleted=fs.readdirSync(path.join(root,'.ezbitu')).filter(n=>n.startsWith('deleted-'))
 expect(deleted.some(n=>fs.existsSync(path.join(root,'.ezbitu',n,'数据模型','用户模块','账户表.json')))).toBe(true)
 await page.screenshot({path:'test-results/file-manager-after-delete.png'})
})

test('saved files persist and external changes cannot be overwritten',async({page})=>{
 await page.goto('/');await expect(page.locator('.x6-node').first()).toBeVisible();const root=await rootPath(page)
 const active=await page.locator('.diagram-nav.active').getAttribute('data-entry-path');expect(active).toBeTruthy()
 const title=page.getByRole('textbox',{name:'图表名称',exact:true});await title.fill('已保存的磁盘标题');await title.press('Tab');await page.getByRole('button',{name:'保存文件',exact:true}).click();await expect(page.getByRole('status')).toContainText('文件已保存')
 expect(JSON.parse(fs.readFileSync(path.join(root,active!),'utf8')).title).toBe('已保存的磁盘标题')
 await page.reload();await expect(title).toHaveValue('已保存的磁盘标题')
 await title.fill('编辑器中的修改');await title.press('Tab')
 const file=path.join(root,active!);const external=JSON.parse(fs.readFileSync(file,'utf8'));external.title='外部程序的新标题';fs.writeFileSync(file,JSON.stringify(external))
 await page.getByRole('button',{name:'保存文件',exact:true}).click();await expect(page.getByRole('status')).toContainText('其他程序修改')
 await expect(title).toHaveValue('编辑器中的修改');expect(JSON.parse(fs.readFileSync(file,'utf8')).title).toBe('外部程序的新标题')
})

test('file navigation protects unsaved edits and refresh reads real external folders',async({page})=>{
 await page.goto('/');await expect(page.locator('.x6-node').first()).toBeVisible();const root=await rootPath(page)
 const title=page.getByRole('textbox',{name:'图表名称',exact:true});await title.fill('尚未保存');await title.press('Tab')
 await page.locator('.diagram-nav[data-type="table"]').first().click()
 const warning=page.locator('.compact-modal').filter({hasText:'切换当前文件？'});await expect(warning).toBeVisible();await warning.getByRole('button',{name:'取消',exact:true}).click();await expect(title).toHaveValue('尚未保存')
 fs.mkdirSync(path.join(root,'外部新建文件夹'));await page.getByRole('button',{name:'刷新目录',exact:true}).click();await expect(row(page,'外部新建文件夹')).toBeVisible()
 await page.screenshot({path:'test-results/file-manager-main.png'})
})
