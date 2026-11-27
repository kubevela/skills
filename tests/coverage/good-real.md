# widget: coverage and runtime verification

## Coverage

| Cloud API lever | CRD field | Definition parameter |
|---|---|---|
| Name | spec.name | name |

## Runtime verification

| Lever | Outcome | Evidence or reason |
|---|---|---|
| name | verified end to end | DescribeWidget |
| size, at creation and changed later | verified end to end | DescribeWidget |
| encryption | endpoint-limited | UpdateWidget returns `UnsupportedOperation: encryption is not emulated` |
| replicas | untested | needs a second region, and the emulator serves one |

Counts: 2 verified end to end, 1 endpoint-limited, 1 untested.

## Real-cloud verification

Run on 2026-11-25 against the widget service, us-east-1, through the widget controller 1.0.

| Lever | Outcome | Evidence or reason |
|---|---|---|
| name | verified end to end | DescribeWidget against the account |
| size large | endpoint-limited | not orderable in the account: DescribeOrderableWidgets lists small only |
| replicas | untested | one extra widget-hour per run; not in the budget |

Counts: 1 verified end to end, 1 endpoint-limited, 1 untested.

## What consumers should know

- Nothing.
