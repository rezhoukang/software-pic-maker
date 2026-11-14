import {test,expect} from '@playwright/test'
import fs from 'node:fs'
import path from 'node:path'
const version=JSON.parse(fs.readFileSync(new URL('../package.json',import.meta.url),'utf8')).version

test('node dragging is opt-in and resets after auto layout, navigation and reload',async({page})=>{
 await page.goto('/');await expect(page.locator('.x6-node').first()).toBeVisible()
 const toggle=page.getByRole('button',{name:'手动调整',exact:true})
 const node=page.locator('.x6-node[data-cell-id="reader"]')
 const drag=async()=>{
  const box=await node.boundingBox();expect(box).not.toBeNull()
  await page.mouse.move(box!.x+box!.width/2,box!.y+box!.height/2)
  await page.mouse.down();await page.mouse.move(box!.x+box!.width/2+45,box!.y+box!.height/2+35,{steps:8});await page.mouse.up()
 }
 await expect(toggle).toHaveAttribute('aria-pressed','false')
 const original=await node.getAttribute('transform')
 await drag();expect(await node.getAttribute('transform')).toBe(original)
 await node.click();await expect(page.locator('.property-heading strong')).toHaveText('读者')
 await page.getByRole('button',{name:'JSON',exact:true}).click()
 const editor=page.getByRole('textbox',{name:'图表 JSON'})
 expect(JSON.parse(await editor.inputValue()).view.positions).toEqual({})
 await page.getByRole('button',{name:'画布',exact:true}).click()
 await toggle.click();await expect(toggle).toHaveAttribute('aria-pressed','true')
 await drag();await expect(page.locator('.render-progress')).toHaveCount(0)
 expect(await node.getAttribute('transform')).not.toBe(original)
 await toggle.click();await expect(toggle).toHaveAttribute('aria-pressed','false')
 const adjusted=await node.getAttribute('transform')
 await drag();expect(await node.getAttribute('transform')).toBe(adjusted)
 await page.getByRole('button',{name:'JSON',exact:true}).click()
 expect(JSON.parse(await editor.inputValue()).view.positions.reader).toBeDefined()
 await page.getByRole('button',{name:'画布',exact:true}).click()
 await toggle.click()
 await page.getByRole('button',{name:'自动排版',exact:true}).click()
 await expect(toggle).toHaveAttribute('aria-pressed','false')
 await page.getByRole('button',{name:'保存文件',exact:true}).click()
 await expect(page.getByRole('status')).toContainText('文件已保存')
 await toggle.click()
 await page.locator('.diagram-nav[data-type="table"]').first().click()
 await expect(page.getByTestId('diagram-canvas')).toHaveAttribute('data-rendered-type','table')
 await expect(toggle).toHaveAttribute('aria-pressed','false')
 await toggle.click();await page.reload();await expect(toggle).toHaveAttribute('aria-pressed','false')
})

