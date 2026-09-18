<script setup lang="ts">
export interface BadgeProps {
  count: number
}

defineProps<BadgeProps>()
</script>

<template>
  <span v-if="count > 0" class="badge">{{ count > 999 ? '999+' : count }}</span>
</template>

<style scoped>
.badge {
  min-width: 18px;
  padding: 1px 5px;
  border-radius: 9px;
  background: #8a94a1;
  color: #fff;
  font-size: 11px;
  font-weight: 600;
  line-height: 1.4;
  text-align: center;
}
</style>
