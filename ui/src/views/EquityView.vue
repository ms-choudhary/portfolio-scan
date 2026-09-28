<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import AllocationDashboard, { type HoldingItem } from '@/components/AllocationDashboard.vue'

type AssetTargets = { equity: number; debt: number; gold: number }

const input = ref<Array<{ name: string; amount: number }>>([])
const loading = ref(true)
const error = ref('')

const largeCapTarget = ref(50)
const midCapTarget = ref(30)
const smallCapTarget = ref(20)
const assetTargets = ref<AssetTargets>({ equity: 70, debt: 20, gold: 10 })

const savingTargets = ref(false)
const targetsSaved = ref(false)
const targetsError = ref('')

const fetchPortfolio = async () => {
  try {
    loading.value = true
    error.value = ''
    const response = await fetch('/api/portfolio/equity/categories')
    if (!response.ok) {
      throw new Error(`HTTP error! status: ${response.status}`)
    }
    input.value = await response.json()
  } catch (e) {
    error.value = `Failed to load equity categories: ${e instanceof Error ? e.message : 'Unknown error'}`
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
      asset: AssetTargets
      equity: { large_cap: number; mid_cap: number; small_cap: number }
    }
    largeCapTarget.value = data.equity.large_cap
    midCapTarget.value = data.equity.mid_cap
    smallCapTarget.value = data.equity.small_cap
    assetTargets.value = data.asset
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
        asset: assetTargets.value,
        equity: {
          large_cap: largeCapTarget.value,
          mid_cap: midCapTarget.value,
          small_cap: smallCapTarget.value,
        },
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

const totalTargetPercent = computed(() => largeCapTarget.value + midCapTarget.value + smallCapTarget.value)

const totalAmount = computed(() => input.value.reduce((sum, item) => sum + item.amount, 0))

const holdings = computed<HoldingItem[]>(() => {
  const result: HoldingItem[] = []

  for (const item of input.value) {
    if (item.name === 'large cap') {
      result.push({
        name: 'large-cap',
        label: `Large Cap - ${largeCapTarget.value}%`,
        currentAmount: item.amount,
        percent: totalAmount.value === 0 ? 0 : (item.amount / totalAmount.value) * 100,
        rebalanceAmount: (totalAmount.value * largeCapTarget.value) / 100 - item.amount,
      })
    } else if (item.name === 'mid cap') {
      result.push({
        name: 'mid-cap',
        label: `Mid Cap - ${midCapTarget.value}%`,
        currentAmount: item.amount,
        percent: totalAmount.value === 0 ? 0 : (item.amount / totalAmount.value) * 100,
        rebalanceAmount: (totalAmount.value * midCapTarget.value) / 100 - item.amount,
      })
    } else if (item.name === 'small cap') {
      result.push({
        name: 'small-cap',
        label: `Small Cap - ${smallCapTarget.value}%`,
        currentAmount: item.amount,
        percent: totalAmount.value === 0 ? 0 : (item.amount / totalAmount.value) * 100,
        rebalanceAmount: (totalAmount.value * smallCapTarget.value) / 100 - item.amount,
      })
    }
  }

  return result
})

const targets = computed(() => [
  { key: 'large-cap', label: 'Large %', value: largeCapTarget.value },
  { key: 'mid-cap', label: 'Mid %', value: midCapTarget.value },
  { key: 'small-cap', label: 'Small %', value: smallCapTarget.value },
])

const updateTarget = ({ key, value }: { key: string; value: number }) => {
  targetsSaved.value = false
  if (key === 'large-cap') largeCapTarget.value = value
  if (key === 'mid-cap') midCapTarget.value = value
  if (key === 'small-cap') smallCapTarget.value = value
  if (totalTargetPercent.value === 100) saveTargets()
}
</script>

<template>
  <div class="max-w-2xl mx-auto mt-4 px-4">
    <a href="/" class="underline">← Back to Portfolio</a>
  </div>

  <div v-if="loading" class="text-center mt-8">Loading...</div>
  <div v-else-if="error" class="text-center mt-8 text-red-600">{{ error }}</div>
  <AllocationDashboard
    v-else
    title="Equity"
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
