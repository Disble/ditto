# Experiment — <the question, in one line>

Written before the measurement.

## Why this is being measured

What decision depends on the answer. If no decision depends on it, do not run it.

## Method

The fixture, and why it is a throwaway project rather than a checkout with work
in it. The control: what is removed so that a broken harness can be told apart
from a broken design. How rounds are ordered and which one is discarded.

## Predictions, and what kills each one

**H1 — <the claim in one sentence>.** <numeric prediction>.
*Falsified if <the exact result that kills it>.*

**H2 — <…>.** <numeric prediction>.
*Falsified if <…>.*

State which hypothesis decides the design, and what each outcome would mean.

## Decision rule, fixed in advance

- <outcome> → <what is done>
- <outcome> → <what is done>, including the outcome that closes the work entirely.

## Results

Raw per-round numbers. Exact counters and wall clock in separate columns; the
counters are what may be gated, the clock is reported only.

| Round | … | … | ratio |
| --- | --- | --- | --- |

### H1 <held / is dead>

The measured numbers against the prediction. If it died, say so first, then why
the original claim was wrong.

## What this does NOT establish

The limits, named explicitly: what was not covered, which probe was approximate,
and which question this result is silent about.
