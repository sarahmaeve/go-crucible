# Unit 04: Build Dependency Graphs

Open the HTML guide for the foundations lesson, production examples, diagrams,
and source notes:

**[Open the build dependency graphs guide](../public/04-build-dependency-graphs/index.html)**

Use these files for the runnable work:

- [Lab: Build Graph Explorer](./lab/README.md)
- [Wheel of Misfortune scenarios](./wheel/README.md)

The lab stores target declarations in both directions. You will use the graph
to find paths with the fewest edges, report cycles, find ready work, and build
a dependency-first order. The three Wheels cover failures that can all produce
a bad plan but have different causes: a real cycle, a relationship the system
could not determine, and two different resources combined under one logical
name.

The examples are small Go programs. They do not read BUILD files or reproduce
Bazel, BuildKit, Prometheus, or Pulumi internals.
