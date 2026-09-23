<div align="center">

# ebay-cli

[![License](https://img.shields.io/badge/LICENSE-MIT-5C9E31?style=for-the-badge)](LICENSE)
[![Built by](https://img.shields.io/badge/BUILT%20BY-REBELCODE-8A2BE2?style=for-the-badge)](https://rebelcode.com)

**Search any eBay marketplace from the terminal, and get told when something new is listed.**

</div>

Built on eBay's official Browse API, so there is no scraping and nothing to break when the site's markup changes. Cross-border shipping costs are quoted to wherever you actually live, not to the marketplace's own country.

## Install

```
go install github.com/jgalea/ebay-cli/cmd/ebay@latest
```

Works on macOS, Linux and Windows.

## Credentials

You need a Production keyset from https://developer.ebay.com/my/keys. The App ID is the client id, the Cert ID is the client secret.

On macOS, store both in the Keychain, entering each at the prompt so the value never lands in shell history:

```
security add-generic-password -a "$USER" -s claude-ebay-client-id -U -w
security add-generic-password -a "$USER" -s claude-ebay-client-secret -U -w
```

On Linux and Windows, which have no Keychain, save them in `credentials.json` in the config directory (`~/.config/ebay-cli/` on Linux, `%AppData%\ebay-cli\` on Windows). On Linux, `chmod 600` it. This file works on macOS too, in `~/Library/Application Support/ebay-cli/`.

```json
{"client_id": "...", "client_secret": "..."}
```

`EBAY_CLIENT_ID` and `EBAY_CLIENT_SECRET` are read first if set. `ebay auth` prints the exact path for your machine, and `ebay auth --check` confirms the credentials work.

## Search

```
ebay search flair 58 espresso --market de --max 450
ebay search cafelat robot -m es --condition used --sort price
ebay search "1zpresso k-ultra" -m it --auctions --sort ending
```

Results show the price a buyer would act on right now, which for an auction with bids is the standing bid rather than the opening price, plus the cheapest quoted shipping to your destination.

| Flag | Meaning |
| --- | --- |
| `-m`, `--market` | `de`, `es`, `it`, `fr`, `gb`, `nl`, `at`, `us` and others; see `ebay markets` |
| `--ship-to` | destination country for shipping quotes, default `PT` |
| `--min`, `--max` | price bounds |
| `--currency` | currency for the price bounds, default `EUR` |
| `--condition` | `new`, `used` or `both` |
| `--seller-in` | only sellers located in this country |
| `--auctions` | include auction listings |
| `--fixed` | buy-it-now only |
| `--sort` | `price`, `-price`, `new`, `ending` |
| `--limit` | 1 to 200, default 50 |
| `--json` | raw API response |

### Auctions are hidden by default

This is eBay's behaviour, not a bug in the tool. The Browse API returns only listings that offer Buy It Now. Once an auction receives a qualifying bid it loses `FIXED_PRICE` and disappears from a default search, which is exactly when it becomes interesting. Pass `--auctions` to see them.

## Watch

A watch is a saved search that reports only what it has not seen before, so it can run on a schedule without repeating itself.

```
ebay watch add flair58 flair 58 espresso -m de --max 450 --auctions
ebay watch run
ebay watch list
ebay watch remove flair58
```

The first `run` records everything already listed and reports nothing. Every run after that prints new arrivals only, and prints nothing at all when there are none, which makes it quiet enough for cron:

```
*/30 * * * * /path/to/ebay watch run
```

Watches live in `watches.json` in the same config directory as the credentials: `~/Library/Application Support/ebay-cli/` on macOS, `~/.config/ebay-cli/` on Linux, `%AppData%\ebay-cli\` on Windows. On Windows, run `ebay watch run` from Task Scheduler instead of cron.

## Limits

eBay allows 1,000 token requests a day per application. The tool mints one token and reuses it until it expires, so a run costs one token request regardless of how many searches it performs.

Sold and completed listings are not available through the Browse API, so this searches live listings only.
