# Tasks

## 1. Exhaustive domain enumeration

- [x] 1.1 `cjtr.ExhaustiveValues`: finite `one_of` list, or a decimal range
      with a declared `Places`, capped at a `maxCount` argument
- [x] 1.2 Tests: `one_of`, bounded decimal range, unbounded decimal
      (`HasRange` without `HasPlaces`), no domain at all, count exceeding
      the cap

## 2. Bounded proof

- [x] 2.1 `conformancecycle.BoundedProofInputs`: vary one input concept's
      own full enumerated domain at a time, baseline from the first
      worked example (mirrors `DirectedSamplingInputs`)
- [x] 2.2 `conformancecycle.CheckHypothesisBoundedProof`: run every case,
      same verdict aggregation as `CheckHypothesisDirectedSampling`
- [x] 2.3 Tests mirroring `directedsampling_test.go`'s own coverage

## 3. Complete proof

- [ ] 3.1 `conformancecycle.CompleteProofInputs`: full cross-product
      across every boundable input concept, capped at a `maxTotalCount`
- [ ] 3.2 `conformancecycle.CheckHypothesisCompleteProof`
- [ ] 3.3 Tests: full coverage when every concept is boundable; falls
      back (found=false) when even one concept is not; cap enforcement

## 4. Rigor-level selection and wiring

- [x] 4.1 Selection function: law tries complete -> bounded -> directed
      sampling, in that order; plain hypothesis always directed sampling
- [x] 4.2 Wire into `check.go` and `implement.go` in place of the direct
      `CheckHypothesisDirectedSampling` call
- [x] 4.3 `recordEvidence` records which level actually ran
- [x] 4.4 Full existing test suite passes unchanged (parity proof, same
      discipline as the expression-core wiring)

## 5. Decision record

- [x] 5.1 Write this proposal/design/tasks set documenting the rigor-level
      interpretation and its rationale, before writing code
- [ ] 5.2 Archive this change once merged
