<script setup lang="ts">
export interface ModalProps {
  title?: string
  wide?: boolean
}

withDefaults(defineProps<ModalProps>(), {
  title: '',
  wide: false,
})

const open = defineModel<boolean>({ default: false })

function close() {
  open.value = false
}
</script>

<template>
  <Teleport to="body">
    <div v-if="open" class="overlay" @click.self="close">
      <div class="dialog" :class="{ 'dialog--wide': wide }" role="dialog" aria-modal="true">
        <h3 v-if="title" class="dialog__title">{{ title }}</h3>
        <slot />
      </div>
    </div>
  </Teleport>
</template>

<style scoped>
.overlay {
  position: fixed;
  inset: 0;
  z-index: 100;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--overlay);
}
.dialog {
  min-width: 320px;
  max-width: 420px;
  padding: 20px;
  border-radius: 10px;
  background: var(--surface);
  box-shadow: 0 12px 40px rgba(0, 0, 0, 0.25);
}
.dialog__title {
  margin: 0 0 14px;
  font-size: 16px;
}
.dialog--wide {
  width: min(880px, 94vw);
  max-width: 880px;
}
@media (max-width: 768px) {
  .overlay { padding: 16px; align-items: flex-start; overflow-y: auto; }
  .dialog { min-width: 0; width: 100%; max-width: 100%; max-height: calc(100vh - 32px); overflow-y: auto; padding: 18px 16px; }
  .dialog--wide { width: 100%; }
}
</style>
