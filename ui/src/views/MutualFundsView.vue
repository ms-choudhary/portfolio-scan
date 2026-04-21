<script setup lang="ts">
import { onMounted, ref } from 'vue'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'

type MutualFundLot = {
  number: number
  units: number
  price: number
  age: number
  pnl: number
  exitLoad: number
  stcg: number
  ltcg: number
}

type MutualFund = {
  name: string
  totalValue: number
  totalPnL: number
  lots: MutualFundLot[]
}

type ApiLot = {
  Number?: number
  Units?: number
  Price?: number
  Age?: number
  PnL?: number
  ExitLoad?: number
  STCG?: number
  LTCG?: number
}

type ApiFund = {
  Name?: string
  TotalValue?: number
  TotalPnL?: number
  Lots?: ApiLot[]
}

const funds = ref<MutualFund[]>([])
const expandedRows = ref<Set<string>>(new Set())
const loading = ref(true)
const error = ref('')

const formatCurrency = (value: number) =>
  `₹${value.toLocaleString('en-IN', { maximumFractionDigits: 0 })}`

const formatNumber = (value: number, decimals = 2) =>
  value.toLocaleString('en-IN', { minimumFractionDigits: decimals, maximumFractionDigits: decimals })

const amountClass = (value: number) => {
  if (value > 0) return 'text-green-700'
  if (value < 0) return 'text-red-700'
  return 'text-foreground'
}

const rowKey = (index: number, name: string) => `${index}:${name}`

const isExpanded = (key: string) => expandedRows.value.has(key)

const toggleExpanded = (key: string) => {
  if (expandedRows.value.has(key)) {
    expandedRows.value.delete(key)
  } else {
    expandedRows.value.add(key)
  }
  expandedRows.value = new Set(expandedRows.value)
}

const normalizeLot = (lot: ApiLot): MutualFundLot => ({
  number: lot.Number ?? 0,
  units: lot.Units ?? 0,
  price: lot.Price ?? 0,
  age: lot.Age ?? 0,
  pnl: lot.PnL ?? 0,
  exitLoad: lot.ExitLoad ?? 0,
  stcg: lot.STCG ?? 0,
  ltcg: lot.LTCG ?? 0,
})

const normalizeFund = (fund: ApiFund): MutualFund => ({
  name: fund.Name ?? 'Unnamed Fund',
  totalValue: fund.TotalValue ?? 0,
  totalPnL: fund.TotalPnL ?? 0,
  lots: (fund.Lots ?? []).map(normalizeLot),
})

const fetchMutualFunds = async () => {
  try {
    loading.value = true
    error.value = ''
    const response = await fetch('/api/mutual_funds')
    if (!response.ok) {
      throw new Error(`HTTP error! status: ${response.status}`)
    }

    const raw = (await response.json()) as ApiFund[]
    funds.value = raw.map(normalizeFund)
  } catch (e) {
    error.value = `Failed to load mutual funds: ${e instanceof Error ? e.message : 'Unknown error'}`
  } finally {
    loading.value = false
  }
}

onMounted(fetchMutualFunds)
</script>

<template>
  <div class="max-w-6xl mx-auto mt-4 px-4">
    <a href="/" class="underline">← Back to Portfolio</a>
    <h1 class="text-2xl font-bold mt-4">Mutual Funds</h1>
  </div>

  <div v-if="loading" class="text-center mt-8">Loading mutual funds...</div>
  <div v-else-if="error" class="text-center mt-8 text-red-600">{{ error }}</div>

  <div v-else class="max-w-6xl mx-auto mt-6 px-4">
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead class="w-10" />
          <TableHead>Fund Name</TableHead>
          <TableHead class="text-right">Total Value</TableHead>
          <TableHead class="text-right">Total PnL</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        <template v-if="funds.length > 0">
          <template v-for="(fund, index) in funds" :key="rowKey(index, fund.name)">
            <TableRow>
              <TableCell>
                <button
                  type="button"
                  class="h-7 w-7 rounded border text-sm"
                  :aria-expanded="isExpanded(rowKey(index, fund.name))"
                  :aria-label="`Toggle lots for ${fund.name}`"
                  @click="toggleExpanded(rowKey(index, fund.name))"
                >
                  <span class="inline-block">{{ isExpanded(rowKey(index, fund.name)) ? '-' : '+' }}</span>
                </button>
              </TableCell>
              <TableCell class="font-medium whitespace-normal">{{ fund.name }}</TableCell>
              <TableCell class="text-right">{{ formatCurrency(fund.totalValue) }}</TableCell>
              <TableCell class="text-right">
                <span :class="amountClass(fund.totalPnL)">{{ formatCurrency(fund.totalPnL) }}</span>
              </TableCell>
            </TableRow>

            <TableRow v-if="isExpanded(rowKey(index, fund.name))">
              <TableCell :colspan="4" class="p-0">
                <div class="px-4 py-3 bg-muted/20">
                  <Table>
                    <TableHeader>
                      <TableRow>
                        <TableHead>Units</TableHead>
                        <TableHead class="text-right">Price</TableHead>
                        <TableHead class="text-right">Age</TableHead>
                        <TableHead class="text-right">PnL</TableHead>
                        <TableHead class="text-right">Exit Load</TableHead>
                        <TableHead class="text-right">STCG</TableHead>
                        <TableHead class="text-right">LTCG</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      <template v-if="fund.lots.length === 0">
                        <TableRow>
                          <TableCell :colspan="7" class="text-center text-muted-foreground py-3">
                            No lots found for this fund.
                          </TableCell>
                        </TableRow>
                      </template>
                      <template v-else>
                        <TableRow v-for="lot in fund.lots" :key="lot.number">
                          <TableCell>{{ formatNumber(lot.units, 3) }}</TableCell>
                          <TableCell class="text-right">{{ formatNumber(lot.price) }}</TableCell>
                          <TableCell class="text-right">{{ lot.age }} days</TableCell>
                          <TableCell class="text-right">
                            <span :class="amountClass(lot.pnl)">{{ formatCurrency(lot.pnl) }}</span>
                          </TableCell>
                          <TableCell class="text-right">{{ formatCurrency(lot.exitLoad) }}</TableCell>
                          <TableCell class="text-right">{{ formatCurrency(lot.stcg) }}</TableCell>
                          <TableCell class="text-right">{{ formatCurrency(lot.ltcg) }}</TableCell>
                        </TableRow>
                      </template>
                    </TableBody>
                  </Table>
                </div>
              </TableCell>
            </TableRow>
          </template>
        </template>
        <TableRow v-else>
          <TableCell :colspan="4" class="text-center text-muted-foreground py-4">
            No mutual funds data available.
          </TableCell>
        </TableRow>
      </TableBody>
    </Table>
  </div>
</template>
