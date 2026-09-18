<script setup lang="ts">
export interface ButtonProps {
  variant?: 'primary' | 'ghost' | 'danger'
  size?: 'sm' | 'md'
  disabled?: boolean
  loading?: boolean
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
  loading: false,
  type: 'button',
})

const emit = defineEmits<ButtonEmits>()
</script>

<template>
  <button
    class="btn"
    :class="[`btn--${variant}`, `btn--${size}`, { 'btn--loading': loading }]"
    :disabled="disabled || loading"
    :type="type"
    :title="title"
    :aria-busy="loading || undefined"
    @click="emit('click', $event)"
  >
    <span class="btn__content"><slot /></span>
    <span v-if="loading" class="btn__overlay" aria-hidden="true">
      <span class="btn__spinner i-lucide-loader-circle" />
    </span>
  </button>
</template>

<style scoped>
.btn {
  position: relative;
  display: inline-flex;
  align-items: center;
  border: 1px solid transparent;
  border-radius: 6px;
  cursor: pointer;
  font: inherit;
  line-height: 1;
  white-space: nowrap;
}
.btn__content {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}
.btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
.btn--loading:disabled {
  opacity: 1;
  cursor: wait;
}
.btn--loading .btn__content {
  opacity: 0.25;
}
.btn__overlay {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: inherit;
  background: inherit;
}
.btn__spinner {
  font-size: 1em;
  animation: btn-spin 0.8s linear infinite;
}
@keyframes btn-spin {
  to {
    transform: rotate(360deg);
  }
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
