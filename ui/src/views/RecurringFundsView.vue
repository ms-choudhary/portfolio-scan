<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { Button } from '@/components/ui/button'
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

type RecurringFund = {
  symbol: string
  name: string
  category: string
  subCategory: string
  baseAmount: number
  startFrom: string
  monthlyAmount: number
  totalValue: number
}

type ApiFund = {
  symbol?: string
  name?: string
  category?: string
  sub_category?: string
  base_amount?: number
  start_from?: string
  monthly_amount?: number
  total_value?: number
}

const funds = ref<RecurringFund[]>([])
const loading = ref(true)
const error = ref('')

const editingSymbol = ref<string | null>(null)
const saving = ref(false)
const saveError = ref('')
const draft = reactive({ baseAmount: 0, startFrom: '', monthlyAmount: 0 })

const startFromPattern = /^(0[1-9]|1[0-2])\/\d{4}$/

const formatCurrency = (value: number) =>
  `₹${value.toLocaleString('en-IN', { maximumFractionDigits: 0 })}`

const normalizeFund = (fund: ApiFund): RecurringFund => ({
  symbol: fund.symbol ?? '',
  name: fund.name ?? 'Unnamed Fund',
  category: fund.category ?? '',
  subCategory: fund.sub_category ?? '',
  baseAmount: fund.base_amount ?? 0,
  startFrom: fund.start_from ?? '',
  monthlyAmount: fund.monthly_amount ?? 0,
  totalValue: fund.total_value ?? 0,
})

const fetchRecurringFunds = async () => {
  try {
    loading.value = true
    error.value = ''
    const response = await fetch('/api/recurring_funds')
    if (!response.ok) {
      throw new Error(`HTTP error! status: ${response.status}`)
    }

    const raw = (await response.json()) as ApiFund[]
    funds.value = raw.map(normalizeFund)
  } catch (e) {
    error.value = `Failed to load recurring funds: ${e instanceof Error ? e.message : 'Unknown error'}`
  } finally {
    loading.value = false
  }
}

const toggleEdit = (fund: RecurringFund) => {
  if (editingSymbol.value === fund.symbol) {
    editingSymbol.value = null
    saveError.value = ''
    return
  }

  editingSymbol.value = fund.symbol
  draft.baseAmount = fund.baseAmount
  draft.startFrom = fund.startFrom
  draft.monthlyAmount = fund.monthlyAmount
  saveError.value = ''
}

const cancelEdit = () => {
  editingSymbol.value = null
  saveError.value = ''
}

const saveEdit = async (symbol: string) => {
  if (!startFromPattern.test(draft.startFrom)) {
    saveError.value = 'Start From must be in MM/YYYY format (e.g. 04/2026).'
    return
  }
  if (draft.baseAmount < 0 || draft.monthlyAmount < 0) {
    saveError.value = 'Amounts must be non-negative.'
    return
  }

  try {
    saving.value = true
    saveError.value = ''

    const response = await fetch('/api/recurring_funds/update', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        symbol,
        base_amount: draft.baseAmount,
        start_from: draft.startFrom,
        monthly_amount: draft.monthlyAmount,
      }),
    })

    if (!response.ok) {
      const text = await response.text()
      throw new Error(text || `HTTP error! status: ${response.status}`)
    }

    const updated = normalizeFund((await response.json()) as ApiFund)
    const index = funds.value.findIndex((f) => f.symbol === symbol)
    if (index !== -1) {
      funds.value[index] = updated
    }

    editingSymbol.value = null
  } catch (e) {
    saveError.value = `Failed to save: ${e instanceof Error ? e.message : 'Unknown error'}`
  } finally {
    saving.value = false
  }
}

onMounted(fetchRecurringFunds)
</script>

<template>
  <div class="max-w-4xl mx-auto mt-4 px-4">
    <a href="/" class="underline">← Back to Portfolio</a>
    <h1 class="text-2xl font-bold mt-4">Recurring Funds</h1>
  </div>

  <div v-if="loading" class="text-center mt-8">Loading recurring funds...</div>
  <div v-else-if="error" class="text-center mt-8 text-red-600">{{ error }}</div>

  <div v-else class="max-w-4xl mx-auto mt-6 px-4">
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead>Fund Name</TableHead>
          <TableHead class="text-right">Total Value</TableHead>
          <TableHead class="text-right">Actions</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        <template v-for="fund in funds" :key="fund.symbol">
          <TableRow>
            <TableCell class="font-medium whitespace-normal">{{ fund.name }}</TableCell>
            <TableCell class="text-right">{{ formatCurrency(fund.totalValue) }}</TableCell>
            <TableCell class="text-right">
              <Button size="sm" variant="outline" @click="toggleEdit(fund)">
                {{ editingSymbol === fund.symbol ? 'Close' : 'Edit' }}
              </Button>
            </TableCell>
          </TableRow>

          <TableRow v-if="editingSymbol === fund.symbol" class="hover:bg-transparent">
            <TableCell :colspan="3" class="bg-muted/40 p-4">
              <div class="grid grid-cols-1 gap-4 sm:grid-cols-3">
                <div class="space-y-1.5">
                  <Label :for="`base-${fund.symbol}`">Base Amount</Label>
                  <Input
                    :id="`base-${fund.symbol}`"
                    v-model.number="draft.baseAmount"
                    type="number"
                    min="0"
                  />
                </div>
                <div class="space-y-1.5">
                  <Label :for="`start-${fund.symbol}`">Start From</Label>
                  <Input
                    :id="`start-${fund.symbol}`"
                    v-model="draft.startFrom"
                    placeholder="MM/YYYY"
                  />
                </div>
                <div class="space-y-1.5">
                  <Label :for="`monthly-${fund.symbol}`">Monthly Amount</Label>
                  <Input
                    :id="`monthly-${fund.symbol}`"
                    v-model.number="draft.monthlyAmount"
                    type="number"
                    min="0"
                  />
                </div>
              </div>

              <p v-if="saveError" class="mt-3 text-sm text-red-600">{{ saveError }}</p>

              <div class="mt-4 flex flex-col gap-2 sm:flex-row sm:justify-end">
                <Button
                  variant="outline"
                  class="w-full sm:w-auto"
                  :disabled="saving"
                  @click="cancelEdit"
                >
                  Cancel
                </Button>
                <Button
                  class="w-full !bg-primary !text-primary-foreground hover:!bg-primary/90 sm:w-auto"
                  :disabled="saving"
                  @click="saveEdit(fund.symbol)"
                >
                  {{ saving ? 'Saving...' : 'Save' }}
                </Button>
              </div>
            </TableCell>
          </TableRow>
        </template>

        <TableRow v-if="funds.length === 0">
          <TableCell :colspan="3" class="text-center text-muted-foreground py-4">
            No recurring funds data available.
          </TableCell>
        </TableRow>
      </TableBody>
    </Table>
  </div>
</template>
