<script setup lang="ts">
import {kinds,type WorkspaceEntry} from '../types'
defineProps<{entry:WorkspaceEntry;activePath:string;expanded:string[];level:number}>()
const emit=defineEmits<{open:[path:string];toggle:[path:string];menu:[event:MouseEvent,entry:WorkspaceEntry]}>()
</script>
<template>
 <li role="none">
  <button class="file-tree-row" :class="{'diagram-nav':entry.kind==='file',active:entry.path===activePath,folder:entry.kind==='folder'}" role="treeitem" :aria-level="level" :aria-expanded="entry.kind==='folder'?expanded.includes(entry.path):undefined" :aria-selected="entry.path===activePath" :data-entry-path="entry.path" :data-kind="entry.kind" :data-type="entry.diagram_type" :title="entry.name" :style="{paddingLeft:12+(level-1)*16+'px'}" @click="entry.kind==='folder'?emit('toggle',entry.path):emit('open',entry.path)" @contextmenu.prevent.stop="emit('menu',$event,entry)">
   <span v-if="entry.kind==='folder'" class="tree-disclosure" aria-hidden="true">{{expanded.includes(entry.path)?'▾':'▸'}}</span><span v-else class="tiny-tag file-type-tag">{{entry.diagram_type?kinds[entry.diagram_type].label:'其他文件'}}</span><span class="tree-filename">{{entry.name}}</span>
  </button>
  <ul v-if="entry.kind==='folder' && expanded.includes(entry.path)" role="group">
   <FileTreeNode v-for="child in entry.children" :key="child.path" :entry="child" :active-path="activePath" :expanded="expanded" :level="level+1" @open="emit('open',$event)" @toggle="emit('toggle',$event)" @menu="(event,entry)=>emit('menu',event,entry)" />
  </ul>
 </li>
</template>
