# RegressionEML

A symbolic regression engine that discovers mathematical expressions mapping inputs (**x**) to outputs (**y**) using evolutionary-style search over expression trees.

Instead of fitting coefficients, `regressioneml` searches for **entire formulas** composed of functions like `exp`, `log`, arithmetic composition, and nested transformations.

---

## Overview

`regressioneml` attempts to reconstruct a hidden function from sample points by evolving candidate expressions and scoring them against observed data.

It is especially useful for:

* Symbolic regression problems
* Function discovery from sparse data
* Recovering known mathematical relationships (e.g., successor function, constants like π)
* Exploring interpretable ML-style regression models

---

## Usage

```
./regressioneml -x <value> -y <value> [options]
```

You provide one or more `(x, y)` training pairs.

---

## Arguments

### Data Inputs

* `-x value`
  Input variable(s). Can be repeated.

* `-y value`
  Output variable(s). Must match corresponding `-x` entries.

---

### Optimization Controls

* `-r int`
  **rho (rounds)** – number of evolutionary iterations
  Default: `1`

* `-a int`
  **alpha (function size per round)** – controls expression growth per iteration
  Default: `1`

* `-b int`
  **beta (brute forcing iterations)** – search intensity per round
  Default: `1`

* `-s int`
  Random seed for reproducibility

---

### Output Modes

* `-json`
  Output results in JSON format instead of CLI progress display

---

## Output Format

During execution:

```
<error> [progress bar] <%> <current best formula>
```

At completion:

```
--- RESULTS ---
Final formula: <expression>
Error Sum Of Squares: <value>
Formula eval at x=...: ...
```

---

## Examples

### 1. Successor Function Learning

Recovering:

```
f(x) = x + 1
```

#### Command:

```
./regressioneml -x 0 -y 1 -x 1 -y 2 -x 2 -y 3 -r 20 -a 20 -b 10000000
```

#### Result:

```
Final formula: exp(exp(1)-log(1)) - log(...)
Error Sum Of Squares: 0
Formula eval at 0: 1
Formula eval at 1: 2
Formula eval at 2: 3
```

---

### 2. π Approximation from Single Point

#### Command:

```
./regressioneml -x 0 -y 3.14159265359 -r 20 -a 20 -b 10000000
```

#### Result:

```
Final formula: exp(1) - log(...)
Error Sum Of Squares: ~1e-15
Formula eval at 0: 3.1415925957
```

---

## How It Works (Conceptual)

The system:

1. Initializes random expression trees
2. Evaluates them against `(x, y)` samples
3. Scores them via error (sum of squares)
4. Mutates / grows expressions using:

   * expansion (`alpha`)
   * search depth (`beta`)
   * iteration cycles (`rho`)
5. Selects best-performing symbolic formulas

Over time, expressions evolve into increasingly accurate mathematical representations of the dataset.

---

## Design Philosophy

Unlike numeric regression:

* No coefficients are directly optimized
* The model structure itself is discovered
* Outputs are human-readable formulas
* Solutions are often non-obvious but interpretable

---

## Notes

* Large `-b` values significantly increase search quality but are computationally expensive
* Increasing `-a` allows more complex expressions per round
* More `-r` improves long-term convergence at the cost of runtime
* Results may vary due to stochastic search unless `-s` is fixed

---

## Limitations

* Can produce extremely large symbolic expressions
* May overfit small datasets
* Computational cost grows quickly with search depth
* No guarantee of minimal or elegant expressions

---

## Example Use Cases

* Function discovery in scientific data
* Reverse engineering unknown formulas
* Educational exploration of symbolic regression
* Benchmarking evolutionary search systems

---

## Future Ideas

* Expression pruning / simplification layer
* Operator weighting (favoring simpler math)
* GPU-accelerated evaluation
* Caching of subexpressions
* Multi-objective optimization (accuracy vs simplicity)

