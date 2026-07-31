# Wheel of Misfortune Track

This track begins one step before ordinary debugging. Instead of receiving a
failing test and a complete contract, you receive an incomplete production
report containing observations, omissions, and an unproven theory. Your job is
to turn that report into a supported diagnosis before repairing the bounded
component.

The format is a tactical, code-focused adaptation of the SRE Wheel of
Misfortune. It is not incident-command training: assume that another responder
owns coordination, mitigation, and stakeholder communication unless the
scenario says otherwise.

## What This Track Trains

- Restating reported behavior without strengthening the report's claims
- Separating observations, interpretations, assumptions, and unknowns
- Asking a small number of high-information questions
- Maintaining competing hypotheses and identifying evidence that separates
  them
- Choosing focused commands and artifacts instead of inspecting everything
- Revising a theory when evidence contradicts it
- Distinguishing the visible symptom, local defect, and broader incident
- Making a restrained repair and verifying both the report and regressions
- Giving a concise handoff with appropriate certainty

The scenarios reuse numbered exercises. They add an investigation path, not
new Go subject matter.

## Production Plausibility

A Wheel scenario must center on a defect that could survive ordinary
development and activate under production conditions. Its maintainer record
must name the escape route—for example, a happy path that passes, a rare valid
configuration, a scale threshold, nondeterministic scheduling, or a conforming
replacement implementation that exposes a hidden assumption.

Pager language alone is not enough. The scenario should make it possible to
discover why the defect appeared now, why existing checks missed it, and which
test or delivery control would prevent recurrence.

## Session Format

Allow 35–45 minutes for a full session:

1. **Incoming report:** state what is observed, claimed, and missing. Do not
   inspect source yet.
2. **Clarification:** ask a prioritized set of questions that would change
   your next action.
3. **Evidence:** choose packets one at a time. Before opening each, write what
   outcomes would strengthen or weaken your hypotheses.
4. **Reproduction and localization:** define the expected contract, reproduce
   the symptom, and find the first local violation.
5. **Repair and verification:** make the narrowest justified change, run the
   focused reproduction, and check relevant neighboring behavior.
6. **Handoff:** summarize cause, evidence, repair, verification, uncertainty,
   and one preventive follow-up.

Use [WORKSHEET.template.md](./WORKSHEET.template.md) to record the
investigation. Afterward, assess the process—not just the patch—with
[RUBRIC.md](./RUBRIC.md).

## Facilitated and Self-Study Modes

With a facilitator, ask questions naturally. The facilitator has a prepared
truth table and should answer equivalent questions consistently without
steering you toward the repair.

For self-study, each scenario provides numbered evidence packets. Write down
the question you would ask and why its answer matters before opening the most
relevant packet. This is an honor system, like opening hints progressively.

Do not open the numbered exercise README, hints, tests, source, solution patch,
maintainer registry, or scenario debrief until the scenario directs you to do
so. Those materials are publicly accessible but collapse the report-first
part of the exercise.

## Scenarios

| # | Title | Tier | Reuses | Production trigger |
|---|---|---|---:|---|
| [W01](./01-replay-regression/REPORT.md) | Replay Regression | Intermediate | 20 | Cache-store migration during a retry burst |
