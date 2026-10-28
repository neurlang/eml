# EML — Evolutionary Machine Learning

![EML GOLANG GOPHER IMAGE](elm.png)

EML is a **symbolic regression engine** written in Go. Instead of fitting numeric
coefficients, it *discovers entire mathematical formulas* that map inputs **x** to
outputs **y** — evolving human-readable expressions like `exp(exp(1)-log(1))` until
they reproduce observed data.

The core idea comes from the EML (Exp Minus Log) formulation, where every expression
is a tree of binary nodes `exp(left) - log(right)` built over the input variable and
the constant `1`.

[Whitepaper](https://arxiv.org/pdf/2603.21852)

---

## What It Does

Given one or more `(x, y)` training pairs, EML searches an enormous space of
expression trees for one that maps every `x` to its `y`. Scoring is by
**sum-of-squared error** (SOS), and the search is evolutionary:

1. Generate random expression trees.
2. Evaluate them against the samples.
3. Score them by error.
4. Mutate / grow / search for better formulas.
5. Keep the best.

Over time, expressions evolve into increasingly accurate representations of the data.

It's especially useful for:

- Symbolic regression problems
- Function discovery from sparse data
- Recovering known relationships (e.g. the successor function `x + 1`, constants like π)
- Exploring interpretable ML-style regression models

---

## Repository Layout

- `regression/` — the core library (`github.com/neurlang/eml/regression`): byte-encoded
  expression programs, randomized generation, and the threaded minimizer.
- `cmd/regressioneml/` — the CLI solver.
- `cmd/emlserver/` — a static web server that serves the WASM UI (`emlui/`).
- `cmd/emlserver/emlui/` — a browser frontend compiled to WebAssembly with the engine.

---

## Quick Start

### CLI solver

```sh
go build ./cmd/regressioneml/
```

Recover the successor function `f(x) = x + 1` from three samples:

```sh
./regressioneml -x 0 -y 1 -x 1 -y 2 -x 2 -y 3 -r 20 -a 20 -b 10000000
```

Output:

```
--- RESULTS ---
Final formula: exp(exp(1)-log(1)) - log(...)
Error Sum Of Squares: 0
Formula eval at 0: 1
Formula eval at 1: 2
Formula eval at 2: 3
```

Approximate π from a single point:

```sh
./regressioneml -x 0 -y 3.14159265359 -r 20 -a 20 -b 10000000
```

```
--- RESULTS ---
Final formula: exp(1) - log(...)
Error Sum Of Squares: ~1e-15
Formula eval at 0: 3.1415925957
```

### WASM UI + server

```sh
cd cmd/emlserver
GOOS=js GOARCH=wasm go build -o emlui.wasm ./emlui/
go build -o emlserver .
./emlserver          # serves the UI on :8080
```

Or run it in Docker:

```sh
docker compose up --build   # from cmd/emlserver
```

---

## CLI Flags

| Flag | Meaning | Default |
| --- | --- | --- |
| `-x value` | Input variable (repeatable) | — |
| `-y value` | Output variable (repeatable, must match `-x`) | — |
| `-r int` | **rho** — number of evolutionary rounds | `1` |
| `-a int` | **alpha** — function size per round | `1` |
| `-b int` | **beta** — brute-force iterations per round | `1` |
| `-g int` | **gamma** — consecutive stalled rounds before early stop | `2` |
| `-s int` | Random seed for reproducibility | random |
| `-t int` | Second seed component (thread) | `0` |
| `-json` | Emit JSON results instead of the CLI progress display | off |
| `-e name` / `-l name` | Display names for exp / log in formulas | `exp` / `log` |

During a run you see a progress bar and the current best formula; at completion the
final formula, its error sum of squares, and per-point evaluations are printed.

---

## How It Works

A candidate is a **byte-coded expression tree** (`Program`, a `[]byte`):

- `0` (`EXX`) — the input variable `x`
- `1` (`ONE`) — the constant `1`
- `2` (`EML`) — a binary node meaning `exp(left) - log(right)`

Evaluation pops opcodes off the front of the slice and recursively builds the
expression. The minimizer (`MinimizeRounds`) runs residual-fitting rounds in
parallel across CPU cores, pruning candidates against a shared global best score and
stopping early once several consecutive rounds produce no improvement.

---

## Design Philosophy

- **No coefficients are optimized** — the model structure itself is discovered.
- **Outputs are human-readable formulas**, often non-obvious but interpretable.
- Real and complex domains are supported in lockstep (`Problem` / `ProblemComplex`).

---

## Notes & Limitations

- Large `-b` values improve search quality but are computationally expensive.
- Increasing `-a` allows more complex expressions per round.
- More `-r` improves long-term convergence at the cost of runtime.
- Results vary with the random seed unless `-s` is fixed.
- Expressions can become extremely large; there is no guarantee of minimal or elegant
  formulas, and small datasets can overfit.

---

## Building & Testing

```sh
go test ./regression/        # unit + fuzz seed corpus
go build ./cmd/regressioneml/
```

CI (`1.22.x` through `latest`) runs the unit/fuzz tests and builds the CLI, WASM UI,
and server.

---

## License

[Apache-2.0](LICENSE)
