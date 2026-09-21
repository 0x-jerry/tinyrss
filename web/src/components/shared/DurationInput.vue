<script setup lang="ts">
import type { DurationUnit } from '../../helpers/duration'

export interface DurationInputProps {
  disabled?: boolean
  min?: number
}

export interface DurationInputEmits {
  change: []
}

const props = defineProps<DurationInputProps>()
const emit = defineEmits<DurationInputEmits>()
const value = defineModel<number>('value', { required: true })
const unit = defineModel<DurationUnit>('unit', { required: true })

const units: DurationUnit[] = ['seconds', 'minutes', 'hours', 'days']
</script>

<template>
  <span class="duration">
    <input
      v-model="value"
      class="duration__input"
      type="number"
      :min="min ?? 1"
      step="1"
      :disabled="disabled"
      @change="emit('change')"
    />
    <select v-model="unit" class="duration__select" :disabled="disabled" @change="emit('change')">
      <option v-for="u in units" :key="u" :value="u">{{ u }}</option>
    </select>
  </span>
</template>

<style scoped>
.duration {
  display: inline-flex;
  align-items: center;
  gap: 5px;
}
.duration__input {
  width: 58px;
  padding: 6px 7px;
  border: 1px solid var(--border-strong);
  border-radius: 6px;
  background: var(--surface);
  color: var(--text);
  font: inherit;
  font-size: 12px;
  text-align: right;
}
.duration__input:focus,
.duration__select:focus {
  border-color: var(--accent);
  outline: 2px solid var(--focus-ring);
  outline-offset: 1px;
}
.duration__input:disabled,
.duration__select:disabled {
  background: var(--bg-hover);
  cursor: wait;
  opacity: 0.65;
}
.duration__select {
  padding: 6px 6px;
  border: 1px solid var(--border-strong);
  border-radius: 6px;
  background: var(--surface);
  color: var(--text-muted);
  font: inherit;
  font-size: 12px;
}
</style>
