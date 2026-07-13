<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import AllocationDashboard, { type HoldingItem } from '@/components/AllocationDashboard.vue'

type EquityTargets = {
  large_cap: number
  mid_cap: number
  small_cap: number
}

const input = ref<Array<{ name: string; amount: number }>>([])
const equityTarget = ref(70)
const debtTarget = ref(20)
const goldTarget = ref(10)
const equitySubTargets = ref<EquityTargets>({ large_cap: 50, mid_cap: 30, small_cap: 20 })

const loading = ref(true)
const error = ref('')
const savingTargets = ref(false)
const targetsSaved = ref(false)
const targetsError = ref('')

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

const fetchTargets = async () => {
  try {
    const response = await fetch('/api/target_allocations')
    if (!response.ok) {
      throw new Error(`HTTP error! status: ${response.status}`)
    }
    const data = (await response.json()) as {
      asset: { equity: number; debt: number; gold: number }
      equity: EquityTargets
    }
    equityTarget.value = data.asset.equity
    debtTarget.value = data.asset.debt
    goldTarget.value = data.asset.gold
    equitySubTargets.value = data.equity
  } catch (e) {
    targetsError.value = `Failed to load targets: ${e instanceof Error ? e.message : 'Unknown error'}`
  }
}

const saveTargets = async () => {
  try {
    savingTargets.value = true
    targetsSaved.value = false
    targetsError.value = ''
    const response = await fetch('/api/target_allocations/update', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        asset: {
          equity: equityTarget.value,
          debt: debtTarget.value,
          gold: goldTarget.value,
        },
        equity: equitySubTargets.value,
      }),
    })
    if (!response.ok) {
      const text = await response.text()
      throw new Error(text || `HTTP error! status: ${response.status}`)
    }
    targetsSaved.value = true
  } catch (e) {
    targetsError.value = `Failed to save targets: ${e instanceof Error ? e.message : 'Unknown error'}`
  } finally {
    savingTargets.value = false
  }
}

onMounted(() => {
  fetchTargets()
  fetchPortfolio()
})

const totalTargetPercent = computed(() => equityTarget.value + debtTarget.value + goldTarget.value)

const totalAmount = computed(() => input.value.reduce((sum, item) => sum + item.amount, 0))

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

  return result
})

const targets = computed(() => [
  { key: 'equity', label: 'Equity %', value: equityTarget.value },
  { key: 'debt', label: 'Debt %', value: debtTarget.value },
  { key: 'gold', label: 'Gold %', value: goldTarget.value },
])

const updateTarget = ({ key, value }: { key: string; value: number }) => {
  targetsSaved.value = false
  if (key === 'equity') equityTarget.value = value
  if (key === 'debt') debtTarget.value = value
  if (key === 'gold') goldTarget.value = value
  if (totalTargetPercent.value === 100) saveTargets()
}
</script>

<template>
  <div class="max-w-2xl mx-auto mt-4 px-4">
    <a href="/mutual-funds" class="underline">Mutual Funds</a>
    <span class="mx-2 text-muted-foreground">|</span>
    <a href="/recurring-funds" class="underline">Recurring Funds</a>
    <span class="mx-2 text-muted-foreground">|</span>
    <a href="/smart-redemption" class="underline">Smart Redemption</a>
    <span class="mx-2 text-muted-foreground">|</span>
    <a href="/invest" class="underline">Invest</a>
  </div>

  <div v-if="loading" class="text-center mt-8">Loading...</div>
  <div v-else-if="error" class="text-center mt-8 text-red-600">{{ error }}</div>
  <AllocationDashboard
    v-else
    title="All"
    :total-amount="totalAmount"
    :holdings="holdings"
    :donut-colors="['orange', 'blue', 'green', 'gray']"
    :show-target-allocation="true"
    :targets="targets"
    :total-target-percent="totalTargetPercent"
    :saving-targets="savingTargets"
    :targets-saved="targetsSaved"
    :targets-error="targetsError"
    @update:target="updateTarget"
  />
</template>
