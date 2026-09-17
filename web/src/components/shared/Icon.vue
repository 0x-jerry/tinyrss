<script setup lang="ts">
export type IconName =
  | 'refresh'
  | 'plus'
  | 'edit'
  | 'trash'
  | 'folder'
  | 'folder-open'
  | 'rss'
  | 'star'
  | 'star-filled'
  | 'check'
  | 'search'
  | 'x'
  | 'chevron-right'
  | 'logout'
  | 'link'
  | 'external'
  | 'filter'

export interface IconProps {
  name: IconName
  size?: number
}

const props = withDefaults(defineProps<IconProps>(), { size: 16 })

// Feather-style stroke icons, keyed by name -> path data.
const PATHS: Record<IconName, string> = {
  refresh: 'M21 12a9 9 0 1 1-2.64-6.36M21 3v6h-6',
  plus: 'M12 5v14M5 12h14',
  edit: 'M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7M18.5 2.5a2.12 2.12 0 0 1 3 3L12 15l-4 1 1-4Z',
  trash: 'M3 6h18M8 6V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2m3 0v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6',
  folder: 'M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2Z',
  'folder-open': 'M6 14l1.5-3.9A2 2 0 0 1 9.36 9H21l2 5v4a2 2 0 0 1-2 2H3a2 2 0 0 1-2-2v-5a2 2 0 0 1 2-2Z',
  rss: 'M4 11a9 9 0 0 1 9 9M4 4a16 16 0 0 1 16 16M5 20a1 1 0 1 0 0-2 1 1 0 0 0 0 2Z',
  star: 'M12 2l3.09 6.26L22 9.27l-5 4.87 1.18 6.88L12 17.77l-6.18 3.25L7 14.14 2 9.27l6.91-1.01Z',
  'star-filled': 'M12 2l3.09 6.26L22 9.27l-5 4.87 1.18 6.88L12 17.77l-6.18 3.25L7 14.14 2 9.27l6.91-1.01Z',
  check: 'M20 6L9 17l-5-5',
  search: 'M11 19a8 8 0 1 0 0-16 8 8 0 0 0 0 16Zm10 2-4.35-4.35',
  x: 'M18 6L6 18M6 6l12 12',
  'chevron-right': 'M9 18l6-6-6-6',
  logout: 'M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4M16 17l5-5-5-5M21 12H9',
  link: 'M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71',
  external: 'M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6M15 3h6v6M10 14L21 3',
  filter: 'M22 3H2l8 9.46V19l4 2v-8.54Z',
}
</script>

<template>
  <svg
    class="icon"
    :width="props.size"
    :height="props.size"
    viewBox="0 0 24 24"
    fill="none"
    stroke="currentColor"
    stroke-width="2"
    stroke-linecap="round"
    stroke-linejoin="round"
    aria-hidden="true"
  >
    <path :d="PATHS[props.name]" />
  </svg>
</template>

<style scoped>
.icon {
  flex: none;
  vertical-align: middle;
}
</style>
