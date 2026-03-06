<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import AllocationDashboard, { type HoldingItem } from '@/components/AllocationDashboard.vue'

const cashAmount = ref('')
const input = ref<Array<{ name: string; amount: number }>>([])
const equityTarget = ref(70)
const debtTarget = ref(20)
const goldTarget = ref(10)

const loading = ref(true)
const error = ref('')

const fetchPortfolio = async () => {
  try {
    loading.value = true
    error.value = ''
    const response = await fetch('/api/portfolio')
    if (!response.ok) {
      throw new Error(`HTTP error! status: ${response.status}`)
    }
    input.value = await response.json()
  } catch (e) {
    error.value = `Failed to load portfolio: ${e instanceof Error ? e.message : 'Unknown error'}`
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  cashAmount.value = localStorage.getItem('portfolioCashAmount') ?? ''
  equityTarget.value = Number(localStorage.getItem('equityTarget') ?? '70')
  debtTarget.value = Number(localStorage.getItem('debtTarget') ?? '20')
  goldTarget.value = Number(localStorage.getItem('goldTarget') ?? '10')
  fetchPortfolio()
})

watch(cashAmount, (value) => localStorage.setItem('portfolioCashAmount', value))
watch(equityTarget, (value) => localStorage.setItem('equityTarget', value.toString()))
watch(debtTarget, (value) => localStorage.setItem('debtTarget', value.toString()))
watch(goldTarget, (value) => localStorage.setItem('goldTarget', value.toString()))

const totalTargetPercent = computed(() => equityTarget.value + debtTarget.value + goldTarget.value)

const totalAmount = computed(() => {
  const portfolioTotal = input.value.reduce((sum, item) => sum + item.amount, 0)
  const cash = parseFloat(cashAmount.value) || 0
  return portfolioTotal + cash
})

const holdings = computed<HoldingItem[]>(() => {
  const result: HoldingItem[] = []

  for (const item of input.value) {
    if (item.name === 'equity') {
      result.push({
        name: 'equity',
        label: `Equity - ${equityTarget.value}%`,
        currentAmount: item.amount,
        percent: (item.amount / totalAmount.value) * 100,
        rebalanceAmount: (totalAmount.value * equityTarget.value) / 100 - item.amount,
        linkTo: '/equity',
      })
    } else if (item.name === 'debt') {
      result.push({
        name: 'debt',
        label: `Debt - ${debtTarget.value}%`,
        currentAmount: item.amount,
        percent: (item.amount / totalAmount.value) * 100,
        rebalanceAmount: (totalAmount.value * debtTarget.value) / 100 - item.amount,
      })
    } else if (item.name === 'gold') {
      result.push({
        name: 'gold',
        label: `Gold - ${goldTarget.value}%`,
        currentAmount: item.amount,
        percent: (item.amount / totalAmount.value) * 100,
        rebalanceAmount: (totalAmount.value * goldTarget.value) / 100 - item.amount,
      })
    }
  }

  const cash = parseFloat(cashAmount.value) || 0
  if (cash > 0) {
    result.push({
      name: 'cash',
      label: 'Cash - 0%',
      currentAmount: cash,
      percent: (cash / totalAmount.value) * 100,
      rebalanceAmount: -cash,
    })
  }

  return result
})

const targets = computed(() => [
  { key: 'equity', label: 'Equity %', value: equityTarget.value },
  { key: 'debt', label: 'Debt %', value: debtTarget.value },
  { key: 'gold', label: 'Gold %', value: goldTarget.value },
])

const updateTarget = ({ key, value }: { key: string; value: number }) => {
  if (key === 'equity') equityTarget.value = value
  if (key === 'debt') debtTarget.value = value
  if (key === 'gold') goldTarget.value = value
}

const updateCashAmount = (value: string) => {
  cashAmount.value = value
}
</script>

<template>
  <div v-if="loading" class="text-center mt-8">Loading...</div>
  <div v-else-if="error" class="text-center mt-8 text-red-600">{{ error }}</div>
  <AllocationDashboard
    v-else
    title="Total Portfolio"
    :total-amount="totalAmount"
    :holdings="holdings"
    :donut-colors="['orange', 'blue', 'green', 'gray']"
    :show-cash-input="true"
    :cash-amount="cashAmount"
    :show-target-allocation="true"
    :targets="targets"
    :total-target-percent="totalTargetPercent"
    @update:cashAmount="updateCashAmount"
    @update:target="updateTarget"
  />
</template>
