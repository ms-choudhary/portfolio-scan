package funds

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
)

const targetAllocationsFile = "target_allocations.json"

type AssetTargets struct {
	Equity float64 `json:"equity"`
	Debt   float64 `json:"debt"`
	Gold   float64 `json:"gold"`
}

type EquityTargets struct {
	LargeCap float64 `json:"large_cap"`
	MidCap   float64 `json:"mid_cap"`
	SmallCap float64 `json:"small_cap"`
}

type TargetAllocations struct {
	Asset  AssetTargets  `json:"asset"`
	Equity EquityTargets `json:"equity"`
}

func defaultTargetAllocations() TargetAllocations {
	return TargetAllocations{
		Asset:  AssetTargets{Equity: 70, Debt: 20, Gold: 10},
		Equity: EquityTargets{LargeCap: 50, MidCap: 30, SmallCap: 20},
	}
}

func GetTargetAllocations() (TargetAllocations, error) {
	data, err := os.ReadFile(targetAllocationsFile)
	if errors.Is(err, os.ErrNotExist) {
		return defaultTargetAllocations(), nil
	}
	if err != nil {
		return TargetAllocations{}, err
	}

	var targets TargetAllocations
	if err := json.Unmarshal(data, &targets); err != nil {
		return TargetAllocations{}, err
	}
	return targets, nil
}

func validateTargets(targets TargetAllocations) error {
	groups := []struct {
		name   string
		values []float64
	}{
		{"asset", []float64{targets.Asset.Equity, targets.Asset.Debt, targets.Asset.Gold}},
		{"equity", []float64{targets.Equity.LargeCap, targets.Equity.MidCap, targets.Equity.SmallCap}},
	}

	for _, g := range groups {
		sum := 0.0
		for _, v := range g.values {
			if v < 0 {
				return fmt.Errorf("%s targets must be non-negative", g.name)
			}
			sum += v
		}
		if math.Abs(sum-100) > 0.01 {
			return fmt.Errorf("%s targets must sum to 100, got %g", g.name, sum)
		}
	}
	return nil
}

func SaveTargetAllocations(targets TargetAllocations) (TargetAllocations, error) {
	if err := validateTargets(targets); err != nil {
		return TargetAllocations{}, err
	}
	if err := saveTargetAllocations(targetAllocationsFile, targets); err != nil {
		return TargetAllocations{}, err
	}
	return targets, nil
}

func saveTargetAllocations(fileName string, targets TargetAllocations) error {
	data, err := json.MarshalIndent(targets, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')

	dir := filepath.Dir(fileName)
	tmp, err := os.CreateTemp(dir, ".target_allocations-*.json")
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
