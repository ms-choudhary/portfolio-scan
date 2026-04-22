<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'

type MutualFundOption = {
  name: string
  symbol: string
}

type RedemptionFund = {
  fundName: string
  unitsToSell: number
  totalValue: number
  pnl: number
  exitLoad: number
  stcgTax: number
  ltcgTax: number
}

type RedemptionResponse = {
  funds: RedemptionFund[]
  totalValue: number
  totalExitLoad: number
  totalTax: number
  actualValue: number
}

type ApiMutualFund = {
  Name?: string
  Symbol?: string
}

type ApiRedemptionFund = {
  fund_name?: string
  units_to_sell?: number
  total_value?: number
  pnl?: number
  exit_load?: number
  stcg_tax?: number
  ltcg_tax?: number
}

type ApiRedemptionResponse = {
  funds?: ApiRedemptionFund[]
  total_value?: number
  total_exit_load?: number
  total_tax?: number
  actual_value?: number
}

const redemptionAmount = ref('')
const avoidQuery = ref('')
const availableFunds = ref<MutualFundOption[]>([])
const selectedFunds = ref<MutualFundOption[]>([])
const result = ref<RedemptionResponse | null>(null)

const loadingFunds = ref(true)
const submitting = ref(false)
const fundsError = ref('')
const submitError = ref('')
const amountPresets = [500000, 1000000, 1500000]

const formatCurrency = (value: number) =>
  `₹${value.toLocaleString('en-IN', { maximumFractionDigits: 0 })}`

const formatNumber = (value: number, decimals = 3) =>
  value.toLocaleString('en-IN', { minimumFractionDigits: decimals, maximumFractionDigits: decimals })

const formatAmountValue = (value: string) => {
  if (!value) {
    return ''
  }

  return Number(value).toLocaleString('en-IN')
}

const formatAmountPreset = (value: number) => `₹${value / 100000}L`

const amountClass = (value: number) => {
  if (value > 0) return 'text-green-700'
  if (value < 0) return 'text-red-700'
  return 'text-foreground'
}

const parsedRedemptionAmount = computed(() => Number(redemptionAmount.value || '0'))

const formattedRedemptionAmount = computed({
  get: () => formatAmountValue(redemptionAmount.value),
  set: (value: string) => {
    redemptionAmount.value = value.replace(/\D/g, '')
  },
})

const filteredFunds = computed(() => {
  const query = avoidQuery.value.trim().toLowerCase()
  const selectedSymbols = new Set(selectedFunds.value.map((fund) => fund.symbol))

  return availableFunds.value
    .filter((fund) => !selectedSymbols.has(fund.symbol))
    .filter((fund) => {
      if (!query) {
        return false
      }

      return (
        fund.name.toLowerCase().includes(query) ||
        fund.symbol.toLowerCase().includes(query)
      )
    })
    .slice(0, 8)
})

const summaryCards = computed(() => {
  if (!result.value) {
    return []
  }

  return [
    { label: 'Total Value', value: result.value.totalValue },
    { label: 'Total Exit Load', value: result.value.totalExitLoad },
    { label: 'Total Tax', value: result.value.totalTax },
    { label: 'Actual Value', value: result.value.actualValue },
  ]
})

const canSubmit = computed(() => parsedRedemptionAmount.value > 0 && !submitting.value)
const hasFundsToSell = computed(() => Boolean(result.value && result.value.funds.length > 0))

const normalizeFundOption = (fund: ApiMutualFund): MutualFundOption => ({
  name: fund.Name ?? 'Unnamed Fund',
  symbol: fund.Symbol ?? '',
})

const normalizeRedemptionFund = (fund: ApiRedemptionFund): RedemptionFund => ({
  fundName: fund.fund_name ?? 'Unnamed Fund',
  unitsToSell: fund.units_to_sell ?? 0,
  totalValue: fund.total_value ?? 0,
  pnl: fund.pnl ?? 0,
  exitLoad: fund.exit_load ?? 0,
  stcgTax: fund.stcg_tax ?? 0,
  ltcgTax: fund.ltcg_tax ?? 0,
})

