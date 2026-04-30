# Backfill stress tests (CP-40824)

Verifies that Go GC laziness is the cause of backfill OOM kills, and that
setting `GOMEMLIMIT` fixes it.

Uses a fake Kubernetes clientset — no cluster, no Docker, no network.

## Usage

```sh
cd tests/stress/backfill

# Run both tests (baseline + with GOMEMLIMIT).
go test -v -count=1 -timeout 10m ./...

# Skip in CI / -short mode.
go test -short ./...

# Benchmark throughput.
go test -bench=. -benchtime=3x ./...
```

## What the tests do

- **TestBackfillMemoryBaseline** — 2000 namespaces × 10 pods, no
  `GOMEMLIMIT`. Reports peak heap, total allocation, and GC cycles.
- **TestBackfillMemoryWithLimit** — same workload with
  `GOMEMLIMIT=128MiB`. Verifies the GC keeps the heap in check.
- **BenchmarkBackfill** — smaller workload (500 × 5) for throughput
  measurement.

## Results (2026-04-29)

| Scenario     | Resources | GOMEMLIMIT | Peak Inuse | Total Alloc | NumGC | Duration |
| ------------ | --------- | ---------- | ---------- | ----------- | ----- | -------- |
| 2000ns × 10p | 22k       | none       | 169 MiB    | 7.5 GiB     | 150   | 4.4s     |
| 2000ns × 10p | 22k       | 128 MiB    | 121 MiB    | 7.5 GiB     | 781   | 7.3s     |
| 5000ns × 20p | 105k      | none       | 770 MiB    | 35.1 GiB    | 144   | 21s      |
| 5000ns × 20p | 105k      | 256 MiB    | 441 MiB    | 35.3 GiB    | 3187  | 81s      |

Total allocation is identical — there's no leak, just churn. `GOMEMLIMIT`
forces the GC to collect more aggressively, keeping the heap below the pod
memory limit at the cost of throughput.
