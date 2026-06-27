# Foodcraft Nakama Backend

Go runtime module for [Nakama](https://heroiclabs.com/docs/nakama/server-framework/introduction/) — handles discovery validation, player state, and leaderboards.

## Architecture

| Nakama feature | Usage |
|----------------|--------|
| **Leaderboards** | `culinary_fame` (discovery points), `explorer` (discovery count) |
| **Storage** `player/state` | `{ discovered[], craft_count, discovery_points }` |
| **Storage** `first_discover` | Global first discoverer per item |
| **RPCs** | `discover_recipe`, `sync_discoveries`, `record_craft`, `get_profile`, `get_leaderboard` |

Server embeds `modules/data/game_data.json` and validates every discovery (recipe A+B→result, tier-0 ingredients, prerequisite discoveries).

## Quick start

```bash
make build   # compile backend.so via Docker
make run     # start CockroachDB + Nakama
make logs    # tail Nakama logs
make stop    # tear down
```

## Local endpoints

| Service | URL |
|---------|-----|
| HTTP API | http://127.0.0.1:7350 |
| Console | http://127.0.0.1:7351 (admin / password) |
| Server key | `foodcraft_dev_key` |

## Godot client

Set matching values in `foodcraft-2/services/nakama_config.gd`. On home screen, guest login calls device auth; discoveries and crafts sync via RPC while playing offline-first with `user://player_progress.json` as local cache.
