# Appunti — Modelli OpenCode Go per gli agenti di Peculium

> Note personali per la futura personalizzazione degli agenti (`.opencode/agents/`
> e `opencode.json`). Fonte: docs ufficiali OpenCode Go + benchmark web (giu-set 2026).

## Limitazioni del piano (importante)

- Limiti globali: **\$12 / 5h** · **\$30 / settimana** · **\$60 / mese** (metri a dollari).
- Il console mostra la **quota per-modello** = limite finestra ÷ `costMultiplier`
  (es. modelli con multiplier 2 → \$6/5h e \$30/mese; multiplier 4 → \$3/5h e \$15/mese).
- Limite **mensile per modello**:
  - `$15/mese`: Grok 4.6, GPT 5.6 Luna, GLM-5.3, Kimi K3, Qwen3.8 Max, DeepSeek V4 Pro, DeepSeek V4 Flash Vision, MiMo-V2.5-Pro
  - `$30/mese`: DeepSeek V4 Flash, Qwen3.8 Flash, Qwen3.7 Max, Hy4 preview
  - `$60/mese`: gli altri (GLM-5.2/5.1, Kimi K2.7/K2.6, LongCat, MiMo-V2.5, MiniMax, Muse, Qwen Plus, Hy3)
  - `$100/mese`: Omen Alpha
- DeepSeek V4 Flash/Pro: prezzi **peak** (01:00-04:00 e 06:00-10:00 UTC lun-ven) = doppio dell'off-peak.
- `Muse Spark 1.2/1.3 Contributor`: tier **contributor** → i prompt vengono usati per training Meta. **Sconsigliato per un'app finanziaria.**
- I modelli "frontier" (K3, Grok 4.6, Q3.7 Max, GLM-5.3) hanno quote 5h bassissime: riservarli a task difficili.

## Tabella modelli — dal più economico al più caro

Prezzi \$/1M token (in/out) · quota 5h = \$12 ÷ multiplier · token/5h calcolati sui pattern tipici (in+cached+out).

| Modello | In | Out | Mult | Quota 5h | Richieste/5h | Token/5h |
|---|---|---|---|---|---|---|
| Muse Spark 1.3 Contributor | 0.10 | 0.20 | 1.0 | $12.00 | 45,300 | ~3.28B |
| Muse Spark 1.2 Contributor | 0.10 | 0.20 | 1.0 | $12.00 | 45,300 | ~3.28B |
| MiMo-V2.5 | 0.14 | 0.28 | 1.0 | $12.00 | 30,100 | ~2.19B |
| Hy3 | 0.14 | 0.58 | 1.0 | $12.00 | 4,300 | ~312M |
| GLM-5.3-Flash | 0.15 | 0.50 | 4.0 | $3.00 | 1,580 | ~89M |
| Qwen3.8 Flash | 0.15 | 0.47 | 2.0 | $6.00 | 5,400 | ~318M |
| Omen Alpha | 0.20 | 0.66 | 0.6 | $20.00* | 11,600 | ~469M |
| GPT 5.6 Luna | 0.20 | 1.20 | 4.0 | $3.00 | 2,050 | ~105M |
| DeepSeek V4 Flash | 0.22 | 0.66 | 2.0 | $6.00 | 7,600 | ~547M |
| DeepSeek V4 Flash Vision | 0.22 | 0.66 | 4.0 | $3.00 | 3,800 | ~274M |
| LongCat-2.0 | 0.30 | 1.20 | 1.0 | $12.00 | 11,400 | ~1.03B |
| MiniMax M3 | 0.30 | 1.20 | 1.0 | $12.00 | 3,200 | ~181M |
| MiniMax M2.7 | 0.30 | 1.20 | 1.0 | $12.00 | 3,400 | ~188M |
| Qwen3.7 Plus | 0.40 | 1.60 | 1.0 | $12.00 | 4,300 | ~248M |
| MiMo-V2.5-Pro | 0.435 | 0.87 | 4.0 | $3.00 | 3,250 | ~283M |
| Qwen3.6 Plus | 0.50 | 3.00 | 1.0 | $12.00 | 3,300 | ~190M |
| DeepSeek V4 Pro | 0.66 | 1.98 | 4.0 | $3.00 | 1,050 | ~87M |
| Hy4 preview | 0.834 | 2.50 | 2.0 | $6.00 | 1,350 | ~98M |
| Kimi K2.7 Code | 0.95 | 4.00 | 1.0 | $12.00 | 1,350 | ~76M |
| Kimi K2.6 | 0.95 | 4.00 | 1.0 | $12.00 | 1,150 | ~64M |
| GLM-5.1 | 1.40 | 4.40 | 1.0 | $12.00 | 880 | ~47M |
| GLM-5.2 | 1.40 | 4.40 | 1.0 | $12.00 | 880 | ~47M |
| GLM-5.3 | 1.40 | 4.40 | 4.0 | $3.00 | 220 | ~12M |
| Grok 4.6 | 2.00 | 6.00 | 4.0 | $3.00 | 169 | ~5.6M |
| Qwen3.8 Max | 2.00 | 6.00 | 4.0 | $3.00 | 160 | ~11M |
| Qwen3.7 Max | 2.50 | 7.50 | 2.0 | $6.00 | 170 | ~11M |
| Kimi K3 | 3.00 | 15.00 | 4.0 | $3.00 | 110 | ~8.6M |

