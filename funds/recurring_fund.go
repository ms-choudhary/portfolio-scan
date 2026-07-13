package funds

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const recurringFundsFile = "recurring_funds.json"

type RecurringFund struct {
	Category      Category `json:"category"`
	SubCategory   Category `json:"sub_category"`
	Symbol        string   `json:"symbol"`
	Name          string   `json:"name"`
	BaseAmount    float64  `json:"base_amount"`
	StartFrom     string   `json:"start_from"`
	MonthlyAmount float64  `json:"monthly_amount"`
	TotalValue    float64  `json:"-"`
}

type RecurringFundResponse struct {
	Symbol        string   `json:"symbol"`
	Name          string   `json:"name"`
	Category      Category `json:"category"`
	SubCategory   Category `json:"sub_category"`
	BaseAmount    float64  `json:"base_amount"`
	StartFrom     string   `json:"start_from"`
	MonthlyAmount float64  `json:"monthly_amount"`
	TotalValue    float64  `json:"total_value"`
}

type RecurringFundUpdate struct {
	Symbol        string  `json:"symbol"`
	BaseAmount    float64 `json:"base_amount"`
	StartFrom     string  `json:"start_from"`
	MonthlyAmount float64 `json:"monthly_amount"`
}

func (f RecurringFund) ToResponse() RecurringFundResponse {
	return RecurringFundResponse{
		Symbol:        f.Symbol,
		Name:          f.Name,
		Category:      f.Category,
		SubCategory:   f.SubCategory,
		BaseAmount:    f.BaseAmount,
		StartFrom:     f.StartFrom,
		MonthlyAmount: f.MonthlyAmount,
		TotalValue:    f.TotalValue,
	}
}

func loadRecurringFunds(fileName string) ([]RecurringFund, error) {
	var funds struct {
		RecurringFunds []RecurringFund `json:"recurring_funds"`
	}

	data, err := os.ReadFile(fileName)
	if err != nil {
		return []RecurringFund{}, err
	}
	if err := json.Unmarshal(data, &funds); err != nil {
		return []RecurringFund{}, err
	}

	return funds.RecurringFunds, nil
}

func GetRecurringFunds() ([]RecurringFund, error) {
	funds, err := loadRecurringFunds(recurringFundsFile)
	if err != nil {
		return []RecurringFund{}, err
	}

	for i, f := range funds {
		months, err := monthsElapsed(f.StartFrom)
		if err != nil {
			return []RecurringFund{}, err
		}
		funds[i].TotalValue = f.BaseAmount + (f.MonthlyAmount * float64(months))
	}
	return funds, nil
}

func validateStartFrom(s string) error {
	parts := strings.Split(s, "/")
	if len(parts) != 2 {
		return fmt.Errorf("start_from must be in MM/YYYY format, got %q", s)
	}
	month, err := strconv.Atoi(parts[0])
	if err != nil || month < 1 || month > 12 {
		return fmt.Errorf("start_from month must be 1-12, got %q", parts[0])
	}
	year, err := strconv.Atoi(parts[1])
	if err != nil || year < 1900 || year > 9999 {
		return fmt.Errorf("start_from year is invalid, got %q", parts[1])
	}
	return nil
}

func UpdateRecurringFund(update RecurringFundUpdate) (RecurringFund, error) {
	if update.BaseAmount < 0 || update.MonthlyAmount < 0 {
		return RecurringFund{}, fmt.Errorf("amounts must be non-negative")
	}
	if err := validateStartFrom(update.StartFrom); err != nil {
		return RecurringFund{}, err
	}

	funds, err := loadRecurringFunds(recurringFundsFile)
	if err != nil {
		return RecurringFund{}, err
	}

	idx := -1
	for i := range funds {
		if funds[i].Symbol == update.Symbol {
			idx = i
			break
		}
	}
	if idx == -1 {
		return RecurringFund{}, fmt.Errorf("recurring fund with symbol %q not found", update.Symbol)
	}

	funds[idx].BaseAmount = update.BaseAmount
	funds[idx].StartFrom = update.StartFrom
	funds[idx].MonthlyAmount = update.MonthlyAmount

	if err := saveRecurringFunds(recurringFundsFile, funds); err != nil {
		return RecurringFund{}, err
	}

	months, err := monthsElapsed(funds[idx].StartFrom)
	if err != nil {
		return RecurringFund{}, err
	}
	funds[idx].TotalValue = funds[idx].BaseAmount + (funds[idx].MonthlyAmount * float64(months))
	return funds[idx], nil
}

func saveRecurringFunds(fileName string, funds []RecurringFund) error {
	payload := struct {
		RecurringFunds []RecurringFund `json:"recurring_funds"`
	}{RecurringFunds: funds}

	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')

	dir := filepath.Dir(fileName)
	tmp, err := os.CreateTemp(dir, ".recurring_funds-*.json")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, fileName)
}

func (f RecurringFund) CategoryName() Category {
	return f.Category
}

func (f RecurringFund) SubCategoryName() Category {
	return f.SubCategory
}

func (f RecurringFund) Value() float64 {
	return f.TotalValue
}

func monthsElapsed(since string) (int, error) {
	currentMonth := int(time.Now().Month())
	currentYear := int(time.Now().Year())

	sinceMonth, err := strconv.Atoi(strings.Split(since, "/")[0])
	if err != nil {
		return 0, err
	}

	sinceYear, err := strconv.Atoi(strings.Split(since, "/")[1])
	if err != nil {
		return 0, err
	}

	return ((currentYear-sinceYear)*12 + currentMonth) - sinceMonth, nil
}
