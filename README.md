# Thought Processor

**Current Go TASS integration (November 6, 2026): experimental, not a released or owner-accepted build.** Thought Processor is the human/admin interface and Go Core for one local-first companion application. Thought Engine supplies the cognitive runtime; Identity supplies canonical authority; Development proposes/tests; Cognition is the **Core-hosted read-only** evidence view, not another service.

**Start at the [current stack and documentation map](core/docs/CURRENT_GO_STACK_20261106.md).** It identifies the exact pinned Core, Engine and Identity revisions, open conflict gates, links to the TASS mind source and Development prototype, and what is **not** implemented. The [Cognition status](core/docs/COGNITION_STATUS_20261106.md) distinguishes actual Engine records from visual demonstrations.

## Status and next acceptance

The integrated draft [PR #88](https://github.com/Arameair/Thought-Processor/pull/88) reports component/fixture tests but **not deployment or a real-Nemotron owner acceptance**. First verify installed versions and safe restore [#79](https://github.com/Arameair/Thought-Processor/issues/79); then test one approved real skill invocation, inspect it in Cognition and compare a controlled Brain graph change [#89](https://github.com/Arameair/Thought-Processor/issues/89). [Portability issues](https://github.com/Arameair/Thought-Processor/issues/90) and [cross-repo conflicts](https://github.com/Arameair/Thought-Processor/pull/91) remain open.

## How to run or inspect

For the **candidate only**, review [Core runbook](core/docs/OWNER_SESSION_FOUNDATION_20261105.md) and [launcher](core/tass.sh). `./tass.sh status` is a read-only inventory; install/update/rollback instructions are **not approved simply because they are documented**. On a fresh installation, Identity creates an entity; GitHub sync is optional. Do not expose the loopback API to a phone or the internet without a separate authentication design.

## Historical documents

This repository also contains earlier P0/P1 Processor launcher, Chat-Jimmy/Thought Controller, Atrium and Workshop experiments. Their README instructions, models and ports may be valid **for their own dated branches only**. They are not the current Go integrated TASS setup; retain them as Git history and experiment-specific documentation, not active startup advice.

**Safety:** no code, deployment, Identity grant, owner data or policy change is implied by these documentation links. Historical CI passes and fixture-model results are not real live tests.
