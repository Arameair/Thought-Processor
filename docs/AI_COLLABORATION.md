# AI collaboration and verification

**Disclosure:** Thought Processor is a human-directed project developed with assistance from AI systems. This disclosure applies to the project's early design and documentation, including the initial repository foundation. Future code, tests, diagrams, or explanatory materials may also be AI-assisted; their origin and verification status should be documented rather than assumed.

## Roles and responsibility

- **Human owner:** defines the product vision, scope, architecture preferences, priorities, tests to accept, and whether experimental work is promoted to `alpha`, `stable`, or `main`.
- **AI assistants:** may help explore alternatives, compare designs, draft documentation, produce experimental changes, generate tests, and review proposed solutions. Their suggestions are not approvals, independent evidence, or a substitute for the owner's decisions.
- **Contributors:** should accurately acknowledge meaningful AI assistance in commit messages, PR/experiment notes, or related documentation where practical. Do not imply that every line in a file was AI-generated if that is not established; likewise, do not present AI-assisted work as entirely unaided.

## Verification policy

An AI-produced statement or code change must be treated like any other unverified contribution:

1. Record its purpose, affected files, experiment branch and exact commit.
2. Distinguish proposed behavior, implemented behavior, automated tests, and real-system observations.
3. Check against actual source code, running dependencies, logs and reproducible tests. Cite evidence when declaring success.
4. Record known limitations and failures instead of smoothing them over.
5. Require explicit owner acceptance for `alpha`, `stable`, and `main` promotions.

A model's claim that an integration works **is not evidence that it works**. In particular, documentation-only commits are not demonstrations of a runnable product.

## Attribution and limitations

This notice acknowledges AI collaboration; it does not assert legal authorship, copyright ownership, licensing rights, or that a particular AI provider endorses the project. It does not identify every individual prompt or tool used. The commit history and experiment notes should provide more specific provenance where available.

**Status updated October 27, 2026:** P0 is an owner-promoted runnable Alpha development baseline, while the unpromoted P1 cleanup branch has Claude-reported deterministic-fixture real-process integration and negative-test evidence. That is **not** an owner-accepted end-user product, a validated live AI system or a production security certification. See [current status](STATUS.md), [project case study](PROJECT_JOURNEY.md), and [credits](CREDITS.md).
