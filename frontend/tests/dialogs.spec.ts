import {test,expect,type Locator} from '@playwright/test'
import fs from 'node:fs'
import path from 'node:path'
const version=JSON.parse(fs.readFileSync(new URL('../package.json',import.meta.url),'utf8')).version
const output=path.resolve('..','test-results',`${version}-previews`)

async function checkFrame(dialog:Locator,height:number){
 await expect(dialog).toHaveCSS('transform','none')
 const card=await dialog.boundingBox(),header=await dialog.locator('.n-card-header').boundingBox(),footer=await dialog.locator('.dialog-actions').boundingBox()
 expect(card!.y).toBeGreaterThanOrEqual(10);expect(card!.y+card!.height).toBeLessThanOrEqual(height-10)
 expect(header!.y).toBeGreaterThanOrEqual(card!.y);expect(footer!.y+footer!.height).toBeLessThanOrEqual(height-10)
 const overflow=await dialog.evaluate(el=>({overflow:getComputedStyle(el).overflowY,scrollTop:el.scrollTop}))
 expect(overflow.overflow).toBe('visible');expect(overflow.scrollTop).toBe(0)
}

for(const height of [740,440])test(`new-file dropdown preserves modal header/footer at ${height}px`,async({page})=>{
 fs.mkdirSync(output,{recursive:true});await page.setViewportSize({width:1120,height})
 await page.goto('/');await expect(page.locator('.x6-node').first()).toBeVisible()
 await page.getByRole('button',{name:'新增文件',exact:true}).click()
 const dialog=page.getByRole('dialog',{name:'新增文件',exact:true})
 await expect(dialog).toBeVisible();await checkFrame(dialog,height)
 await dialog.getByRole('textbox',{name:'文件名称',exact:true}).fill('测试功能图')
 await page.screenshot({path:path.join(output,`new-file-${height}.png`)})
 const before=await dialog.boundingBox()
 await dialog.locator('.n-base-selection').click()
 const menu=page.locator('.n-base-select-menu:visible')
 await expect(menu).toBeVisible();await checkFrame(dialog,height)
 const after=await dialog.boundingBox();expect(Math.abs(after!.y-before!.y)).toBeLessThan(1)
 const option=page.locator('.n-base-select-option:visible').filter({hasText:/^功能结构图$/})
 await option.scrollIntoViewIfNeeded();await expect(option).toBeInViewport()
 const title=await dialog.getByText('新增文件',{exact:true}).boundingBox(),popup=await menu.boundingBox()
 expect(popup!.y+popup!.height<=title!.y||popup!.y>=title!.y+title!.height).toBe(true)
 await page.screenshot({path:path.join(output,`new-file-dropdown-${height}.png`)})
 await option.click();await expect(dialog.locator('.n-base-selection')).toContainText('功能结构图')
 await dialog.getByRole('button',{name:'创建文件',exact:true}).click();await expect(dialog).toBeHidden()
 await expect(page.getByTestId('diagram-canvas')).toHaveAttribute('data-rendered-type','structure')
})

test('long dialog bodies scroll independently from their titles and actions',async({page})=>{
 fs.mkdirSync(output,{recursive:true});await page.setViewportSize({width:1120,height:440})
 await page.goto('/');await page.locator('.diagram-nav[data-type="usecase"]').first().click()
 await expect(page.getByTestId('diagram-canvas')).toHaveAttribute('data-rendered-type','usecase')
 await page.getByRole('button',{name:'连线',exact:true}).click()
 const dialog=page.getByRole('dialog',{name:'建立关联关系'})
 await expect(dialog).toBeVisible();await checkFrame(dialog,440)
 const content=dialog.locator('.dialog-scroll-body')
 expect(await content.evaluate(el=>el.scrollHeight>el.clientHeight)).toBe(true)
 const before=await dialog.locator('.n-card-header').boundingBox()
 await content.evaluate(el=>el.scrollTop=el.scrollHeight)
 expect(await dialog.locator('.n-card-header').boundingBox()).toEqual(before)
 await checkFrame(dialog,440)
 await page.screenshot({path:path.join(output,'relation-dialog-short.png')})
 await dialog.getByRole('button',{name:'取消',exact:true}).click();await expect(dialog).toBeHidden()
 await page.getByRole('button',{name:'MCP 接入',exact:true}).click()
 const mcp=page.getByRole('dialog',{name:'MCP 接入',exact:true})
 await expect(mcp).toBeVisible();await checkFrame(mcp,440)
 await mcp.locator('.dialog-actions').getByRole('button',{name:'关闭',exact:true}).click()

})
