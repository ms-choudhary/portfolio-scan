<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'

type Allocation = { name: string; amount: number }

type TargetAllocations = {
  asset: { equity: number; debt: number; gold: number }
  equity: { large_cap: number; mid_cap: number; small_cap: number }
}

type FundMeta = {
  symbol: string
  name: string
  category: string
  sub_category: string
}

type RecurringFund = {
  symbol: string
  category: string
  sub_category: string
  monthly_amount: number
}

type FundLine = { symbol: string; name: string; amount: number }

type SubGroup = {
  key: string
  label: string
  amount: number
  funds: FundLine[]
  unallocated: number
}

type AssetGroup = {
  key: 'equity' | 'debt' | 'gold'
  label: string
  amount: number
  subGroups: SubGroup[]
  funds: FundLine[]
  unallocated: number
  recurring: FundLine[]
}

type Bucket = { key: string; current: number; target: number; locked: number }

const AVOID_KEY = 'investFundsToAvoid'
const EPS = 1e-6

const cashInput = ref('')
const rebalance = ref(true)
const pfChecked = ref(false)
const npsChecked = ref(false)

const targets = ref<TargetAllocations>({
  asset: { equity: 70, debt: 20, gold: 10 },
  equity: { large_cap: 50, mid_cap: 30, small_cap: 20 },
})
const assetHoldings = ref<Allocation[]>([])
const equityHoldings = ref<Allocation[]>([])
const recurringFunds = ref<RecurringFund[]>([])
const fundMeta = ref<FundMeta[]>([])
const fundsToAvoid = ref<string[]>([])
const avoidQuery = ref('')

const loading = ref(true)
const error = ref('')

const formatCurrency = (value: number) =>
  `₹${value.toLocaleString('en-IN', { maximumFractionDigits: 0 })}`

const fetchJson = async <T,>(url: string): Promise<T> => {
  const response = await fetch(url)
  if (!response.ok) {
    throw new Error(`HTTP error! status: ${response.status}`)
  }
  return (await response.json()) as T
}

const loadAll = async () => {
  try {
    loading.value = true
    error.value = ''
    const [t, a, e, r, m] = await Promise.all([
      fetchJson<TargetAllocations>('/api/target_allocations'),
      fetchJson<Allocation[]>('/api/portfolio'),
      fetchJson<Allocation[]>('/api/portfolio/equity/categories'),
      fetchJson<RecurringFund[]>('/api/recurring_funds'),
      fetchJson<FundMeta[]>('/api/mutual_funds/metadata'),
    ])
    targets.value = t
    assetHoldings.value = a
    equityHoldings.value = e
    recurringFunds.value = r
    fundMeta.value = m
  } catch (e) {
    error.value = `Failed to load: ${e instanceof Error ? e.message : 'Unknown error'}`
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  const saved = localStorage.getItem(AVOID_KEY)
  if (saved) {
    fundsToAvoid.value = JSON.parse(saved) as string[]
  }
  loadAll()
})

watch(
  fundsToAvoid,
  (value) => localStorage.setItem(AVOID_KEY, JSON.stringify(value)),
  { deep: true },
)

const cash = computed(() => Number(cashInput.value || '0'))

const pf = computed(() => recurringFunds.value.find((f) => f.symbol === 'pf') ?? null)
const nps = computed(() => recurringFunds.value.find((f) => f.symbol === 'nps') ?? null)

const pfAmount = computed(() => (pfChecked.value && pf.value ? pf.value.monthly_amount : 0))
const npsAmount = computed(() => (npsChecked.value && nps.value ? nps.value.monthly_amount : 0))

const deployable = computed(() => Math.max(0, cash.value - pfAmount.value - npsAmount.value))
const shortfall = computed(() => cash.value < pfAmount.value + npsAmount.value)

const sumByName = (list: Allocation[], name: string) =>
  list.filter((item) => item.name === name).reduce((sum, item) => sum + item.amount, 0)

const at = (rec: Record<string, number>, key: string): number => rec[key] ?? 0

const withoutCurrent = (buckets: Bucket[]): Bucket[] =>
  buckets.map((b) => ({ ...b, current: 0 }))

