package price

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/shopspring/decimal"

	"github.com/alv67/peculium/internal/model"
)

const sparkChunkSize = 50

type sparkQuote struct {
	Timestamp []int64    `json:"timestamp"`
	Close     []*float64 `json:"close"`
	Symbol    string     `json:"symbol"`
}

// fetchSpark performs one Yahoo spark request for the given symbols and returns
// the per-symbol close series.
func (f *YahooFetcher) fetchSpark(ctx context.Context, symbols []string) (map[string]sparkQuote, error) {
	url := "https://query1.finance.yahoo.com/v8/finance/spark?symbols=" +
		strings.Join(symbols, ",") + "&range=5d&interval=1d&indicators=close&includeTimestamp=true"

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)")

	resp, err := f.doRequest(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("yahoo request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		io.Copy(io.Discard, resp.Body)
		return nil, &fetchStatusError{status: resp.StatusCode, message: fmt.Sprintf("yahoo returned status %d", resp.StatusCode)}
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	var parsed map[string]sparkQuote
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("unmarshal: %w (body: %s)", err, string(body))
	}
	return parsed, nil
}

// fetchQuotesBatch fetches the latest close for every asset with Yahoo spark
// requests (chunked), falling back to a per-symbol chart call for symbols
// missing from the response. Returns the tickers that were updated and the
// per-symbol issues.
func (f *YahooFetcher) fetchQuotesBatch(ctx context.Context, assets []*model.Asset) ([]string, []FetchIssue) {
	refreshed := make([]string, 0)
	issues := make([]FetchIssue, 0)

	for start := 0; start < len(assets); start += sparkChunkSize {
		end := start + sparkChunkSize
		if end > len(assets) {
			end = len(assets)
		}
		chunk := assets[start:end]

		resp, err := f.fetchSpark(ctx, tickers(chunk))
		if err != nil {
			for _, a := range chunk {
				issues = append(issues, FetchIssue{Symbol: a.Ticker, RequestType: "spark", AssetID: &a.ID, Code: issueCode(err), Message: fmt.Sprintf("spark %s: %v", a.Ticker, err)})
			}
			continue
		}

		for _, a := range chunk {
			sq, ok := resp[a.Ticker]
			if !ok {
				// Missing from the batch response: fall back to a single call.
				if err := f.fetchQuote(ctx, a); err != nil {
					issues = append(issues, FetchIssue{Symbol: a.Ticker, RequestType: "chart", AssetID: &a.ID, Code: issueCode(err), Message: fmt.Sprintf("chart %s: %v", a.Ticker, err)})
				} else {
					refreshed = append(refreshed, a.Ticker)
				}
				continue
			}

			idx := -1
			for i := len(sq.Close) - 1; i >= 0; i-- {
				if sq.Close[i] != nil {
					idx = i
					break
				}
			}
			if idx == -1 {
				issues = append(issues, FetchIssue{Symbol: a.Ticker, RequestType: "spark", AssetID: &a.ID, Code: "error", Message: fmt.Sprintf("spark %s: no close data", a.Ticker)})
				continue
			}
			if idx >= len(sq.Timestamp) {
				issues = append(issues, FetchIssue{Symbol: a.Ticker, RequestType: "spark", AssetID: &a.ID, Code: "error", Message: fmt.Sprintf("spark %s: no close timestamp", a.Ticker)})
				continue
			}

			closePrice := *sq.Close[idx]
			if _, err := f.repos.Price.Create(ctx, &model.Price{
				AssetID: a.ID,
				Date:    priceDate(sq.Timestamp[idx]),
				Open:    decimal.NewFromFloat(closePrice),
				High:    decimal.NewFromFloat(closePrice),
				Low:     decimal.NewFromFloat(closePrice),
				Close:   decimal.NewFromFloat(closePrice),
				Volume:  0,
				Source:  "yahoo",
			}); err != nil {
				issues = append(issues, FetchIssue{Symbol: a.Ticker, RequestType: "spark", AssetID: &a.ID, Code: "error", Message: fmt.Sprintf("spark %s: save price: %v", a.Ticker, err)})
				continue
			}
			refreshed = append(refreshed, a.Ticker)
		}
	}
	return refreshed, issues
}
