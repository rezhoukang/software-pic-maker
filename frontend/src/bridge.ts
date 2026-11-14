import type { Diagram, DiagramTheme, Project, RenderResult, Session, WorkspaceTree, WorkspaceDocument } from './types'
import library from '../../shared/library.json'
import { clone } from './types'
import { EventsOn } from '../wailsjs/runtime/runtime'

declare global { interface Window { go?: { main: { App: Record<string, (...args: any[]) => Promise<any>> } } } }
export const desktop = () => Boolean(window.go?.main?.App)
const api = () => window.go!.main.App
export async function loadSession(): Promise<Session> {
  if (desktop()) return api().GetSession()
  let recoveryError = ''
  const stored = localStorage.getItem('ezruantu-session-v1') ?? localStorage.getItem('tuxu-session-v1')
  if (stored) { try { const s = JSON.parse(stored); await validatePreviewProject(s.project); return { ...s, path: typeof s.path === 'string' ? s.path : '', recovered: true } } catch (e) { recoveryError = e instanceof Error ? e.message : String(e) } }
  return { project: { schema_version: '1.0', name: '图书借阅系统', diagrams: clone(library) as Diagram[] }, path: '', recovered: false, recovery_error: recoveryError }
}
async function validatePreviewProject(project: Project) {
  const r = await fetch('/api/project/validate', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(project ?? null) })
  if (!r.ok) throw new Error(await r.text())
}
export async function renderDiagram(d: Diagram): Promise<RenderResult> {
  if (desktop()) return api().RenderDiagram(JSON.stringify(d))
  const r = await fetch('/api/render', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(d) })
  if (!r.ok) throw new Error(await r.text())
  return r.json()
}
let recoveryRevision = 0
export async function recover(project: Project, path: string, fileRevision='') { if (desktop()) return api().SaveWorkspaceRecovery ? api().SaveWorkspaceRecovery(JSON.stringify(project),path,fileRevision) : api().SaveRecovery(JSON.stringify(project), path); const revision = ++recoveryRevision; await validatePreviewProject(project); if (revision === recoveryRevision) localStorage.setItem('ezruantu-session-v1', JSON.stringify({ project, path,revision:fileRevision })) }
export function markDirty(dirty: boolean) { if (desktop()) void api().SetDirty(dirty) }
export function onCloseRequested(callback: () => void): () => void { return desktop() ? EventsOn('app:close-requested', callback) : () => {} }
export async function confirmQuit() { if (desktop()) await api().ConfirmQuit() }
async function workspaceCall<T>(method:string,operation:string,args:Record<string,string>,nativeArgs:string[]=[]):Promise<T> {
 if (desktop() && api()[method]) return api()[method](...nativeArgs)
 let key=localStorage.getItem('ezbitu-preview-workspace');if(!key){key=crypto.randomUUID();localStorage.setItem('ezbitu-preview-workspace',key)}
 const response=await fetch('/api/workspace/'+operation,{method:'POST',headers:{'Content-Type':'application/json','X-EZ-Workspace':key},body:JSON.stringify(args)})
 if(!response.ok)throw new Error(await response.text());return response.json()
}
export const listWorkspace=()=>workspaceCall<WorkspaceTree>('ListWorkspace','list',{})
export const readWorkspaceFile=(path:string)=>workspaceCall<WorkspaceDocument>('ReadWorkspaceFile','read',{path},[path])
export const createWorkspaceFolder=(parent:string,name:string)=>workspaceCall<string>('CreateWorkspaceFolder','create-folder',{parent,name},[parent,name])
export const createWorkspaceFile=(parent:string,name:string,diagram:Diagram)=>{const raw=JSON.stringify(diagram);return workspaceCall<WorkspaceDocument>('CreateWorkspaceFile','create-file',{parent,name,raw},[parent,name,raw])}
export const writeWorkspaceFile=(path:string,diagram:Diagram,revision:string)=>{const raw=JSON.stringify(diagram);return workspaceCall<WorkspaceDocument>('WriteWorkspaceFile','write',{path,raw,revision},[path,raw,revision])}
export const renameWorkspaceEntry=(path:string,name:string)=>workspaceCall<string>('RenameWorkspaceEntry','rename',{path,name},[path,name])
export const deleteWorkspaceEntry=(path:string)=>workspaceCall<boolean>('DeleteWorkspaceEntry','delete',{path},[path])
export const importWorkspaceProject=(project:Project)=>{const raw=JSON.stringify(project);return workspaceCall<string>('ImportWorkspaceProject','import',{raw},[raw])}
export async function saveProject(project: Project, path: string, saveAs = false): Promise<string> {
  if (desktop()) return api().SaveProject(JSON.stringify(project), path, saveAs)
  await validatePreviewProject(project)
  download(JSON.stringify(project, null, 2), project.name + '.ezruantu.json', 'application/json'); return 'browser-download'
}
export async function openProject(): Promise<Session | undefined> {
  if (desktop()) { const s = await api().OpenProject(); return s.path ? s : undefined }
  return new Promise((resolve, reject) => { const input = document.createElement('input'); input.type = 'file'; input.accept = '.json'; input.oncancel = () => resolve(undefined); input.onchange = async () => { const f = input.files?.[0]; if (!f) return resolve(undefined); try { if (f.size > 32 * 1024 * 1024) throw new Error('项目不能超过 32 MB'); resolve({ project: JSON.parse(await f.text()), path: '', recovered: false }) } catch (e) { reject(e) } }; input.click() })
}
export async function exportDiagram(d: Diagram, format: string, svg: string): Promise<string> {
  let png = ''
  if (format === 'png') png = await svgToPng(svg)
  if (desktop()) return api().ExportFile(JSON.stringify(d), format, png)
  if (format === 'png') { download(Uint8Array.from(atob(png), c => c.charCodeAt(0)), d.title + '.png', 'image/png'); return 'browser-download' }
  const r = await fetch('/api/export?format=' + format, { method: 'POST', body: JSON.stringify(d) }); if (!r.ok) throw new Error(await r.text()); download(await r.text(), d.title + '.' + format, format === 'svg' ? 'image/svg+xml' : format === 'json' ? 'application/json' : 'application/xml'); return 'browser-download'
}
export async function environment() { if (desktop()) return api().GetEnvironment(); return { version: '0.2.14', mcp_path: 'EZ软图/EZ软图.exe', data_dir: 'EZ软图/data', mcp_ready: false } }
export async function loadSettings():Promise<{theme:DiagramTheme}> {
 if(desktop() && api().GetSettings)return api().GetSettings()
 const theme=localStorage.getItem('ezruantu-global-theme')
 return {theme:theme==='mono'||theme==='sage'?theme:'blue'}
}
export async function saveGlobalTheme(theme:DiagramTheme):Promise<void> {
 if(desktop() && api().SetGlobalTheme){await api().SetGlobalTheme(theme);return}
 localStorage.setItem('ezruantu-global-theme',theme)
}
export async function copyText(text: string) { if (desktop()) return api().CopyText(text); return navigator.clipboard.writeText(text) }
function download(content: string | Uint8Array, name: string, type: string) { const blob = new Blob([content as BlobPart], { type }); const url = URL.createObjectURL(blob); const a = document.createElement('a'); a.href = url; a.download = name; a.click(); setTimeout(() => URL.revokeObjectURL(url), 1000) }
async function svgToPng(svg: string): Promise<string> {
  await document.fonts.ready
  const url = URL.createObjectURL(new Blob([svg], { type: 'image/svg+xml' }))
  try { const image = new Image(); image.src = url; await image.decode(); const scale = Math.min(2, 8000 / Math.max(image.width, image.height), Math.sqrt(24000000 / (image.width * image.height))); const canvas = document.createElement('canvas'); canvas.width = Math.ceil(image.width * scale); canvas.height = Math.ceil(image.height * scale); const ctx = canvas.getContext('2d')!; ctx.fillStyle = '#fff'; ctx.fillRect(0, 0, canvas.width, canvas.height); ctx.drawImage(image, 0, 0, canvas.width, canvas.height); return canvas.toDataURL('image/png').split(',')[1] } finally { URL.revokeObjectURL(url) }
}

