package funds

type Holding interface {
	CategoryName() Category
	SubCategoryName() Category
	Value() float64
}

type Category string

const (
	Equity            Category = "equity"
	Debt              Category = "debt"
	Gold              Category = "gold"
	LargeCapEquity    Category = "large cap"
	MidCapEquity      Category = "mid cap"
	LargeMidCapEquity Category = "large mid cap"
	SmallCapEquity    Category = "small cap"
)

func GetPortfolio() ([]Holding, error) {
	var holdings []Holding

	recurringFunds, err := GetRecurringFunds()
	if err != nil {
		return []Holding{}, err
	}

	for _, f := range recurringFunds {
		holdings = append(holdings, f)
	}

	mutualFunds, err := GetMutualFunds()
	if err != nil {
		return []Holding{}, err
	}

	for _, f := range mutualFunds {
		holdings = append(holdings, f)
	}

	return holdings, nil
}