const normalizeRedemptionResponse = (payload: ApiRedemptionResponse): RedemptionResponse => ({
  funds: (payload.funds ?? []).map(normalizeRedemptionFund),
  totalValue: payload.total_value ?? 0,
  totalExitLoad: payload.total_exit_load ?? 0,
  totalTax: payload.total_tax ?? 0,
  actualValue: payload.actual_value ?? 0,
})

const fetchMutualFunds = async () => {
  try {
    loadingFunds.value = true
    fundsError.value = ''

    const response = await fetch('/api/mutual_funds')
    if (!response.ok) {
      throw new Error(`HTTP error! status: ${response.status}`)
    }

    const raw = (await response.json()) as ApiMutualFund[]
    availableFunds.value = raw
      .map(normalizeFundOption)
      .filter((fund) => fund.symbol.length > 0)
      .sort((a, b) => a.name.localeCompare(b.name))
  } catch (error) {
    fundsError.value = `Failed to load mutual funds: ${error instanceof Error ? error.message : 'Unknown error'}`
  } finally {
    loadingFunds.value = false
  }
}

const addFundToAvoid = (fund: MutualFundOption) => {
  if (selectedFunds.value.some((selected) => selected.symbol === fund.symbol)) {
    avoidQuery.value = ''
    return
  }

  selectedFunds.value = [...selectedFunds.value, fund]
  avoidQuery.value = ''
}

const removeFundToAvoid = (symbol: string) => {
  selectedFunds.value = selectedFunds.value.filter((fund) => fund.symbol !== symbol)
}

const addFirstSuggestion = () => {
  if (filteredFunds.value.length === 0) {
    return
  }

  addFundToAvoid(filteredFunds.value[0])
}

const setRedemptionAmount = (value: number) => {
  redemptionAmount.value = value.toString()
}

const submitRedemption = async () => {
  try {
    submitting.value = true
    submitError.value = ''
    result.value = null

    const response = await fetch('/api/smart_redemption', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        amount: parsedRedemptionAmount.value,
        funds_to_avoid: selectedFunds.value.map((fund) => fund.symbol),
      }),
    })

    if (!response.ok) {
      const errorText = await response.text()
      throw new Error(errorText || `HTTP error! status: ${response.status}`)
    }

    const raw = (await response.json()) as ApiRedemptionResponse
    result.value = normalizeRedemptionResponse(raw)
  } catch (error) {
    submitError.value = `Failed to calculate smart redemption: ${error instanceof Error ? error.message : 'Unknown error'}`
  } finally {
    submitting.value = false
  }
}

onMounted(fetchMutualFunds)
</script>