const allocateWorstGapFirst = (buckets: Bucket[], free: number): Record<string, number> => {
  const totalTarget = buckets.reduce((s, b) => s + b.target, 0) || 1
  const baseTotal = buckets.reduce((s, b) => s + b.current + b.locked, 0)
  const newTotal = baseTotal + free

  const added: Record<string, number> = {}
  const level: Record<string, number> = {}
  for (const b of buckets) {
    added[b.key] = b.locked
    level[b.key] = b.current + b.locked
  }

  let remaining = free
  while (remaining > EPS) {
    const deficits = buckets
      .map((b) => ({ key: b.key, deficit: newTotal * (b.target / totalTarget) - at(level, b.key) }))
      .filter((d) => d.deficit > EPS)
      .sort((a, b) => b.deficit - a.deficit)
    const worstEntry = deficits[0]
    if (!worstEntry) {
      break
    }

    const worst = worstEntry.deficit
    const leaders = deficits.filter((d) => worst - d.deficit < EPS)
    const next = deficits[leaders.length]?.deficit ?? 0
    const step = Math.min((worst - next) * leaders.length, remaining)
    const perLeader = step / leaders.length
    for (const l of leaders) {
      added[l.key] = at(added, l.key) + perLeader
      level[l.key] = at(level, l.key) + perLeader
    }
    remaining -= step
  }

  if (remaining > EPS) {
    for (const b of buckets) {
      added[b.key] = at(added, b.key) + remaining * (b.target / totalTarget)
    }
  }
  return added
}

const assetBuckets = computed<Bucket[]>(() => [
  { key: 'equity', current: sumByName(assetHoldings.value, 'equity'), target: targets.value.asset.equity, locked: npsAmount.value },
  { key: 'debt', current: sumByName(assetHoldings.value, 'debt'), target: targets.value.asset.debt, locked: pfAmount.value },
  { key: 'gold', current: sumByName(assetHoldings.value, 'gold'), target: targets.value.asset.gold, locked: 0 },
])

const assetAdded = computed(() =>
  allocateWorstGapFirst(
    rebalance.value ? assetBuckets.value : withoutCurrent(assetBuckets.value),
    deployable.value,
  ),
)

const equityMf = computed(() => at(assetAdded.value, 'equity') - npsAmount.value)
const debtMf = computed(() => at(assetAdded.value, 'debt') - pfAmount.value)
const goldMf = computed(() => at(assetAdded.value, 'gold'))

const equitySubBuckets = computed<Bucket[]>(() => [
  { key: 'large cap', current: sumByName(equityHoldings.value, 'large cap'), target: targets.value.equity.large_cap, locked: 0 },
  { key: 'mid cap', current: sumByName(equityHoldings.value, 'mid cap'), target: targets.value.equity.mid_cap, locked: 0 },
  { key: 'small cap', current: sumByName(equityHoldings.value, 'small cap'), target: targets.value.equity.small_cap, locked: 0 },
])

const equitySubAdded = computed(() =>
  allocateWorstGapFirst(
    rebalance.value ? equitySubBuckets.value : withoutCurrent(equitySubBuckets.value),
    equityMf.value,
  ),
)

const eligibleFunds = (predicate: (f: FundMeta) => boolean): FundMeta[] =>
  fundMeta.value.filter(predicate).filter((f) => !fundsToAvoid.value.includes(f.symbol))

const splitEqually = (amount: number, list: FundMeta[]): FundLine[] => {
  if (list.length === 0 || amount <= EPS) {
    return []
  }
  const per = amount / list.length
  return list.map((f) => ({ symbol: f.symbol, name: f.name, amount: per }))
}

const subCapLabels: Record<'large cap' | 'mid cap' | 'small cap', string> = {
  'large cap': 'Large Cap',
  'mid cap': 'Mid Cap',
  'small cap': 'Small Cap',
}

const subCapTargets = computed<Record<'large cap' | 'mid cap' | 'small cap', number>>(() => ({
  'large cap': targets.value.equity.large_cap,
  'mid cap': targets.value.equity.mid_cap,
  'small cap': targets.value.equity.small_cap,
}))