export type ExportFormat = 'png' | 'svg' | 'drawio' | 'json'
export interface ExportBundleFile { path: string; data: string }
declare global { interface Window { showDirectoryPicker?: (options: { mode: 'readwrite' }) => Promise<FileSystemDirectoryHandle> } }

function base64Text(text: string) {
 const bytes = new TextEncoder().encode(text)
 let binary = ''
 for (let i = 0; i < bytes.length; i += 32768) binary += String.fromCharCode(...bytes.subarray(i, i + 32768))
 return btoa(binary)
}
export async function prepareExportFile(d: Diagram, format: ExportFormat, svg: string, path: string): Promise<ExportBundleFile> {
 if (format === 'png') return { path, data: await svgToPng(svg) }
 if (format === 'svg') return { path, data: base64Text(svg) }
 if (format === 'json') return { path, data: base64Text(JSON.stringify(d, null, 2)) }
 let text: string
 if (desktop()) text = await api().ExportText(JSON.stringify(d), format)
 else {
  const response = await fetch('/api/export?format=' + format, { method: 'POST', body: JSON.stringify(d) })
  if (!response.ok) throw new Error(await response.text())
  text = await response.text()
 }
 return { path, data: base64Text(text) }
}
export async function pickBrowserExportFolder(): Promise<FileSystemDirectoryHandle | undefined> {
 if (desktop()) return undefined
 if (!window.showDirectoryPicker) throw new Error('此浏览器不支持保存文件夹，请选择 ZIP 压缩包，或使用桌面应用。')
 return window.showDirectoryPicker({ mode: 'readwrite' })
}
export async function exportBundle(files: ExportBundleFile[], name: string, folder: boolean, zipped: boolean, directory?: FileSystemDirectoryHandle): Promise<boolean> {
 if (!folder && !zipped) throw new Error('请至少选择一种输出方式')
 if (desktop()) return Boolean((await api().ExportBundle(name, files, folder, zipped))?.length)
 let archive: Uint8Array<ArrayBuffer> | undefined
 if (zipped) {
  const response = await fetch('/api/export/zip', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ files }) })
  if (!response.ok) throw new Error(await response.text())
  archive = new Uint8Array(await response.arrayBuffer())
 }
 let outputName = '导出-' + name.replace(/[\\/:*?"<>|]/g, '-').replace(/[. ]+$/g, '')
 if (folder) {
  if (!directory) throw new Error('尚未选择导出位置')
  const base = outputName
  for (let i = 1; ; i++) {
   if (i > 10000) throw new Error('同名导出结果过多，请更换保存位置')
   outputName = i === 1 ? base : `${base} (${i})`
   let exists = false
   for (const [entry, kind] of [[outputName, 'folder'], [outputName + '.zip', 'file']] as const) {
    try { if (kind === 'folder') await directory.getDirectoryHandle(entry); else await directory.getFileHandle(entry); exists = true }
    catch (e) { if (!(e instanceof DOMException) || !['NotFoundError', 'TypeMismatchError'].includes(e.name)) throw e; if (e.name === 'TypeMismatchError') exists = true }
   }
   if (!exists) break
  }
  const root = await directory.getDirectoryHandle(outputName, { create: true })
  for (const file of files) {
   const parts = file.path.split('/'); const filename = parts.pop()!
   let parent = root
   for (const part of parts) parent = await parent.getDirectoryHandle(part, { create: true })
   const handle = await parent.getFileHandle(filename, { create: true }); const writer = await handle.createWritable()
   await writer.write(Uint8Array.from(atob(file.data), c => c.charCodeAt(0))); await writer.close()
  }
  if (archive) {
   const handle = await directory.getFileHandle(outputName + '.zip', { create: true }); const writer = await handle.createWritable()
   await writer.write(archive); await writer.close()
  }
 } else if (archive) download(archive, outputName + '.zip', 'application/zip')
 return true
}
