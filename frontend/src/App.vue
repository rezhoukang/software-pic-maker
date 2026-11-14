<script setup lang="ts">
import { computed, onMounted, onBeforeUnmount, ref, watch } from 'vue'
import { NConfigProvider, NSelect, NButton, NForm, NFormItem, NInput, NAlert, NBadge, zhCN } from 'naive-ui'
import { Users, Plus, Save, Undo2, Redo2, ZoomIn, ZoomOut, CircleDot, Braces, SlidersHorizontal, Copy, Trash2, Check, AlertCircle, X, PanelLeftOpen, PanelRightClose, PanelRightOpen, File, FileJson, Link2, Move, BookOpen, Sparkles, RotateCcw, Maximize, Minimize } from 'lucide-vue-next'
import Ajv from 'ajv/dist/2020'
import DiagramCanvas from './components/DiagramCanvas.vue'
import AppDialog from './components/AppDialog.vue'
import ExportDialog from './components/ExportDialog.vue'
import FileExplorer from './components/FileExplorer.vue'
import schema from '../../shared/diagram.schema.json'
import library from '../../shared/library.json'
import { clone, itemsOf, kinds, newDiagram, type Diagram, type DiagramType, type Item, type Project, type Relation, type RenderResult } from './types'
import * as bridge from './bridge'
import { buildSkillHandoff } from './mcp-skill'

const schemaValidator = new Ajv({ allErrors: true, strict: false }).compile(schema)
const project = ref<Project>({ schema_version: '1.0', name: '图书借阅系统', diagrams: clone(library) as Diagram[] })
const activeId = ref(project.value.diagrams[0].id)
const active = computed(() => project.value.diagrams.find(d => d.id === activeId.value) ?? project.value.diagrams[0])
const meta = computed(() => kinds[active.value.type])
const selected = ref(''); const dirty = ref(false)
const result = ref<RenderResult | null>(null); const busy = ref(false); const saving = ref(false); const applying = ref(false); const failure = ref('')
const view = ref<'canvas' | 'json'>('canvas'); const draft = ref(''); const jsonError = ref(''); const jsonDirty = computed(() => draft.value !== JSON.stringify(active.value, null, 2))
const canvasFocus = ref(false)
const exportModal = ref(false)
const showFileManager = ref(true)
const manualAdjust = ref(false)
const canvas = ref<InstanceType<typeof DiagramCanvas>>(); let fitOnRender = false; const zoom = ref(1); const showInspector = ref(true); const inspectorTab = ref('design')
type HistoryEntry = { kind: 'selection'; selected: string } | { kind: 'content'; project: Project; activeId: string; selected: string }
const undoStack = ref<HistoryEntry[]>([]); const redoStack = ref<HistoryEntry[]>([])
const newModal = ref(false); const mcpModal = ref(false); const relationModal = ref(false); const confirmModal = ref(false)
const confirmText = ref(''); const confirmError = ref(''); const switchSaving = ref(false); const saveError = ref(''); let confirmedAction: (() => void | Promise<void>) | null = null
const closeModal = ref(false); const quitting = ref(false); let stopCloseListener: (() => void) | undefined
async function quitApp() { if (quitting.value) return; quitting.value = true; try { await bridge.confirmQuit() } catch (e) { notify(errorText(e), true) } finally { quitting.value = false } }
onMounted(() => { stopCloseListener = bridge.onCloseRequested(() => { closeModal.value = true }) })
onBeforeUnmount(() => stopCloseListener?.())
const newType = ref<DiagramType>('er'); const newTitle = ref(''); const newParent=ref(''); const newError=ref(''); const creatingFile=ref(false); const pendingCopy=ref<Diagram|null>(null)
const explorer=ref<InstanceType<typeof FileExplorer>>(); const activeFile=ref('');const fileRevision=ref('');const hasDocument=ref(false);const workspaceReady=ref(false);const fileBusy=ref(false);let navigationToken=0;let recoveryFile='';let recoveryDiagram:Diagram|null=null;let recoveryRevision=''
const fileBreadcrumb=computed(()=>activeFile.value?activeFile.value.split('/'):[])
const fileUnsaved=computed(()=>dirty.value || jsonDirty.value)
const relationSource = ref<string>(''); const relationTarget = ref<string>(''); const relationLabel = ref(''); const relationKind = ref('association')
const toast = ref<{ text: string; error: boolean } | null>(null); let toastTimer: ReturnType<typeof setTimeout>; let recoveryTimer: ReturnType<typeof setTimeout>; let renderTimer: ReturnType<typeof setTimeout>; let renderToken = 0; let ready = false
const env = ref({ version: '0.2.14', mcp_path: '', data_dir: '', mcp_ready: false })
const dialogSelectTheme = { peers: { InternalSelectMenu: { height: 'min(220px, max(96px, calc(100dvh - 300px)))' } } }
const themeOverrides = { common: { primaryColor: '#5268ec', primaryColorHover: '#4159de', primaryColorPressed: '#394bd1', borderRadius: '8px', fontFamily: '"Microsoft YaHei", "Segoe UI", sans-serif', fontSize: '12px', fontSizeSmall: '12px', fontSizeMedium: '12px', heightSmall: '32px', heightMedium: '34px' } }
const allItems = computed(() => itemsOf(active.value))
const allRelations = computed(() => active.value.model.relations ?? active.value.model.messages ?? [])
const relationTypes = computed(() => active.value.type === 'sequence' ? [{label:'同步调用',value:'call'},{label:'返回消息',value:'return'}] : active.value.type === 'usecase' ? [{label:'关联 association',value:'association'},{label:'包含 «include»',value:'include'},{label:'扩展 «extend»',value:'extend'}] : [{label:'关联',value:'association'},{label:'依赖',value:'dependency'},{label:'继承（起点继承终点）',value:'inheritance'}])
function endpointOptions(kind: string | undefined, source?: string) {
 const items = active.value.type !== 'usecase' ? allItems.value : kind === 'include' || kind === 'extend' ? active.value.model.use_cases! : source ? (active.value.model.actors!.some(a => a.id === source) ? active.value.model.use_cases! : active.value.model.actors!) : allItems.value
 return items.filter(n => active.value.type !== 'usecase' || !source || n.id !== source).map(n => ({label:n.name,value:n.id}))
}
function normalizeRelation(r: Relation): Relation { const next = {...r}; const sources = endpointOptions(next.kind); if (!sources.some(n => n.value === next.source)) next.source = sources[0]?.value ?? ''; const targets = endpointOptions(next.kind,next.source); if (!targets.some(n => n.value === next.target)) next.target = targets[0]?.value ?? ''; return next }
function relationHint(kind?: string) { return kind === 'include' ? '起点为基础用例，箭头指向被包含用例。必经步骤使用 «include»。' : kind === 'extend' ? '起点为扩展用例，箭头指向基础用例。条件触发的附加行为使用 «extend»。' : '关联连接参与者与用例，无方向箭头。' }
watch([relationKind, relationSource], () => { const r = normalizeRelation({id:'draft',source:relationSource.value,target:relationTarget.value,kind:relationKind.value}); relationSource.value=r.source; relationTarget.value=r.target })
const selectedItem = computed(() => { const id = selected.value.startsWith('$attr:') ? selected.value.split(':')[1] : selected.value; return allItems.value.find(n => n.id === id) })
const selectedRelation = computed(() => allRelations.value.find(r => r.id === selected.value || '$relation:' + r.id === selected.value || '$edge:' + r.id + ':a' === selected.value || '$edge:' + r.id + ':b' === selected.value))
const validationIssues = computed(() => result.value?.validation.issues ?? [])
const nodeWord = computed(() => ({ er: '实体', table: '表', usecase: '用例', flowchart: '步骤', sequence: '参与者', class: '类', architecture: '组件', structure: '模块' })[active.value.type])
const mcpConfig = computed(() => JSON.stringify({ mcpServers: { ezruantu: { command: env.value.mcp_path || '请填写 EZ软图.exe 的绝对路径', args: ['--mcp', '--workspace', env.value.data_dir ? env.value.data_dir : '请填写项目工作目录'] } } }, null, 2))

