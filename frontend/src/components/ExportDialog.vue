<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { NAlert, NButton, NCheckbox, NCheckboxGroup, NForm, NFormItem, NRadioGroup, NRadioButton, NSelect } from 'naive-ui'
import AppDialog from './AppDialog.vue'
import { clone, type Diagram, type WorkspaceEntry, type WorkspaceTree } from '../types'
import * as bridge from '../bridge'

const props = defineProps<{ show: boolean; current?: Diagram; currentPath: string }>()
const emit = defineEmits<{ 'update:show': [value: boolean]; complete: [message: string] }>()
type Scope = 'current' | 'parent' | 'folder' | 'all'
const scope = ref<Scope | null>(null)
const format = ref<bridge.ExportFormat | null>(null)
const theme = ref('blue')
const themes = [{ id: 'blue', label: '蓝图', color: '#586fe8' }, { id: 'mono', label: '论文', color: '#343b46' }, { id: 'sage', label: '青绿', color: '#6a8c7c' }]
const outputs = ref<string[]>(['folder'])
const tree = ref<WorkspaceTree | null>(null)
const loading = ref(false), exporting = ref(false), error = ref(''), progress = ref('')
const folderPicker = ref(false), chosenFolder = ref<string | null>(null)
let loadToken = 0
const scopeOptions = computed(() => [
 { label: '当前图', value: 'current', disabled: !props.current },
 { label: '当前文件夹', value: 'parent', disabled: !props.currentPath },
 { label: '选择文件夹', value: 'folder' },
 { label: '全部文件', value: 'all' },
])
const formatOptions = [
 { label: 'PNG 图片', value: 'png' }, { label: 'SVG 矢量图', value: 'svg' },
 { label: 'draw.io', value: 'drawio' }, { label: 'JSON 设计数据', value: 'json' },
]
const selectTheme = { peers: { InternalSelectMenu: { height: 'min(220px, 40dvh)' } } }
function flatten(entries: WorkspaceEntry[]): WorkspaceEntry[] { return entries.flatMap(entry => [entry, ...flatten(entry.children || [])]) }
const entries = computed(() => flatten(tree.value?.entries || []))
const folders = computed(() => [{ path: '', name: '主目录' }, ...entries.value.filter(e => e.kind === 'folder').map(e => ({ path: e.path, name: '主目录 / ' + e.path.replaceAll('/', ' / ') }))])
const root = computed(() => scope.value === 'parent' ? props.currentPath.split('/').slice(0, -1).join('/') : scope.value === 'folder' ? chosenFolder.value : '')
const candidates = computed(() => {
 if (!scope.value || scope.value === 'folder' && chosenFolder.value === null) return []
 if (scope.value === 'current') return props.current ? [{ path: props.currentPath, name: props.current.title }] : []
 return entries.value.filter(e => e.kind === 'file' && e.diagram_type && (!root.value || e.path.startsWith(root.value + '/')))
})
const count = computed(() => candidates.value.length)
const multiple = computed(() => count.value > 1)
const canExport = computed(() => !loading.value && !exporting.value && !!scope.value && !!format.value && count.value > 0 && (!multiple.value || outputs.value.length > 0))

watch(() => props.show, async show => {
 const token = ++loadToken
 if (!show) return
 scope.value = null; format.value = null; chosenFolder.value = null; outputs.value = ['folder']; error.value = ''; progress.value = ''; tree.value = null; loading.value = true
 theme.value = props.current?.view.theme || 'blue'
 try { const next = await bridge.listWorkspace(); if (token === loadToken) tree.value = next }
 catch (e) { if (token === loadToken) error.value = e instanceof Error ? e.message : String(e) }
 finally { if (token === loadToken) loading.value = false }
})

