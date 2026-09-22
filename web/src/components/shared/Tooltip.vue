<script setup lang="ts">
import { computed, onBeforeUnmount, ref } from 'vue'
import { useEventListener } from '@vueuse/core'
import { autoUpdate, computePosition, flip, offset, shift, type Placement } from '@floating-ui/dom'

export interface TooltipProps {
  text: string
  disabled?: boolean
  placement?: Placement
}

const props = withDefaults(defineProps<TooltipProps>(), {
  disabled: false,
  placement: 'top',
})

const open = ref(false)
const triggerRef = ref<HTMLElement | null>(null)
const bubbleRef = ref<HTMLElement | null>(null)
let cleanupAutoUpdate: (() => void) | null = null

// The wrapper span is `display: contents`, so it has no box to measure or hover;
// anchor to the slot's root element (e.g. the feed row) instead.
const anchor = computed(() => {
  const wrap = triggerRef.value
  return (wrap?.firstElementChild as HTMLElement | null) ?? wrap ?? null
})

async function updatePosition() {
  const el = anchor.value
  const bubble = bubbleRef.value
  if (!el || !bubble) return
  const { x, y } = await computePosition(el, bubble, {
    strategy: 'fixed',
    placement: props.placement,
    middleware: [offset(8), flip(), shift({ padding: 8 })],
  })
  bubble.style.left = `${x}px`
  bubble.style.top = `${y}px`
}

function show() {
  if (props.disabled || !props.text || open.value) return
  const el = anchor.value
  const bubble = bubbleRef.value
  if (!el || !bubble) return
  open.value = true
  // Re-position on scroll/resize and while the bubble stays visible.
  cleanupAutoUpdate = autoUpdate(el, bubble, updatePosition)
}

function hide() {
  open.value = false
  cleanupAutoUpdate?.()
  cleanupAutoUpdate = null
}

useEventListener(anchor, 'mouseenter', show)
useEventListener(anchor, 'mouseleave', hide)
useEventListener(anchor, 'focusin', show)
useEventListener(anchor, 'focusout', hide)

onBeforeUnmount(() => cleanupAutoUpdate?.())
</script>

<template>
  <span ref="triggerRef" class="tooltip"><slot /></span>
  <Teleport to="body">
    <div
      ref="bubbleRef"
      class="tooltip__bubble"
      :class="{ shown: open }"
      role="tooltip"
    >
      {{ text }}
    </div>
  </Teleport>
</template>

<style scoped>
.tooltip {
  display: contents;
}
.tooltip__bubble {
  position: fixed;
  z-index: 1000;
  max-width: 260px;
  padding: 6px 9px;
  border-radius: 6px;
  background: var(--text);
  color: var(--bg);
  font-size: 12px;
  line-height: 1.4;
  word-break: break-word;
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.18);
  pointer-events: none;
  opacity: 0;
  visibility: hidden;
  transition: opacity 0.12s ease;
}
.tooltip__bubble.shown {
  opacity: 1;
  visibility: visible;
}
</style>