function notify(text: string, error = false) { toast.value = { text, error }; clearTimeout(toastTimer); toastTimer = setTimeout(() => toast.value = null, error ? 6000 : 3000) }
function errorText(e: unknown) { return e instanceof Error ? e.message : String(e) }
function uid(prefix: string) { return prefix + '-' + Date.now().toString(36) + Math.random().toString(36).slice(2, 6) }
function syncDraft() { draft.value = JSON.stringify(active.value, null, 2); jsonError.value = '' }
function guardDraft() { if (view.value === 'json' && jsonDirty.value) { notify('请先应用或重置当前 JSON 编辑', true); return false }; return true }
function historyEntry(kind: HistoryEntry['kind']): HistoryEntry { return kind === 'selection' ? { kind, selected: selected.value } : { kind, project: clone(project.value), activeId: activeId.value, selected: selected.value } }
function pushHistory(entry: HistoryEntry) { undoStack.value.push(entry); if (undoStack.value.length > 40) undoStack.value.shift(); redoStack.value = [] }
function selectElement(id: string) { if (selected.value !== id) { pushHistory(historyEntry('selection')); selected.value = id }; inspectorTab.value = 'design'; if (id === '$boundary') showInspector.value = true }
function commit(change: () => void) { const preserveDraft = view.value === 'json' && jsonDirty.value; const before = historyEntry('content'); change(); if (before.kind !== 'content' || JSON.stringify(before.project) === JSON.stringify(project.value)) return; pushHistory(before); dirty.value = true; bridge.markDirty(true); if (!preserveDraft) syncDraft() }
async function refresh() { const token = ++renderToken; busy.value = true; failure.value = ''; try { const next = await bridge.renderDiagram(clone(active.value)); if (token !== renderToken) return; result.value = next; if (fitOnRender && next.validation.valid) { fitOnRender = false; setTimeout(() => canvas.value?.fit(), 0) }; if (!next.validation.valid) failure.value = next.validation.issues.find(i => i.severity === 'error')?.message ?? '图表校验失败' } catch (e) { if (token === renderToken) failure.value = errorText(e) } finally { if (token === renderToken) busy.value = false } }
watch(() => JSON.stringify(active.value), () => { if (!ready) return; busy.value = true; ++renderToken; clearTimeout(renderTimer); renderTimer = setTimeout(refresh, 60) })
watch([draft, dirty], () => bridge.markDirty(dirty.value || (view.value === 'json' && jsonDirty.value)))
watch(project, () => { if (!ready) return; clearTimeout(recoveryTimer); recoveryTimer = setTimeout(async () => { try { await bridge.recover(clone(project.value), activeFile.value,fileRevision.value) } catch (e) { notify('自动恢复副本保存失败：' + errorText(e), true) } }, 700) }, { deep: true })

