// Command allocation-alert checks the running portfolio-scan server's current
// allocation against the saved targets and posts a Slack alert when any asset
// class or equity sub-category drifts beyond the allowed skew.
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"strings"
	"time"
)

type allocation struct {
	Name   string  `json:"name"`
	Amount float64 `json:"amount"`
}

type targetAllocations struct {
	Asset struct {
		Equity float64 `json:"equity"`
		Debt   float64 `json:"debt"`
		Gold   float64 `json:"gold"`
	} `json:"asset"`
	Equity struct {
		LargeCap float64 `json:"large_cap"`
		MidCap   float64 `json:"mid_cap"`
		SmallCap float64 `json:"small_cap"`
	} `json:"equity"`
}

type line struct {
	name    string
	current float64
	target  float64
}

func getBody(url string) ([]byte, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s: status %d: %s", url, resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return body, nil
}

func fetchTargets(baseURL string) (targetAllocations, error) {
	var targets targetAllocations
	body, err := getBody(baseURL + "/api/target_allocations")
	if err != nil {
		return targets, err
	}
	if err := json.Unmarshal(body, &targets); err != nil {
		return targets, err
	}
	return targets, nil
}

func fetchAllocations(baseURL, path string) ([]allocation, error) {
	body, err := getBody(baseURL + path)
	if err != nil {
		return nil, err
	}
	var holdings []allocation
	if err := json.Unmarshal(body, &holdings); err != nil {
		return nil, err
	}
	return holdings, nil
}

func computeGroup(holdings []allocation, targets map[string]float64) []line {
	total := 0.0
	for _, h := range holdings {
		total += h.Amount
	}

	var lines []line
	if total == 0 {
		return lines
	}
	for _, h := range holdings {
		target, ok := targets[h.Name]
		if !ok {
			continue
		}
		lines = append(lines, line{name: h.Name, current: h.Amount / total * 100, target: target})
	}
	return lines
}

func anyBreach(lines []line, skew float64) bool {
	for _, l := range lines {
		if math.Abs(l.current-l.target) > skew {
			return true
		}
	}
	return false
}

func section(title string, lines []line) string {
	var b strings.Builder
	b.WriteString("*" + title + "*\n")
	for _, l := range lines {
		b.WriteString(fmt.Sprintf("%s: %.1f%%\n", l.name, l.current))
	}
	return b.String()
}

func formatMessage(assetLines, equityLines []line, assetSkew, equitySkew float64) string {
	var b strings.Builder
	b.WriteString(":rotating_light: *Portfolio allocation drift detected*\n\n")
	b.WriteString(section(fmt.Sprintf("Asset allocation (%.1f %%)", assetSkew), assetLines))
	b.WriteString("\n")
	b.WriteString(section(fmt.Sprintf("Equity allocation (%.1f %%)", equitySkew), equityLines))
	return b.String()
}

func postToSlack(channel, text string) error {
	token := os.Getenv("SLACK_TOKEN")
	if token == "" {
		return fmt.Errorf("SLACK_TOKEN environment variable is not set")
	}
	if channel == "" {
		return fmt.Errorf("no Slack channel provided (use -channel or set SLACK_CHANNEL)")
	}

	payload, err := json.Marshal(map[string]string{"channel": channel, "text": text})
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, "https://slack.com/api/chat.postMessage", bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var result struct {
		OK    bool   `json:"ok"`
		Error string `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}
	if !result.OK {
		return fmt.Errorf("slack rejected the message: %s", result.Error)
	}
	return nil
}

func main() {
	baseURL := flag.String("url", "http://localhost:9876", "base URL of the running portfolio-scan server")
	assetSkew := flag.Float64("asset-skew", 5, "max allowed asset-allocation drift in percentage points")
	equitySkew := flag.Float64("equity-skew", 5, "max allowed equity sub-allocation drift in percentage points")
	channel := flag.String("channel", os.Getenv("SLACK_CHANNEL"), "Slack channel ID or name to alert (defaults to $SLACK_CHANNEL)")
	dryRun := flag.Bool("dry-run", false, "print the alert but do not post to Slack")
	flag.Parse()

	targets, err := fetchTargets(*baseURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: failed to fetch target allocations: %v\n", err)
		os.Exit(1)
	}

	assetHoldings, err := fetchAllocations(*baseURL, "/api/portfolio")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: failed to fetch portfolio: %v\n", err)
		os.Exit(1)
	}

	equityHoldings, err := fetchAllocations(*baseURL, "/api/portfolio/equity/categories")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: failed to fetch equity categories: %v\n", err)
		os.Exit(1)
	}

	assetTargets := map[string]float64{
		"equity": targets.Asset.Equity,
		"debt":   targets.Asset.Debt,
		"gold":   targets.Asset.Gold,
	}
	equityTargets := map[string]float64{
		"large cap": targets.Equity.LargeCap,
		"mid cap":   targets.Equity.MidCap,
		"small cap": targets.Equity.SmallCap,
	}

	assetLines := computeGroup(assetHoldings, assetTargets)
	equityLines := computeGroup(equityHoldings, equityTargets)

	if !anyBreach(assetLines, *assetSkew) && !anyBreach(equityLines, *equitySkew) {
		fmt.Printf("Allocation within tolerance (asset ±%.1fpp, equity ±%.1fpp); no alert.\n", *assetSkew, *equitySkew)
		return
	}

	message := formatMessage(assetLines, equityLines, *assetSkew, *equitySkew)
	fmt.Println(message)

	if *dryRun {
		fmt.Println("(dry-run: not posting to Slack)")
		return
	}

	if err := postToSlack(*channel, message); err != nil {
		fmt.Fprintf(os.Stderr, "error: failed to post to Slack: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Alert posted to Slack.")
}
