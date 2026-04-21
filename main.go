package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"net/url"
	"portfolio-scan/funds"
	"strings"
)

type Allocation struct {
	Name   string  `json:"name"`
	Amount float64 `json:"amount"`
}

//go:embed ui/dist/*
var frontendFS embed.FS

func handleHTTPError(w http.ResponseWriter, err error) {
	fmt.Fprintf(w, "error: %v", err)
	log.Printf("error: %v", err)
	return
}

func handlePortfolio(w http.ResponseWriter, req *http.Request) {
	holdings, err := funds.GetPortfolio()
	if err != nil {
		handleHTTPError(w, err)
		return
	}

	allocations := []Allocation{
		Allocation{Name: string(funds.Equity), Amount: 0.0},
		Allocation{Name: string(funds.Debt), Amount: 0.0},
		Allocation{Name: string(funds.Gold), Amount: 0.0},
	}

	for _, h := range holdings {
		if h.CategoryName() == funds.Equity {
			allocations[0].Amount += h.Value()
		} else if h.CategoryName() == funds.Debt {
			allocations[1].Amount += h.Value()
		} else if h.CategoryName() == funds.Gold {
			allocations[2].Amount += h.Value()
		}
	}

	data, err := json.Marshal(allocations)
	if err != nil {
		handleHTTPError(w, err)
		return
	}

	w.Header().Set("Access-Control-Allow-Origin", "*")
	fmt.Fprint(w, string(data))

	log.Printf("200 ok: %v", string(data))
}

func handleEquityCategories(w http.ResponseWriter, req *http.Request) {
	holdings, err := funds.GetPortfolio()
	if err != nil {
		handleHTTPError(w, err)
		return
	}

	allocations := []Allocation{
		{Name: string(funds.LargeCapEquity), Amount: 0.0},
		{Name: string(funds.MidCapEquity), Amount: 0.0},
		{Name: string(funds.SmallCapEquity), Amount: 0.0},
	}

	for _, h := range holdings {
		if _, ok := h.(funds.MutualFund); !ok {
			continue
		}

		if h.CategoryName() != funds.Equity {
			continue
		}

		amount := h.Value()
		switch h.SubCategoryName() {
		case funds.LargeCapEquity:
			allocations[0].Amount += amount
		case funds.MidCapEquity:
			allocations[1].Amount += amount
		case funds.SmallCapEquity:
			allocations[2].Amount += amount
		case funds.LargeMidCapEquity:
			allocations[0].Amount += amount / 2
			allocations[1].Amount += amount / 2
		}
	}

	data, err := json.Marshal(allocations)
	if err != nil {
		handleHTTPError(w, err)
		return
	}

	w.Header().Set("Access-Control-Allow-Origin", "*")
	fmt.Fprint(w, string(data))

	log.Printf("200 ok: %v", string(data))
}

func main() {
	distFS, err := fs.Sub(frontendFS, "ui/dist")
	if err != nil {
		log.Fatal(err)
	}

	frontendHandler := http.FileServer(http.FS(distFS))

	http.HandleFunc("/api/portfolio", handlePortfolio)
	http.HandleFunc("/api/portfolio/equity/categories", handleEquityCategories)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api") {
			return
		}

		// serve index.html for any path that does not exists
		// and let vue handle the routing
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path == "" {
			path = "index.html"
		}

		_, err := distFS.Open(path)
		if err != nil {
			r2 := new(http.Request)
			*r2 = *r
			r2.URL = new(url.URL)
			*r2.URL = *r.URL
			r2.URL.Path = "/"
			frontendHandler.ServeHTTP(w, r2)
			return
		}

		frontendHandler.ServeHTTP(w, r)
	})

	log.Printf("listening on: localhost:9876")
	log.Fatal(http.ListenAndServe(":9876", nil))
}