const assetGroups = computed<AssetGroup[]>(() => {
  const equitySubGroups: SubGroup[] = (['large cap', 'mid cap', 'small cap'] as const).map((key) => {
    const mf = at(equitySubAdded.value, key)
    const funds = splitEqually(mf, eligibleFunds((f) => f.category === 'equity' && f.sub_category === key))
    return {
      key,
      label: `${subCapLabels[key]} - ${subCapTargets.value[key]}%`,
      amount: mf,
      funds,
      unallocated: funds.length === 0 && mf > EPS ? mf : 0,
    }
  })

  const equityRecurring: FundLine[] =
    npsChecked.value && npsAmount.value > 0
      ? [{ symbol: 'nps', name: 'NPS', amount: npsAmount.value }]
      : []

  const debtFunds = splitEqually(debtMf.value, eligibleFunds((f) => f.category === 'debt'))
  const debtRecurring: FundLine[] =
    pfChecked.value && pfAmount.value > 0
      ? [{ symbol: 'pf', name: 'PF', amount: pfAmount.value }]
      : []

  const goldFunds = splitEqually(goldMf.value, eligibleFunds((f) => f.category === 'gold'))

  return [
    {
      key: 'equity',
      label: `Equity - ${targets.value.asset.equity}%`,
      amount: at(assetAdded.value, 'equity'),
      subGroups: equitySubGroups,
      funds: [],
      unallocated: 0,
      recurring: equityRecurring,
    },
    {
      key: 'debt',
      label: `Debt - ${targets.value.asset.debt}%`,
      amount: at(assetAdded.value, 'debt'),
      subGroups: [],
      funds: debtFunds,
      unallocated: debtFunds.length === 0 && debtMf.value > EPS ? debtMf.value : 0,
      recurring: debtRecurring,
    },
    {
      key: 'gold',
      label: `Gold - ${targets.value.asset.gold}%`,
      amount: at(assetAdded.value, 'gold'),
      subGroups: [],
      funds: goldFunds,
      unallocated: goldFunds.length === 0 && goldMf.value > EPS ? goldMf.value : 0,
      recurring: [],
    },
  ]
})

const selectedAvoidFunds = computed(() =>
  fundMeta.value.filter((f) => fundsToAvoid.value.includes(f.symbol)),
)

const filteredFunds = computed(() => {
  const query = avoidQuery.value.trim().toLowerCase()
  if (!query) {
    return []
  }
  return fundMeta.value
    .filter((f) => !fundsToAvoid.value.includes(f.symbol))
    .filter((f) => f.name.toLowerCase().includes(query) || f.symbol.toLowerCase().includes(query))
    .slice(0, 8)
})

const addFundToAvoid = (symbol: string) => {
  if (!fundsToAvoid.value.includes(symbol)) {
    fundsToAvoid.value = [...fundsToAvoid.value, symbol]
  }
  avoidQuery.value = ''
}

const removeFundToAvoid = (symbol: string) => {
  fundsToAvoid.value = fundsToAvoid.value.filter((s) => s !== symbol)
}

const addFirstSuggestion = () => {
  const first = filteredFunds.value[0]
  if (first) {
    addFundToAvoid(first.symbol)
  }
}
</script>

