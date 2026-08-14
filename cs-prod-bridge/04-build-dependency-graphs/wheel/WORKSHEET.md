# Unit 04 Investigation Worksheet

Copy these questions into your notes before starting a Wheel.

## Read the report

**What did users or operators observe?**

**Where in the system does the failure occur: graph construction, validation,
ordering, or execution?**

**What can you change? What must remain valid?**

## State the graph

**What does one node represent?**

**Complete the sentence `A -> B means ...`:**

**Which direction must this operation follow?**

**Which relationships or states does this graph leave out?**

## Draw the smallest relevant input

**Known nodes:**

**Edges found from declarations or analysis:**

**Unknown or undetermined relationships:**

**Distinct records that share a logical name:**

**Expected path, cycle report, or order:**

## List possible causes

**Possible cause 1:**

**Possible cause 2:**

**Possible cause 3, if needed:**

For each cause, write one result that supports it and one result that makes it
less likely.

## Choose the next investigation step

Repeat these questions before opening another packet, test, trace, or source
file.

**What question should this step answer?**

**Why is it the most useful next step?**

**What result would support each remaining cause?**

**What actually happened?**

**Which causes became more or less likely?**

## Check the proposed repair

**Does it keep a shared dependency that is not a cycle?**

**Does it keep “known to have none” separate from “unknown” or “not
determined”?**

**Does every edge join the intended individual records?**

**Does the operation require dependency-first, dependent-first, creation, or
deletion order?**

**Can input order or Go map order change a result that should stay the same?**

**Which ordinary tests protect valid behavior?**

**Which tagged test reproduces the reported failure without using wall-clock
timing?**

## Three-minute handoff

State what a node and an edge mean. Name the first incorrect edge or state.
Explain which evidence separated the possible causes, the repair, the checks
you ran, and one limit of the model.
