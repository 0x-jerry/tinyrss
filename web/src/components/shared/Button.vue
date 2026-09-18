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
  background: var(--accent);
  color: var(--on-accent);
}
.btn--primary:hover:not(:disabled) {
  background: var(--accent-hover);
}
.btn--ghost {
  background: transparent;
  color: var(--text-muted);
  border-color: var(--border-strong);
}
.btn--ghost:hover:not(:disabled) {
  background: var(--bg-hover);
}
.btn--danger {
  background: transparent;
  color: var(--danger);
  border-color: var(--danger-border);
}
.btn--danger:hover:not(:disabled) {
  background: var(--danger-bg);
  border-color: var(--danger-border);
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
