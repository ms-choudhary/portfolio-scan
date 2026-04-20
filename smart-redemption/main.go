package main

import (
	"cmp"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os"
	"slices"
	"time"

	"smart-redemption/nav"
)

type Transaction struct {
	Symbol string    `json:"symbol"`
	Price  float64   `json:"price"`
	Qty    float64   `json:"qty"`
	Date   time.Time `json:"date"`
}

type Fund struct {
	Symbol          string  `json:"symbol"`
	Name            string  `json:"name"`
	ExitLoadPercent float64 `json:"exit_load_percent,omitempty"`
	ExitLoadDays    int     `json:"exit_load_days,omitempty"`
	LTP             float64 `json:"ltp,omitempty"` // Last Traded Price
	Lots            []Lot   `json:"lots,omitempty"`
}

type Accumulated struct {
	TotalExitLoad float64
	TotalQty      float64
	TotalSTCGTax  float64
	TotalLTCGTax  float64
	TotalPnL      float64
}

type Lot struct {
	Fund     *Fund
	Number   int
	Price    float64
	Qty      float64
	Age      int
	PnL      float64
	ExitLoad float64
	STCGTax  float64
	LTCGTax  float64
	Accumulated
}

type Candidate struct {
	Fund        *Fund
	LotNumber   int
	UnitsToSell float64
	TotalValue  float64
	PnL         float64
	ExitLoad    float64
	STCGTax     float64
	LTCGTax     float64
}

type RedemptionRequest struct {
	Amount       float64
	FundsToAvoid []string
}

type RedemptionResponse struct {
	Funds         []Candidate
	TotalValue    float64
	TotalExitLoad float64
	TotalTax      float64
	ActualValue   float64
}

func loadFunds(fileName string) ([]Fund, error) {
	var funds struct {
		Funds []Fund
	}
	data, err := os.ReadFile(fileName)
	if err != nil {
		return []Fund{}, err
	}
	if err := json.Unmarshal(data, &funds); err != nil {
		return []Fund{}, err
	}

	symbols := []string{}
	for _, f := range funds.Funds {
		symbols = append(symbols, f.Symbol)
	}

	if err := nav.UpdateNAVs(symbols); err != nil {
		log.Printf("failed to update navs: %v", err)
	}

	for i, f := range funds.Funds {
		ltp, err := nav.Get(f.Symbol)
		if err != nil {
			return []Fund{}, err
		}

		funds.Funds[i].LTP = ltp
	}

	return funds.Funds, nil
}

func loadTxns(fileName string) ([]Transaction, error) {
	var txns struct {
		Transactions []Transaction
	}
	data, err := os.ReadFile(fileName)
	if err != nil {
		return []Transaction{}, err
	}
	if err := json.Unmarshal(data, &txns); err != nil {
		return []Transaction{}, err
	}

	return txns.Transactions, nil
}

func daysFromToday(date time.Time) int {
	d := time.Now().Sub(date)
	if d < 0 {
		return 0
	}
	return int(d.Hours() / 24)
}

func calculatePnL(lot Lot, units float64) float64 {
	return (lot.Fund.LTP - lot.Price) * units
}

func calculateExitLoad(lot Lot, units float64) float64 {
	if lot.Age < lot.Fund.ExitLoadDays {
		return (lot.Fund.LTP * units * lot.Fund.ExitLoadPercent) / 100
	}
	return 0.0
}

func calculateSTCGTax(age int, pnl float64) float64 {
	if age < 365 && pnl > 0 {
		return pnl * 0.2
	}
	return 0.0
}

func calculateLTCGTax(age int, pnl float64) float64 {
	if age >= 365 && pnl > 0 {
		return pnl * 0.125
	}
	return 0.0
}

func (f *Fund) computeAccumulatedValues() {
	for i, lot := range f.Lots {
		if i == 0 {
			f.Lots[i].TotalExitLoad = lot.ExitLoad
			f.Lots[i].TotalQty = lot.Qty
			f.Lots[i].TotalSTCGTax = lot.STCGTax
			f.Lots[i].TotalLTCGTax = lot.LTCGTax
			f.Lots[i].TotalPnL = lot.PnL
		} else {
			lastLot := f.Lots[i-1]
			f.Lots[i].TotalExitLoad = lot.ExitLoad + lastLot.TotalExitLoad
			f.Lots[i].TotalQty = lot.Qty + lastLot.TotalQty
			f.Lots[i].TotalSTCGTax = lot.STCGTax + lastLot.TotalSTCGTax
			f.Lots[i].TotalLTCGTax = lot.LTCGTax + lastLot.TotalLTCGTax
			f.Lots[i].TotalPnL = lot.PnL + lastLot.TotalPnL
		}
	}
}

