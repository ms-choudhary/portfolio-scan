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
	"time"

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
	Month    string  `json:"month,omitempty"`
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

func fetchFunds(account, apikey, requestToken, apisecret string) ([]Fund, error) {
	kc := kiteconnect.New(apikey)

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
		} else if account == Debt && strings.Contains(h.Fund, "LIQUID FUND") {
			funds = append(funds, Fund{Class: Debt, Name: h.Fund, Quantity: h.Quantity, Symbol: h.Tradingsymbol})
		} else if account == Debt && strings.Contains(h.Fund, "GOLD ETF FUND") {
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
		if f.Symbol == "NA" {
			if strings.HasSuffix(f.Name, "monthly") {
				month, err := monthsElapsed(f.Month)
				if err != nil {
					return err
				}

				p.Funds[i].Quantity = float64(month)
			}
			continue
		}

		nav, err := parseNAV(lines, f.Symbol)
		if err != nil {
			return fmt.Errorf("err, failed to fetch value: %v for %s", err, f.Name)
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

func loadFunds(fileName string) ([]Fund, error) {
	var p Portfolio
	data, err := os.ReadFile(fileName)
	if err != nil {
		return []Fund{}, err
	}
	if err := json.Unmarshal(data, &p); err != nil {
		return []Fund{}, err
	}

	return p.Funds, nil
}

func loadPortfolios() (Portfolio, error) {
	p := Portfolio{Funds: []Fund{}}

	dir, err := os.ReadDir(".")
	if err != nil {
		return Portfolio{}, err
	}

	for _, entry := range dir {
		if !entry.IsDir() && strings.HasPrefix(entry.Name(), ".portfolio") && !strings.Contains(entry.Name(), "example") {
			funds, err := loadFunds(entry.Name())
			if err != nil {
				return Portfolio{}, err
			}

			p.Funds = append(p.Funds, funds...)
		}
	}

	return p, nil
}

func handlePortfolio(w http.ResponseWriter, req *http.Request) {
	p, err := loadPortfolios()
	if err != nil {
		handleHTTPError(w, err)
		return
	}

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

func getEnvOrFail(envvar string) string {
	val := os.Getenv(envvar)
	if val == "" {
		log.Fatalf("env %s not found", envvar)
	}

	return val
}

func handleLogin(w http.ResponseWriter, req *http.Request) {
	if strings.Contains(req.URL.Path, "debt") {
		kc := kiteconnect.New(getEnvOrFail("DEBT_KITE_API_KEY"))
		http.Redirect(w, req, kc.GetLoginURL(), http.StatusMovedPermanently)
		return
	} else if strings.Contains(req.URL.Path, "equity") {
		kc := kiteconnect.New(getEnvOrFail("EQ_KITE_API_KEY"))
		http.Redirect(w, req, kc.GetLoginURL(), http.StatusMovedPermanently)
		return
	}

	fmt.Fprintf(w, "<html><h1>Error: invalid path, expected (/login/debt or /login/equity)</h1></html>")
}

func savePortfolio(account, requestToken string) error {
	var funds []Fund
	var err error
	if account == Equity {
		funds, err = fetchFunds(Equity, getEnvOrFail("EQ_KITE_API_KEY"), requestToken, getEnvOrFail("EQ_KITE_API_SECRET"))
		if err != nil {
			return err
		}
	} else if account == Debt {
		funds, err = fetchFunds(Debt, getEnvOrFail("DEBT_KITE_API_KEY"), requestToken, getEnvOrFail("DEBT_KITE_API_SECRET"))
		if err != nil {
			return err
		}
	} else {
		return fmt.Errorf("invalid account: %s", account)
	}

	data, err := json.Marshal(&Portfolio{Funds: funds})
	if err != nil {
		return err
	}

	fileName := fmt.Sprintf(".portfolio_%s", strings.ToLower(account))
	return os.WriteFile(fileName, data, 0644)
}

func handleAuthRedirect(w http.ResponseWriter, req *http.Request) {
	if strings.Contains(req.URL.Path, "debt") {
		if err := savePortfolio(Debt, req.URL.Query().Get("request_token")); err != nil {
			log.Printf("err: %v", err)
			fmt.Fprintf(w, "<html><h1>%s</h1></html>", err.Error())
			return
		}
	} else if strings.Contains(req.URL.Path, "equity") {
		if err := savePortfolio(Equity, req.URL.Query().Get("request_token")); err != nil {
			log.Printf("err: %v", err)
			fmt.Fprintf(w, "<html><h1>%s</h1></html>", err.Error())
			return
		}
	} else {
		fmt.Fprintf(w, "<html><h1>Error: invalid auth redirect url, %s expected (/auth/debt or /auth/equity)</h1></html>", req.URL.Path)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, "<html><h1>Success!</h1></html>")
	log.Print("logged in successfully")
}

func main() {
	distFS, err := fs.Sub(frontendFS, "ui/dist")
	if err != nil {
		log.Fatal(err)
	}

	frontendHandler := http.FileServer(http.FS(distFS))

	http.HandleFunc("/api/portfolio", handlePortfolio)
	http.HandleFunc("/login/", handleLogin)
	http.HandleFunc("/auth/", handleAuthRedirect)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/portfolio" {
			return
		}
		if strings.HasPrefix(r.URL.Path, "/login") || strings.HasPrefix(r.URL.Path, "/return") {
			return
		}
		frontendHandler.ServeHTTP(w, r)
	})

	log.Printf("listening on: localhost:9876")
	log.Fatal(http.ListenAndServe(":9876", nil))
}
