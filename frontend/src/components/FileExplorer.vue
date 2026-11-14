<script setup lang="ts">
import {computed,ref,onMounted,onBeforeUnmount,watch,nextTick} from 'vue'
import {NDropdown,NSelect,NInput,NForm,NFormItem,NButton,NAlert,type InputInst} from 'naive-ui'
import {PanelLeftClose} from 'lucide-vue-next'
import FileTreeNode from './FileTreeNode.vue'
import AppDialog from './AppDialog.vue'
import {kinds,type WorkspaceEntry,type WorkspaceTree} from '../types'
import * as bridge from '../bridge'
const props=defineProps<{activePath:string;dirty:boolean}>()
const emit=defineEmits<{collapse:[];open:[path:string,initial:boolean];design:[path:string];create:[parent:string];renamed:[oldPath:string,newPath:string];deleted:[path:string];error:[message:string]}>()
const tree=ref<WorkspaceTree>({root:'',entries:[]});const loading=ref(false);const loadError=ref('');const expanded=ref<string[]>([]);const rootOpen=ref(true);const selectedFolder=ref('');const search=ref('');const category=ref('all')
const menu=ref(false);const menuX=ref(0);const menuY=ref(0);const target=ref<WorkspaceEntry>({name:'主目录',path:'',kind:'folder'})
const action=ref<'folder'|'rename'|'delete'|''>('');const name=ref('');const input=ref<InputInst>();const working=ref(false);const actionError=ref('')
const categoryOptions=[{label:'全部分类',value:'all'},...Object.entries(kinds).map(([value,meta])=>({label:meta.label,value})),{label:'其他文件',value:'other'}]
const rootEntry:WorkspaceEntry={name:'主目录',path:'',kind:'folder'}
const options=computed(()=>target.value.kind==='folder'?[{label:'新增文件夹',key:'folder'},{label:'新增文件',key:'file'},{label:'重命名',key:'rename',disabled:!target.value.path},{label:'删除',key:'delete',disabled:!target.value.path}]:[{label:'重命名',key:'rename'},{label:'删除',key:'delete'},{label:'设计表',key:'design'}])
function allEntries(entries:WorkspaceEntry[]):WorkspaceEntry[]{return entries.flatMap(e=>[e,...allEntries(e.children||[])])}
const filtered=computed(()=>{
 const visit=(entries:WorkspaceEntry[],parentMatch=false):WorkspaceEntry[]=>entries.flatMap(entry=>{
  const matches=parentMatch || entry.name.toLowerCase().includes(search.value.trim().toLowerCase())
  if(entry.kind==='file')return matches && (category.value==='all' || category.value===(entry.diagram_type||'other'))?[entry]:[]
  const children=visit(entry.children||[],matches && Boolean(search.value.trim()))
  return children.length || matches?[{...entry,children}]:[]
 });return visit(tree.value.entries)
})
const visibleExpanded=computed(()=>search.value.trim() || category.value!=='all'?allEntries(filtered.value).filter(e=>e.kind==='folder').map(e=>e.path):expanded.value)
function reveal(path:string){const parts=path.split('/');for(let i=1;i<parts.length;i++){const p=parts.slice(0,i).join('/');if(!expanded.value.includes(p))expanded.value.push(p)}rootOpen.value=true}
watch(()=>props.activePath,p=>{reveal(p);if(p)selectedFolder.value=p.split('/').slice(0,-1).join('/')})
let refreshTask:Promise<void>|null=null
function refresh(initial=false,silent=false):Promise<void>{
 if(refreshTask)return refreshTask
 loading.value=!silent;loadError.value=''
 refreshTask=(async()=>{try{tree.value=await bridge.listWorkspace();if(initial){expanded.value=tree.value.entries.filter(e=>e.kind==='folder').map(e=>e.path);const files=allEntries(tree.value.entries).filter(e=>e.kind==='file'&&e.diagram_type);const first=files.find(e=>e.path===props.activePath)||files.find(e=>e.diagram_type==='er')||files[0];if(first)emit('open',first.path,true)}reveal(props.activePath)}catch(e){loadError.value=String(e instanceof Error?e.message:e)}finally{loading.value=false;refreshTask=null}})()
 return refreshTask
}
function toggle(path:string){selectedFolder.value=path;expanded.value=expanded.value.includes(path)?expanded.value.filter(p=>p!==path):[...expanded.value,path]}
function showMenu(event:MouseEvent,entry:WorkspaceEntry){target.value=entry;menuX.value=Math.min(event.clientX,window.innerWidth-180);menuY.value=Math.min(event.clientY,window.innerHeight-180);menu.value=true}
async function selectMenu(key:string){menu.value=false;if(key==='file'){emit('create',target.value.path);return};if(key==='design'){emit('design',target.value.path);return};action.value=key as typeof action.value;name.value=key==='rename'?target.value.name.replace(/\.json$/i,''):'';actionError.value='';await nextTick();input.value?.focus();input.value?.select()}
async function submit(){if(working.value)return;working.value=true;actionError.value='';try{
 if(action.value==='folder'){const path=await bridge.createWorkspaceFolder(target.value.path,name.value);expanded.value.push(target.value.path,path);selectedFolder.value=path}
 else if(action.value==='rename'){const old=target.value.path;const path=await bridge.renameWorkspaceEntry(old,name.value);expanded.value=expanded.value.map(p=>p===old||p.startsWith(old+'/')?path+p.slice(old.length):p);if(selectedFolder.value===old||selectedFolder.value.startsWith(old+'/'))selectedFolder.value=path+selectedFolder.value.slice(old.length);emit('renamed',old,path)}
 else if(action.value==='delete'){await bridge.deleteWorkspaceEntry(target.value.path);if(selectedFolder.value===target.value.path||selectedFolder.value.startsWith(target.value.path+'/'))selectedFolder.value=target.value.path.split('/').slice(0,-1).join('/');emit('deleted',target.value.path)}
 action.value='';await refresh()
 }catch(e){actionError.value=e instanceof Error?e.message:String(e)}finally{working.value=false}}