<template>
  <div class="max-w-6xl mx-auto mt-4 px-4">
    <a href="/" class="underline">← Back to Portfolio</a>
    <h1 class="mt-4 text-2xl font-bold">Smart Redemption</h1>
    <p class="mt-2 text-muted-foreground">
      Optimize selling funds for lower exit load and tax amounts.
    </p>
  </div>

  <div class="max-w-6xl mx-auto mt-6 px-4 grid gap-6 lg:grid-cols-[22rem_minmax(0,1fr)]">
    <Card class="h-fit">
      <CardHeader>
        <CardTitle>How much do you need?</CardTitle>
      </CardHeader>
      <CardContent class="space-y-5">
        <div class="space-y-4">
          <div class="flex items-center rounded-[1.8rem] border-4 border-slate-200 bg-slate-50 px-8 py-6 shadow-[inset_0_1px_0_rgba(255,255,255,0.8)]">
            <span class="mr-6 text-4xl font-medium text-slate-400">₹</span>
            <Input
              id="redemption-amount"
              v-model="formattedRedemptionAmount"
              type="text"
              inputmode="numeric"
              placeholder="5,00,000"
              class="h-auto border-0 bg-transparent px-0 py-0 text-5xl font-semibold tracking-tight !text-slate-900 shadow-none focus-visible:ring-0 focus-visible:border-transparent"
            />
          </div>

          <div class="grid grid-cols-3 gap-4">
            <button
              v-for="amount in amountPresets"
              :key="amount"
              type="button"
              class="rounded-2xl border px-4 py-4 text-2xl font-medium transition-colors"
              :class="parsedRedemptionAmount === amount
                ? 'border-emerald-500 bg-emerald-100 text-emerald-700'
                : 'border-slate-200 bg-slate-50 text-slate-600 hover:border-slate-300 hover:bg-slate-100'"
              @click="setRedemptionAmount(amount)"
            >
              {{ formatAmountPreset(amount) }}
            </button>
          </div>
        </div>

        <div class="space-y-2">
          <Label for="funds-to-avoid">Funds to avoid</Label>
          <div class="relative">
            <Input
              id="funds-to-avoid"
              v-model="avoidQuery"
              placeholder="Type a fund name"
              :disabled="loadingFunds"
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
                class="flex w-full items-center justify-between px-3 py-2 text-left text-sm hover:bg-muted"
                @click="addFundToAvoid(fund)"
              >
                <span class="pr-4">{{ fund.name }}</span>
                <span class="text-xs text-muted-foreground">{{ fund.symbol }}</span>
              </button>
            </div>
          </div>

          <p v-if="loadingFunds" class="text-sm text-muted-foreground">Loading funds...</p>
          <p v-else-if="fundsError" class="text-sm text-red-600">{{ fundsError }}</p>
          <p v-else class="text-sm text-muted-foreground">
            Start typing to see matching fund names. 
          </p>

          <div v-if="selectedFunds.length > 0" class="flex flex-wrap gap-2 pt-2">
            <button
              v-for="fund in selectedFunds"
              :key="fund.symbol"
              type="button"
              class="inline-flex items-center gap-2 rounded-full border px-3 py-1 text-sm"
              @click="removeFundToAvoid(fund.symbol)"
            >
              <span>{{ fund.name }}</span>
              <span class="text-muted-foreground">×</span>
            </button>
          </div>
        </div>

        <Button
          class="w-full !border-transparent !bg-primary !text-primary-foreground hover:!bg-primary/90"
          :disabled="!canSubmit || loadingFunds"
          @click="submitRedemption"
        >
          {{ submitting ? 'Calculating...' : 'Show Funds' }}
        </Button>

        <p v-if="submitError" class="text-sm text-red-600">{{ submitError }}</p>
      </CardContent>
    </Card>

    <div class="space-y-6">
      <div v-if="result" class="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
        <Card v-for="card in summaryCards" :key="card.label">
          <CardHeader class="pb-2">
            <CardDescription>{{ card.label }}</CardDescription>
            <CardTitle class="text-2xl">{{ formatCurrency(card.value) }}</CardTitle>
          </CardHeader>
        </Card>
      </div>

      <Card v-if="hasFundsToSell">
        <CardHeader>
          <CardTitle>Recommended Funds to Sell</CardTitle>
        </CardHeader>
        <CardContent>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Fund Name</TableHead>
                <TableHead class="text-right">Units to Sell</TableHead>
                <TableHead class="text-right">Total Value</TableHead>
                <TableHead class="text-right">PnL</TableHead>
                <TableHead class="text-right">Exit Load</TableHead>
                <TableHead class="text-right">STCG</TableHead>
                <TableHead class="text-right">LTCG</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              <TableRow v-for="fund in result?.funds" :key="fund.fundName">
                <TableCell class="font-medium whitespace-normal">{{ fund.fundName }}</TableCell>
                <TableCell class="text-right">{{ formatNumber(fund.unitsToSell) }}</TableCell>
                <TableCell class="text-right">{{ formatCurrency(fund.totalValue) }}</TableCell>
                <TableCell class="text-right">
                  <span :class="amountClass(fund.pnl)">{{ formatCurrency(fund.pnl) }}</span>
                </TableCell>
                <TableCell class="text-right">{{ formatCurrency(fund.exitLoad) }}</TableCell>
                <TableCell class="text-right">{{ formatCurrency(fund.stcgTax) }}</TableCell>
                <TableCell class="text-right">{{ formatCurrency(fund.ltcgTax) }}</TableCell>
              </TableRow>
            </TableBody>
          </Table>
        </CardContent>
      </Card>
    </div>
  </div>
</template>
