import {test,expect} from '@playwright/test'

test('MCP configuration and copied Skill use the same EXE with MCP mode',async({page})=>{
 await page.addInitScript(()=>{
  Object.defineProperty(navigator,'clipboard',{value:{writeText:async(text:string)=>{(window as any).copiedText=text}}})
 })
 await page.goto('/')
 await expect(page.locator('.x6-node').first()).toBeVisible()
 await page.getByRole('button',{name:'MCP 接入',exact:true}).click()
 const dialog=page.getByRole('dialog',{name:'MCP 接入',exact:true})
 const configText=await dialog.locator('.config-code').innerText()
 const config=JSON.parse(configText)
 expect(Object.keys(config.mcpServers)).toEqual(['ezruantu'])
 expect(config.mcpServers.ezruantu.command).toBe('EZ软图/EZ软图.exe')
 expect(config.mcpServers.ezruantu.args).toEqual(['--mcp','--workspace','EZ软图/data'])
 await expect(dialog).toContainText('MCP 内置于 EZ软图.exe')
 await dialog.getByRole('button',{name:'复制配置',exact:true}).click()
 expect(await page.evaluate(()=>(window as any).copiedText)).toBe(configText)
 await dialog.getByRole('button',{name:'复制 Skill',exact:true}).click()
 const skill=await page.evaluate(()=>(window as any).copiedText) as string
 expect(skill).toContain(configText)
 expect(skill).toContain('export_project')
 expect(skill).not.toContain('tuxu-mcp')
 await page.screenshot({path:'test-results/integrated-mcp-dialog.png'})
})
