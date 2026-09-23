# Verification report

Date: 2026-09-17. These are observed results for the delivered source, not a claim that CI has already run on GitHub.

| Check | Result | Evidence / limitation |
|---|---|---|
| Java 21 Maven reactor `verify` | Passed | Parent, common library and four service modules built successfully |
| Java unit tests | 5 passed | Product mapping/validation, reservation transitions/duplicate lines, shipment state guard |
| Java PostgreSQL context tests | 4 explicitly skipped | No Docker daemon in the creation environment |
| Executable Java JAR structure and CRC | Passed | All four JARs opened successfully after a clean isolated build |
| Go `go test -race -count=1 ./...` | Passed | Four unit-test functions across HTTP, inventory domain and scoring |
| Go `go vet ./...` | Passed | No diagnostics |
| Go `go build -buildvcs=false ./...` | Passed | All packages / three service executables compile |
| Go integration test compilation | Passed | Compiled with `-tags=integration -run '^$'`; tests were not executed |
| Docker Compose configuration | Passed | Validated using Compose v2.35.1 and temporary non-secret interpolation values |
| Static OpenAPI | Passed | OpenAPI 3.0 validator accepted all 34 paths |
| Python scripts | Passed syntax checks | Full demo/acceptance need the running services |
| Full Docker image build/start | Not executed | Docker daemon unavailable |
| PostgreSQL/Kafka integration and acceptance execution | Not executed | Docker daemon unavailable |
| GitHub Actions run | Not executed | Repository has not been published from this environment |

Logs: [Java](evidence/java-build.txt), [Go unit tests](evidence/go-tests.txt), [integration compilation](evidence/go-integration-compile.txt), [static checks](evidence/static-validation.txt).

The source includes real implementations and runnable tests; integration success must be established by running the supplied pipeline or these commands on a Docker-capable machine:

```bash
python3 scripts/init_env.py
docker compose up --build -d --wait --wait-timeout 600
python3 tests/acceptance.py
mvn -B verify
go test -race -tags=integration ./...
```

`go build -buildvcs=false` was used for local validation because this environment's surrounding workspace is not a normal Git checkout. This disables VCS metadata stamping, not compilation or tests. Maven verification was run from a clean temporary source copy to keep build output separate from synchronized workspace files. No test failures were hidden as successes; Docker-dependent Java tests report explicit skips.

The archive contains source, not locally built JARs, installed toolchains, caches, `.env` or database contents. `SHA256SUMS` records its source-file hashes.
