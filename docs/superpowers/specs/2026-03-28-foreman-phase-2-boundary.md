# Foreman Phase 2 Boundary

Date: 2026-03-28
Status: approved from interactive architecture review

## Summary

Foreman remains the local embedded control plane, not the primary human-facing manager agent.

Phase 2 must preserve a strict layer boundary:

- upstream manager agents own human-facing PM experience, chat/session routing, and client protocol bridges
- Foreman owns repo execution truth, task governance, approvals, leases, runs, artifacts, and board projections
- downstream runners execute work and report state back into Foreman

This boundary is intended to prevent Foreman from re-implementing gateway, ACP, or session-routing responsibilities that already belong in systems such as OpenClaw.

## Core Decision

Foreman should not grow into a replacement for OpenClaw, Nanobot, or ZeroClaw.

Foreman should instead operate beneath those systems as the local execution control plane:

- manager agent decides what work should happen
- Foreman decides what actually happened in the repo runtime
- runner executes work
- Foreman persists the result as source of truth

## Roles by System

### OpenClaw

OpenClaw remains responsible for:

- gateway sessions
- channel routing
- human-facing PM interaction
- ACP bridge and ACP-facing integrations

When Foreman is used with OpenClaw:

- ACP should stay in OpenClaw's layer
- OpenClaw should call Foreman through a Foreman-native adapter or gateway interface
- Foreman should not re-implement OpenClaw's ACP bridge as part of its core product

### Nanobot and ZeroClaw

Nanobot and ZeroClaw remain upstream manager-agent entrypoints.

When Foreman is used with them:

- they keep their own manager / orchestration role
- Foreman provides the missing repo-governance and execution-truth layer
- Foreman should not attempt to absorb their gateway, daemon, or manager UX responsibilities

### Direct Foreman Entry

Foreman can still expose its own local CLI and board UI.

Those remain valid secondary entrypoints for:

- self-hosted usage
- debugging
- direct local control
- users who want the control plane without a separate manager agent

This does not change the product definition: direct Foreman entry is secondary, not the primary PM experience.

## ACP Boundary

ACP is a client / manager / IDE protocol concern, not the core of Foreman's control-plane identity.

Phase 2 rule:

- do not build a first-class ACP stack into Foreman unless a specific thin adapter is required

If ACP support is needed later, it should be implemented as:

- a narrow adapter
- mapping Foreman state and commands into ACP-compatible session semantics
- without making ACP the internal model of the system

## Source of Truth Rule

Foreman is the source of truth for:

- projects
- modules
- tasks
- approvals
- leases
- runs
- artifacts
- board state

Manager agents are not the source of truth for repo execution state.

Manager agents may hold:

- conversational context
- user intent
- planning context
- routing/session context

But durable execution truth must resolve to Foreman storage, not chat history or gateway session memory.

## What Foreman Must Not Rebuild

Phase 2 should avoid rebuilding the following inside Foreman core:

- multi-channel gateway infrastructure
- chat transport adapters as first-class PM channels
- ACP bridge functionality already provided by OpenClaw
- manager-agent session ownership
- human-facing PM orchestration UX as the primary product mode

If any of these appear necessary, they should be justified as thin adapters around Foreman's control-plane surface, not as a new product direction.

## What Foreman Should Continue Building

Phase 2 should continue deepening Foreman's actual product layer:

- richer task and module governance
- stronger approval and lease policy
- better artifact indexing and summaries
- deeper runner lifecycle handling
- improved board usability
- cleaner integration contracts for upstream manager agents

## Integration Patterns

Recommended integration patterns:

### Pattern A: OpenClaw over Foreman

- user talks to OpenClaw
- OpenClaw manages channel/session/protocol concerns
- OpenClaw sends normalized work requests into Foreman
- Foreman dispatches runners and persists truth
- OpenClaw reads status from Foreman

### Pattern B: Nanobot or ZeroClaw over Foreman

- user talks to Nanobot or ZeroClaw
- upstream manager agent owns interaction and orchestration
- Foreman handles execution governance and persistence

### Pattern C: Direct Foreman

- user interacts with Foreman CLI / board directly
- Foreman acts as both local control plane and local operator surface
- this is supported, but remains secondary in the product definition

## Phase 2 Product Constraint

Any Phase 2 proposal should answer this question before implementation:

> Is this making Foreman a better control plane, or is it pulling Foreman upward into gateway / PM / protocol territory?

If the change primarily improves:

- project truth
- execution governance
- runner control
- approvals
- board state
- integration clarity

then it likely belongs in Foreman.

If the change primarily improves:

- chat UX
- session routing
- transport protocols
- ACP bridging
- manager-agent conversational orchestration

then it likely belongs outside Foreman core.