test('drag reroutes sibling rail before release and saves as a single undo step',async({page})=>{
 await page.goto('/');await page.locator('.diagram-nav[data-type="structure"]').first().click()
 await expect(page.locator('.render-progress')).toHaveCount(0)
 await page.getByRole('button',{name:'手动调整',exact:true}).click()
 const node=page.locator('.x6-node[data-cell-id="search"]')
 const box=await node.boundingBox();expect(box).not.toBeNull()
 const response=page.waitForResponse(r=>r.url().endsWith('/api/render')&&Boolean(r.request().postDataJSON()?.view?.positions?.search))
 await page.mouse.move(box!.x+box!.width/2,box!.y+box!.height/2);await page.mouse.down()
 await page.mouse.move(box!.x+box!.width/2+45,box!.y+box!.height/2+85,{steps:10})
 await expect(page.getByTestId('diagram-canvas')).toHaveAttribute('data-dragging','search')
 const preview=await (await response).json();expect(preview.validation.valid).toBe(true)
 const edge=(id:string)=>page.locator(`.x6-edge[data-cell-id="${id}"] path`).nth(1)
 const expected=preview.scene.edges.find((e:any)=>e.id==='s3')
 await expect.poll(async()=>{
  const d=await edge('s3').getAttribute('d');const numbers=d?.match(/-?\d+(?:\.\d+)?/g)?.map(Number)||[]
  // Scene uses the single shared rail, not the moved child's midpoint.
  return Math.abs(numbers[3]-expected.points[1].y)
 }).toBeLessThan(.1)
 await expect.poll(async()=>{
  const d=await edge('s3').getAttribute('d');const n=d?.match(/-?\d+(?:\.\d+)?/g)?.map(Number)||[]
  for(let i=2;i<n.length;i+=2)if(Math.abs(n[i]-n[i-2])>.1&&Math.abs(n[i+1]-n[i-1])>.1)return false
  return true
 }).toBe(true)
 const output=path.resolve('..','test-results',`${version}-previews`);fs.mkdirSync(output,{recursive:true})
 await page.locator('.canvas-area').screenshot({path:path.join(output,'structure-drag-held.png')})
 await page.mouse.up()
 await expect(page.getByTestId('diagram-canvas')).toHaveAttribute('data-dragging','')
 await expect(page.locator('.render-progress')).toHaveCount(0)
 await page.getByRole('button',{name:'JSON',exact:true}).click()
 const editor=page.getByRole('textbox',{name:'图表 JSON'});const moved=JSON.parse(await editor.inputValue())
 expect(moved.view.positions.search).toBeDefined()
 await page.getByRole('button',{name:'画布',exact:true}).click()
 await page.getByRole('button',{name:'撤销',exact:true}).click()
 await page.getByRole('button',{name:'JSON',exact:true}).click()
 expect(JSON.parse(await editor.inputValue()).view.positions.search).toBeUndefined()
 await page.getByRole('button',{name:'画布',exact:true}).click()
 await page.getByRole('button',{name:'重做',exact:true}).click()
 await page.getByRole('button',{name:'保存文件',exact:true}).click()
 await expect(page.getByRole('status')).toContainText('文件已保存')
 await page.reload();await expect(node).toBeVisible()
 await page.getByRole('button',{name:'JSON',exact:true}).click()
 expect(JSON.parse(await editor.inputValue()).view.positions.search).toEqual(moved.view.positions.search)
 await page.getByRole('button',{name:'画布',exact:true}).click()
 await page.getByRole('button',{name:'全屏画布',exact:true}).click()
 await page.getByRole('button',{name:'适应画布',exact:true}).click()
 await page.locator('.canvas-area').screenshot({path:path.join(output,'structure-drag-saved.png')})
})

test('late drag preview cannot overwrite the released class layout',async({page})=>{
 await page.goto('/');await page.locator('.diagram-nav[data-type="class"]').first().click()
 await expect(page.locator('.render-progress')).toHaveCount(0)
 await page.getByRole('button',{name:'手动调整',exact:true}).click()
 let release!:()=>void,started!:()=>void
 const gate=new Promise<void>(resolve=>release=resolve),pending=new Promise<void>(resolve=>started=resolve)
 let held=false
 await page.route('**/api/render',async route=>{
  if(!held&&route.request().postDataJSON()?.view?.positions?.service){
   held=true;const response=await route.fetch();started();await gate;await route.fulfill({response})
  }else await route.continue()
 })
 const node=page.locator('.x6-node[data-cell-id="service"]');const box=await node.boundingBox();expect(box).not.toBeNull()
 const x=box!.x+box!.width/2,y=box!.y+box!.height/2
 await page.mouse.move(x,y);await page.mouse.down();await page.mouse.move(x+35,y+20,{steps:5});await pending
 await page.mouse.move(x+85,y+85,{steps:5});await page.mouse.up()
 await expect(page.getByTestId('diagram-canvas')).toHaveAttribute('data-dragging','')
 await expect(page.locator('.render-progress')).toHaveCount(0)
 const line=page.locator('.x6-edge[data-cell-id="c1"] path').nth(1)
 const before=await line.getAttribute('d'),nodeBefore=await node.boundingBox()
 const finished=page.waitForResponse(r=>r.url().endsWith('/api/render'))
 release();await finished
 await page.evaluate(()=>new Promise<void>(resolve=>requestAnimationFrame(()=>requestAnimationFrame(()=>resolve()))))
 expect(await line.getAttribute('d')).toBe(before)
 expect(await node.boundingBox()).toEqual(nodeBefore)
})
