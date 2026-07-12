# Self-hosted OSM stack (tiles + Overpass) on Kubernetes

An open, no-billing Mapbox-equivalent:

| Service          | Purpose                                   | Endpoint (via gateway)        |
| ---------------- | ----------------------------------------- | ----------------------------- |
| **TileServer GL**| Basemap tiles/styles (rendered by MapLibre)| `/maps/…`                     |
| **Overpass API** | OSM feature queries (Overpass QL)         | `/overpass/api/interpreter`   |

All service traffic goes through an **nginx gateway** with route-scoped
`X-API-Key` permissions. Missing, invalid, or cross-route keys return `401`.

| Credential | `/maps/` | `/overpass/` | Owner |
| --- | --- | --- | --- |
| iOS maps key | allowed | denied | iOS configuration |
| backend Overpass key | denied | allowed | Railway backend secret |

An app-bundled key is extractable, so the iOS key is deliberately incapable of
running Overpass queries. The Railway backend owns and bounds those queries.

```
Ingress ──► gateway (checks X-API-Key)
               ├─ /maps/…     ──► tileserver  (Deployment + PVC)
               └─ /overpass/… ──► overpass     (StatefulSet + PVC)
```

## Files

| File                    | Committed? | What                                             |
| ----------------------- | ---------- | ------------------------------------------------ |
| `secrets.example.yaml`  | ✅ yes     | Template Secret — copy to `secrets.yaml`.        |
| `secrets.yaml`          | 🚫 ignored | **Real API keys.** Git-ignored; never commit.    |
| `gateway.yaml`          | ✅ yes     | nginx routing + API-key gate.                    |
| `tileserver.yaml`       | ✅ yes     | Tile server, PVC, initContainer downloader.      |
| `overpass.yaml`         | ✅ yes     | Overpass StatefulSet + PVC.                      |
| `ingress.yaml`          | ✅ yes     | Public host → gateway.                            |

## Prerequisites

- A cluster with an **nginx ingress controller** and a default **StorageClass**.
- (Optional) cert-manager for the `maps-tls` cert, or drop the `tls:` block for plain HTTP.

## Before you apply — edit these placeholders

1. **`secrets.yaml`** — separate real iOS-maps and backend-Overpass keys (`openssl rand -hex 24`).
2. **`tileserver.yaml`** — `TILES_URL` → your region's `.mbtiles`
   (from [OpenMapTiles](https://openmaptiles.org/) or [Planetiler](https://github.com/onthegomap/planetiler)),
   and `TILESERVER_PUBLIC_URL` → the same public host as the ingress with the
   required `/maps/` suffix.
3. **`overpass.yaml`** — `OVERPASS_PLANET_URL` / `OVERPASS_DIFF_URL` → your
   [Geofabrik](https://download.geofabrik.de/) region.
4. **`ingress.yaml`** — your `host` and TLS secret.
5. **PVC sizes + Overpass RAM** — scale to coverage (see table below).

### Sizing by coverage

| Coverage | tiles PVC | overpass PVC | overpass RAM | First import |
| -------- | --------- | ------------ | ------------ | ------------ |
| City     | ~2Gi      | ~5Gi         | 2Gi          | minutes      |
| Country  | ~20Gi     | ~50Gi        | 4–8Gi        | hours        |
| Planet   | 100Gi+    | 500Gi+ SSD   | 16Gi+        | many hours   |

## Apply

If upgrading an existing installation, migrate the ignored `secrets.yaml` from
the old `$api_key_valid` map to the two maps in `secrets.example.yaml` **before**
restarting or applying the new gateway. Otherwise nginx cannot resolve the new
variables and will not start.

Bring Overpass up first (long first import), then the rest:

```bash
cp secrets.example.yaml secrets.yaml    # then edit real keys
kubectl apply -f secrets.yaml

kubectl apply -f overpass.yaml
kubectl logs -f statefulset/overpass    # wait until it answers queries

kubectl apply -f tileserver.yaml -f gateway.yaml -f ingress.yaml
```

> After the first successful Overpass import, edit `overpass.yaml` and change
> `OVERPASS_MODE` from `init` to `clone` so a pod restart doesn't re-import.

## Key rotation

Edit `secrets.yaml`, then:

```bash
kubectl apply -f secrets.yaml
kubectl rollout restart deployment/gateway
```

## Client usage

```js
const BASE = 'https://maps.example.com';
const headers = { 'X-API-Key': 'ios-maps-key' };

// MapLibre GL — transformRequest attaches the key to EVERY tile/style request.
const map = new maplibregl.Map({
  container: 'map',
  style: `${BASE}/maps/styles/basic/style.json`,
  transformRequest: (url) => ({ url, headers }),
});

// Overpass is not called from iOS. The Railway backend uses its separate key.
```

Keep the key in the **`X-API-Key` header**, not a `?key=` query param — query
params leak into access logs.

`TILESERVER_PUBLIC_URL` must stay under `/maps/`. TileServer uses it for the
tile, glyph, and sprite URLs embedded in its responses; keeping those resources
on the configured public map host lets the iOS client scope its API-key header
to exactly that host.

## Verify

```bash
# YAML syntax validation is fully offline (run from server/k8s/):
ruby -ryaml -e 'ARGV.each { |path| YAML.load_stream(File.read(path)); puts "#{path}: ok" }' \
  secrets.example.yaml gateway.yaml tileserver.yaml overpass.yaml ingress.yaml

# Kubernetes schema/admission validation uses your current cluster:
kubectl apply --dry-run=server -f secrets.example.yaml -f gateway.yaml \
  -f tileserver.yaml -f overpass.yaml -f ingress.yaml

# On the cluster (positive and negative permission checks):
curl -H "X-API-Key: <ios-maps-key>" https://maps.example.com/maps/styles/basic/style.json   # 200
curl -i https://maps.example.com/maps/styles/basic/style.json                       # 401 (no key)
curl -i -H "X-API-Key: <backend-key>" https://maps.example.com/maps/styles/basic/style.json # 401
curl -H "X-API-Key: <backend-key>" --data 'data=[out:json];out;' \
  https://maps.example.com/overpass/api/interpreter                                  # 200
curl -i -H "X-API-Key: <ios-maps-key>" --data 'data=[out:json];out;' \
  https://maps.example.com/overpass/api/interpreter                                  # 401
curl -i https://maps.example.com/healthz                                             # 200
```

## Not included

- **Nominatim** (address search / geocoding) — separate service
  (`mediagis/nominatim`), own import + PVC. Add if you need geocoding.