\* Omen Alpha: quota teorica \$20 ma tetto globale reale \$12/5h.

I token/5h includono **cached read** (costo bassissimo); i token "nuovi" effettivi per richiesta sono ~300-1.000 input + ~100-310 output.

## Benchmark utili (riferimento)

- **SWE-bench Verified** (isolamento bug): DS V4 Pro 80.6% (record open), V4 Flash ~79%, GLM-5.1 77.8%.
- **SWE-bench Pro** (multi-file/repo): **Qwen3.8 Max 67.7%** (top Go), Qwen3.8 Flash 62.5%, GPT 5.6 Luna 62.7%, GLM-5.2 62.1%, Qwen3.7 Max 60.6%, MiniMax M3 59.0%.
- **Terminal-Bench 2.1** (CLI agent): Kimi K3 88.3%, GLM-5.3 88.2%, **Qwen3.8 Max 86.6%**, GLM-5.2 81.0%.
- **DeepSWE v1.1** (long-horizon): GLM-5.3 69.0%, Kimi K3 67.5%, Qwen3.8 Flash 58.7%, Qwen3.8 Max 56.6%.
- **LiveCodeBench**: DS V4 Pro 93.5% (#1), Qwen3.8 Flash 91.9%.
- **PaperBench** (ricerca/replicazione): **Qwen3.8 Max 93.0%** (top, sopra GPT-5.6 Sol 90.5 e Fable 5 88.8).
- **AA Intelligence Index**: K3 57 · Grok 4.6 61 · GLM-5.3 59.5 · **Qwen3.8 Max 58.1** · GLM-5.2 52.6 · DS V4 Pro 45.3.
- **Frontend Code Arena**: Kimi K3 #1 (preferenza umana). Hy3 batte GLM-5.1 su frontend (blind eval Tencent). Qwen3.8 Max: QwenReactBench 1724 / QwenSVGBench 1713 (build React/SVG).
- **Velocità** (tok/s): Qwen3.8 Flash 195, GPT 5.6 Luna 175, GLM-5.2 168, DS V4 Flash 72.

### Qwen3.8 in breve (verificato con benchmark ufficiali Qwen, ago-set 2026)

- **Qwen3.8 Flash** (= "Flash-Next" su Go: 125B totali / 6B attivi, architettura preview Qwen4, 1M ctx, 195 tok/s):
  a \$0.15/\$0.47 è oggi il **miglior coder per prezzo del piano** — SWE-Pro 62.5%, DeepSWE 58.7%,
  LiveCodeBench 91.9%, SWE-Multilingual 81.0%. Supera DS V4 Flash (SWE-Pro ~56-59%) e pareggia
  GLM-5.2 (62.1%) a ~1/9 del costo, con quota 5.400 rq/5h. Caveat: NL2Repo sotto a V4 Flash (48.1 vs 54.2).
- **Qwen3.8 Max** (2.4T / 95B attivi, 1M ctx): il coder **SWE-Pro più forte del piano (67.7%)** ed eccelle
  su ricerca/long-horizon: PaperBench 93.0 (#1), Terminal-Bench 86.6, GPQA 92.6, IFBench 82.8, OSWorld 86.1.
  Punti deboli: HLE 43.6 e DeepSWE 56.6 (sotto Opus/Fable). Quota scarsa: 160 rq/5h, \$15/mese → **premium**.

## Modelli consigliati per agente (3 per agente, costo crescente)

| Agente | 1 (economico) | 2 (medio) | 3 (premium) |
|---|---|---|---|
| **backend** (Go/PG/Redis/API) | **Qwen3.8 Flash (primario)** | DeepSeek V4 Pro | GLM-5.2 |
| **frontend** (SvelteKit/TS/Tailwind) | Hy3 | **Qwen3.8 Flash (primario)** | Kimi K3 |
| **python** (FastAPI/scraping) | MiMo-V2.5 | **Qwen3.8 Flash (primario)** | Qwen3.7 Plus |
| **finanza** (analisi, read-only) | **MiniMax M3 (primario)** | GLM-5.2 | Qwen3.8 Max |
| **ux-ui** (design, read-only) | Hy3 | LongCat-2.0 | **Qwen3.8 Max (primario)** |
| **general** (built-in) | **Qwen3.8 Flash (primario)** | MiniMax M3 | GLM-5.2 |
| **explore** (built-in, rapido) | MiMo-V2.5 | Hy3 | **Qwen3.8 Flash (primario)** |

### Razionale sintetico per agente

- **backend**: tante chiamate tool → **Qwen3.8 Flash** come workhorse (SWE-Pro 62.5%, DeepSWE 58.7%, 195 tok/s, 5.400 rq/5h) supera DS V4 Flash quasi a parità di prezzo; DS V4 Pro per i task Go più complessi (SWE-V 80.6%); GLM-5.2 come orchestratore 1M ctx (SWE-Pro 62.1%). GLM-5.3 più capace ma quota 220 rq/5h → troppo scarso.
- **frontend**: Hy3 economico e forte su UI; **Qwen3.8 Flash** bilanciato (SWE-Multilingual 81.0%, veloce); K3 per UI critiche (#1 Frontend Code Arena) ma 110 rq/5h. Qwen3.8 Max alternativo premium (QwenReactBench 1724 / QwenSVGBench 1713).
- **python**: scraping = tante richieste, difficoltà media → MiMo-V2.5 (30k rq/5h, 1M ctx) e **Qwen3.8 Flash** come primario (miglior rapporto prezzo/prestazioni del piano).
- **finanza**: poche chiamate, reasoning → MiniMax M3 (BrowseComp 83.5, 1M ctx), poi GLM-5.2; **Qwen3.8 Max** per analisi dure/ricerca (GPQA 92.6, PaperBench 93.0) — alternativa Grok 4.6 (AA 61).
- **ux-ui**: creativo → Hy3/LongCat per idee; **Qwen3.8 Max** per proposte visuali forti (QwenSVGBench 1713, QwenReactBench 1724) — alternativa Grok 4.6 (lavoro visuale/interattivo).
- **general/explore**: volume e costo → MiMo/Hy3; **Qwen3.8 Flash** come workhorse di qualità; GLM-5.2 come orchestratore.

## Come applicare (quando vorrai)

Nei file `.opencode/agents/<nome>.md` aggiungi nel frontmatter:

```yaml
---
model: opencode-go/glm-5.2
---
```

Per i built-in (general, explore) aggiungi in `opencode.json`:

```json
{
  "agent": {
    "explore": { "model": "opencode-go/deepseek-v4-flash" },
    "general": { "model": "opencode-go/glm-5.2" }
  }
}
```

Formato ID: `opencode-go/<model-id>` (es. `opencode-go/kimi-k3`). Verifica i disponibili con `opencode models`.