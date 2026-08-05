+++
title = 'Investigation worksheet'
description = 'Questions for investigating a Wheel before changing its code.'
weight = 1
+++

# Investigation worksheet

Copy these questions into your notes before starting a Wheel.

## Read the report

**What did users or operators observe?**

**What explanation does the report suggest without proving?**

**What important facts are still missing?**

**Which code are you responsible for, and which parts must you leave
unchanged?**

## List possible causes

**Possible cause 1:**

**Possible cause 2:**

**Possible cause 3, if needed:**

## Write down the ordering rules

**Which fields are compared, and in which direction?**

**How are ties or duplicate keys resolved?**

**Which function creates or checks the order?**

**Can the data change while the operation or page traversal is in progress?**

## Choose the next investigation step

Repeat these questions whenever you inspect another packet, test, benchmark,
or profile.

**What question should this step answer?**

**Why is it the most useful next step?**

**What result would support each remaining explanation?**

**What actually happened?**

**Which explanations became more or less likely?**

## Explain and repair the failure

**What is the smallest input that reproduces it?**

**Where does the program first behave incorrectly or do too much work?**

**Why does that happen?**

**What did you change?**

**Which tests check the repaired behavior?**

**What did the benchmark or profile add to the diagnosis?**

**What risk or unanswered question remains?**

## Three-minute handoff

Summarize what happened, the evidence that identified the cause, the change you
made, the tests or measurements you ran, and any follow-up work.
