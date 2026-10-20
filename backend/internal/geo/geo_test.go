package geo

import "testing"

func TestCanonicalCountries(t *testing.T) {
	seen := map[string]bool{}
	for _, c := range Countries {
		if seen[c] {
			t.Fatalf("Countries contains duplicate code %q", c)
		}
		seen[c] = true
		if _, ok := regionByCountry[c]; !ok {
			t.Fatalf("Countries %q is not a key of regionByCountry", c)
		}
	}
	for code := range regionByCountry {
		if !seen[code] {
			t.Fatalf("regionByCountry key %q missing from Countries", code)
		}
	}
	if len(Countries) != len(regionByCountry) {
		t.Fatalf("Countries len = %d, regionByCountry len = %d", len(Countries), len(regionByCountry))
	}
}

func TestClassifyAssetClass(t *testing.T) {
	cases := []struct {
		name      string
		assetType string
		assetName string
		category  string
		want      string
	}{
		{"stock defaults to equity", "stock", "Apple Inc.", "", "equity"},
		{"bond type", "bond", "Some Bond", "", "bond"},
		{"commodity type", "commodity", "Gold", "", "commodity"},
		{"crypto type", "crypto", "Bitcoin", "", "crypto"},
		{"cash type", "cash", "Cash account", "", "currency"},
		{"bond ETF by name", "etf", "Vanguard Total Bond Market ETF", "", "bond"},
		{"bond ETF by category", "etf", "iShares Core US Agg", "Intermediate Core Bond", "bond"},
		{"government bond ETF", "etf", "Vanguard Short Treasury ETF", "Short Government", "bond"},
		{"money market lands on bond", "etf", "Money Market Fund", "", "bond"},
		{"commodity ETF by category", "etf", "SPDR Gold Shares", "Commodities Focused", "commodity"},
		{"commodity ETF by name", "etf", "iShares Silver Trust", "", "commodity"},
		{"real estate ETF", "etf", "Vanguard Real Estate Index Fund", "Real Estate", "real_estate"},
		{"REIT ETF", "etf", "iShares U.S. Real Estate ETF", "", "real_estate"},
		{"crypto ETF by name", "etf", "ProShares Bitcoin Strategy ETF", "", "crypto"},
		{"currency ETF", "etf", "Invesco CurrencyShares Euro Trust", "Currency", "currency"},
		{"balanced fund", "mutual_fund", "Vanguard Balanced Fund", "Moderate Allocation", "mixed"},
		{"all-world equity ETF defaults to equity", "etf", "Vanguard FTSE All-World UCITS ETF USD Acc", "", "equity"},
		{"index ETF defaults to equity", "etf", "SPDR S&P 500 ETF", "Large Blend", "equity"},
		{"stock via etf name with oil in category word is avoided", "etf", "Global X Oil Equities", "Equity", "equity"},
	}

	for _, tc := range cases {
		got := ClassifyAssetClass(tc.assetType, tc.assetName, tc.category)
		if got != tc.want {
			t.Errorf("%s: ClassifyAssetClass(%q, %q, %q) = %q, want %q",
				tc.name, tc.assetType, tc.assetName, tc.category, got, tc.want)
		}
	}
}

func TestFundClassifiable(t *testing.T) {
	cases := map[string]bool{
		"stock":       false,
		"bond":        false,
		"commodity":   false,
		"crypto":      false,
		"cash":        false,
		"etf":         true,
		"mutual_fund": true,
		"other":       true,
		"":            true,
	}
	for typ, want := range cases {
		if got := FundClassifiable(typ); got != want {
			t.Errorf("FundClassifiable(%q) = %v, want %v", typ, got, want)
		}
	}
}

// TestRegionsCanonical asserts the canonical region list matches the Morningstar
// taxonomy: 10 regions plus the residual bucket, in display order.
func TestRegionsCanonical(t *testing.T) {
	want := []string{
		"North America",
		"Latin America",
		"United Kingdom",
		"Europe Developed",
		"Europe Emerging",
		"Africa / Middle East",
		"Japan",
		"Australasia",
		"Asia Developed",
		"Asia Emerging",
		"Other / Not Classified",
	}
	if len(Regions) != len(want) {
		t.Fatalf("Regions len = %d, want %d", len(Regions), len(want))
	}
	for i, name := range want {
		if Regions[i] != name {
			t.Fatalf("Regions[%d] = %q, want %q", i, Regions[i], name)
		}
	}
	if Regions[len(Regions)-1] != OtherRegion {
		t.Fatalf("last region = %q, want %q", Regions[len(Regions)-1], OtherRegion)
	}
}

