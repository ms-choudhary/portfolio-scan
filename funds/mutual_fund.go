package funds

import (
	"cmp"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os"
	"slices"
	"strings"
	"time"

	"portfolio-scan/nav"
)

type Transaction struct {
	Account string    `json:"account"`
	Symbol  string    `json:"symbol"`
	Price   float64   `json:"price"`
	Qty     float64   `json:"qty"`
	Date    time.Time `json:"date"`
}

type MutualFund struct {
	Account         string   `json:"account"`
	Category        Category `json:"category"`
	SubCategory     Category `json:"sub_category"`
	Symbol          string   `json:"symbol"`
	Name            string   `json:"name"`
	ExitLoadPercent float64  `json:"exit_load_percent,omitempty"`
	ExitLoadDays    int      `json:"exit_load_days,omitempty"`
	LTP             float64  `json:"-"` // Last Traded Price
	Lots            []Lot    `json:"-"`
}

type Accumulated struct {
	TotalExitLoad float64
	TotalQty      float64
	TotalSTCGTax  float64
	TotalLTCGTax  float64
	TotalPnL      float64
}

type Lot struct {
	Fund     *MutualFund
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

type LotResponse struct {
	Number   int
	Units    float64
	Price    float64
	Age      int
	PnL      float64
	ExitLoad float64
	STCG     float64
	LTCG     float64
}

type FundResponse struct {
	Symbol     string
	Name       string
	Units      float64
	TotalValue float64
	TotalPnL   float64
	Lots       []LotResponse
}

type Candidate struct {
	Fund        *MutualFund `json:"-"`
	FundName    string      `json:"fund_name"`
	LotNumber   int         `json:"lot_number"`
	UnitsToSell float64     `json:"units_to_sell"`
	TotalValue  float64     `json:"total_value"`
	PnL         float64     `json:"pnl"`
	ExitLoad    float64     `json:"exit_load"`
	STCGTax     float64     `json:"stcg_tax"`
	LTCGTax     float64     `json:"ltcg_tax"`
}

type RedemptionRequest struct {
	Amount       float64  `json:"amount"`
	FundsToAvoid []string `json:"funds_to_avoid"`
}

type RedemptionResponse struct {
	Funds         []Candidate `json:"funds"`
	TotalValue    float64     `json:"total_value"`
	TotalExitLoad float64     `json:"total_exit_load"`
	TotalTax      float64     `json:"total_tax"`
	TotalPnL      float64     `json:"total_pnl"`
	ActualValue   float64     `json:"actual_value"`
}

func (f MutualFund) CategoryName() Category {
	return f.Category
}

func (f MutualFund) SubCategoryName() Category {
	return f.SubCategory
}

func (f MutualFund) Value() float64 {
	return f.Lots[len(f.Lots)-1].TotalQty * f.LTP
}

func (f MutualFund) ToResponse() FundResponse {
	response := FundResponse{
		Symbol:     f.Symbol,
		Name:       f.Name,
		Units:      f.Lots[len(f.Lots)-1].TotalQty,
		TotalValue: f.Lots[len(f.Lots)-1].TotalQty * f.LTP,
		TotalPnL:   f.Lots[len(f.Lots)-1].TotalPnL,
		Lots:       []LotResponse{},
	}

	for i := len(f.Lots) - 1; i >= 0; i-- {
		lotResponse := LotResponse{
			Number:   f.Lots[i].Number,
			Units:    f.Lots[i].Qty,
			Price:    f.Lots[i].Price,
			Age:      f.Lots[i].Age,
			PnL:      f.Lots[i].PnL,
			ExitLoad: f.Lots[i].TotalExitLoad,
			STCG:     f.Lots[i].TotalSTCGTax,
			LTCG:     f.Lots[i].TotalLTCGTax,
		}
		response.Lots = append(response.Lots, lotResponse)
	}

	return response
}

func loadMutualFunds(fileName string) ([]MutualFund, error) {
	var funds struct {
		MutualFunds []MutualFund `json:"mutual_funds"`
	}
	data, err := os.ReadFile(fileName)
	if err != nil {
		return []MutualFund{}, err
	}
	if err := json.Unmarshal(data, &funds); err != nil {
		return []MutualFund{}, err
	}

	symbols := []string{}
	for _, f := range funds.MutualFunds {
		symbols = append(symbols, f.Symbol)
	}

	if err := nav.UpdateNAVs(symbols); err != nil {
		log.Printf("failed to update navs: %v", err)
	}

	for i, f := range funds.MutualFunds {
		ltp, err := nav.Get(f.Symbol)
		if err != nil {
			return []MutualFund{}, err
		}

		funds.MutualFunds[i].LTP = ltp
	}

	return funds.MutualFunds, nil
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

func getAllTransactions() ([]Transaction, error) {
	dir, err := os.ReadDir(".")
	if err != nil {
		return []Transaction{}, err
	}

	var result struct {
		Transactions []Transaction
	}
	result.Transactions = []Transaction{}

	for _, entry := range dir {
		if !entry.IsDir() && strings.HasPrefix(entry.Name(), "transactions") && strings.HasSuffix(entry.Name(), ".json") {
			txns, err := loadTxns(entry.Name())
			if err != nil {
				return []Transaction{}, err
			}

			result.Transactions = append(result.Transactions, txns...)
		}
	}

	return result.Transactions, nil
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

func (f *MutualFund) computeAccumulatedValues() {
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

func (f *MutualFund) removeRedeemedLots(units float64) {
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

func GetMutualFunds() ([]MutualFund, error) {
	transactions, err := getAllTransactions()
	if err != nil {
		return []MutualFund{}, err
	}

	funds, err := loadMutualFunds("mutual_funds.json")
	if err != nil {
		return []MutualFund{}, err
	}

	fundsBySymbol := map[string]*MutualFund{}
	for i, f := range funds {
		fundsBySymbol[f.Symbol] = &funds[i]
	}

	for _, t := range transactions {
		f, ok := fundsBySymbol[t.Symbol]
		if !ok {
			continue
		}

		if t.Account != f.Account {
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
		if f.Category == Equity {
			lot.STCGTax = calculateSTCGTax(lot.Age, lot.PnL)
			lot.LTCGTax = calculateLTCGTax(lot.Age, lot.PnL)
		}

		f.Lots = append(f.Lots, lot)
	}

	result := []MutualFund{}
	for _, f := range fundsBySymbol {
		if len(f.Lots) == 0 {
			continue
		}
		f.computeAccumulatedValues()
		result = append(result, *f)
	}

	return result, nil
}

// req := RedemptionRequest{
// Amount:       4500000.0,
// FundsToAvoid: []string{},
// }
func SmartRedemption(req RedemptionRequest) (RedemptionResponse, error) {
	funds, err := GetMutualFunds()
	if err != nil {
		return RedemptionResponse{}, err
	}

	log.Printf("===== received redemption request, amount: %f, funds to avoid: %v", req.Amount, req.FundsToAvoid)

	allLots := []Lot{}
	for _, f := range funds {
		if f.Category == Equity {
			allLots = append(allLots, f.Lots...)
		}
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

	i := 0
	remaining := req.Amount
	fundsToSell := map[string]Candidate{}

	for i < len(allLots) && remaining > 1e-2 {
		lot := allLots[i]
		i++

		if slices.Contains(req.FundsToAvoid, lot.Fund.Symbol) {
			continue
		}

		prevCandidateValue := 0.0
		if candidate, exists := fundsToSell[lot.Fund.Symbol]; exists {
			if lot.Number < candidate.LotNumber {
				continue
			} else {
				prevCandidateValue = candidate.UnitsToSell * candidate.Fund.LTP
				remaining += prevCandidateValue
			}
		}

		prevLotsUnits := lot.TotalQty - lot.Qty
		lotUnitsShare := lot.Qty

		unitsToSell := lot.TotalQty
		if remaining < lot.TotalQty*lot.Fund.LTP {
			if remaining < prevLotsUnits*lot.Fund.LTP {
				log.Printf("invalid: cannot satisfy remaining just by selling this lot (age: %d) of fund %s", lot.Age, lot.Fund.Name[:10])
				remaining -= prevCandidateValue
				continue
			}

			lotUnitsShare := (remaining - prevLotsUnits*lot.Fund.LTP) / lot.Fund.LTP
			unitsToSell = prevLotsUnits + lotUnitsShare
		} else if unitsToSell*lot.Fund.LTP < 5000 {
			log.Printf("skipping %d lot of fund %s as val %f less than 5k", lot.Number, lot.Fund.Name[:10], unitsToSell*lot.Fund.LTP)
			continue
		}

		fundsToSell[lot.Fund.Symbol] = Candidate{
			Fund:        lot.Fund,
			FundName:    lot.Fund.Name,
			LotNumber:   lot.Number,
			UnitsToSell: unitsToSell,
			TotalValue:  unitsToSell * lot.Fund.LTP,
			PnL:         lot.TotalPnL - lot.PnL + calculatePnL(lot, lotUnitsShare),
			ExitLoad:    lot.TotalExitLoad - lot.ExitLoad + calculateExitLoad(lot, lotUnitsShare),
			STCGTax:     lot.TotalSTCGTax - lot.STCGTax + calculateSTCGTax(lot.Age, calculatePnL(lot, lotUnitsShare)),
			LTCGTax:     lot.TotalLTCGTax - lot.LTCGTax + calculateLTCGTax(lot.Age, calculatePnL(lot, lotUnitsShare)),
		}

		remaining -= unitsToSell * lot.Fund.LTP
		log.Printf("sell %f units of fund %s lotAge %d | pnl: %f | exit: %f | stcg: %f | ltcg: %f | remaining: %f\n", unitsToSell, lot.Fund.Name[:10], lot.Age, lot.TotalPnL, lot.TotalExitLoad, lot.TotalSTCGTax, lot.TotalLTCGTax, remaining)
	}

	if i == len(allLots) {
		return RedemptionResponse{}, fmt.Errorf("insufficient balance")
	}

	response := RedemptionResponse{Funds: []Candidate{}}
	for _, f := range fundsToSell {
		response.Funds = append(response.Funds, f)
		response.TotalValue += f.TotalValue
		response.TotalExitLoad += f.ExitLoad
		response.TotalTax += f.STCGTax + f.LTCGTax
		response.TotalPnL += f.PnL
	}

	sortByValueDesc := func(a, b Candidate) int {
		return cmp.Compare(b.TotalValue, a.TotalValue)
	}

	slices.SortFunc(response.Funds, sortByValueDesc)

	response.ActualValue = response.TotalValue - response.TotalExitLoad - response.TotalTax

	return response, nil
}
