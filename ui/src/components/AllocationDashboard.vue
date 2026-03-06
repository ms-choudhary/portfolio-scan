<script setup lang="ts">
import { computed } from 'vue'
import { DonutChart } from '@/components/ui/chart-donut'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import {
  Table,
  TableBody,
  TableCaption,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'

export type AllocationItem = {
  name: string
  amount: number
}

export type HoldingItem = {
  name: string
  label: string
  currentAmount: number
  percent: number
  rebalanceAmount: number
  linkTo?: string
}

const props = defineProps<{
  title: string
  totalAmount: number
  donutColors: string[]
  holdings: HoldingItem[]
  showCashInput?: boolean
  cashAmount?: string
  showTargetAllocation?: boolean
  targets?: Array<{ key: string; label: string; value: number }>
  totalTargetPercent?: number
}>()

const emit = defineEmits<{
  'update:cashAmount': [value: string]
  'update:target': [payload: { key: string; value: number }]
}>()

const chartData = computed(() => props.holdings)

const formatCurrency = (amount: number) => Math.round(amount).toLocaleString('en-IN')
</script>

<template>
  <h1 class="block ml-auto mr-auto text-2xl font-bold text-center mb-4">
    {{ title }}: ₹ {{ formatCurrency(totalAmount) }}
  </h1>

  <div class="block ml-auto mr-auto mb-8 mt-8 max-w-md">
    <DonutChart
      index="name"
      category="percent"
      :colors="donutColors"
      :valueFormatter="(tick) => `${tick.toFixed(1)}%`"
      :data="chartData"
    />
  </div>

  <div v-if="showCashInput" class="block ml-auto mr-auto mb-8 mt-8 max-w-md">
    <h2 class="text-xl font-semibold text-center mb-4">Cash Amount</h2>
    <Textarea
      :model-value="cashAmount"
      placeholder="Enter cash amount"
      type="number"
      @update:model-value="(value) => emit('update:cashAmount', String(value ?? ''))"
    />
  </div>

  <div v-if="showTargetAllocation" class="block ml-auto mr-auto mb-8 mt-8 max-w-md p-6 border rounded-lg">
    <h2 class="text-xl font-semibold text-center mb-4">Target Allocation</h2>

    <div class="grid grid-cols-3 gap-4 mb-4">
      <div v-for="target in targets" :key="target.key">
        <Label :for="target.key" class="text-sm">{{ target.label }}</Label>
        <Input
          :id="target.key"
          :model-value="target.value"
          type="number"
          min="0"
          max="100"
          step="1"
          class="mt-1"
          @update:model-value="(value) => emit('update:target', { key: target.key, value: Number(value) || 0 })"
        />
      </div>
    </div>

    <div class="text-center font-semibold" :class="totalTargetPercent !== 100 ? 'text-red-600' : 'text-green-600'">
      Total: {{ totalTargetPercent }}%
      <span v-if="totalTargetPercent !== 100">(should equal 100%)</span>
      <span v-else>✓</span>
    </div>
  </div>

  <div class="max-w-2xl mx-auto">
    <Table>
      <TableCaption>Current Holdings.</TableCaption>
      <TableHeader>
        <TableRow>
          <TableHead>Class</TableHead>
          <TableHead class="text-right">Current</TableHead>
          <TableHead class="text-right">Rebalance</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        <TableRow v-for="holding in holdings" :key="holding.name">
          <TableCell class="font-medium">
            <a v-if="holding.linkTo" class="underline" :href="holding.linkTo">{{ holding.label }}</a>
            <span v-else>{{ holding.label }}</span>
          </TableCell>
          <TableCell class="text-right">₹{{ formatCurrency(holding.currentAmount) }}</TableCell>
          <TableCell class="text-right">
            <div :class="holding.rebalanceAmount < 0 ? 'text-red-700' : 'text-green-700'">
              ₹{{ formatCurrency(holding.rebalanceAmount) }}
            </div>
          </TableCell>
        </TableRow>
      </TableBody>
    </Table>
  </div>
</template>