const deletingActive=computed(()=>action.value==='delete' && (props.activePath===target.value.path || props.activePath.startsWith(target.value.path+'/')))
function requestDelete(path:string){const entry=allEntries(tree.value.entries).find(e=>e.path===path);if(entry){target.value=entry;void selectMenu('delete')}}
function contextRoot(event:MouseEvent){showMenu(event,rootEntry)}
let focusTimer:ReturnType<typeof setTimeout>|undefined
const refreshOnFocus=()=>{clearTimeout(focusTimer);focusTimer=setTimeout(()=>void refresh(false,true),150)}
onMounted(()=>{void refresh(true);window.addEventListener('focus',refreshOnFocus)})
onBeforeUnmount(()=>{clearTimeout(focusTimer);window.removeEventListener('focus',refreshOnFocus)})
defineExpose({refresh,requestDelete,selectedFolder,reveal})
</script>
<template>
 <section class="file-explorer" aria-label="文件管理">
  <div class="files-heading"><span>文件管理</span><button @click="refresh(false)" :disabled="loading" aria-label="刷新目录">刷新</button><button @click="emit('create',selectedFolder)" aria-label="新增文件">新建</button></div>
  <div class="file-filters">
   <NSelect class="file-category" v-model:value="category" :options="categoryOptions" :virtual-scroll="false" role="combobox" aria-label="分类筛选" />
   <NInput v-model:value="search" placeholder="搜索文件或文件夹" :input-props="{'aria-label':'搜索文件或文件夹'}" clearable />
  </div>
  <nav class="file-tree-scroll" aria-label="主目录文件树" @contextmenu.self.prevent="contextRoot">
   <ul role="tree" aria-label="主目录">
    <li role="none"><div class="file-tree-root-row"><button class="file-tree-row root-row" role="treeitem" aria-level="1" :aria-expanded="rootOpen" :title="tree.root" data-entry-path="" data-kind="folder" @click="rootOpen=!rootOpen;selectedFolder=''" @contextmenu.prevent.stop="contextRoot"><span class="tree-disclosure">{{rootOpen?'▾':'▸'}}</span><span>主目录</span></button><button class="icon-button file-manager-collapse" title="收起文件管理" aria-label="收起文件管理" aria-expanded="true" @click="emit('collapse')"><PanelLeftClose :size="18" /></button></div>
     <ul v-if="rootOpen" role="group"><FileTreeNode v-for="entry in filtered" :key="entry.path" :entry="entry" :active-path="activePath" :expanded="visibleExpanded" :level="2" @open="emit('open',$event,false)" @toggle="toggle" @menu="showMenu" /></ul>
    </li>
   </ul>
   <p v-if="loading" class="tree-message">正在读取目录…</p><p v-else-if="loadError" class="tree-message error">{{loadError}}</p><p v-else-if="!filtered.length" class="tree-message">{{search||category!=='all'?'没有匹配的文件':'右键主目录，新建文件或文件夹'}}</p>
  </nav>
  <div class="file-root-path" :title="tree.root">{{tree.root||'主目录'}}</div>
  <NDropdown trigger="manual" placement="bottom-start" :show="menu" :x="menuX" :y="menuY" :options="options" @select="selectMenu" @clickoutside="menu=false" />
  <AppDialog :show="Boolean(action)" :mask-closable="!working" :closable="!working" :title="action==='folder'?'新增文件夹':action==='rename'?'重命名':'删除'" class="compact-modal file-action-modal" @update:show="v=>{if(!v&&!working)action=''}">
   <template v-if="action==='delete'"><p class="dialog-message">删除“{{target.name}}”{{target.kind==='folder'?'及其全部子项':''}}？</p><NAlert v-if="deletingActive&&dirty" type="warning" :show-icon="false">当前文件有未保存的编辑，删除后将关闭编辑页。</NAlert><p class="dialog-description">文件会移入内部回收目录，可手动找回。</p></template>
   <NForm v-else label-placement="top" :show-feedback="false" @submit.prevent="submit"><NFormItem :label="action==='folder'?'文件夹名称':'名称'"><NInput ref="input" v-model:value="name" :input-props="{'aria-label':action==='folder'?'文件夹名称':'名称'}" :maxlength="120" @keydown.enter.prevent="submit" /></NFormItem></NForm>
   <NAlert v-if="actionError" type="error" :show-icon="false" role="alert">{{actionError}}</NAlert>
   <template #footer><NButton :disabled="working" @click="action=''">取消</NButton><NButton :type="action==='delete'?'error':'primary'" :loading="working" :disabled="working||(!name.trim()&&action!=='delete')" @click="submit">{{action==='delete'?'确认删除':'确定'}}</NButton></template>
  </AppDialog>
 </section>
</template>
