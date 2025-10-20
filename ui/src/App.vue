<script setup lang="ts">
import { ref, computed, watch, onMounted } from 'vue'
import { Textarea } from "@/components/ui/textarea"
import { DonutChart } from "@/components/ui/chart-donut"
import { Input } from "@/components/ui/input"
import {
  Table,
  TableBody,
  TableCaption,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"

const cashAmount = ref('')
const input = ref<Array<{name: string, amount: number}>>([])
const loading = ref(true)
const error = ref('')

// Target allocation percentages
const equityTarget = ref(75)
const debtTarget = ref(20)
const goldTarget = ref(5)

// Fetch portfolio data from API
const fetchPortfolio = async () => {
  try {
    loading.value = true
    error.value = ''
    const response = await fetch('/api/portfolio')
    if (!response.ok) {
      throw new Error(`HTTP error! status: ${response.status}`)
    }
    const data = await response.json()
    input.value = data
  } catch (e) {
    error.value = `Failed to load portfolio: ${e instanceof Error ? e.message : 'Unknown error'}`
    console.error('Error fetching portfolio:', error.value)
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  const savedCash = localStorage.getItem('portfolioCashAmount')
  if (savedCash) {
    cashAmount.value = savedCash
  }
  
  const savedEquity = localStorage.getItem('equityTarget')
  if (savedEquity) {
    equityTarget.value = parseFloat(savedEquity)
  }
  
  const savedDebt = localStorage.getItem('debtTarget')
  if (savedDebt) {
    debtTarget.value = parseFloat(savedDebt)
  }
  
  const savedGold = localStorage.getItem('goldTarget')
  if (savedGold) {
    goldTarget.value = parseFloat(savedGold)
  }
  
  fetchPortfolio()
})

watch(cashAmount, (newValue) => {
  localStorage.setItem('portfolioCashAmount', newValue)
})

watch(equityTarget, (newValue) => {
  localStorage.setItem('equityTarget', newValue.toString())
})

watch(debtTarget, (newValue) => {
  localStorage.setItem('debtTarget', newValue.toString())
})

watch(goldTarget, (newValue) => {
  localStorage.setItem('goldTarget', newValue.toString())
})

const totalTargetPercent = computed(() => {
  return equityTarget.value + debtTarget.value + goldTarget.value
})

const totalAmount = computed(() => {
  let total = 0
  for (const item of input.value) {
    total += item.amount
  }
  const cash = parseFloat(cashAmount.value) || 0
  return total + cash
})

const holdings = computed(() => {
  const result = []
  
  for (const item of input.value) {
    if (item.name == "equity") {
      result.push({
          name: "equity", 
          label: `EQUITY - ${equityTarget.value}%`, 
          currentAmount: item.amount, 
          percent: (item.amount / totalAmount.value) * 100,
          rebalanceAmount: (totalAmount.value * equityTarget.value / 100) - item.amount
      })
    } else if (item.name == "debt") {
      result.push({
          name: "debt", 
          label: `DEBT - ${debtTarget.value}%`, 
          currentAmount: item.amount, 
          percent: (item.amount / totalAmount.value) * 100,
          rebalanceAmount: (totalAmount.value * debtTarget.value / 100) - item.amount
      })
    } else if (item.name == "gold") {
      result.push({
          name: "gold",
          label: `GOLD - ${goldTarget.value}%`,
          currentAmount: item.amount, 
          percent: (item.amount / totalAmount.value) * 100,
          rebalanceAmount: (totalAmount.value * goldTarget.value / 100) - item.amount
      })
    }
  }
  
  // Add cash holding if cash amount is populated
  const cash = parseFloat(cashAmount.value) || 0
  if (cash > 0) {
    result.push({
      name: "cash",
      label: "CASH - 0%",
      currentAmount: cash,
      percent: (cash / totalAmount.value) * 100,
      rebalanceAmount: -cash
    })
  }
  
  return result
})

// Helper function to format currency
const formatCurrency = (amount: number) => {
  return Math.round(amount).toLocaleString('en-IN')
}
</script>

<template>
  <h1 class="block ml-auto mr-auto text-2xl font-bold text-center mb-4">
    ₹ {{ formatCurrency(totalAmount) }}
  </h1>
  
	<div class="block ml-auto mr-auto mb-8 mt-8 max-w-md">
  	<DonutChart
  	  index="name"
  	  category="percent"
			:colors="['orange', 'blue', 'green', 'gray']"
			:valueFormatter="(tick) => `${tick.toFixed(1)}%`"
  	  :data="holdings"
  	/>
	</div>
  
  <div class="block ml-auto mr-auto mb-8 mt-8 max-w-md">
	 	 <h2 class="text-xl font-semibold text-center mb-4">Cash Amount</h2>
     <Textarea 
       v-model="cashAmount" 
       placeholder="Enter cash amount" 
       type="number"
     />
  </div>

  <div class="block ml-auto mr-auto mb-8 mt-8 max-w-md p-6 border rounded-lg">
    <h2 class="text-xl font-semibold text-center mb-4">Target Allocation</h2>
    
    <div class="grid grid-cols-3 gap-4 mb-4">
      <div>
        <Label for="equity" class="text-sm">Equity %</Label>
        <Input 
          id="equity"
          v-model.number="equityTarget" 
          type="number"
          min="0"
          max="100"
          step="1"
          class="mt-1"
        />
      </div>
      <div>
        <Label for="debt" class="text-sm">Debt %</Label>
        <Input 
          id="debt"
          v-model.number="debtTarget" 
          type="number"
          min="0"
          max="100"
          step="1"
          class="mt-1"
        />
      </div>
      <div>
        <Label for="gold" class="text-sm">Gold %</Label>
        <Input 
          id="gold"
          v-model.number="goldTarget" 
          type="number"
          min="0"
          max="100"
          step="1"
          class="mt-1"
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
  	        {{ holding.label }}
  	      </TableCell>
  	      <TableCell class="text-right">
  	        ₹{{ formatCurrency(holding.currentAmount) }}
  	      </TableCell>
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