// TestSectorsCanonical asserts the canonical GICS sector list: 11 sectors
// plus the residual bucket, in display order.
func TestSectorsCanonical(t *testing.T) {
	want := []string{
		"Energy",
		"Materials",
		"Industrials",
		"Consumer Discretionary",
		"Consumer Staples",
		"Health Care",
		"Financials",
		"Information Technology",
		"Communication Services",
		"Utilities",
		"Real Estate",
		"Other / Not Classified",
	}
	if len(GICSSectors) != len(want) {
		t.Fatalf("GICSSectors len = %d, want %d", len(GICSSectors), len(want))
	}
	for i, name := range want {
		if GICSSectors[i] != name {
			t.Fatalf("GICSSectors[%d] = %q, want %q", i, GICSSectors[i], name)
		}
	}
	if GICSSectors[len(GICSSectors)-1] != OtherSector {
		t.Fatalf("last sector = %q, want %q", GICSSectors[len(GICSSectors)-1], OtherSector)
	}
}

// TestChangedRegionMappings asserts the country-to-region mappings that moved
// when aligning the taxonomy with Morningstar.
func TestChangedRegionMappings(t *testing.T) {
	cases := map[string]string{
		"GB": "United Kingdom",
		"JP": "Japan",
		"AU": "Australasia",
		"NZ": "Australasia",
		"KR": "Asia Developed",
		"TW": "Asia Developed",
		"SI": "Europe Developed",
	}
	for code, want := range cases {
		if got := RegionForCountry(code); got != want {
			t.Errorf("RegionForCountry(%q) = %q, want %q", code, got, want)
		}
	}
}

func TestRegionForCountry(t *testing.T) {
	cases := []struct {
		name    string
		country string
		want    string
	}{
		{"US", "US", "North America"},
		{"us lowercase", "us", "North America"},
		{"Canada", "CA", "North America"},
		{"Italy", "IT", "Europe Developed"},
		{"Italy lowercase", "it", "Europe Developed"},
		{"United Kingdom", "GB", "United Kingdom"},
		{"Japan", "JP", "Japan"},
		{"Australia", "AU", "Australasia"},
		{"New Zealand", "NZ", "Australasia"},
		{"South Korea", "KR", "Asia Developed"},
		{"Taiwan", "TW", "Asia Developed"},
		{"Slovenia", "SI", "Europe Developed"},
		{"China", "CN", "Asia Emerging"},
		{"empty", "", "Other / Not Classified"},
		{"unknown", "XYZ", "Other / Not Classified"},
		{"spaces", "  US  ", "North America"},
	}
	for _, tc := range cases {
		if got := RegionForCountry(tc.country); got != tc.want {
			t.Errorf("%s: RegionForCountry(%q) = %q, want %q", tc.name, tc.country, got, tc.want)
		}
	}
}

func TestNormalizeCountry(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"full name", "United States", "US"},
		{"full name lowercase", "united kingdom", "GB"},
		{"already a code", "US", "US"},
		{"code lowercase", "it", "IT"},
		{"full name with spaces", "  Switzerland ", "CH"},
		{"full name multi-word", "Hong Kong", "HK"},
		{"unknown", "XYZ", ""},
		{"unknown name", "Atlantis", ""},
		{"empty", "", ""},
		{"one letter is unknown", "U", ""},
		{"usa alias resolved via map", "USA", "US"},
		{"justetf name", "South Korea", "KR"},
		{"justetf name", "Saudi Arabia", "SA"},
		{"justetf name", "United Arab Emirates", "AE"},
		{"justetf name", "Thailand", "TH"},
		{"justetf name", "Malaysia", "MY"},
	}
	for _, tc := range cases {
		if got := NormalizeCountry(tc.in); got != tc.want {
			t.Errorf("%s: NormalizeCountry(%q) = %q, want %q", tc.name, tc.in, got, tc.want)
		}
	}
}

func TestNormalizeSectorJustETF(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"Technology", "Information Technology"},
		{"Finance", "Financials"},
		{"Non-Energy Materials", "Materials"},
		{"Consumer Non-Cyclicals", "Consumer Staples"},
		{"Consumer Cyclicals", "Consumer Discretionary"},
		{"Consumer Services", "Consumer Discretionary"},
		{"Business Services", "Industrials"},
		{"Telecommunication", "Communication Services"},
		{"Healthcare", "Health Care"},
		{"Industrials", "Industrials"},
		{"Energy", "Energy"},
		{"Utilities", "Utilities"},
	}
	for _, tc := range cases {
		if got := NormalizeSector(tc.in); got != tc.want {
			t.Errorf("NormalizeSector(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
