import requests
from fastapi import FastAPI, HTTPException, Query

from . import morningstar, scraper
from .morningstar import MorningstarDataError
from .schemas import EtfSearchResult, Exposure, Holdings

app = FastAPI(title="Peculium ETF metadata service", version="1.0.0")


@app.get("/healthz")
def healthz():
    return {"status": "ok"}


@app.get("/api/v1/etf/search", response_model=list[EtfSearchResult])
def search_etf(q: str = Query(...)):
    try:
        # Tickers with a recognized market suffix resolve via Morningstar (the
        # ISIN can differ by market); bare tickers fall back to JustETF.
        if morningstar.has_market_suffix(q):
            return morningstar.search_etf_morningstar(q)
        return scraper.search_etf(q)
    except requests.RequestException as exc:
        raise HTTPException(status_code=502, detail=f"upstream search failed: {exc}")


@app.get("/api/v1/etf/{isin}/exposure", response_model=Exposure)
def get_exposure(isin: str):
    try:
        exposure = scraper.fetch_exposure(isin)
    except requests.RequestException as exc:
        raise HTTPException(status_code=502, detail=f"upstream fetch failed: {exc}")
    except Exception as exc:
        raise HTTPException(status_code=502, detail=f"exposure parse failed: {exc}")
    if not exposure.countries:
        raise HTTPException(status_code=502, detail="no country data found for ISIN")
    return exposure


@app.get("/api/v1/etf/{isin}/holdings", response_model=Holdings)
def get_holdings(isin: str):
    return Holdings(isin=isin, holdings=[])


@app.get("/api/v1/etf/{isin}/morningstar-exposure", response_model=Exposure)
def get_morningstar_exposure(isin: str):
    try:
        exposure = morningstar.fetch_morningstar_exposure(isin)
    except MorningstarDataError as exc:
        raise HTTPException(status_code=502, detail=str(exc))
    except requests.RequestException as exc:
        raise HTTPException(status_code=502, detail=f"upstream fetch failed: {exc}")
    except Exception as exc:
        raise HTTPException(status_code=502, detail=f"exposure parse failed: {exc}")
    if not exposure.countries:
        raise HTTPException(status_code=502, detail="no country data found for ISIN")
    return exposure