<template>
  <div class="max-w-2xl mx-auto mt-4 px-4">
    <a href="/" class="underline">← Back to Portfolio</a>
    <h1 class="text-2xl font-bold mt-4">Invest</h1>
  </div>

  <div v-if="loading" class="text-center mt-8">Loading...</div>
  <div v-else-if="error" class="text-center mt-8 text-red-600">{{ error }}</div>

  <div v-else class="max-w-2xl mx-auto mt-6 px-4 space-y-6">
    <div class="space-y-2">
      <Label for="cash">Cash</Label>
      <Input id="cash" v-model="cashInput" type="number" inputmode="numeric" placeholder="Enter amount" />
    </div>

    <label class="flex items-center gap-2">
      <input v-model="rebalance" type="checkbox" />
      <span>Rebalance</span>
    </label>

    <div class="flex gap-6">
      <label class="flex items-center gap-2">
        <input v-model="pfChecked" type="checkbox" :disabled="!pf" />
        <span>PF</span>
      </label>
      <label class="flex items-center gap-2">
        <input v-model="npsChecked" type="checkbox" :disabled="!nps" />
        <span>NPS</span>
      </label>
    </div>

    <div class="space-y-2">
      <Label for="avoid">Funds to avoid</Label>
      <div class="relative">
        <Input
          id="avoid"
          v-model="avoidQuery"
          placeholder="Type a fund name"
          @keydown.enter.prevent="addFirstSuggestion"
        />
        <div
          v-if="filteredFunds.length > 0"
          class="absolute z-10 mt-2 max-h-64 w-full overflow-auto rounded-md border bg-card shadow-lg"
        >
          <button
            v-for="fund in filteredFunds"
            :key="fund.symbol"
            type="button"
            class="flex w-full items-center justify-between gap-2 !border-0 !bg-white px-3 py-2 text-left text-sm shadow-none hover:!bg-slate-100"
            @click="addFundToAvoid(fund.symbol)"
          >
            <span class="min-w-0 truncate">{{ fund.name }}</span>
            <span class="shrink-0 text-xs text-muted-foreground">{{ fund.sub_category }}</span>
          </button>
        </div>
      </div>

      <div v-if="selectedAvoidFunds.length > 0" class="flex flex-col gap-2 pt-2">
        <button
          v-for="fund in selectedAvoidFunds"
          :key="fund.symbol"
          type="button"
          class="flex w-full items-center justify-between gap-2 rounded-full !border-slate-300 !bg-white px-3 py-2 text-left text-sm text-slate-800 shadow-none hover:!bg-slate-50"
          @click="removeFundToAvoid(fund.symbol)"
        >
          <span class="min-w-0 truncate">{{ fund.name }}</span>
          <span class="shrink-0 text-muted-foreground">×</span>
        </button>
      </div>
    </div>

    <p v-if="shortfall" class="text-sm text-red-600">
      Cash is less than the PF + NPS contribution.
    </p>

    <div class="border rounded-lg divide-y">
      <div v-for="group in assetGroups" :key="group.key" class="p-4">
        <div class="flex justify-between gap-2 font-semibold">
          <span class="min-w-0 truncate">{{ group.label }}</span>
          <span class="shrink-0 tabular-nums">{{ formatCurrency(group.amount) }}</span>
        </div>

        <div v-for="sub in group.subGroups" :key="sub.key" class="mt-3 pl-4">
          <div class="flex justify-between gap-2 text-sm font-medium">
            <span class="min-w-0 truncate">{{ sub.label }}</span>
            <span class="shrink-0 tabular-nums">{{ formatCurrency(sub.amount) }}</span>
          </div>
          <div
            v-for="line in sub.funds"
            :key="line.symbol"
            class="flex justify-between gap-2 text-sm text-muted-foreground pl-4"
          >
            <span class="min-w-0 break-words">{{ line.name }}</span>
            <span class="shrink-0 tabular-nums">{{ formatCurrency(line.amount) }}</span>
          </div>
          <p v-if="sub.unallocated > 0" class="text-xs text-amber-600 pl-4">
            No eligible funds — {{ formatCurrency(sub.unallocated) }} unallocated.
          </p>
        </div>

        <div
          v-for="line in group.funds"
          :key="line.symbol"
          class="flex justify-between gap-2 text-sm text-muted-foreground pl-4 mt-2"
        >
          <span class="min-w-0 break-words">{{ line.name }}</span>
          <span class="shrink-0 tabular-nums">{{ formatCurrency(line.amount) }}</span>
        </div>
        <p v-if="group.unallocated > 0" class="text-xs text-amber-600 pl-4 mt-2">
          No eligible funds — {{ formatCurrency(group.unallocated) }} unallocated.
        </p>

        <div
          v-for="line in group.recurring"
          :key="line.symbol"
          class="flex justify-between gap-2 text-sm text-muted-foreground pl-4 mt-2"
        >
          <span class="min-w-0 break-words">{{ line.name }} (recurring)</span>
          <span class="shrink-0 tabular-nums">{{ formatCurrency(line.amount) }}</span>
        </div>
      </div>
    </div>
  </div>
</template>
