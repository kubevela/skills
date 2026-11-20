#!/usr/bin/env python3
"""Summarise the AWS Emulator's fidelity report for the API operations a definition uses.

usage: emulator-fidelity.py --service dynamodb [--operations CreateTable,UpdateTable,...]
                            (--version 1.0.533 | --file scores.json[.gz])

AWS scores every release of the emulator against the live service and publishes
the result as scores.json.gz on that release. Read it before testing a lever: an
operation the report scores as failing is a known limit of the endpoint, to be
recorded as endpoint-limited, not something to discover at runtime.

--version downloads the report of that emulator version, which must be the
version the cluster runs, not the latest. --file reads a copy, for example
/opt/aws-emulator/release-info/scores.json.gz from the ghcr.io/kubevela/aws-emulator image.

Prints one line per operation: how many scored steps passed and how many did
not, then the scenarios that did not pass. Without --operations it prints every
operation of the service. Exits 0 when the report was read, 2 when it could not
be or when the service or an operation is not in it.
"""
import argparse
import collections
import gzip
import json
import sys
import urllib.request

URL = "https://github.com/aws/aws-emulator/releases/download/v{version}/scores.json.gz"


def load(args):
    if args.file:
        raw = open(args.file, "rb").read()
    else:
        with urllib.request.urlopen(URL.format(version=args.version), timeout=120) as response:
            raw = response.read()
    if raw[:2] == b"\x1f\x8b":
        raw = gzip.decompress(raw)
    return json.loads(raw)


def service_matches(wanted, services):
    # Entries look like "com.amazonaws.dynamodb#DynamoDB_20120810".
    wanted = wanted.lower()
    return any(s.lower().split("#")[0].rsplit(".", 1)[-1] == wanted for s in services)


def main():
    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    parser.add_argument("--service", required=True, help="AWS service name as in the SDK namespace, such as dynamodb, s3 or rds")
    parser.add_argument("--operations", default="", help="comma-separated API operations; default is all of the service")
    source = parser.add_mutually_exclusive_group(required=True)
    source.add_argument("--version", help="emulator version the cluster runs")
    source.add_argument("--file", help="a copy of scores.json or scores.json.gz")
    args = parser.parse_args()

    try:
        report = load(args)
    except Exception as err:  # noqa: BLE001 - any failure to read the report is the same outcome
        print(f"cannot read the fidelity report: {err}", file=sys.stderr)
        return 2

    units = [u for u in report.get("units", []) if service_matches(args.service, u.get("services", []))]
    if not units:
        print(f'no scenario for service "{args.service}" in the report', file=sys.stderr)
        return 2

    steps = collections.defaultdict(collections.Counter)
    failing = collections.defaultdict(list)
    for unit in units:
        detail = unit.get("detail") or {}
        seen = set()
        for step in detail.get("steps") or []:
            operation = step.get("operation")
            if not operation:
                continue
            passed = step.get("outcome") == "pass"
            steps[operation]["pass" if passed else "not pass"] += 1
            if not passed and operation not in seen:
                seen.add(operation)
                failing[operation].append(f'{unit.get("unitId")} ({step.get("mismatch") or step.get("outcome")})')

    wanted = [o for o in args.operations.split(",") if o] or sorted(steps)
    missing = [o for o in wanted if o not in steps]

    print(f'AWS Emulator fidelity for {args.service}, scored {report.get("scoredAt", "?")}: {len(units)} scenarios')
    for operation in wanted:
        if operation in missing:
            print(f"{operation}: not in the report")
            continue
        count = steps[operation]
        print(f'{operation}: {count["pass"]} passed, {count["not pass"]} did not')
        for scenario in failing[operation][:8]:
            print(f"  - {scenario}")
        if len(failing[operation]) > 8:
            print(f"  - and {len(failing[operation]) - 8} more")
    return 2 if missing else 0


if __name__ == "__main__":
    sys.exit(main())
