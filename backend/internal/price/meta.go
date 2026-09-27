package price

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/shopspring/decimal"

	"github.com/alv67/peculium/internal/geo"
	"github.com/alv67/peculium/internal/model"
)

type AssetMeta struct {
	Ticker     string `json:"ticker"`
	Name       string `json:"name"`
	Type       string `json:"type"`
	Currency   string `json:"currency"`
	Exchange   string `json:"exchange"`
	Country    string `json:"country,omitempty"`
	AssetClass string `json:"asset_class,omitempty"`
}

// FetchMeta resolves metadata (currency, name, type, country) for a ticker via
// the Yahoo chart endpoint. Country is only derived for equities, since ETFs
// usually span multiple countries.
func (f *YahooFetcher) FetchMeta(ctx context.Context, ticker string) (*AssetMeta, error) {
	chart, err := f.fetchChart(ctx, ticker)
	if err != nil {
		return nil, err
	}

	meta := chart.Meta
	name := meta.LongName
	if name == "" {
		name = meta.ShortName
	}

	assetType := mapType(meta.InstrumentType)
	assetMeta := &AssetMeta{
		Ticker:     meta.Symbol,
		Name:       name,
		Type:       assetType,
		Currency:   meta.Currency,
		Exchange:   meta.ExchangeName,
		AssetClass: geo.ClassifyAssetClass(assetType, name, ""),
	}

	// Country only makes sense for a single equity; ETFs/other types span
	// multiple countries. For stocks the exposure country comes from the
	// issuer's domicile (assetProfile.country) so cross-listings keep the
	// issuer home market; the exchange-based derivation is only a fallback.
	if assetType == "stock" {
		if _, _, country, err := f.FetchAssetProfile(ctx, ticker); err == nil && country != "" {
			assetMeta.Country = country
		} else {
			assetMeta.Country = exchangeCountry(meta.ExchangeName)
		}
	}

	// Best-effort refinement of the class for funds from the Morningstar
	// category (non-blocking: chart meta already gives a sensible default).
	if geo.FundClassifiable(assetType) {
		if fund, err := f.FetchFundCategory(ctx, ticker); err == nil && fund != "" {
			assetMeta.AssetClass = geo.ClassifyAssetClass(assetType, name, fund)
		}
	}

	return assetMeta, nil
}

// exchangeCountry maps a Yahoo exchange code to an ISO country code for equities.
func exchangeCountry(exchange string) string {
	return exchangeCountries[strings.ToUpper(exchange)]
}

// ExchangeCountry maps a Yahoo exchange code to the ISO country code of its
// home market. Unknown exchanges return "".
func ExchangeCountry(exchange string) string {
	return exchangeCountry(exchange)
}

type yahooAssetProfile struct {
	Sector   string `json:"sector"`
	Industry string `json:"industry"`
	Country  string `json:"country"`
}

type yahooTopHoldings struct {
	SectorWeightings []map[string]struct {
		Raw float64 `json:"raw"`
	} `json:"sectorWeightings"`
}

type yahooDefaultKeyStatistics struct {
	Category string `json:"category"`
}

type yahooFundProfile struct {
	CategoryName string `json:"categoryName"`
}

type yahooQuoteSummaryResponse struct {
	QuoteSummary struct {
		Result []struct {
			AssetProfile         yahooAssetProfile         `json:"assetProfile"`
			TopHoldings          yahooTopHoldings          `json:"topHoldings"`
			DefaultKeyStatistics yahooDefaultKeyStatistics `json:"defaultKeyStatistics"`
			FundProfile          yahooFundProfile          `json:"fundProfile"`
		} `json:"result"`
		Error interface{} `json:"error"`
	} `json:"quoteSummary"`
}

// crumbTTL is how long a fetched crumb is reused before being refreshed. The
// crumb is session-bound, so keeping it too long risks 401s on stale sessions.
const crumbTTL = 5 * time.Minute

// FetchAssetProfile returns the GICS sector, industry and issuer domicile
// country for a ticker from the Yahoo v10 quoteSummary assetProfile module.
// Empty strings are returned without error when the profile module is absent.
func (f *YahooFetcher) FetchAssetProfile(ctx context.Context, ticker string) (sector, industry, country string, err error) {
	result, err := f.fetchQuoteSummary(ctx, ticker, "assetProfile")
	if err != nil {
		return "", "", "", err
	}
	if result == nil {
		return "", "", "", nil
	}
	profile := result.QuoteSummary.Result[0].AssetProfile
	return profile.Sector, profile.Industry, profile.Country, nil
}

