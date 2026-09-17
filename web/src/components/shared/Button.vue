<script setup lang="ts">
export interface ButtonProps {
  variant?: 'primary' | 'ghost' | 'danger'
  size?: 'sm' | 'md'
  disabled?: boolean
  type?: 'button' | 'submit'
  title?: string
}
export interface ButtonEmits {
  click: [event: MouseEvent]
}

withDefaults(defineProps<ButtonProps>(), {
  variant: 'primary',
  size: 'md',
  disabled: false,
  type: 'button',
})

const emit = defineEmits<ButtonEmits>()
</script>

<template>
  <button
    class="btn"
    :class="[`btn--${variant}`, `btn--${size}`]"
    :disabled="disabled"
    :type="type"
    :title="title"
    @click="emit('click', $event)"
  >
    <slot />
  </button>
</template>

<style scoped>
.btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  border: 1px solid transparent;
  border-radius: 6px;
  cursor: pointer;
  font: inherit;
  line-height: 1;
  white-space: nowrap;
}
.btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
.btn--primary {
  background: #2f6fed;
  color: #fff;
}
.btn--primary:hover:not(:disabled) {
  background: #285fd0;
}
.btn--ghost {
  background: transparent;
  color: #5b6472;
  border-color: #d8dde6;
}
.btn--ghost:hover:not(:disabled) {
  background: #f2f4f8;
}
.btn--danger {
  background: transparent;
  color: #d64545;
  border-color: #e8b4b4;
}
.btn--sm {
  padding: 4px 8px;
  font-size: 12px;
}
.btn--md {
  padding: 7px 12px;
  font-size: 13px;
}
</style>
