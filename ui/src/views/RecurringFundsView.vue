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

type RecurringFund = {
  name: string
  totalValue: number
}

type ApiFund = {
  Name?: string
  TotalValue?: number
}

const funds = ref<RecurringFund[]>([])
const loading = ref(true)
const error = ref('')

const formatCurrency = (value: number) =>
  `₹${value.toLocaleString('en-IN', { maximumFractionDigits: 0 })}`

const normalizeFund = (fund: ApiFund): RecurringFund => ({
  name: fund.Name ?? 'Unnamed Fund',
  totalValue: fund.TotalValue ?? 0,
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
        </TableRow>
      </TableHeader>
      <TableBody>
        <TableRow v-for="fund in funds" :key="fund.name">
          <TableCell class="font-medium whitespace-normal">{{ fund.name }}</TableCell>
          <TableCell class="text-right">{{ formatCurrency(fund.totalValue) }}</TableCell>
        </TableRow>
        <TableRow v-if="funds.length === 0">
          <TableCell :colspan="2" class="text-center text-muted-foreground py-4">
            No recurring funds data available.
          </TableCell>
        </TableRow>
      </TableBody>
    </Table>
  </div>
</template>
