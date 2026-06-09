# My Diagnosis — D03

Fill this in from the artifact alone, **before** opening any source
file. Then eliminate hypotheses against the code and compare with
`DIAGNOSIS_NOTES.md`.

## What the traceback proves

[ Statements you can defend from the artifact alone: where the panic
originated, what path it travelled, where recovery *should* have
intervened and observably did not. Be precise about what each frame's
line number means. ]

## What the absences prove

[ The artifact is as notable for what is missing as for what is
present. What does "zero `processor panicked` log lines, ever, with a
real logger attached" eliminate? What does the absence of any
recovery-related frame in the traceback tell you — and what does it
NOT tell you? ]

## Hypotheses, ranked

[ List the explanations that fit all the evidence, most likely first.
For each: what in the artifact supports it, and what single
observation in the source would confirm or kill it. ]

1.
2.
3.

## Mechanism (after reading the source)

[ Which hypothesis survived, and the precise mechanism — cite the
language rule at work. ]

## Proposed fix

[ The change you would make, in a sentence or a short snippet. ]

## Confidence and confirmation

[ How you would prove the fix beyond the exercise test — what log
line and what Result state should the reprocess team see on the next
malformed sample? ]
