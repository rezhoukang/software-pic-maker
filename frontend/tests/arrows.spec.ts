import { test, expect } from '@playwright/test'
import fs from 'node:fs'
import path from 'node:path'

const library=JSON.parse(fs.readFileSync(new URL('../../shared/library.json',import.meta.url),'utf8'))
const version=JSON.parse(fs.readFileSync(new URL('../package.json',import.meta.url),'utf8')).version
const output=path.resolve('..','test-results',`${version}-previews`)

for(const diagram of library) test(`${diagram.type} actual canvas arrow endpoints and vector preview`,async({page})=>{
 fs.mkdirSync(output,{recursive:true})
 await page.goto('/')
 await page.locator(`.diagram-nav[data-type="${diagram.type}"]`).first().click()
 await expect(page.getByTestId('diagram-canvas')).toHaveAttribute('data-rendered-type',diagram.type)
 await expect(page.locator('.render-progress')).toHaveCount(0)
 const response=await page.request.post('http://127.0.0.1:34116/api/render',{data:diagram})
 const result=await response.json();expect(result.validation.valid).toBe(true)
 fs.writeFileSync(path.join(output,`${diagram.type}.svg`),result.svg)
 for(const edge of result.scene.edges){
  const line=page.locator(`.x6-edge[data-cell-id="${edge.id}"] path`).nth(1)
  await expect(line).toBeAttached()
  const actual=await line.evaluate(el=>{const p=el as SVGPathElement;const a=p.getPointAtLength(0),b=p.getPointAtLength(p.getTotalLength());return{start:{x:a.x,y:a.y},end:{x:b.x,y:b.y}}})
  expect(Math.hypot(actual.start.x-edge.points[0].x,actual.start.y-edge.points[0].y),`${edge.id} source attachment`).toBeLessThan(.1)
  const marker=await line.getAttribute('marker-end')
  if(edge.arrow==='none'){expect(marker||'').not.toMatch(/url\(/);const end=edge.points.at(-1);expect(Math.hypot(actual.end.x-end.x,actual.end.y-end.y)).toBeLessThan(.1)}
  else {
   expect(marker).toMatch(/url\(/)
   const id=marker!.match(/#([^)]*)/)![1]
   const markerElement=page.locator(`[id="${id}"]`)
   const shape=markerElement.locator('path')
   const tipOffset=await markerElement.evaluate(el=>{const marker=el as SVGMarkerElement;const shape=marker.querySelector('path') as SVGPathElement;const box=shape.getBBox();return -box.x-marker.refX.baseVal.value})
   const end=edge.points.at(-1),prior=edge.points.at(-2),length=Math.hypot(end.x-prior.x,end.y-prior.y)
   const tip={x:actual.end.x+(end.x-prior.x)/length*tipOffset,y:actual.end.y+(end.y-prior.y)/length*tipOffset}
   expect(Math.hypot(tip.x-end.x,tip.y-end.y),`${edge.id} visible arrow tip`).toBeLessThan(.1)
   if(edge.arrow==='open')await expect(shape).toHaveAttribute('fill','none')
   if(edge.arrow==='triangle')await expect(shape).toHaveAttribute('fill','#fff')
  }
 }
 await page.getByRole('button',{name:'全屏画布',exact:true}).click()
 await expect(page.locator('.app-shell')).toHaveClass(/canvas-focus/)
 await expect.poll(()=>page.locator('.diagram-canvas').evaluate(el=>Math.round(el.getBoundingClientRect().width))).toBe(1480)
 await page.getByRole('button',{name:'适应画布',exact:true}).click()
 await page.locator('.canvas-area').screenshot({path:path.join(output,`${diagram.type}.png`)})
 // Render the exported SVG in the same browser to compare it visually with X6.
 const vector=await page.context().newPage()
 await vector.setContent(result.svg)
 await vector.locator('svg').screenshot({path:path.join(output,`${diagram.type}-svg.png`)})
 await vector.close()
})

test('class inheritance marker and three incoming arrows remain distinct in X6',async({page})=>{
 await page.goto('/');await page.locator('.diagram-nav[data-type="class"]').first().click()
 const d=structuredClone(library.find((d:any)=>d.type==='class'))
 d.model.classes=[{id:'base',name:'Base'},{id:'first',name:'First'},{id:'second',name:'Second'},{id:'third',name:'Third'}]
 d.model.relations=['first','second','third'].map((source,i)=>({id:`inherit${i}`,source,target:'base',kind:'inheritance'}))
 d.view.positions={base:{x:650,y:240},first:{x:80,y:80},second:{x:80,y:300},third:{x:80,y:520}}
 await page.getByRole('button',{name:'JSON',exact:true}).click()
 await page.getByRole('textbox',{name:'图表 JSON'}).fill(JSON.stringify(d))
 await page.getByRole('button',{name:'校验并应用',exact:true}).click()
 await expect(page.getByRole('status')).toContainText('JSON 已应用')
 await page.getByRole('button',{name:'画布',exact:true}).click()
 const points=[]
 for(let i=0;i<3;i++){
  const line=page.locator(`.x6-edge[data-cell-id="inherit${i}"] path`).nth(1)
  await expect(line).toHaveAttribute('marker-end',/url\(/)
  points.push(await line.evaluate(el=>{const p=el as SVGPathElement;const end=p.getPointAtLength(p.getTotalLength());return`${end.x.toFixed(1)},${end.y.toFixed(1)}`}))
 }
 expect(new Set(points).size).toBe(3)
 await page.getByRole('button',{name:'全屏画布',exact:true}).click()
 await page.getByRole('button',{name:'适应画布',exact:true}).click()
 await page.locator('.canvas-area').screenshot({path:path.join(output,'class-inheritance.png')})
})
