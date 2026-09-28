package nav

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

type NAV struct {
	UpdatedAt time.Time
	Funds     map[string]float64
}

var ErrNAVFileNotFound = errors.New("nav.json not found")

func load() (NAV, error) {
	var n NAV
	data, err := os.ReadFile("nav.json")
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return NAV{}, ErrNAVFileNotFound
		}
		return NAV{}, err
	}

	if err := json.Unmarshal(data, &n); err != nil {
		return NAV{}, err
	}

	return n, nil
}

func store(nav *NAV) error {
	data, err := json.MarshalIndent(nav, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile("nav.json", data, 0644)
}

func parseNAV(lines []string, sym string) (float64, error) {
	for _, line := range lines {
		if strings.Contains(line, sym) {
			// nav is 7 field, semi-colon separated
			nav, err := strconv.ParseFloat(strings.Split(line, ";")[6], 64)
			if err != nil {
				return 0, fmt.Errorf("could not parse float: %v", err)
			}

			return nav, nil
		}
	}

	return 0, fmt.Errorf("fund not found")
}

func fetchLatest(symbols []string) error {
	resp, err := http.Get("https://portal.amfiindia.com/spages/NAVAll.txt")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	lines := strings.Split(string(body), "\n")

	n := NAV{Funds: map[string]float64{}}
	for _, s := range symbols {
		price, err := parseNAV(lines, s)
		if err != nil {
			return fmt.Errorf("err, failed to fetch nav for %s:  %v", s, err)
		}

		n.Funds[s] = price
	}

	n.UpdatedAt = time.Now()

	return store(&n)
}

func UpdateNAVs(symbols []string) error {
	nav, err := load()
	if err == ErrNAVFileNotFound {
		log.Printf("nav.json doesn't exists, fetching...")
		return fetchLatest(symbols)
	} else if err != nil {
		return err
	}

	if time.Now().Sub(nav.UpdatedAt) > time.Hour*1 {
		log.Printf("stale nav, refetching...")
		return fetchLatest(symbols)
	}

	for _, s := range symbols {
		if _, ok := nav.Funds[s]; !ok {
			log.Printf("found new fund, refetching...")
			return fetchLatest(symbols)
		}
	}

	return nil
}

func Get(symbol string) (float64, error) {
	nav, err := load()
	if err != nil {
		return 0.0, err
	}

	ltp, ok := nav.Funds[symbol]
	if !ok {
		return 0.0, fmt.Errorf("sym %s not found", symbol)
	}
	return ltp, nil
}