async function submit() {
 if (!canExport.value) return
 const chosenFormat = format.value!, chosenTheme = theme.value, files = clone(candidates.value), base = root.value || '', current = props.current ? clone(props.current) : undefined, currentPath = props.currentPath
 const folder = outputs.value.includes('folder'), zipped = outputs.value.includes('zip')
 exporting.value = true; error.value = ''; progress.value = ''
 try {
  // Browser directory access needs the original click gesture. The desktop
  // bridge opens its native destination dialog after preparing all files.
  const directory = files.length > 1 && folder ? await bridge.pickBrowserExportFolder() : undefined
  const prepared: bridge.ExportBundleFile[] = []
  let total = 0
  for (let i = 0; i < files.length; i++) {
   const entry = files[i]
   progress.value = `正在准备 ${i + 1} / ${files.length} 张图…`
   let diagram: Diagram
   try {
    diagram = entry.path === currentPath && current ? current : clone((await bridge.readWorkspaceFile(entry.path)).diagram)
    diagram.view.theme = chosenTheme
    const rendered = await bridge.renderDiagram(diagram)
    if (!rendered.validation.valid) throw new Error(rendered.validation.issues.find(issue => issue.severity === 'error')?.message || '图表校验失败')
    if (files.length === 1) {
     const saved = await bridge.exportDiagram(diagram, chosenFormat, rendered.svg)
     if (saved) { emit('update:show', false); emit('complete', '已导出 1 张图') }
     return
    }
    const relative = (base ? entry.path.slice(base.length + 1) : entry.path).replace(/\.json$/i, '') + '.' + chosenFormat
    const file = await bridge.prepareExportFile(diagram, chosenFormat, rendered.svg, relative)
    total += file.data.length * .75
    if (total > 128 * 1024 * 1024) throw new Error('批量导出内容超过 128 MB，请缩小范围')
    prepared.push(file)
   } catch (e) { throw new Error(`${entry.path || entry.name}：${e instanceof Error ? e.message : String(e)}`) }
  }
  progress.value = '正在保存导出结果…'
  const name = base ? base.split('/').at(-1)! : '全部文件'
  if (await bridge.exportBundle(prepared, name, folder, zipped, directory)) {
   emit('update:show', false); emit('complete', `已导出 ${files.length} 张图${folder && zipped ? '，已生成文件夹和 ZIP 压缩包' : ''}`)
  }
 } catch (e) {
  if (!(e instanceof DOMException && e.name === 'AbortError')) error.value = e instanceof Error ? e.message : String(e)
 } finally { exporting.value = false; progress.value = '' }
}
</script>

<template>
 <AppDialog :show="show" title="导出" class="export-modal" :width="480" :closable="!exporting" :mask-closable="!exporting" @update:show="value => emit('update:show', value)">
  <NForm label-placement="top" :show-feedback="false">
   <NFormItem label="导出范围"><NSelect v-model:value="scope" :options="scopeOptions" :theme-overrides="selectTheme" :disabled="exporting" :virtual-scroll="false" placeholder="请选择导出范围" aria-label="导出范围" /></NFormItem>
   <div v-if="scope === 'folder'" class="export-folder-choice"><NButton size="small" :disabled="exporting || loading" @click="folderPicker = true">选择文件夹</NButton><span :title="chosenFolder ?? ''">{{ chosenFolder === null ? '尚未选择' : chosenFolder || '主目录' }}</span></div>
   <NFormItem label="导出形式"><NSelect v-model:value="format" :options="formatOptions" :theme-overrides="selectTheme" :disabled="exporting" :virtual-scroll="false" placeholder="请选择导出形式" aria-label="导出形式" /></NFormItem>
   <NFormItem label="风格"><NRadioGroup v-model:value="theme" class="theme-picker" size="small" name="export-theme" aria-label="导出风格" :disabled="exporting"><NRadioButton v-for="option in themes" :key="option.id" :value="option.id"><span class="theme-label"><span class="theme-swatch" :style="{ background: option.color }" aria-hidden="true"></span>{{ option.label }}</span></NRadioButton></NRadioGroup></NFormItem>
   <NFormItem v-if="multiple" label="输出方式"><div><NCheckboxGroup v-model:value="outputs" :disabled="exporting"><NCheckbox value="folder">文件夹</NCheckbox><NCheckbox value="zip">ZIP 压缩包</NCheckbox></NCheckboxGroup><p v-if="!outputs.length" class="export-output-error">请至少选择一种输出方式</p></div></NFormItem>
  </NForm>
  <p class="export-summary" aria-live="polite">{{ loading ? '正在读取导出范围…' : progress || (scope === 'folder' && chosenFolder === null ? '请选择要导出的文件夹' : scope ? `共 ${count} 张图${count > 1 ? '，保留目录结构' : ''}` : '请选择导出范围和形式') }}</p>
  <NAlert v-if="error" type="error" :show-icon="false" role="alert">{{ error }}</NAlert>
  <template #footer><NButton :disabled="exporting" @click="emit('update:show', false)">取消</NButton><NButton type="primary" :disabled="!canExport" :loading="exporting" @click="submit">导出</NButton></template>
 </AppDialog>
 <AppDialog v-model:show="folderPicker" title="选择导出文件夹" :width="440">
  <div class="export-folder-list"><button v-for="folder in folders" :key="folder.path" type="button" :title="folder.name" @click="chosenFolder = folder.path; folderPicker = false">{{ folder.name }}</button></div>
  <template #footer><NButton @click="folderPicker = false">取消</NButton></template>
 </AppDialog>
</template>
