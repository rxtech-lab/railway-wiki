# Railway Wiki for iOS

The app targets iOS 26 and adapts between an iPad `NavigationSplitView` and a compact `NavigationStack`. It uses MapLibre Native for the protected OpenStreetMap tile stack, RxAuthSwift for OAuth, and the backend JSON Schema contract for management forms.

## Configuration

Copy `Config/Secrets.example.xcconfig` to both `Config/Secrets.Debug.xcconfig` and `Config/Secrets.Release.xcconfig`, then supply environment-specific values. The secret files are ignored by Git.

- `OVERPASS_API_BASE_URL` must point to the authenticated Railway backend proxy, never directly to the Kubernetes Overpass service.
- `OSM_MAPS_API_KEY` must be a maps-only gateway key. MapLibre sends it only to the configured OSM host.
- Release builds intentionally have no endpoint defaults and show a configuration error if a required value is missing or malformed.

## Verification

Open `iOS.xcodeproj` and run the `iOS` scheme, or use `xcodebuild` with an iOS 26 simulator. The pinned SwiftLint build-tool plugin runs with every app build. CI resolves that same pinned artifact for strict linting, runs the unit and UI targets on representative iPad and iPhone simulators, and executes Maestro on both the iPad and compact iPhone layouts.

Maestro uses a deterministic local fixture and an already installed Debug build:

```sh
./scripts/maestro-test.sh
```

See `Tests/README.md` for fixture details and launch overrides.
