#!/usr/bin/env python3
"""Check the runtime verification record of a cloud-service definition module.

usage: check-coverage.py <docs/coverage.md> [--cue <generated definition.cue>]

The record is the table under the "## Runtime verification" heading:

    | Lever | Outcome | Evidence or reason |

Every row must have exactly one of three outcomes, and the outcomes that are not
a pass must say why:

    verified end to end   the third cell names the API call that read it back
    endpoint-limited      the third cell quotes the error or names the limitation
    untested              the third cell names what blocks the test

Placeholders such as "not exercised yet", "later" or "TODO" are rejected:
nothing blocks such a lever, so the fix is a test, not a table row. The check
knows a fixed list of placeholders; whether another reason is a real blocker
is for the reviewer to judge. The line that starts with
"Counts:" must agree with the table.

With --cue, every top-level parameter of the generated definition must be named
in the Lever column, so a parameter cannot ship without an outcome.

A module that was also run against a real cloud account keeps a second section,
"## Real-cloud verification", in the same format. When it is present it is checked
by the same rules, except that it may cover a subset of the parameters: the
emulator table stays the complete record.

Exits 0 when the record is complete, 1 with one line per problem otherwise.
"""
import argparse
import re
import sys

OUTCOMES = ("verified end to end", "endpoint-limited", "untested")
EMULATOR_SECTION = "Runtime verification"
REAL_CLOUD_SECTION = "Real-cloud verification"
# What people write when nothing actually blocks the test.
NOT_A_REASON = re.compile(
    r"^\W*((not (yet )?(exercised|tested|run|done|covered)\b.*)|((not yet|todo|tbd|later|pending|n/?a|none|unknown)\W*)|)$",
    re.IGNORECASE,
)


def cells(line):
    # Split on "|" but not on the escaped "\|" a cell may contain.
    line = line.strip()
    if line.startswith("|"):
        line = line[1:]
    if line.endswith("|") and not line.endswith("\\|"):
        line = line[:-1]
    return [c.strip().replace("\\|", "|") for c in re.split(r"(?<!\\)\|", line)]


def section_rows(text, heading):
    """Rows of the first table under the given "## <heading>" section."""
    section = re.search(r"^##\s+" + re.escape(heading) + r"\s*$(.*?)(?=^##\s|\Z)", text, re.MULTILINE | re.DOTALL | re.IGNORECASE)
    if not section:
        return None, ""
    body = section.group(1)
    rows, in_table = [], False
    for line in body.splitlines():
        if line.lstrip().startswith("|"):
            in_table = True
            row = cells(line)
            if set("".join(row)) <= set("-: "):
                continue
            rows.append(row)
        elif in_table:
            break
    return rows, body


def check_table(rows, body, label):
    """Problems with one record table, and its counts and lever names."""
    problems = []
    if len(rows) < 2:
        return [f"the {label} section has no table rows"], None, []
    header = [h.lower() for h in rows[0]]
    if len(header) < 3 or header[0] != "lever" or header[1] != "outcome":
        problems.append(f'{label}: the table header must be "| Lever | Outcome | Evidence or reason |"')

    counts = dict.fromkeys(OUTCOMES, 0)
    levers = []
    for row in rows[1:]:
        lever = row[0] if row else ""
        levers.append(lever)
        if len(row) != 3:
            problems.append(f'{label}: "{lever}": the row has {len(row)} cells, not 3; a "|" inside a cell must be written as "\\|"')
            continue
        outcome = row[1].lower()
        reason = row[2]
        if outcome not in OUTCOMES:
            problems.append(f'{label}: "{lever}": outcome "{row[1]}" is not one of: {", ".join(OUTCOMES)}')
            continue
        counts[outcome] += 1
        if NOT_A_REASON.match(reason):
            what = {
                "verified end to end": "name the API call that read it back",
                "endpoint-limited": "quote the error or name the limitation",
                "untested": "name what blocks the test, or write the test",
            }[outcome]
            problems.append(f'{label}: "{lever}": {outcome} without a reason ("{reason}"): {what}')

    stated = re.search(
        r"^Counts:\s*(\d+)\s+verified end to end,\s*(\d+)\s+endpoint-limited,\s*(\d+)\s+untested", body, re.MULTILINE | re.IGNORECASE
    )
    if not stated:
        problems.append(f'{label}: no line "Counts: N verified end to end, N endpoint-limited, N untested"')
    else:
        for outcome, number in zip(OUTCOMES, stated.groups()):
            if int(number) != counts[outcome]:
                problems.append(f"{label}: Counts says {number} {outcome}, the table has {counts[outcome]}")
    return problems, counts, levers


def top_level_parameters(cue):
    """Names declared directly inside the `parameter: {` block of generated CUE."""
    # [ \t]*, not \s*: the indent must stay on the line of `parameter:`.
    match = re.search(r"^([ \t]*)parameter:[ \t]*\{[ \t]*$", cue, re.MULTILINE)
    if not match:
        return []
    indent = match.group(1)
    names = []
    for line in cue[match.end():].splitlines():
        if line.startswith(indent + "}"):
            break
        field = re.match(r"^" + re.escape(indent) + r"\t([A-Za-z_][A-Za-z0-9_]*)[?!]?:", line)
        # A name starting with "_" is a hidden field, such as a validator block.
        if field and not field.group(1).startswith("_") and field.group(1) not in names:
            names.append(field.group(1))
    return names


def summary(counts):
    return ", ".join(f"{counts[o]} {o}" for o in OUTCOMES)


def main():
    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    parser.add_argument("coverage")
    parser.add_argument("--cue")
    args = parser.parse_args()

    text = open(args.coverage, encoding="utf-8").read()
    rows, body = section_rows(text, EMULATOR_SECTION)
    if rows is None:
        print(f'FAIL no "## {EMULATOR_SECTION}" section')
        return 1
    problems, counts, levers = check_table(rows, body, EMULATOR_SECTION)
    if counts is None:
        print("FAIL " + problems[0])
        return 1

    real_rows, real_body = section_rows(text, REAL_CLOUD_SECTION)
    real_counts = None
    if real_rows is not None:
        real_problems, real_counts, _ = check_table(real_rows, real_body, REAL_CLOUD_SECTION)
        problems.extend(real_problems)

    if args.cue:
        named = " ".join(levers)
        for parameter in top_level_parameters(open(args.cue, encoding="utf-8").read()):
            if not re.search(r"(?<![A-Za-z0-9_])" + re.escape(parameter) + r"(?![A-Za-z0-9_])", named):
                problems.append(f'parameter "{parameter}" has no row in the {EMULATOR_SECTION} table')

    for problem in problems:
        print("FAIL " + problem)
    if problems:
        return 1
    line = "OK " + summary(counts)
    if real_counts is not None:
        line += "; real cloud: " + summary(real_counts)
    print(line)
    return 0


if __name__ == "__main__":
    sys.exit(main())
