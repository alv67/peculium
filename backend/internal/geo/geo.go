// Package geo defines the canonical macro-regions and GICS sectors used to
// classify asset exposure, plus the mappings from country codes and provider
// sector names to those canonical values.
package geo

import "strings"

// Regions is the canonical list of macro-regions in display order. It follows
// Morningstar's official region taxonomy so a provider prefill can map 1:1.
var Regions = []string{
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

// OtherRegion is the canonical name of the fallback macro-region that absorbs
// country weights that map to no specific region (or the residual when the
// country weights of a provider do not cover 100%).
const OtherRegion = "Other / Not Classified"

// Countries is the canonical list of ISO alpha-2 country codes in display
// order (grouped by macro-region, then code). It contains exactly the keys of
// the regionByCountry map below, so every canonical country maps to a region.
var Countries = []string{
	// North America
	"CA", "US",
	// Latin America
	"AR", "BO", "BR", "CL", "CO", "CR", "CU", "DO", "EC", "GT", "HN", "MX", "NI", "PA", "PE", "PY", "SV", "UY", "VE",
	// United Kingdom
	"GB",
	// Europe Developed
	"AT", "BE", "CH", "CY", "DE", "DK", "ES", "FI", "FR", "GR", "IE", "IS", "IT", "LU", "MT", "NL", "NO", "PT", "SE", "SI",
	// Europe Emerging
	"BG", "CZ", "EE", "HR", "HU", "LT", "LV", "PL", "RO", "RS", "RU", "SK", "TR", "UA",
	// Africa / Middle East
	"AE", "BH", "DZ", "EG", "IL", "JO", "KE", "KW", "LB", "MA", "NG", "OM", "QA", "SA", "TN", "ZA",
	// Japan
	"JP",
	// Australasia
	"AU", "NZ",
	// Asia Developed
	"HK", "KR", "SG", "TW",
	// Asia Emerging
	"BD", "CN", "ID", "IN", "LK", "MY", "PH", "PK", "TH", "VN",
}

// GICSSectors is the canonical list of GICS sectors in display order, ending
// with the residual bucket so a provider prefill can map 1:1.
var GICSSectors = []string{
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

// OtherSector is the canonical name of the fallback sector that absorbs
// provider sector labels that map to no GICS sector (or the residual when the
// sector weights of a provider do not cover 100%).
const OtherSector = "Other / Not Classified"

var regionByCountry = map[string]string{
	// North America
	"US": "North America",
	"CA": "North America",
	// Latin America
	"BR": "Latin America",
	"MX": "Latin America",
	"AR": "Latin America",
	"CL": "Latin America",
	"CO": "Latin America",
	"PE": "Latin America",
	"VE": "Latin America",
	"UY": "Latin America",
	"PY": "Latin America",
	"EC": "Latin America",
	"BO": "Latin America",
	"CR": "Latin America",
	"PA": "Latin America",
	"DO": "Latin America",
	"GT": "Latin America",
	"HN": "Latin America",
	"SV": "Latin America",
	"NI": "Latin America",
	"CU": "Latin America",
	// United Kingdom
	"GB": "United Kingdom",
	// Europe Developed
	"FR": "Europe Developed",
	"DE": "Europe Developed",
	"IT": "Europe Developed",
	"ES": "Europe Developed",
	"CH": "Europe Developed",
	"NL": "Europe Developed",
	"BE": "Europe Developed",
	"AT": "Europe Developed",
	"SE": "Europe Developed",
	"DK": "Europe Developed",
	"NO": "Europe Developed",
	"FI": "Europe Developed",
	"IE": "Europe Developed",
	"PT": "Europe Developed",
	"LU": "Europe Developed",
	"IS": "Europe Developed",
	"GR": "Europe Developed",
	"MT": "Europe Developed",
	"CY": "Europe Developed",
	"SI": "Europe Developed",
	// Europe Emerging
	"PL": "Europe Emerging",
	"CZ": "Europe Emerging",
	"HU": "Europe Emerging",
	"TR": "Europe Emerging",
	"RU": "Europe Emerging",
	"RO": "Europe Emerging",
	"BG": "Europe Emerging",
	"SK": "Europe Emerging",
	"HR": "Europe Emerging",
	"RS": "Europe Emerging",
	"UA": "Europe Emerging",
	"LT": "Europe Emerging",
	"LV": "Europe Emerging",
	"EE": "Europe Emerging",
	// Africa / Middle East
	"ZA": "Africa / Middle East",
	"NG": "Africa / Middle East",
	"EG": "Africa / Middle East",
	"SA": "Africa / Middle East",
	"AE": "Africa / Middle East",
	"IL": "Africa / Middle East",
	"KE": "Africa / Middle East",
	"MA": "Africa / Middle East",
	"DZ": "Africa / Middle East",
	"TN": "Africa / Middle East",
	"QA": "Africa / Middle East",
	"KW": "Africa / Middle East",
	"OM": "Africa / Middle East",
	"BH": "Africa / Middle East",
	"JO": "Africa / Middle East",
	"LB": "Africa / Middle East",
	// Japan
	"JP": "Japan",
	// Australasia
	"AU": "Australasia",
	"NZ": "Australasia",
	// Asia Developed
	"KR": "Asia Developed",
	"TW": "Asia Developed",
	"SG": "Asia Developed",
	"HK": "Asia Developed",
	// Asia Emerging
	"CN": "Asia Emerging",
	"IN": "Asia Emerging",
	"TH": "Asia Emerging",
	"ID": "Asia Emerging",
	"MY": "Asia Emerging",
	"PH": "Asia Emerging",
	"VN": "Asia Emerging",
	"LK": "Asia Emerging",
	"PK": "Asia Emerging",
	"BD": "Asia Emerging",
}

// RegionForCountry maps an ISO alpha-2 country code to its macro-region.
// Matching is case-insensitive; unknown or empty codes fall back to
// "Other / Not Classified".
func RegionForCountry(country string) string {
	if region, ok := regionByCountry[strings.ToUpper(strings.TrimSpace(country))]; ok {
		return region
	}
	return "Other / Not Classified"
}

// countryCodeByName maps full country names (Yahoo assetProfile style, e.g.
// "United States") to their ISO alpha-2 code. Lookups are case-insensitive.
var countryCodeByName = map[string]string{
	"UNITED STATES":            "US",
	"UNITED STATES OF AMERICA": "US",
	"USA":                      "US",
	"CANADA":                   "CA",
	"UNITED KINGDOM":           "GB",
	"GREAT BRITAIN":            "GB",
	"ENGLAND":                  "GB",
	"FRANCE":                   "FR",
	"GERMANY":                  "DE",
	"ITALY":                    "IT",
	"SPAIN":                    "ES",
	"NETHERLANDS":              "NL",
	"SWITZERLAND":              "CH",
	"SWEDEN":                   "SE",
	"DENMARK":                  "DK",
	"NORWAY":                   "NO",
	"FINLAND":                  "FI",
	"BELGIUM":                  "BE",
	"AUSTRIA":                  "AT",
	"PORTUGAL":                 "PT",
	"IRELAND":                  "IE",
	"JAPAN":                    "JP",
	"AUSTRALIA":                "AU",
	"CHINA":                    "CN",
	"HONG KONG":                "HK",
	"INDIA":                    "IN",
	"BRAZIL":                   "BR",
	"MEXICO":                   "MX",
	"ARGENTINA":                "AR",
	"TAIWAN":                   "TW",
	"KOREA":                    "KR",
	"REPUBLIC OF KOREA":        "KR",
	"SOUTH KOREA":              "KR",
	"SINGAPORE":                "SG",
	"SOUTH AFRICA":             "ZA",
	"NEW ZEALAND":              "NZ",
	"RUSSIA":                   "RU",
	"POLAND":                   "PL",
	"TURKEY":                   "TR",
	"ISRAEL":                   "IL",
	"LUXEMBOURG":               "LU",
	"GREECE":                   "GR",
	"SAUDI ARABIA":             "SA",
	"UNITED ARAB EMIRATES":     "AE",
	"THAILAND":                 "TH",
	"MALAYSIA":                 "MY",
	"INDONESIA":                "ID",
	"VIETNAM":                  "VN",
	"PHILIPPINES":              "PH",
	"PAKISTAN":                 "PK",
	"COLOMBIA":                 "CO",
	"CHILE":                    "CL",
	"PERU":                     "PE",
	"CZECH REPUBLIC":           "CZ",
	"CZECHIA":                  "CZ",
	"HUNGARY":                  "HU",
	"ROMANIA":                  "RO",
	"QATAR":                    "QA",
	"KUWAIT":                   "KW",
	"EGYPT":                    "EG",
	"NIGERIA":                  "NG",
	"KENYA":                    "KE",
	"BANGLADESH":               "BD",
}

// NormalizeCountry trims and upper-cases the input, returning the ISO alpha-2
// code. An input that is already a two-letter code is returned unchanged; other
// values are resolved against the full-name map. Unknown values return "".
func NormalizeCountry(codeOrName string) string {
	s := strings.ToUpper(strings.TrimSpace(codeOrName))
	if s == "" {
		return ""
	}
	if isISOCode(s) {
		return s
	}
	return countryCodeByName[s]
}

// isISOCode reports whether s is a two-letter ASCII code.
func isISOCode(s string) bool {
	if len(s) != 2 {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < 'A' || s[i] > 'Z' {
			return false
		}
	}
	return true
}

// IsValidCountry reports whether s is a recognizable country value: either an
// ISO alpha-2 code or a resolvable full name.
func IsValidCountry(s string) bool {
	return NormalizeCountry(s) != ""
}

var sectorAliases = map[string]string{
	"Technology":             "Information Technology",
	"Information Technology": "Information Technology",
	"Financial Services":     "Financials",
	"Finance":                "Financials",
	"Financials":             "Financials",
	"Healthcare":             "Health Care",
	"Health Care":            "Health Care",
	"Consumer Cyclical":      "Consumer Discretionary",
	"Consumer Cyclicals":     "Consumer Discretionary",
	"Consumer Services":      "Consumer Discretionary",
	"Consumer Discretionary": "Consumer Discretionary",
	"Consumer Defensive":     "Consumer Staples",
	"Consumer Non-Cyclicals": "Consumer Staples",
	"Consumer Staples":       "Consumer Staples",
	"Non-Energy Materials":   "Materials",
	"Basic Materials":        "Materials",
	"Materials":              "Materials",
	"Communication Services": "Communication Services",
	"Telecommunication":      "Communication Services",
	"Telecommunications":     "Communication Services",
	"Business Services":      "Industrials",
}

// NormalizeSector maps a provider sector name to the canonical GICS name.
// Names absent from the alias map are returned unchanged.
func NormalizeSector(sector string) string {
	s := strings.TrimSpace(sector)
	if s == "" {
		return ""
	}
	if canonical, ok := sectorAliases[s]; ok {
		return canonical
	}
	return s
}

var sectorKeyToGICS = map[string]string{
	"realestate":             "Real Estate",
	"consumer_cyclical":      "Consumer Discretionary",
	"basic_materials":        "Materials",
	"consumer_defensive":     "Consumer Staples",
	"technology":             "Information Technology",
	"communication_services": "Communication Services",
	"financial_services":     "Financials",
	"utilities":              "Utilities",
	"industrials":            "Industrials",
	"energy":                 "Energy",
	"healthcare":             "Health Care",
}

// SectorKeyToGICS maps a Yahoo sectorWeightings key to the canonical GICS
// sector. Unknown keys return "".
func SectorKeyToGICS(key string) string {
	return sectorKeyToGICS[strings.ToLower(strings.TrimSpace(key))]
}

// Investment-class keywords used by ClassifyAssetClass. Operated on the
// lower-cased concatenation of name + fund category.
var (
	classCryptoKeywords   = []string{"crypto", "bitcoin", "ethereum", "blockchain"}
	classMoneyMarketWords = []string{"money market", "cash reserve", "cash mgmt", " mmf"}
	classCommodityWords   = []string{"commodit", "gold", "silver", "platinum", "palladium", "precious metals", "crude oil", "natural gas", "uranium", "copper", "agricultur"}
	classRealEstateWords  = []string{"real estate", " reit", "property "}
	classBondWords        = []string{"bond", "treasury", "governm", "aggregate", "corporate", "municipal", "muni", "fixed income", "high yield", "inflation-protected", " tips", "ultra-short", "duration", "credit"}
	classCurrencyWords    = []string{"currency", "foreign exchange", " fxetf"}
	classMixedWords       = []string{"allocation", "balanced", "target-date", "target date", "multi-asset", "moderate", "conservative", "lifestyle"}
)

// ClassifyAssetClass guesses the investment class of an asset from its type,
// name and (for funds) the provider's fund category (Morningstar-style).
//
// Stocks, bonds, commodities and crypto map by type; ETFs and mutual funds are
// classified by keyword heuristics on name + category, defaulting to "equity".
// The result is best-effort: a manual override on the asset page always wins.
func ClassifyAssetClass(assetType, name, fundCategory string) string {
	switch assetType {
	case "stock":
		return "equity"
	case "bond":
		return "bond"
	case "commodity":
		return "commodity"
	case "crypto":
		return "crypto"
	case "cash":
		return "currency"
	}

	hay := strings.ToLower(name + " " + fundCategory)
	switch {
	case containsAnyWord(hay, classCryptoKeywords):
		return "crypto"
	case containsAnyWord(hay, classMoneyMarketWords):
		// Money market is not a class in our taxonomy: it lands on bond (debt).
		return "bond"
	case containsAnyWord(hay, classCommodityWords):
		return "commodity"
	case containsAnyWord(hay, classRealEstateWords):
		return "real_estate"
	case containsAnyWord(hay, classBondWords):
		return "bond"
	case containsAnyWord(hay, classCurrencyWords):
		return "currency"
	case containsAnyWord(hay, classMixedWords):
		return "mixed"
	default:
		return "equity"
	}
}

func containsAnyWord(hay string, words []string) bool {
	for _, w := range words {
		if strings.Contains(hay, w) {
			return true
		}
	}
	return false
}

// FundClassifiable reports whether an asset type needs fund-category-based
// classification, i.e. its class is not already fixed by the type itself.
func FundClassifiable(assetType string) bool {
	switch assetType {
	case "stock", "bond", "commodity", "crypto", "cash":
		return false
	default:
		return true
	}
}