function chooseView(v: 'canvas' | 'json') { if (v === view.value) return; if (v === 'canvas' && view.value === 'json' && jsonDirty.value) { notify('请先应用 JSON，或点击重置编辑', true); return }; view.value = v; if (v === 'json') { manualAdjust.value = false; syncDraft() } }
function restoreHistory(entry: HistoryEntry) { if (entry.kind === 'content') { project.value = entry.project; activeId.value = entry.activeId; dirty.value = true; bridge.markDirty(true); syncDraft() }; selected.value = entry.selected; inspectorTab.value = 'design'; if (entry.selected === '$boundary') showInspector.value = true }
function undo() { if (!guardDraft() || !undoStack.value.length) return; const entry = undoStack.value.pop()!; redoStack.value.push(historyEntry(entry.kind)); restoreHistory(entry) }
function redo() { if (!guardDraft() || !redoStack.value.length) return; const entry = redoStack.value.pop()!; undoStack.value.push(historyEntry(entry.kind)); restoreHistory(entry) }
async function exportProject() { if (!guardDraft()) return; try { const snapshot=clone(project.value); snapshot.name=active.value.title; const path=await bridge.saveProject(snapshot,'',true); if(path)notify('项目副本已导出') } catch(e){notify(errorText(e),true)} }
async function save(saveAs=false):Promise<boolean> {
 if(saveAs){await exportProject();return false}
 saveError.value=''
 if(saving.value || !hasDocument.value || !activeFile.value || !guardDraft())return false
 saving.value=true
 const path=activeFile.value,snapshot=clone(active.value)
 try{
  const saved=await bridge.writeWorkspaceFile(path,snapshot,fileRevision.value)
  if(activeFile.value!==path)return false
  fileRevision.value=saved.revision
  dirty.value=JSON.stringify(snapshot)!==JSON.stringify(active.value)
  bridge.markDirty(dirty.value||jsonDirty.value)
  await bridge.recover(clone(project.value),activeFile.value,fileRevision.value)
  notify(dirty.value?'文件已保存，后续编辑仍待保存':'文件已保存')
  await explorer.value?.refresh()
  return !dirty.value && !(view.value==='json'&&jsonDirty.value)
 }catch(e){saveError.value=errorText(e);notify(saveError.value,true);return false}
 finally{saving.value=false}
}
function requestReplace(action:()=>void | Promise<void>){
 if(switchSaving.value)return
 if(dirty.value || jsonDirty.value){
  confirmText.value='当前文件有未保存的修改。可以保存后切换，或放弃修改直接切换。'
  confirmError.value='';confirmedAction=action;confirmModal.value=true
 }else void action()
}
async function discardAndSwitch(){
 if(switchSaving.value)return
 const action=confirmedAction;confirmedAction=null;confirmModal.value=false
 await action?.()
}
async function saveAndSwitch(){
 if(switchSaving.value)return
 const action=confirmedAction,path=activeFile.value
 switchSaving.value=true;confirmError.value=''
 try{
  if(view.value==='json'&&jsonDirty.value){
   await applyJSON()
   if(jsonDirty.value){confirmError.value=jsonError.value||'JSON 尚未通过校验，请先修正后再保存。';return}
  }
  if(!await save(false)){
   confirmError.value=saveError.value||'仍有未保存的修改，请重试。';return
  }
  if(activeFile.value!==path||fileUnsaved.value){confirmError.value='文件在保存期间发生变化，请重试。';return}
  confirmedAction=null;confirmModal.value=false
  await action?.()
 }finally{switchSaving.value=false}
}
async function loadWorkspaceFile(path:string,initial=false){
 if(!initial && path===activeFile.value && hasDocument.value)return
 const load=async()=>{const token=++navigationToken;fileBusy.value=true;try{const doc=await bridge.readWorkspaceFile(path);if(token!==navigationToken)return
 const restore=initial&&recoveryFile===doc.path&&recoveryDiagram?.id===doc.diagram.id
 const d=restore?clone(recoveryDiagram!):doc.diagram;const restoredDirty=Boolean(restore&&JSON.stringify(d)!==JSON.stringify(doc.diagram))
 manualAdjust.value=false;project.value={schema_version:'1.0',name:d.title,diagrams:[d]};activeId.value=d.id;activeFile.value=doc.path;fileRevision.value=restoredDirty&&recoveryRevision?recoveryRevision:doc.revision;hasDocument.value=true;dirty.value=restoredDirty;bridge.markDirty(dirty.value);selected.value='';undoStack.value=[];redoStack.value=[];view.value='canvas';syncDraft();fitOnRender=true;await refresh()
 }catch(e){notify(errorText(e),true)}finally{if(token===navigationToken)fileBusy.value=false}}
 if(initial)await load();else requestReplace(load)
}
function designWorkspaceFile(path:string){if(path===activeFile.value&&hasDocument.value)chooseView('canvas');else void loadWorkspaceFile(path)}
function requestNewFile(parent=''){requestReplace(()=>{newParent.value=parent;newTitle.value='';newError.value='';pendingCopy.value=null;newModal.value=true})}
async function open(){requestReplace(async()=>{try{const session=await bridge.openProject();if(!session)return;const first=await bridge.importWorkspaceProject(session.project);await explorer.value?.refresh();await loadWorkspaceFile(first,true);notify('项目已导入主目录')}catch(e){notify(errorText(e),true)}})}
async function addDiagram(){if(creatingFile.value || !newTitle.value.trim())return;creatingFile.value=true;newError.value='';try{
 const d=pendingCopy.value?clone(pendingCopy.value):newDiagram(newType.value,newTitle.value.trim().replace(/\.json$/i,''));d.id=uid(d.type);d.title=newTitle.value.trim().replace(/\.json$/i,'');
 const doc=await bridge.createWorkspaceFile(newParent.value,newTitle.value.trim(),d);newModal.value=false;await explorer.value?.refresh();await loadWorkspaceFile(doc.path,true);notify('文件已创建')
 }catch(e){newError.value=errorText(e)}finally{creatingFile.value=false}}
