package funds

import (
	"encoding/json"
	"os"
	"strconv"
	"strings"
	"time"
)

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
	Name       string
	TotalValue float64
}

func (f RecurringFund) ToResponse() RecurringFundResponse {
	return RecurringFundResponse{
		Name:       f.Name,
		TotalValue: f.TotalValue,
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
	funds, err := loadRecurringFunds("recurring_funds.json")
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
