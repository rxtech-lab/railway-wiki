# iOS acceptance fixtures

Unit tests use a custom `URLProtocol`, while XCUITests start an in-process loopback HTTP fixture. Both paths verify requests without requiring a deployed Railway API or exposing a real map key.

Maestro flows use the same deterministic contract through `Tests/Fixtures/railway_mock_server.py`. Build and install the Debug app on the target simulator, then run:

```sh
./scripts/maestro-test.sh
```

The flow launch arguments override configured endpoints, map key, OAuth client ID, and auth token only for that launched process. Run the dashboard/resource and map/route flows in `.maestro` on an iPad simulator. CI runs `Tests/Maestro/compact-navigation.yaml` separately on iPhone, and the XCUITest target also exercises an orientation transition.
