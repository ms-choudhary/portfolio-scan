package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	kiteconnect "github.com/zerodha/gokiteconnect/v4"
)

const (
	Equity = "equity"
	Debt   = "debt"
	Gold   = "gold"
)

type Fund struct {
	Class    string  `json:"class"`
	Name     string  `json:"name"`
	Symbol   string  `json:"symbol"`
	Quantity float64 `json:"quantity"`
	Price    float64 `json:"price"`
}

type Portfolio struct {
	Funds []Fund `json:"funds"`
}

type Allocation struct {
	Name   string  `json:"name"`
	Amount float64 `json:"amount"`
}

//go:embed ui/dist/*
var frontendFS embed.FS

func fetchFunds(account, apikey, apisecret string) ([]Fund, error) {
	kc := kiteconnect.New(apikey)

	fmt.Printf("Login to %s account and paste the requestToken:\n", account)
	fmt.Println(kc.GetLoginURL())
	fmt.Printf("requestToken: ")

	var requestToken string
	fmt.Scanf("%s\n", &requestToken)

	data, err := kc.GenerateSession(requestToken, apisecret)
	if err != nil {
		return []Fund{}, fmt.Errorf("could not generate kite session: %v", err)
	}

	kc.SetAccessToken(data.AccessToken)

	holdings, err := kc.GetMFHoldings()
	if err != nil {
		return []Fund{}, fmt.Errorf("cannot get holdings: %v", err)
	}

	funds := []Fund{}
	for _, h := range holdings {
		if account == Equity {
			funds = append(funds, Fund{Class: Equity, Name: h.Fund, Quantity: h.Quantity, Symbol: h.Tradingsymbol})
		} else if account == Debt && strings.HasPrefix(h.Fund, "AXIS LIQUID FUND") {
			funds = append(funds, Fund{Class: Debt, Name: h.Fund, Quantity: h.Quantity, Symbol: h.Tradingsymbol})
		} else if account == Debt && strings.HasPrefix(h.Fund, "UTI GOLD ETF FUND") {
			funds = append(funds, Fund{Class: Gold, Name: h.Fund, Quantity: h.Quantity, Symbol: h.Tradingsymbol})
		}
	}

	return funds, nil
}

func parseNAV(lines []string, sym string) (float64, error) {
	for _, line := range lines {
		if strings.Contains(line, sym) {
			// nav is 5 field, semi-colon separated
			nav, err := strconv.ParseFloat(strings.Split(line, ";")[4], 64)
			if err != nil {
				return 0, fmt.Errorf("could not parse float: %v", err)
			}

			return nav, nil
		}
	}

	return 0, fmt.Errorf("fund not found")
}

func (p *Portfolio) updateCurrentPrice() error {
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

	for i, f := range p.Funds {
		nav, err := parseNAV(lines, f.Symbol)
		if err != nil {
			return fmt.Errorf("err: %v", err)
		}

		p.Funds[i].Price = nav
	}

	return nil
}

func handleHTTPError(w http.ResponseWriter, err error) {
	fmt.Fprintf(w, "error: %v", err)
	log.Printf("error: %v", err)
	return
}

func (p *Portfolio) Handler(w http.ResponseWriter, req *http.Request) {
	if err := p.updateCurrentPrice(); err != nil {
		handleHTTPError(w, err)
		return
	}

	allocations := []Allocation{
		Allocation{Name: Equity, Amount: 0.0},
		Allocation{Name: Debt, Amount: 0.0},
		Allocation{Name: Gold, Amount: 0.0},
	}

	for _, f := range p.Funds {
		if f.Class == Equity {
			allocations[0].Amount += f.Quantity * f.Price
		} else if f.Class == Debt {
			allocations[1].Amount += f.Quantity * f.Price
		} else if f.Class == Gold {
			allocations[2].Amount += f.Quantity * f.Price
		}
	}

	data, err := json.Marshal(allocations)
	if err != nil {
		handleHTTPError(w, err)
		return
	}

	w.Header().Set("Access-Control-Allow-Origin", "*")
	fmt.Fprintf(w, string(data))

	log.Printf("200 ok: %v", string(data))
}

func fetchPortfolio(eqAPIKey, eqAPISecret, debtAPIKey, debtAPISecret string) (*Portfolio, error) {
	if eqAPIKey == "" || eqAPISecret == "" {
		return &Portfolio{}, fmt.Errorf("please set EQ_KITE_API_KEY and EQ_KITE_API_SECRET")
	}

	if debtAPIKey == "" || debtAPISecret == "" {
		return &Portfolio{}, fmt.Errorf("please set DEBT_KITE_API_KEY and DEBT_KITE_API_SECRET")
	}

	p := Portfolio{Funds: []Fund{}}
	funds, err := fetchFunds(Equity, eqAPIKey, eqAPISecret)
	if err != nil {
		return &Portfolio{}, fmt.Errorf("could not login equity account: %v", err)
	}

	p.Funds = append(p.Funds, funds...)

	funds, err = fetchFunds(Debt, debtAPIKey, debtAPISecret)
	if err != nil {
		return &Portfolio{}, fmt.Errorf("could not login debt account: %v", err)
	}

	p.Funds = append(p.Funds, funds...)
	return &p, nil
}

func main() {
	var portfolio *Portfolio
	data, err := os.ReadFile(".portfolio")
	if os.IsNotExist(err) {
		portfolio, err = fetchPortfolio(os.Getenv("EQ_KITE_API_KEY"), os.Getenv("EQ_KITE_API_SECRET"), os.Getenv("DEBT_KITE_API_KEY"), os.Getenv("DEBT_KITE_API_SECRET"))
		if err != nil {
			log.Fatal(err)
		}

		data, err = json.Marshal(*portfolio)
		if err != nil {
			log.Fatalf("error marshalling portfolio: %v", err)
		}

		if err := os.WriteFile(".portfolio", data, 0644); err != nil {
			log.Fatalf("error writing portfolio: %v", err)
		}
	} else if err != nil {
		log.Fatalf("error reading portfolio: %v", err)
	} else {
		var p Portfolio
		if err := json.Unmarshal(data, &p); err != nil {
			log.Fatalf("failed to unmarhsal portfolio: %v", err)
		}

		portfolio = &p

		log.Printf("loaded portfolio")
	}

	distFS, err := fs.Sub(frontendFS, "ui/dist")
	if err != nil {
		log.Fatal(err)
	}

	frontendHandler := http.FileServer(http.FS(distFS))

	http.HandleFunc("/api/portfolio", portfolio.Handler)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/portfolio" {
			return
		}
		frontendHandler.ServeHTTP(w, r)
	})

	log.Printf("listening on: localhost:9876")
	log.Fatal(http.ListenAndServe(":9876", nil))
}
