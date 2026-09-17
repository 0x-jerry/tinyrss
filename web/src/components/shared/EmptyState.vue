<script setup lang="ts">
export interface EmptyStateProps {
  message: string
  icon?: string
}

withDefaults(defineProps<EmptyStateProps>(), { icon: 'i-lucide-rss' })
</script>

<template>
  <div class="empty">
    <span aria-hidden="true" :class="icon" class="empty__icon" />
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
  font-size: 28px;
  opacity: 0.5;
}
.empty__msg {
  margin: 0;
  font-size: 14px;
}
</style>
