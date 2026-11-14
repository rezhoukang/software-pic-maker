<script setup lang="ts">
import { ref } from 'vue'
import { NModal } from 'naive-ui'

withDefaults(defineProps<{
  show: boolean
  title: string
  width?: number
  closable?: boolean
  maskClosable?: boolean
}>(), { width: 460, closable: true, maskClosable: true })
const emit = defineEmits<{ 'update:show': [value: boolean] }>()
const content = ref<HTMLElement>()
function focusContent() {
  if (content.value?.closest('.app-dialog')?.contains(document.activeElement)) return
  content.value?.querySelector<HTMLElement>('input:not([disabled]),textarea:not([disabled])')?.focus({ preventScroll: true })
}
</script>

<template>
  <NModal :show="show" preset="card" class="app-dialog" :title="title" :aria-label="title"
    :bordered="false" :closable="closable" :mask-closable="maskClosable" :close-on-esc="closable"
    :auto-focus="false" :style="{ width: `${width}px`, maxWidth: 'calc(100vw - 40px)', maxHeight: 'calc(100dvh - 40px)' }"
    header-class="dialog-title-bar" content-class="dialog-scroll-body" footer-class="dialog-footer"
    :content-style="{ minHeight: '0', overflowY: 'auto' }"
    @update:show="value => emit('update:show', value)" @after-enter="focusContent">
    <div ref="content" class="dialog-content"><slot /></div>
    <template v-if="$slots.footer" #footer><div class="dialog-actions"><slot name="footer" /></div></template>
  </NModal>
</template>
