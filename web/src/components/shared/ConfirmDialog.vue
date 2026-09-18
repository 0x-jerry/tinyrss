<script setup lang="ts">
import Button from './Button.vue'

export interface ConfirmDialogProps {
  title?: string
  message?: string
  confirmText?: string
}

export interface ConfirmDialogEmits {
  confirm: []
  cancel: []
}

withDefaults(defineProps<ConfirmDialogProps>(), {
  title: 'Are you sure?',
  confirmText: 'Confirm',
})

const open = defineModel<boolean>({ default: false })
const emit = defineEmits<ConfirmDialogEmits>()

function close() {
  open.value = false
  emit('cancel')
}
</script>

<template>
  <Teleport to="body">
    <div v-if="open" class="overlay" @click.self="close">
      <div class="dialog" role="dialog" aria-modal="true">
        <h3 class="dialog__title">{{ title }}</h3>
        <p v-if="message" class="dialog__msg">{{ message }}</p>
        <div class="dialog__actions">
          <Button variant="ghost" @click="close">Cancel</Button>
          <Button variant="danger" @click="open = false; emit('confirm')">{{ confirmText }}</Button>
        </div>
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
  margin: 0 0 8px;
  font-size: 16px;
}
.dialog__msg {
  margin: 0 0 18px;
  color: #5b6472;
  font-size: 14px;
}
.dialog__actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}
@media (max-width: 768px) {
  .overlay { padding: 16px; align-items: flex-start; }
  .dialog { min-width: 0; width: 100%; max-width: 100%; max-height: calc(100vh - 32px); overflow-y: auto; }
  .dialog__actions { flex-wrap: wrap; }
}
</style>
