<script setup lang="ts">
export interface ModalProps {
  title?: string
}

withDefaults(defineProps<ModalProps>(), {
  title: '',
})

const open = defineModel<boolean>({ default: false })

function close() {
  open.value = false
}
</script>

<template>
  <Teleport to="body">
    <div v-if="open" class="overlay" @click.self="close">
      <div class="dialog" role="dialog" aria-modal="true">
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
  background: rgba(15, 20, 30, 0.45);
}
.dialog {
  min-width: 320px;
  max-width: 420px;
  padding: 20px;
  border-radius: 10px;
  background: #fff;
  box-shadow: 0 12px 40px rgba(0, 0, 0, 0.25);
}
.dialog__title {
  margin: 0 0 14px;
  font-size: 16px;
}
</style>
