<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import HomeView from '@/views/HomeView.vue'
import EquityView from '@/views/EquityView.vue'

const path = ref(window.location.pathname)

const onPopState = () => {
  path.value = window.location.pathname
}

onMounted(() => {
  window.addEventListener('popstate', onPopState)
})

onUnmounted(() => {
  window.removeEventListener('popstate', onPopState)
})

const currentView = computed(() => {
  if (path.value === '/equity') {
    return EquityView
  }

  return HomeView
})
</script>

<template>
  <component :is="currentView" />
</template>
