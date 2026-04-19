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
	LSP             float64 `json:"lsp"` // Last Traded Price
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
	return (lot.Fund.LSP - lot.Price) * units
}

func calculateExitLoad(lot Lot, units float64) float64 {
	if lot.Age < lot.Fund.ExitLoadDays {
		return (lot.Fund.LSP * units * lot.Fund.ExitLoadPercent) / 100
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

func main() {
	transactions, err := loadTxns("transactions.json")
	if err != nil {
		log.Fatalf("err: %v", err)
	}

	funds, err := loadFunds("equity.json")
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

		if len(lot.Fund.Lots) == 0 {
			lot.TotalExitLoad = lot.ExitLoad
			lot.TotalQty = lot.Qty
			lot.TotalSTCGTax = lot.STCGTax
			lot.TotalLTCGTax = lot.LTCGTax
			lot.TotalPnL = lot.PnL
		} else {
			lastLot := f.Lots[len(f.Lots)-1]
			lot.TotalExitLoad = lot.ExitLoad + lastLot.TotalExitLoad
			lot.TotalQty = lot.Qty + lastLot.TotalQty
			lot.TotalSTCGTax = lot.STCGTax + lastLot.TotalSTCGTax
			lot.TotalLTCGTax = lot.LTCGTax + lastLot.TotalLTCGTax
			lot.TotalPnL = lot.PnL + lastLot.TotalPnL
		}

		f.Lots = append(f.Lots, lot)
	}

	for _, f := range fundsBySymbol {
		fmt.Println(f.Name, f.Lots[len(f.Lots)-1].TotalQty)
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
		Amount:       300000.0,
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
				remaining += candidate.UnitsToSell * candidate.Fund.LSP
			}
		}

		unitsToSell := lot.TotalQty
		if remaining < lot.TotalQty*lot.Fund.LSP {
			unitsToSell = remaining / lot.Fund.LSP
		}

		fundsToSell[lot.Fund.Symbol] = Candidate{
			Fund:        lot.Fund,
			LotNumber:   lot.Number,
			UnitsToSell: unitsToSell,
			TotalValue:  unitsToSell * lot.Fund.LSP,
			PnL:         lot.TotalPnL - lot.PnL + calculatePnL(lot, unitsToSell),
			ExitLoad:    lot.TotalExitLoad - lot.ExitLoad + calculateExitLoad(lot, unitsToSell),
			STCGTax:     lot.TotalSTCGTax - lot.STCGTax + calculateSTCGTax(lot.Age, calculatePnL(lot, unitsToSell)),
			LTCGTax:     lot.TotalLTCGTax - lot.LTCGTax + calculateLTCGTax(lot.Age, calculatePnL(lot, unitsToSell)),
		}

		remaining -= unitsToSell * lot.Fund.LSP
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

	// add all transactions to database from cas statement
	// get transactions from database
	// construct funds with lots with accumulated
	// get all lots from all funds, sort them based on (total_exit_load + totaltax) lowest
	// pick lots till request satisfied
	// construct candidates from the lots; accumulate in redemption request
}
