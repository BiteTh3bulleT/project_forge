# PHASE 6 — Neuron Mesh Worker Teams

## Objective

Implement bounded worker teams using typed work orders, leases, validators, and propose-only outputs.

## Instructions

Add worker registry, capabilities, team types, in-process workers, budget/cancellation handling, typed artifacts, and tests.

## Validation

Workers cannot mutate canonical memory. Outputs are typed. Leases and scopes enforced.

## What not to do

Do not build generic chat agents. Do not allow unlimited subjobs. Do not bypass budgets.

## Exit gate

Exit when workers execute bounded memory jobs and emit non-canonical artifacts.

## Agent rule

Do not output implementation code in chat. Write code to files and summarize changed files, tests run, and remaining risks.