// FetchAssetProfileExtended returns sector/industry + issuer domicile country
// (from assetProfile) and the sector weightings (from topHoldings) for a
// ticker. sectorWeightings is empty when the module is absent (e.g. single
// stocks).
func (f *YahooFetcher) FetchAssetProfileExtended(ctx context.Context, ticker string) (sector, industry, country string, sectorWeightings []model.ExposureRow, err error) {
	result, err := f.fetchQuoteSummary(ctx, ticker, "assetProfile,topHoldings")
	if err != nil {
		return "", "", "", nil, err
	}
	if result == nil {
		return "", "", "", nil, nil
	}

	r := result.QuoteSummary.Result[0]
	sectorWeightings = make([]model.ExposureRow, 0, len(r.TopHoldings.SectorWeightings))
	for _, entry := range r.TopHoldings.SectorWeightings {
		for key, val := range entry {
			name := geo.SectorKeyToGICS(key)
			if name == "" {
				continue
			}
			pct := decimal.NewFromFloat(val.Raw).Mul(decimal.NewFromInt(100)).Round(4)
			if !pct.IsPositive() {
				continue
			}
			sectorWeightings = append(sectorWeightings, model.ExposureRow{Name: name, Weight: pct})
		}
	}

	return r.AssetProfile.Sector, r.AssetProfile.Industry, r.AssetProfile.Country, sectorWeightings, nil
}

// FetchFundCategory returns the Morningstar-style fund category for a ticker
// (defaultKeyStatistics.category, falling back to fundProfile.categoryName).
// It is used to guess the investment class of ETFs and mutual funds. An empty
// string is returned without error when the ticker is not a fund or the
// category is unknown. This intentionally reads only the fund-category
// modules, never the sector holdings.
func (f *YahooFetcher) FetchFundCategory(ctx context.Context, ticker string) (string, error) {
	result, err := f.fetchQuoteSummary(ctx, ticker, "defaultKeyStatistics,fundProfile")
	if err != nil {
		return "", err
	}
	if result == nil {
		return "", nil
	}
	r := result.QuoteSummary.Result[0]
	category := r.DefaultKeyStatistics.Category
	if category == "" {
		category = r.FundProfile.CategoryName
	}
	return strings.TrimSpace(category), nil
}

// fetchQuoteSummary fetches the requested quoteSummary modules for a ticker,
// handling the crumb handshake and retrying once with a fresh crumb when the
// cached one is rejected with 401. A nil result is returned when Yahoo returns
// no quoteSummary result.
func (f *YahooFetcher) fetchQuoteSummary(ctx context.Context, ticker, modules string) (*yahooQuoteSummaryResponse, error) {
	crumb, cookie, err := f.getCrumb(ctx)
	if err != nil {
		return nil, err
	}

	result, err := f.fetchQuoteSummaryRaw(ctx, ticker, modules, crumb, cookie)
	if err != nil && statusOf(err) == http.StatusUnauthorized {
		// Cached crumb was rejected: Yahoo rotated the session, re-fetch once.
		f.dropCrumb()
		crumb, cookie, err = f.getCrumb(ctx)
		if err != nil {
			return nil, err
		}
		result, err = f.fetchQuoteSummaryRaw(ctx, ticker, modules, crumb, cookie)
	}
	if err != nil {
		return nil, err
	}
	if len(result.QuoteSummary.Result) == 0 {
		return nil, nil
	}
	return result, nil
}

// fetchQuoteSummaryRaw performs a single quoteSummary request using the given
// crumb and session cookie header.
func (f *YahooFetcher) fetchQuoteSummaryRaw(ctx context.Context, ticker, modules, crumb, cookie string) (*yahooQuoteSummaryResponse, error) {
	url := fmt.Sprintf("https://query1.finance.yahoo.com/v10/finance/quoteSummary/%s?modules=%s&crumb=%s", ticker, modules, crumb)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)")
	req.Header.Set("Cookie", cookie)

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

	var result yahooQuoteSummaryResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("unmarshal: %w (body: %s)", err, string(body))
	}
	if result.QuoteSummary.Error != nil {
		return nil, fmt.Errorf("yahoo error: %v", result.QuoteSummary.Error)
	}
	return &result, nil
}