function duplicateDiagram(){if(!guardDraft())return;newParent.value=activeFile.value.split('/').slice(0,-1).join('/');newTitle.value=active.value.title+' 副本';newType.value=active.value.type;pendingCopy.value=clone(active.value);newError.value='';newModal.value=true}
function deleteDiagram(){if(activeFile.value)explorer.value?.requestDelete(activeFile.value)}
function fileRenamed(oldPath:string,newPath:string){if(activeFile.value===oldPath||activeFile.value.startsWith(oldPath+'/')){activeFile.value=newPath+activeFile.value.slice(oldPath.length);void bridge.recover(clone(project.value),activeFile.value,fileRevision.value).catch(e=>notify(errorText(e),true))}}
function fileDeleted(path:string){if(activeFile.value===path||activeFile.value.startsWith(path+'/')){++navigationToken;hasDocument.value=false;activeFile.value='';fileRevision.value='';dirty.value=false;selected.value='';result.value=null;syncDraft();bridge.markDirty(false);void bridge.recover(clone(project.value),'').catch(()=>{})}notify('已移入内部回收目录')}
async function applyJSON() {
 if (applying.value) return
 const originalProject = project.value, originalId = active.value.id, originalDiagram = JSON.stringify(active.value), originalDraft = draft.value
 applying.value = true
 try {
  const d = JSON.parse(originalDraft) as Diagram
  if (!schemaValidator(d)) { jsonError.value = (schemaValidator.errors ?? []).slice(0, 5).map(e => `${e.instancePath || '/'} ${e.message}`).join('\n'); return }
  if (project.value.diagrams.some(x => x.id === d.id && x.id !== originalId)) throw new Error('图表 ID 与项目内另一张图重复')
  const rendered = await bridge.renderDiagram(d)
  if (project.value !== originalProject || active.value.id !== originalId || JSON.stringify(active.value) !== originalDiagram || draft.value !== originalDraft) { notify('内容已变更，请重新应用当前 JSON', true); return }
  if (!rendered.validation.valid) { jsonError.value = rendered.validation.issues.filter(i => i.severity === 'error').map(i => `${i.path}: ${i.message}`).join('\n'); return }
  const index = project.value.diagrams.findIndex(x => x.id === originalId)
  commit(() => project.value.diagrams[index] = d); activeId.value = d.id; result.value = rendered; selected.value = ''; syncDraft(); notify('JSON 已应用，画布已更新')
 } catch (e) { jsonError.value = errorText(e) } finally { applying.value = false }
}
function editDiagram(field: 'title' | 'description', value: string) { if (!guardDraft()) return; if (field === 'title' && !value.trim()) return notify('图表名称不能为空', true); commit(() => active.value[field] = value) }
function editSystemName(event: Event) { const input = event.target as HTMLInputElement; if (!guardDraft()) { input.value = active.value.model.system_name || ''; return }; const value = input.value.trim(); if (!value) { input.value = active.value.model.system_name || ''; return notify('系统名称不能为空', true) }; if (value !== active.value.model.system_name) commit(() => active.value.model.system_name = value); input.value = value }
function editItem(key: 'name' | 'shape', value: string) { const n = selectedItem.value; if (!n || (key === 'name' && !value.trim())) return; commit(() => n[key] = value) }
function editField(index: number, key: 'name' | 'type' | 'primary', value: string | boolean) { const f = selectedItem.value?.fields?.[index]; if (!f || (key === 'name' && !String(value).trim())) return; commit(() => { if (key === 'primary') f.primary = Boolean(value); else f[key] = String(value) }) }
function addField() { const n = selectedItem.value; if (!n || (n.fields?.length ?? 0) >= 50) return; commit(() => { n.fields ??= []; const names = new Set(n.fields.map(f => f.name)); let i = n.fields.length + 1; while (names.has('field_' + i)) i++; n.fields.push({ name: 'field_' + i }) }) }
function removeField(index: number) { if (selectedItem.value) commit(() => selectedItem.value!.fields!.splice(index, 1)) }
function setMethods(value: string) { if (selectedItem.value) commit(() => selectedItem.value!.methods = value.split('\n').map(s => s.trim()).filter(Boolean).slice(0, 20)) }
function addNode(actor = false) { if (!guardDraft()) return; const type = active.value.type; const key = type === 'er' || type === 'table' ? 'entities' : type === 'class' ? 'classes' : type === 'sequence' ? 'participants' : type === 'usecase' ? actor ? 'actors' : 'use_cases' : 'nodes'; const list = active.value.model[key] as Item[]; const limit = key === 'actors' || key === 'participants' ? 12 : key === 'use_cases' ? 40 : 80; if (list.length >= limit) return notify('已达到该类型的元素数量上限', true); const id = uid(actor ? 'actor' : 'node'); const n: Item = { id, name: (actor ? '参与者' : nodeWord.value) + (list.length + 1) }; if (['er', 'table', 'class'].includes(type)) n.fields = [{ name: 'id', type: 'bigint', primary: true }, { name: 'name', type: 'varchar(100)' }]; if (type === 'class') n.methods = []; if (type === 'flowchart') n.shape = 'process'; commit(() => list.push(n)); selected.value = id; fitOnRender = true }
function removeSelected() { if (!guardDraft()) return; const n = selectedItem.value, r = selectedRelation.value; if (!n && !r) return; if (n && allItems.value.length <= 1) return notify('请至少保留一个元素', true); const type = active.value.type; if (n && type === 'sequence' && allItems.value.length <= 2) return notify('时序图至少保留两个参与者', true); if (n && type === 'usecase' && ((active.value.model.actors?.some(a => a.id === n.id) && active.value.model.actors.length === 1) || (active.value.model.use_cases?.some(a => a.id === n.id) && active.value.model.use_cases.length === 1))) return notify('至少保留一个参与者和一个用例', true); if (type === 'sequence' && ((r && allRelations.value.length <= 1) || (n && allRelations.value.filter(m => m.source !== n.id && m.target !== n.id).length === 0))) return notify('时序图至少保留一条消息', true); commit(() => { for (const key of ['entities','actors','use_cases','classes','participants','nodes'] as const) if (active.value.model[key] && n) active.value.model[key] = active.value.model[key]!.filter(x => x.id !== n.id); for (const key of ['relations','messages'] as const) if (active.value.model[key]) active.value.model[key] = active.value.model[key]!.filter(x => n ? x.source !== n.id && x.target !== n.id : x.id !== r!.id); if (n) for (const id of Object.keys(active.value.view.positions)) if (id === n.id || id.startsWith('$attr:' + n.id + ':')) delete active.value.view.positions[id] }); selected.value = '' }
function openRelation() { relationSource.value = allItems.value[0]?.id ?? ''; relationTarget.value = active.value.type === 'usecase' ? active.value.model.use_cases![0].id : allItems.value[1]?.id ?? relationSource.value; relationLabel.value = ''; relationKind.value = active.value.type === 'sequence' ? 'call' : 'association'; relationModal.value = true }
async function addRelation() { try { const test = clone(active.value); const key = test.type === 'sequence' ? 'messages' : 'relations'; const r: Relation = { id: uid('rel'), source: relationSource.value, target: relationTarget.value, label: relationLabel.value || (key === 'messages' ? '新消息' : ''), kind: relationKind.value }; test.model[key]!.push(r); const rendered = await bridge.renderDiagram(test); if (!rendered.validation.valid) return notify(rendered.validation.issues[0].message, true); commit(() => active.value.model[key]!.push(r)); relationModal.value = false; selected.value = r.id } catch(e) { notify(errorText(e), true) } }
async function editRelation(key: keyof Relation, value: string) { const r = selectedRelation.value; if (!r) return; const diagram=active.value; const before=JSON.stringify(r); const next=normalizeRelation({...r,[key]:value}); const test=clone(diagram); const list=test.model[test.type==='sequence'?'messages':'relations']!; list[list.findIndex(n=>n.id===r.id)]=next; try { const rendered=await bridge.renderDiagram(test); if(active.value!==diagram || JSON.stringify(r)!==before) return; if(!rendered.validation.valid) return notify(rendered.validation.issues.find(i=>i.severity==='error')?.message || '关系无效',true); commit(()=>Object.assign(r,next)) } catch(e) { notify(errorText(e),true) } }
function moveNode(id: string, x: number, y: number) { if (!manualAdjust.value) return; commit(() => active.value.view.positions[id] = { x: Math.round(x), y: Math.round(y) }) }
function autoLayout() { if (!guardDraft()) return; manualAdjust.value = false; commit(() => active.value.view.positions = {}); selected.value = ''; fitOnRender = true; notify('已重新排版') }
function openExport() { if (guardDraft()) exportModal.value = true }
async function openMCP() { try { env.value = await bridge.environment() } catch (e) { notify(errorText(e), true) }; mcpModal.value = true }
async function copy(value: string) { try { await bridge.copyText(value); notify('已复制到剪贴板') } catch (e) { notify(errorText(e), true) } }
async function copySkill() { try { await bridge.copyText(buildSkillHandoff(mcpConfig.value)); notify('已复制 Skill 与 MCP 接入指引，可粘贴给 Agent') } catch (e) { notify(errorText(e), true) } }
function handleKey(e: KeyboardEvent) {
 const target = e.target as HTMLElement | null
 const input = target?.closest('input')
 const editing = Boolean(target?.isContentEditable || target?.closest('textarea,select') || (input && !input.readOnly && !['radio','checkbox','button','submit','reset','range','color','file','hidden','image'].includes(input.type)))
 if (exportModal.value) { if ((e.ctrlKey || e.metaKey) && ['s', 'z'].includes(e.key.toLowerCase())) e.preventDefault(); return }
 if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 's') { e.preventDefault(); void save(e.shiftKey) }
 if (!editing && (e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 'z') {
  e.preventDefault()
  if (e.shiftKey) redo(); else undo()
 }
 if (!editing && !target?.closest('.file-explorer') && e.key === 'Delete') removeSelected()
 if (e.key === 'Escape') { selectElement(''); if(canvasFocus.value) canvasFocus.value=false }
}
onMounted(async()=>{try{const session=await bridge.loadSession();project.value=session.project;activeId.value=project.value.diagrams[0].id;
 if(session.path && !/^(?:[A-Za-z]:|[\\/])/.test(session.path) && session.path.endsWith('.json')){activeFile.value=session.path.replaceAll('\\','/');recoveryFile=activeFile.value;recoveryDiagram=clone(project.value.diagrams[0]);recoveryRevision=session.revision||''}
 if(session.recovery_error)notify('恢复副本无法读取：'+session.recovery_error,true);env.value=await bridge.environment()
 }catch(e){notify(errorText(e),true)};syncDraft();ready=true;workspaceReady.value=true;document.addEventListener('keydown',handleKey)})

