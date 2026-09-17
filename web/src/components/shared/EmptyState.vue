<script setup lang="ts">
import Icon, { type IconName } from './Icon.vue'

export interface EmptyStateProps {
  message: string
  icon?: IconName
}

withDefaults(defineProps<EmptyStateProps>(), { icon: 'rss' })
</script>

<template>
  <div class="empty">
    <Icon :name="icon" :size="28" class="empty__icon" />
    <p class="empty__msg">{{ message }}</p>
    <slot />
  </div>
</template>

<style scoped>
.empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 10px;
  height: 100%;
  padding: 24px;
  text-align: center;
  color: #8a93a3;
}
.empty__icon {
  opacity: 0.5;
}
.empty__msg {
  margin: 0;
  font-size: 14px;
}
</style>