// getCrumb returns the cached crumb plus the session cookie header it was
// issued for, fetching a fresh one when the cache is empty or expired.
func (f *YahooFetcher) getCrumb(ctx context.Context) (crumb, cookie string, err error) {
	f.mu.Lock()
	if f.crumbValue != "" && time.Since(f.crumbAt) < crumbTTL {
		c, k := f.crumbValue, f.crumbCookie
		f.mu.Unlock()
		return c, k, nil
	}
	f.mu.Unlock()

	crumb, cookie, err = f.fetchCrumb(ctx)
	if err != nil {
		return "", "", err
	}

	f.mu.Lock()
	f.crumbValue, f.crumbCookie, f.crumbAt = crumb, cookie, time.Now()
	f.mu.Unlock()
	return crumb, cookie, nil
}

// dropCrumb invalidates the cached crumb so the next call re-runs the flow.
func (f *YahooFetcher) dropCrumb() {
	f.mu.Lock()
	f.crumbValue, f.crumbCookie, f.crumbAt = "", "", time.Time{}
	f.mu.Unlock()
}

// fetchCrumb runs the Yahoo crumb handshake: it grabs the session cookies from
// fc.yahoo.com and exchanges them for a crumb at the getcrumb endpoint. The
// returned cookie header must be sent alongside the crumb on quoteSummary.
func (f *YahooFetcher) fetchCrumb(ctx context.Context) (crumb, cookie string, err error) {
	cookie, err = f.fetchCookies(ctx)
	if err != nil {
		return "", "", err
	}

	req, err := http.NewRequestWithContext(ctx, "GET", "https://query1.finance.yahoo.com/v1/test/getcrumb", nil)
	if err != nil {
		return "", "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)")
	req.Header.Set("Cookie", cookie)

	resp, err := f.doRequest(ctx, req)
	if err != nil {
		return "", "", fmt.Errorf("yahoo request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		io.Copy(io.Discard, resp.Body)
		return "", "", &fetchStatusError{status: resp.StatusCode, message: fmt.Sprintf("yahoo returned status %d", resp.StatusCode)}
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", fmt.Errorf("read response: %w", err)
	}
	crumb = strings.TrimSpace(string(body))
	if crumb == "" {
		return "", "", fmt.Errorf("empty crumb from yahoo")
	}
	return crumb, cookie, nil
}

// fetchCookies obtains the Yahoo session cookies by hitting fc.yahoo.com. The
// endpoint answers 404 but still sets the cookies needed by the crumb flow, so
// the 404 is expected and not treated as an error here.
func (f *YahooFetcher) fetchCookies(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://fc.yahoo.com", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)")

	resp, err := f.doRequest(ctx, req)
	if err != nil {
		return "", fmt.Errorf("yahoo request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNotFound {
		io.Copy(io.Discard, resp.Body)
		return "", &fetchStatusError{status: resp.StatusCode, message: fmt.Sprintf("yahoo returned status %d", resp.StatusCode)}
	}
	io.Copy(io.Discard, resp.Body)

	var parts []string
	for _, c := range resp.Cookies() {
		parts = append(parts, c.Name+"="+c.Value)
	}
	if len(parts) == 0 {
		return "", fmt.Errorf("no session cookies from yahoo")
	}
	return strings.Join(parts, "; "), nil
}

var exchangeCountries = map[string]string{
	// US
	"NMS": "US", "NGM": "US", "NYS": "US", "ASE": "US", "PCX": "US",
	"NYE": "US", "PNK": "US", "OQB": "US", "OQX": "US", "OTC": "US",
	// Canada
	"TOR": "CA", "VAN": "CA",
	// Germany
	"FRA": "DE", "BER": "DE", "HAM": "DE", "MUN": "DE", "DUS": "DE",
	"STU": "DE", "XETRA": "DE", "HAN": "DE", "DRF": "DE", "GER": "DE",
	// Italy
	"MIL": "IT",
	// UK
	"LSE": "GB",
	// France
	"PAR": "FR",
	// Switzerland
	"EBS": "CH", "EBM": "CH",
	// Netherlands
	"AMS": "NL",
	// Spain
	"MCE": "ES", "BME": "ES",
	// Nordic
	"STO": "SE", "CPH": "DK", "OSL": "NO", "HEL": "FI",
	// Benelux / Austria / Portugal / Ireland
	"BRU": "BE", "VIE": "AT", "ELI": "PT", "ISE": "IE",
	// Japan / Hong Kong / China
	"TYO": "JP", "FUK": "JP", "HKG": "HK", "SHA": "CN", "SHE": "CN",
	// APAC
	"ASX": "AU", "SES": "SG", "KSC": "KR", "TPE": "TW",
	// Americas
	"SAO": "BR", "BUE": "AR", "MEX": "MX",
	// India / South Africa
	"BSE": "IN", "NSE": "IN", "JNB": "ZA",
}