onBeforeUnmount(() => { clearTimeout(toastTimer); clearTimeout(recoveryTimer); clearTimeout(renderTimer); document.removeEventListener('keydown', handleKey) })
</script>

<template>
<NConfigProvider class="app-provider" :locale="zhCN" :theme-overrides="themeOverrides">
  <div class="app-shell" :class="{'canvas-focus':canvasFocus}">
    <aside v-show="showFileManager" class="sidebar">
      <FileExplorer v-if="workspaceReady" ref="explorer" :active-path="activeFile" :dirty="fileUnsaved" @collapse="showFileManager = false" @open="loadWorkspaceFile" @design="designWorkspaceFile" @create="requestNewFile" @renamed="fileRenamed" @deleted="fileDeleted" @error="message=>notify(message,true)" />
      <div class="sidebar-bottom"><NButton class="mcp-entry" block @click="openMCP">MCP 接入</NButton></div>
    </aside>
    <aside v-if="!showFileManager && !canvasFocus" class="file-manager-rail"><button class="icon-button" title="展开文件管理" aria-label="展开文件管理" aria-expanded="false" @click="showFileManager = true"><PanelLeftOpen :size="18" /></button></aside>

    <main class="main-area">
      <header class="editor-toolbar"><template v-if="hasDocument"><NButton class="manual-layout-toggle" size="small" :type="manualAdjust?'primary':'default'" :secondary="manualAdjust" :aria-pressed="manualAdjust" :title="manualAdjust?'关闭手动拖拽':'开启手动拖拽'" :disabled="view==='json'" @click="manualAdjust=!manualAdjust"><Move :size="14" />手动调整</NButton><span class="toolbar-separator"></span><div class="view-switch"><button :class="{ chosen: view === 'canvas' }" @click="chooseView('canvas')"><File :size="15" />画布</button><button :class="{ chosen: view === 'json' }" @click="chooseView('json')"><Braces :size="15" />JSON<span v-if="view === 'json' && jsonDirty" class="unsaved-dot"></span></button></div><span class="toolbar-separator"></span><button class="toolbar-button" @click="addNode(false)" :disabled="view === 'json'"><Plus :size="16" />添加{{ nodeWord }}</button><button v-if="active.type === 'usecase'" class="toolbar-button" @click="addNode(true)"><Users :size="15" />参与者</button><button class="toolbar-button" @click="openRelation" :disabled="view === 'json'"><Link2 :size="15" />{{ active.type === 'sequence' ? '消息' : '连线' }}</button><button class="toolbar-button" @click="autoLayout" :disabled="view === 'json'"><Sparkles :size="15" />自动排版</button><div class="toolbar-spacer"></div><button class="icon-button" :title="canvasFocus ? '退出全屏' : '全屏画布'" :aria-label="canvasFocus ? '退出全屏' : '全屏画布'" @click="canvasFocus=!canvasFocus"><Minimize v-if="canvasFocus" :size="17" /><Maximize v-else :size="17" /></button><span class="toolbar-separator"></span><button class="icon-button" title="撤销 Ctrl+Z" aria-label="撤销" :disabled="!undoStack.length" @click="undo"><Undo2 :size="17" /></button><button class="icon-button" title="重做 Ctrl+Shift+Z" aria-label="重做" :disabled="!redoStack.length" @click="redo"><Redo2 :size="17" /></button></template><span v-if="hasDocument" class="toolbar-separator"></span><div class="top-actions toolbar-document-actions"><NButton size="small" quaternary @click="open">导入项目</NButton><NButton size="small" :loading="saving" :disabled="saving||!hasDocument||fileBusy" @click="save(false)">保存文件</NButton><NButton size="small" type="primary" :disabled="fileBusy" data-testid="export-button" @click="openExport">导出</NButton></div></header>
      <div v-if="hasDocument" class="editor-body">
        <section class="canvas-area" :class="{ 'code-mode': view === 'json' }">
          <template v-if="view === 'canvas'"><DiagramCanvas ref="canvas" :result="result" :diagram="active" :allow-node-drag="manualAdjust" :diagram-id="active.id" :selected="selected" @select="selectElement" @move="moveNode" @zoom="value => zoom = value" /><div v-if="busy" class="render-progress">正在更新图表…</div><div v-if="failure" class="canvas-error"><AlertCircle :size="18" /><span>{{ failure }}</span><button @click="refresh">重试</button></div><div class="canvas-bottom"><div class="zoom-controls"><button aria-label="缩小" @click="canvas?.zoom(-.1)"><ZoomOut :size="17" /></button><button class="zoom-value" @click="canvas?.resetZoom()">{{ Math.round(zoom * 100) }}%</button><button aria-label="放大" @click="canvas?.zoom(.1)"><ZoomIn :size="17" /></button><span></span><button class="fit-canvas-button" title="适应画布" aria-label="适应画布" @click="canvas?.fit()"><CircleDot :size="17" /></button></div></div></template>
          <template v-else><div class="code-heading"><span><FileJson :size="16" />{{ activeFile.split('/').at(-1) }}</span><div><button class="button subtle" @click="syncDraft"><RotateCcw :size="14" />重置编辑</button><button class="button primary" @click="applyJSON"><Check :size="15" />校验并应用</button></div></div><textarea class="json-editor" aria-label="图表 JSON" v-model="draft" spellcheck="false"></textarea><div v-if="jsonError" class="json-error"><AlertCircle :size="16" /><pre>{{ jsonError }}</pre></div><div class="code-footer"><span>JSON Schema v1.0</span><span>{{ jsonDirty ? '编辑尚未应用到画布' : '与画布同步' }}</span></div></template>
        </section>

        <aside v-if="showInspector" class="inspector"><div class="inspector-tabs"><button :class="{ active: inspectorTab === 'design' }" @click="inspectorTab = 'design'"><SlidersHorizontal :size="14" />{{ selectedItem || selectedRelation ? '元素属性' : '图表设置' }}</button><button :class="{ active: inspectorTab === 'check' }" @click="inspectorTab = 'check'">检查<span v-if="validationIssues.length" class="count-badge">{{ validationIssues.length }}</span></button><button class="icon-button inspector-collapse" title="收起属性面板" aria-label="收起属性面板" aria-expanded="true" @click="showInspector = false"><PanelRightClose :size="17" /></button></div>
          <div class="inspector-content" :class="{disabled: view === 'json'}" v-if="inspectorTab === 'design'">
            <template v-if="selectedItem"><div class="property-heading"><div><strong>{{ selectedItem.name }}</strong><small>选中元素</small></div><button class="icon-button" title="取消选择" @click="selectElement('')"><X :size="15" /></button></div><label class="form-label">名称<input :value="selectedItem.name" maxlength="120" @change="e => editItem('name',(e.target as HTMLInputElement).value)"></label><label class="form-label">标识符<code class="readonly-code">{{ selectedItem.id }}</code></label><label v-if="active.type === 'flowchart' || active.type === 'architecture'" class="form-label">图形<NSelect :value="selectedItem.shape || 'process'" :options="active.type === 'flowchart' ? [{label:'处理步骤',value:'process'},{label:'条件判断',value:'decision'},{label:'开始 / 结束',value:'terminal'}] : [{label:'服务',value:'service'},{label:'客户端',value:'client'},{label:'数据库',value:'database'}]" @update:value="v => editItem('shape',v)" /></label>
              <template v-if="['er','table','class'].includes(active.type)"><div class="property-section-title"><span>{{ active.type === 'class' ? '属性' : '字段 / 属性' }} <b>{{ selectedItem.fields?.length ?? 0 }}</b></span><button class="text-button" :disabled="(selectedItem.fields?.length ?? 0) >= 50" :title="(selectedItem.fields?.length ?? 0) >= 50 ? '最多 50 个字段 / 属性' : '添加字段 / 属性'" @click="addField"><Plus :size="13" />添加</button></div><div class="fields-list"><div class="field-row" v-for="(f,i) in selectedItem.fields" :key="i"><div><button class="pk-toggle" :class="{on:f.primary}" title="切换主键标记" :aria-label="'切换字段 '+f.name+' 主键'" @click="editField(i,'primary',!f.primary)">PK</button><input :value="f.name" aria-label="字段名称" maxlength="120" @change="e => editField(i,'name',(e.target as HTMLInputElement).value)"><button class="icon-button" title="删除字段" @click="removeField(i)"><X :size="13" /></button></div><input :value="f.type || ''" aria-label="字段类型" placeholder="未设置类型" maxlength="60" @change="e => editField(i,'type',(e.target as HTMLInputElement).value)"></div></div></template>
              <label v-if="active.type === 'class'" class="form-label">方法（每行一个）<textarea :value="selectedItem.methods?.join('\n') || ''" rows="5" @change="e => setMethods((e.target as HTMLTextAreaElement).value)"></textarea></label><button class="danger-button" @click="removeSelected"><Trash2 :size="14" />删除元素及相关连线</button>
            </template>
            <template v-else-if="selectedRelation"><div class="property-heading"><span class="property-icon"><Link2 :size="19" /></span><div><strong>{{ active.type === 'sequence' ? '交互消息' : '关联关系' }}</strong><small>选中连线</small></div><button class="icon-button" @click="selectElement('')"><X :size="15" /></button></div><label v-if="['usecase','class','sequence'].includes(active.type)" class="form-label">关系类型<NSelect :value="selectedRelation.kind || (active.type === 'sequence' ? 'call' : 'association')" :options="relationTypes" @update:value="v => editRelation('kind',v)" /></label><p v-if="active.type === 'usecase'" class="relation-hint">{{ relationHint(selectedRelation.kind) }}</p><label v-if="!['include','extend'].includes(selectedRelation.kind || '')" class="form-label">文字说明<input :value="selectedRelation.label" maxlength="120" @change="e => editRelation('label',(e.target as HTMLInputElement).value)"></label><label class="form-label">起点<NSelect :value="selectedRelation.source" :options="endpointOptions(selectedRelation.kind)" @update:value="v => editRelation('source',v)" /></label><label class="form-label">终点<NSelect :value="selectedRelation.target" :options="endpointOptions(selectedRelation.kind,selectedRelation.source)" @update:value="v => editRelation('target',v)" /></label><template v-if="['er','table'].includes(active.type)"><label class="form-label">起点基数<NSelect :value="selectedRelation.source_cardinality || '1'" :options="['1','0..1','0..N','1..N','N','M'].map(x => ({label:x,value:x}))" @update:value="v => editRelation('source_cardinality',v)" /></label><label class="form-label">终点基数<NSelect :value="selectedRelation.target_cardinality || 'N'" :options="['1','0..1','0..N','1..N','N','M'].map(x => ({label:x,value:x}))" @update:value="v => editRelation('target_cardinality',v)" /></label></template><button class="danger-button" @click="removeSelected"><Trash2 :size="14" />删除关系</button></template>
            <template v-else><div class="property-section-title first"><span>基本信息</span><span class="tiny-tag">{{ meta.label }}</span></div><label class="form-label">图表名称<input :value="active.title" maxlength="120" @change="e => editDiagram('title',(e.target as HTMLInputElement).value)"></label><label v-if="active.type === 'usecase'" class="form-label">系统名称<input :value="active.model.system_name" aria-label="系统名称" maxlength="120" @change="editSystemName"></label><label class="form-label">说明<textarea :value="active.description" rows="3" maxlength="2000" @change="e => editDiagram('description',(e.target as HTMLTextAreaElement).value)"></textarea></label><label v-if="!['er','usecase','sequence'].includes(active.type)" class="form-label">布局方向<NSelect :value="active.view.direction" :options="[{label:'从左到右',value:'LR'},{label:'从上到下',value:'TB'}]" @update:value="v => commit(() => { active.view.direction=v;active.view.positions={} })" /></label><div class="property-divider"></div><div class="property-section-title"><span>图表操作</span></div><button class="inspector-action" @click="duplicateDiagram"><Copy :size="15" />复制当前图表</button><button class="inspector-action" @click="save(true)"><Save :size="15" />导出项目副本</button><button class="inspector-action muted-danger" @click="deleteDiagram"><Trash2 :size="15" />删除当前文件</button></template>
          </div>
          <div v-else class="inspector-content"><div class="validation-summary" :class="{failed:!result?.validation.valid}"><Check v-if="result?.validation.valid" :size="22" /><AlertCircle v-else :size="22" /><strong>{{ result?.validation.valid ? '结构校验通过' : '需要检查' }}</strong><p>{{ validationIssues.length ? '以下提示可以帮助完善图表。' : '字段、引用和关系符合当前图种规则。' }}</p></div><div class="issue-card" v-for="(issue,i) in validationIssues" :key="i" :class="issue.severity"><span>{{ issue.severity === 'error' ? '错误' : '提示' }}</span><p>{{ issue.message }}</p><code>{{ issue.path }}</code></div><div class="layout-note"><BookOpen :size="15" /><p>校验检查结构规则，业务含义仍需结合实际需求确认。</p></div></div>
        </aside>
        <div v-else-if="!canvasFocus" class="inspector-rail"><button class="icon-button" title="展开属性面板" aria-label="展开属性面板" aria-expanded="false" @click="showInspector = true"><PanelRightOpen :size="17" /></button></div>
      </div>
      <div v-else class="empty-workspace"><h1>从主目录选择一个文件</h1><p>右键文件夹创建文件，也可先新建文件夹整理设计内容。</p><button class="button primary" @click="requestNewFile(explorer?.selectedFolder||'')">新增文件</button></div>
      <footer class="statusbar">
        <nav class="breadcrumb file-breadcrumb" aria-label="文件路径" :title="['主目录', ...fileBreadcrumb].join(' / ')"><span>主目录</span><template v-for="(part,index) in fileBreadcrumb" :key="index"><span class="slash">/</span><strong v-if="index===fileBreadcrumb.length-1">{{part}}</strong><span v-else>{{part}}</span></template></nav>
        <span v-if="hasDocument" class="diagram-stats" aria-label="图表统计">{{ allItems.length }} 个元素<i></i>{{ allRelations.length }} {{ active.type === 'sequence' ? '条消息' : '条关系' }}</span>
        <div v-if="hasDocument" class="statusbar-right"><span class="file-save-status" data-testid="file-save-status" aria-live="polite"><NBadge dot :type="fileUnsaved?'warning':'success'" /><span>{{ fileUnsaved ? '未保存' : '已保存' }}</span></span><span class="diagram-validity" aria-label="图表结构状态"><span class="status-dot" :class="{invalid: !result?.validation.valid}"></span>{{ result?.validation.valid ? '图表结构有效' : '等待有效图表' }}</span></div>
      </footer>
    </main>
    <Transition name="toast"><div v-if="toast" class="toast-message" :class="{error:toast.error}" role="status"><AlertCircle v-if="toast.error" :size="18" /><Check v-else :size="18" />{{ toast.text }}<button @click="toast = null"><X :size="14" /></button></div></Transition>
  </div>

  <ExportDialog v-model:show="exportModal" :current="hasDocument ? active : undefined" :current-path="activeFile" @complete="message => notify(message)" />

  <AppDialog v-model:show="newModal" title="新增文件" class="compact-modal new-file-modal" :closable="!creatingFile" :mask-closable="!creatingFile">
    <p class="dialog-description">位置：主目录{{newParent?'/'+newParent:''}}</p>
    <NForm label-placement="top" :show-feedback="false" @submit.prevent="addDiagram">
      <NFormItem label="文件名称"><NInput v-model:value="newTitle" :input-props="{'aria-label':'文件名称'}" placeholder="例如：用户表设计" :maxlength="110" @keydown.enter.prevent="addDiagram" /></NFormItem>
      <NFormItem label="图表分类"><NSelect :theme-overrides="dialogSelectTheme" v-model:value="newType" :disabled="Boolean(pendingCopy)" :virtual-scroll="false" :options="Object.entries(kinds).map(([value,m])=>({label:m.label,value}))" /></NFormItem>
    </NForm>
    <NAlert v-if="newError" type="error" :show-icon="false" role="alert">{{newError}}</NAlert>
    <template #footer><NButton :disabled="creatingFile" @click="newModal=false">取消</NButton><NButton type="primary" :loading="creatingFile" :disabled="creatingFile||!newTitle.trim()" @click="addDiagram">创建文件</NButton></template>
  </AppDialog>

  <AppDialog v-model:show="relationModal" title="建立关联关系" class="compact-modal relation-modal">
    <p class="dialog-description">{{ active.type === 'sequence' ? '添加交互消息，按起点和终点确定消息方向。' : '选择关系类型、起点和终点。' }}</p>
    <NForm label-placement="top" :show-feedback="false" @submit.prevent="addRelation">
      <NFormItem v-if="['usecase','class','sequence'].includes(active.type)" label="关系类型"><NSelect :theme-overrides="dialogSelectTheme" v-model:value="relationKind" :options="relationTypes" /></NFormItem>
      <NAlert v-if="active.type === 'usecase'" class="dialog-hint" type="info" :show-icon="false">{{ relationHint(relationKind) }}</NAlert>
      <NFormItem label="起点"><NSelect :theme-overrides="dialogSelectTheme" v-model:value="relationSource" :options="endpointOptions(relationKind)" /></NFormItem>
      <NFormItem label="终点"><NSelect :theme-overrides="dialogSelectTheme" v-model:value="relationTarget" :options="endpointOptions(relationKind,relationSource)" /></NFormItem>
      <NFormItem v-if="!['include','extend'].includes(relationKind)" label="文字说明"><NInput v-model:value="relationLabel" :input-props="{'aria-label':'文字说明'}" placeholder="例如：创建、借阅、是 / 否" :maxlength="120" /></NFormItem>
    </NForm>
    <template #footer><NButton @click="relationModal=false">取消</NButton><NButton type="primary" @click="addRelation">添加关系</NButton></template>
  </AppDialog>

  <AppDialog v-model:show="mcpModal" title="MCP 接入" class="mcp-modal" :width="640">
    <p class="dialog-description">将以下配置添加到支持 MCP 的客户端，或点击“复制 Skill”将完整指引交给 Agent。</p>
    <NAlert :type="env.mcp_ready?'success':'info'" :show-icon="false">{{ env.mcp_ready ? '内置 MCP 已就绪，与桌面应用共用同一个 EXE' : 'MCP 内置于 EZ软图.exe，发布后可直接接入' }}</NAlert>
    <pre class="config-code">{{ mcpConfig }}</pre>
    <p class="dialog-description">支持项目文件夹、图表新建与修改、版本校验，以及 PNG / SVG / draw.io / JSON 批量导出。可在桌面窗口关闭后独立运行，PNG 需本机安装 Edge 或 Chrome。</p>
    <template #footer><NButton @click="mcpModal=false">关闭</NButton><NButton @click="copy(mcpConfig)">复制配置</NButton><NButton type="primary" @click="copySkill">复制 Skill</NButton></template>
  </AppDialog>

  <AppDialog v-model:show="confirmModal" title="切换当前文件？" class="compact-modal" :closable="!switchSaving" :mask-closable="!switchSaving">
    <p class="dialog-message">{{ confirmText }}</p>
    <NAlert v-if="confirmError" type="error" :show-icon="false" class="switch-error" role="alert">{{ confirmError }}</NAlert>
    <template #footer><NButton :disabled="switchSaving" @click="confirmModal=false">取消</NButton><NButton :disabled="switchSaving" @click="discardAndSwitch">放弃并切换</NButton><NButton type="primary" :loading="switchSaving" :disabled="switchSaving" @click="saveAndSwitch">保存并切换</NButton></template>
  </AppDialog>

  <AppDialog v-model:show="closeModal" title="退出 EZ软图？" class="compact-modal" :mask-closable="false" :closable="!quitting">
    <p class="dialog-message">项目或 JSON 草稿有未保存的修改。恢复副本可能不包含最近的编辑。请选择取消并保存项目，或直接退出。</p>
    <template #footer><NButton :disabled="quitting" @click="closeModal=false">取消</NButton><NButton type="primary" :loading="quitting" :disabled="quitting" @click="quitApp">直接退出</NButton></template>
  </AppDialog>
</NConfigProvider>
</template>
