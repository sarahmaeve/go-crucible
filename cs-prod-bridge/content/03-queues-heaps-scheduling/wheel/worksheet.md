+++
title = 'Investigation worksheet'
description = 'Questions for separating queue activity from useful progress before changing code.'
weight = 1
+++

# Investigation worksheet

Copy these questions into your notes before starting a Wheel.

## Read the report

**What did users or operators observe?**

**Which condition started the incident?**

**Which later condition prevented recovery?**

**Which code are you responsible for, and which parts must remain unchanged?**

## Describe the work lifecycle

**What is the logical work item?**

**Which state owns it now: queued, assigned, delayed, running, completed, or
terminal?**

**What event or time boundary permits the next transition?**

**Does another attempt repeat the same request, or can it select different
work?**

## List possible causes

**Possible cause 1:**

**Possible cause 2:**

**Possible cause 3, if needed:**

## Choose the next investigation step

Repeat these questions whenever you inspect another packet, test, counter, or
trace.

**What question should this step answer?**

**Why is it the most useful next step?**

**What result would support each remaining explanation?**

**What actually happened?**

**Which explanations became more or less likely?**

## Explain and repair the failure

**Which operation is repeated?**

**Can the condition needed for that operation to succeed still occur?**

**What useful capacity does each repeated attempt consume?**

**Which transition should replace the repetition, and who owns that
transition?**

**Which tests preserve genuinely recoverable retries?**

## Three-minute handoff

Summarize the trigger, the condition that prevented recovery, the evidence
that separated them, the state transition you changed, the verification you
ran, and one remaining risk.
