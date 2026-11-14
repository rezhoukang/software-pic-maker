import {test,expect,type Page} from '@playwright/test'
import fs from 'node:fs'
import path from 'node:path'
import type {Diagram} from '../src/types'

const library=JSON.parse(fs.readFileSync(path.resolve('..','shared/library.json'),'utf8')) as Diagram[]

const row=(page:Page,path:string)=>page.locator(`.file-tree-row[data-entry-path="${path}"]`)

test.beforeEach(async({page})=>{
 await page.route('**/api/workspace/list',route=>route.fulfill({json:{root:'test-results/file-workspaces',entries:[
  {name:'项目1',path:'项目1',kind:'folder',children:[
   {name:'设计目录1',path:'项目1/设计目录1',kind:'folder',children:[
    {name:'架构1.json',path:'项目1/设计目录1/架构1.json',kind:'file',diagram_type:'architecture'},
   ]},
   {name:'空目录1',path:'项目1/空目录1',kind:'folder',children:[]},
  ]},
  {name:'空项目1',path:'空项目1',kind:'folder',children:[]},
 ]}}))
 await page.route('**/api/workspace/read',async route=>{
  const {path}=route.request().postDataJSON()
  await route.fulfill({json:{path,diagram:library.find(d=>d.type==='architecture'),revision:'test'}})
 })
 await page.route('**/api/render',route=>route.fulfill({json:{validation:{valid:true,issues:[]},scene:{width:800,height:600,nodes:[],edges:[],type:'architecture',title:'测试',theme:'blue'},svg:'<svg xmlns="http://www.w3.org/2000/svg"/>',node_svgs:{}}}))
 await page.goto('/')
 await expect(row(page,'项目1/设计目录1/架构1.json')).toBeVisible()
})

test('search auto-reveals matches but allows each folder to collapse and expand',async({page})=>{
 const search=page.getByRole('textbox',{name:'搜索文件或文件夹'})
 const project=row(page,'项目1'),folder=row(page,'项目1/设计目录1'),file=row(page,'项目1/设计目录1/架构1.json')
 await folder.click()
 await expect(file).toBeHidden()
 await search.fill('1')
 await expect(file).toBeVisible()
 await page.locator('.sidebar').screenshot({path:'test-results/file-tree-search-expanded.png'})
 await folder.click()
 await expect(folder).toHaveAttribute('aria-expanded','false')
 await expect(file).toBeHidden()
 await project.click()
 await expect(folder).toBeHidden()
 await project.click()
 await expect(folder).toHaveAttribute('aria-expanded','false')
 await folder.click()
 await expect(file).toBeVisible()
 await project.click()
 await page.getByRole('button',{name:'刷新目录',exact:true}).click()
 await expect(folder).toBeHidden()
 await search.fill('')
 await expect(project).toHaveAttribute('aria-expanded','true')
 await expect(folder).toHaveAttribute('aria-expanded','false')
 await search.fill('架构1')
 await expect(file).toBeVisible()
})

test('category filtering allows collapse and empty folders always show the collapsed arrow',async({page})=>{
 const empty=row(page,'空项目1')
 await expect(empty).toHaveAttribute('aria-expanded','false')
 await expect(empty.locator('.tree-disclosure')).toHaveText('▸')
 await empty.click()
 await expect(empty).toHaveAttribute('aria-expanded','false')
 await page.getByRole('combobox',{name:'分类筛选'}).click()
 await page.locator('.n-base-select-option:visible').filter({hasText:/^架构图$/}).click()
 const project=row(page,'项目1'),file=row(page,'项目1/设计目录1/架构1.json')
 await expect(file).toBeVisible()
 await project.click()
 await expect(file).toBeHidden()
 await project.click()
 await expect(file).toBeVisible()
 const filteredEmpty=row(page,'项目1/空目录1')
 await expect(filteredEmpty).toHaveAttribute('aria-expanded','false')
 await expect(filteredEmpty.locator('.tree-disclosure')).toHaveText('▸')
 await page.getByRole('textbox',{name:'搜索文件或文件夹'}).fill('不存在的项目')
 await expect(row(page,'')).toHaveAttribute('aria-expanded','false')
 await expect(row(page,'').locator('.tree-disclosure')).toHaveText('▸')
 await page.locator('.sidebar').screenshot({path:'test-results/file-tree-empty-search.png'})
})
