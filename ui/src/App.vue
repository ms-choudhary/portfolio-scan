<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import HomeView from '@/views/HomeView.vue'
import EquityView from '@/views/EquityView.vue'
import MutualFundsView from '@/views/MutualFundsView.vue'
import RecurringFundsView from '@/views/RecurringFundsView.vue'
import SmartRedemptionView from '@/views/SmartRedemptionView.vue'

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

  if (path.value === '/mutual-funds') {
    return MutualFundsView
  }

  if (path.value === '/recurring-funds') {
    return RecurringFundsView
  }

  if (path.value === '/smart-redemption') {
    return SmartRedemptionView
  }

  return HomeView
})
</script>

<template>
  <component :is="currentView" />
</template>
