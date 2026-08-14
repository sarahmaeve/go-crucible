+++
title = 'Investigation worksheet'
description = 'Questions for separating queue activity from completed work before changing code.'
weight = 1
+++

# Investigation worksheet

Copy these questions into your notes before starting a Wheel.

## Read the report

**What did users or operators observe?**

**Which condition started the incident?**

**Which later condition prevented recovery?**

**What can you change? What must stay unchanged?**

## Trace one work item

**What is the logical work item?**

**Which state owns it now: queued, assigned, delayed, running, completed, or
ended?**

**Which event or time permits the next state change?**

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

**Which state change should replace the repetition, and who owns that change?**

**Which tests preserve genuinely recoverable retries?**

## Three-minute handoff

Summarize the trigger, the condition that stopped recovery, the evidence that
separated them, the state change you made, the checks you ran, and one
remaining risk.
