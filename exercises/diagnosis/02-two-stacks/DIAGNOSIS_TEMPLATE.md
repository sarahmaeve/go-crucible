# My Diagnosis — D02

Fill this in from the artifact alone, **before** opening any source
file. Then confirm against the code and compare with
`DIAGNOSIS_NOTES.md`.

## Signal

[ What in the report is the signal? What is racing with what, and
what does the shared address most likely refer to? ]

## Localization

[ File and line of the racing access(es), and the file and line where
the racing goroutines are created. Name the frames that took you
there. ]

## Mechanism

[ What you believe the racing line does, and why concurrent execution
of it corrupts the result. Bonus: explain the failing assertion's
"got 35" — why fewer, and why a multiple of 5? ]

## Proposed fix

[ The change you would make, in a sentence or a short snippet. ]

## Confidence and confirmation

[ Finding or hypothesis? What will you check first in the source —
and what would you grep the package for to be sure the same pattern
isn't repeated elsewhere? ]