func (f *Fund) removeRedeemedLots(units float64) {
	remaining := units
	i := 0
	for ; i < len(f.Lots) && remaining > 1e-2; i++ {
		if remaining < f.Lots[i].Qty {
			newqty := f.Lots[i].Qty - remaining
			f.Lots[i].Qty = newqty
			f.Lots[i].PnL = calculatePnL(f.Lots[i], newqty)
			f.Lots[i].ExitLoad = calculateExitLoad(f.Lots[i], newqty)
			f.Lots[i].STCGTax = calculateSTCGTax(f.Lots[i].Age, f.Lots[i].PnL)
			f.Lots[i].LTCGTax = calculateLTCGTax(f.Lots[i].Age, f.Lots[i].PnL)
			break
		}
		remaining -= f.Lots[i].Qty
	}
	f.Lots = f.Lots[i:]
}

func main() {
	transactions, err := loadTxns("transactions.json")
	if err != nil {
		log.Fatalf("err: %v", err)
	}

	funds, err := loadFunds("alldebt.json")
	if err != nil {
		log.Fatalf("err: %v", err)
	}

	fundsBySymbol := map[string]*Fund{}
	for i, f := range funds {
		fundsBySymbol[f.Symbol] = &funds[i]
	}

	for _, t := range transactions {
		f, ok := fundsBySymbol[t.Symbol]
		if !ok {
			continue
		}

		if len(f.Lots) == 0 {
			f.Lots = []Lot{}
		}

		// redemption
		if t.Qty < 0 {
			f.removeRedeemedLots(math.Abs(t.Qty))
			continue
		}

		lot := Lot{
			Fund:   f,
			Number: len(f.Lots),
			Price:  t.Price,
			Qty:    t.Qty,
		}

		lot.Age = daysFromToday(t.Date)
		lot.PnL = calculatePnL(lot, lot.Qty)
		lot.ExitLoad = calculateExitLoad(lot, lot.Qty)
		lot.STCGTax = calculateSTCGTax(lot.Age, lot.PnL)
		lot.LTCGTax = calculateLTCGTax(lot.Age, lot.PnL)

		f.Lots = append(f.Lots, lot)
	}

	for _, f := range fundsBySymbol {
		f.computeAccumulatedValues()
	}

	allLots := []Lot{}
	for _, f := range fundsBySymbol {
		allLots = append(allLots, f.Lots...)
	}

	compareLots := func(a, b Lot) int {
		totalCostA := a.TotalExitLoad + a.TotalSTCGTax
		totalCostB := b.TotalExitLoad + b.TotalSTCGTax

		if n := cmp.Compare(totalCostA, totalCostB); n != 0 {
			return n
		}

		return cmp.Compare(math.Abs(a.TotalPnL), math.Abs(b.TotalPnL))
	}

	slices.SortFunc(allLots, compareLots)

	req := RedemptionRequest{
		Amount:       4500000.0,
		FundsToAvoid: []string{},
	}

	i := 0
	remaining := req.Amount
	fundsToSell := map[string]Candidate{}

	for i < len(allLots) && remaining > 1e-2 {
		lot := allLots[i]
		i++

		if slices.Contains(req.FundsToAvoid, lot.Fund.Symbol) {
			continue
		}

		if candidate, exists := fundsToSell[lot.Fund.Symbol]; exists {
			if lot.Number < candidate.LotNumber {
				continue
			} else {
				remaining += candidate.UnitsToSell * candidate.Fund.LTP
			}
		}

		unitsToSell := lot.TotalQty
		if remaining < lot.TotalQty*lot.Fund.LTP {
			unitsToSell = remaining / lot.Fund.LTP
		}

		fundsToSell[lot.Fund.Symbol] = Candidate{
			Fund:        lot.Fund,
			LotNumber:   lot.Number,
			UnitsToSell: unitsToSell,
			TotalValue:  unitsToSell * lot.Fund.LTP,
			PnL:         lot.TotalPnL - lot.PnL + calculatePnL(lot, unitsToSell),
			ExitLoad:    lot.TotalExitLoad - lot.ExitLoad + calculateExitLoad(lot, unitsToSell),
			STCGTax:     lot.TotalSTCGTax - lot.STCGTax + calculateSTCGTax(lot.Age, calculatePnL(lot, unitsToSell)),
			LTCGTax:     lot.TotalLTCGTax - lot.LTCGTax + calculateLTCGTax(lot.Age, calculatePnL(lot, unitsToSell)),
		}

		remaining -= unitsToSell * lot.Fund.LTP
		fmt.Printf("sell %d lot of %f units of fund %s, remaining: %f\n", lot.Number, unitsToSell, lot.Fund.Name, remaining)
	}

	if i == len(allLots) {
		fmt.Println("insufficient balance")
	}

	for _, f := range fundsToSell {
		fmt.Println(f.Fund.Name)
		fmt.Println("===========================")
		fmt.Printf("units: %f, total value: %f, pnl: %f, exit load: %f, stcg: %f, ltcg: %f\n", f.UnitsToSell, f.TotalValue, f.PnL, f.ExitLoad, f.STCGTax, f.LTCGTax)
	}
}
