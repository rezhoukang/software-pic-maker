<script setup lang="ts">
import { ref, onMounted, onBeforeUnmount, watch, nextTick } from 'vue'
import { Graph } from '@antv/x6'
import { clone, type Diagram, type RenderResult } from '../types'
import { renderDiagram } from '../bridge'
const props = defineProps<{ result: RenderResult | null; diagram: Diagram; allowNodeDrag: boolean; diagramId: string; selected: string }>()
const emit = defineEmits<{ select: [id: string]; move: [id: string, x: number, y: number]; zoom: [value: number] }>()
const host = ref<HTMLElement>(); let graph: Graph; let observer: ResizeObserver; let previousId = ''; let rendering = false
const draggingId = ref('')
let dragScene: RenderResult | null = null
let previewTimer: ReturnType<typeof setTimeout> | undefined; let previewToken = 0; let previewRevision = 0; let previewRunning = false
Graph.registerNode('tuxu-node', { inherit: 'rect', markup: [{ tagName: 'rect', selector: 'selection' }, { tagName: 'image', selector: 'image' }, { tagName: 'rect', selector: 'boundaryTitle' }], attrs: { selection: { refWidth: '100%', refHeight: '100%', fill: 'transparent', stroke: 'transparent', strokeWidth: 2, rx: 7 }, image: { refWidth: '100%', refHeight: '100%', x: 0, y: 0, preserveAspectRatio: 'none' } } }, true)
function pointAt(points:{x:number;y:number}[],ratio:number){const lengths=points.slice(1).map((p,i)=>Math.hypot(p.x-points[i].x,p.y-points[i].y));let remaining=lengths.reduce((a,b)=>a+b,0)*ratio;for(let i=0;i<lengths.length;i++){if(lengths[i]>0&&remaining<=lengths[i]){const t=remaining/lengths[i];return {x:points[i].x+(points[i+1].x-points[i].x)*t,y:points[i].y+(points[i+1].y-points[i].y)*t}}remaining-=lengths[i]}return points.at(-1)!}
function edgeColor() { return props.result?.scene.theme === 'mono' ? '#333333' : props.result?.scene.theme === 'sage' ? '#5f7a71' : '#475569' }
function arrowMarker(arrow:string) {
  // X6 shortens the line by its 1.6px stroke. Anchor the visible marker tip
  // at the scene endpoint rather than at the center of the marker path.
  const refX = -1.6
  if (arrow==='none') return null
  if (arrow==='filled') return {name:'block',width:10,height:10,fill:edgeColor(),refX}
  if (arrow==='triangle') return {name:'block',width:13,height:12,fill:'#fff',refX}
  return {name:'path',d:'M 10 -5 L 0 0 L 10 5',offsetX:-5,fill:'none',strokeWidth:1.5,refX}
}
function buildCells(result:RenderResult) {
  const { scene, node_svgs } = result
  const cells: any[] = scene.nodes.map(n => ({ id: n.id, shape: 'tuxu-node', x: n.x, y: n.y, width: n.width, height: n.height, zIndex: n.shape === 'boundary' ? 0 : 2, data: { movable: n.movable, shape: n.shape }, attrs: { boundaryTitle: { x: 0, y: 0, refWidth: '100%', height: 48, fill: 'transparent', stroke: 'none', pointerEvents: n.shape === 'boundary' ? 'all' : 'none', cursor: 'pointer' }, image: { pointerEvents: n.shape === 'boundary' ? 'none' : 'auto', 'xlink:href': 'data:image/svg+xml;charset=utf-8,' + encodeURIComponent(node_svgs[n.id]) }, selection: { pointerEvents: n.shape === 'boundary' ? 'none' : 'auto', stroke: props.selected === n.id ? '#4f6df5' : 'transparent', strokeDasharray: '5 3' } } }))
  for (const e of scene.edges) {
    const pts = e.points; if (pts.length < 2) continue
    const endpoint = (id: string | undefined, point: {x:number;y:number}) => { const n = scene.nodes.find(n => n.id === id); return n ? { cell: id, anchor: { name: 'topLeft', args: { dx: point.x-n.x, dy: point.y-n.y } }, connectionPoint: { name: 'anchor' } } : point }
    const labels: any[] = []
    for (const [text, distance, y, position, angle] of [[e.label, .5, -14, e.label_position, e.label_angle ?? 0], [e.source_label, .12, e.label ? 18 : -15, e.source_label_position, 0], [e.target_label, .88, e.label ? 18 : -15, e.target_label_position, 0]] as [string | undefined, number, number, {x:number;y:number} | undefined, number][]) {
      if (!text) continue
      const base = pointAt(pts,distance)
      labels.push({ position: { distance, angle, offset: position ? {x:position.x-base.x,y:position.y-base.y} : {x:0,y} }, attrs: { label: { text, pointerEvents: 'all', cursor: 'pointer', fill: edgeColor(), fontSize: 14, fontFamily: 'Microsoft YaHei, sans-serif' }, body: { fill: '#fff', stroke: 'none', rx: 3, pointerEvents: 'none' } } })
    }
    cells.push({ id: e.id, shape: 'edge', source: endpoint(e.source, pts[0]), target: endpoint(e.target, pts.at(-1)!), vertices: pts.slice(1, -1), zIndex: 1, router: { name: 'normal' }, connector: { name: 'normal' }, labels, attrs: { line: { stroke: props.selected === e.id ? '#4f6df5' : edgeColor(), strokeWidth: 1.6, strokeDasharray: e.dashed ? '7 5' : '', targetMarker: arrowMarker(e.arrow) } } })
  }
  return cells
}
function update() {
  if (!graph || !props.result?.validation.valid || draggingId.value) return
  rendering = true
  graph.fromJSON({cells:buildCells(props.result)}); rendering = false
  if (previousId !== props.diagramId) { previousId = props.diagramId; void nextTick(fit) }
}
// Patch existing cells during a drag: replacing the graph would destroy X6's
// active mouse gesture. The same backend scene still drives saved/exported data.
function applyDragPreview(result:RenderResult) {
  const cells=buildCells(result)
  rendering=true
  try {graph.batchUpdate(()=>{
    const ids=new Set(cells.map(c=>c.id))
    graph.getCells().forEach(cell=>{if(!ids.has(cell.id))graph.removeCell(cell)})
    for(const config of cells){
      const cell=graph.getCellById(config.id)
      if(!cell){if(config.shape==='edge')graph.addEdge(config);else graph.addNode(config);continue}
      if(cell.isNode()){
        if(cell.id!==draggingId.value)cell.position(config.x,config.y)
        cell.resize(config.width,config.height)
      }else if(cell.isEdge()){
        cell.setSource(config.source);cell.setTarget(config.target);cell.setVertices(config.vertices);cell.setLabels(config.labels)
      }
      cell.setAttrs(config.attrs)
    }
  })
  dragScene=result
  straightenDragEndpoints()
  }finally{rendering=false}
}
function straightenDragEndpoints() {
  const result=dragScene??props.result
  if(!result||!['structure','flowchart','class','table','architecture'].includes(result.scene.type))return
  for(const edge of result.scene.edges){
    if(edge.source!==draggingId.value&&edge.target!==draggingId.value)continue
    const cell=graph.getCellById(edge.id);if(!cell?.isEdge())continue
    const points=edge.points.map(p=>({...p}));if(points.length<2)continue
    const shift=(id:string|undefined,index:number)=>{
      const original=result.scene.nodes.find(n=>n.id===id),node=id?graph.getCellById(id):undefined
      if(original&&node?.isNode()){const p=node.position();points[index].x+=p.x-original.x;points[index].y+=p.y-original.y}
    }
    shift(edge.source,0);shift(edge.target,points.length-1)
    if(points.length>2){
      const old=edge.points,last=points.length-1
      if(Math.abs(old[0].x-old[1].x)<.01)points[1].x=points[0].x;else points[1].y=points[0].y
      if(Math.abs(old[last].x-old[last-1].x)<.01)points[last-1].x=points[last].x;else points[last-1].y=points[last].y
    }else if(Math.abs(points[0].x-points[1].x)>.01&&Math.abs(points[0].y-points[1].y)>.01){
      const [a,b]=points
      if(Math.abs(edge.points[0].y-edge.points[1].y)<.01){const mid=(a.x+b.x)/2;points.splice(1,0,{x:mid,y:a.y},{x:mid,y:b.y})}
      else{const mid=(a.y+b.y)/2;points.splice(1,0,{x:a.x,y:mid},{x:b.x,y:mid})}
    }
    cell.setVertices(points.slice(1,-1))
  }
}
function schedulePreview() {
  ++previewRevision
  if(!previewTimer&&!previewRunning)previewTimer=setTimeout(()=>void previewDrag(),50)
}
async function previewDrag() {
  previewTimer=undefined
  const id=draggingId.value
  if(!id)return
  const node=graph.getCellById(id);if(!node?.isNode())return
  const diagram=clone(props.diagram);const p=node.position()
  diagram.view.positions[id]={x:Math.max(0,p.x),y:Math.max(0,p.y)}
  const token=previewToken,revision=previewRevision
  previewRunning=true
  try{
    const next=await renderDiagram(diagram)
    if(token===previewToken&&draggingId.value===id&&next.validation.valid)applyDragPreview(next)
  }catch{
    // The committed render on release reports backend errors through App.vue.
  }finally{
    previewRunning=false
    if(draggingId.value&&revision!==previewRevision&&!previewTimer)previewTimer=setTimeout(()=>void previewDrag(),50)
  }
}
function endPreview(){++previewToken;clearTimeout(previewTimer);previewTimer=undefined;draggingId.value='';dragScene=null}
function fit() { if (!graph?.getCells().length) return; graph.zoomToFit({ padding: 42, maxScale: 1 }); emit('zoom', graph.zoom()) }
function zoom(delta: number) { if (graph) { graph.zoom(delta); emit('zoom', graph.zoom()) } }
function resetZoom() { if (graph) { graph.zoomTo(1); graph.centerContent(); emit('zoom', 1) } }
onMounted(() => {
  graph = new Graph({ container: host.value!, autoResize: false, background: { color: '#fafbfe' }, grid: { visible: true, type: 'dot', size: 20, args: { color: '#dce2ec', thickness: 1 } }, panning: { enabled: true, eventTypes: ['leftMouseDown'] }, mousewheel: { enabled: true, zoomAtMousePosition: true, minScale: .15, maxScale: 2.5 }, interacting: cellView => ({ nodeMovable: props.allowNodeDrag && cellView.cell.isNode() && cellView.cell.getData()?.movable, edgeLabelMovable: false, edgeMovable: false, arrowheadMovable: false, vertexMovable: false, vertexAddable: false, vertexDeletable: false }), translating: { restrict: (view) => view?.cell.getData()?.shape === 'participant' ? { x: 0, y: 64, width: 20000, height: 58 } : { x: 0, y: 0, width: 20000, height: 20000 } }, connecting: { allowBlank: false } })
  graph.on('node:click', ({ node }) => { if (node.getData()?.movable || node.getData()?.shape === 'boundary') emit('select', node.id) })
  graph.on('edge:click', ({ edge }) => emit('select', edge.id.startsWith('$junction-edge:')?'':edge.id))
  graph.on('blank:click', () => emit('select', ''))
  graph.on('node:move', ({node})=>{if(!rendering&&props.allowNodeDrag){draggingId.value=node.id;dragScene=props.result;++previewToken}})
  graph.on('node:moving', ()=>{if(!rendering&&props.allowNodeDrag&&draggingId.value){straightenDragEndpoints();schedulePreview()}})
  graph.on('node:moved', ({ node }) => { if (!rendering && props.allowNodeDrag && draggingId.value === node.id) { endPreview(); const p = node.position(); emit('move', node.id, Math.max(0, p.x), Math.max(0, p.y)) } })
  graph.on('scale', ({ sx }) => emit('zoom', sx))
  observer = new ResizeObserver(entries => { const r = entries[0].contentRect; graph.resize(r.width, r.height); if (previousId) fit() }); observer.observe(host.value!.parentElement!)
  update()
})
watch(() => props.result, update)
watch(() => props.allowNodeDrag, enabled => { if (!enabled && draggingId.value) { endPreview(); update() } })
watch(() => props.selected, id => { if (!graph) return; graph.getNodes().forEach(n => n.attr('selection/stroke', n.id === id ? '#4f6df5' : 'transparent')); graph.getEdges().forEach(e => e.attr('line/stroke', e.id === id ? '#4f6df5' : edgeColor())) })
onBeforeUnmount(() => { endPreview(); observer?.disconnect(); graph?.dispose() })
defineExpose({ fit, zoom, resetZoom })
</script>
<template><div ref="host" class="diagram-canvas" data-testid="diagram-canvas" :data-rendered-type="result?.scene.type" :data-dragging="draggingId" :data-node-drag-enabled="allowNodeDrag"></div></template>
