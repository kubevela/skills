# Security Policy

This repository holds skills, client setup templates and scripts that AI agents follow
when they operate KubeVela. It ships no server and no binary, but its content steers
agents that hold cluster credentials, so it is handled under the same security policy as
the rest of the KubeVela project.

## What counts as a security issue here

- A skill, appendix or client template that leads an agent to mutate a cluster without
  the approval step and the write flag the skills require, to disclose secret values or
  customer data, or to bypass the access policy the skills quote.
- A script under a skill's `scripts/` or under `scripts/` that can be made to run
  arbitrary commands or exfiltrate data through the inputs it reads.
- Retrieved content (logs, annotations, definition descriptions, tool results) that a
  skill treats as instructions rather than as evidence.
- A pinned server version or configuration in `kubevela-sre/references/servers.yaml` or
  `clients/` that is known to be vulnerable or weakens the read-only guarantees.

Vulnerabilities in `vela mcp` itself or in `vela-core` belong to
[kubevela/kubevela](https://github.com/kubevela/kubevela/blob/master/SECURITY.md) and are
reported to the same address below.

## Reporting a Vulnerability

If you've found a security issue that you'd like to disclose confidentially please send
email to cncf-kubevela-security@lists.cncf.io. Do not open a public issue or pull request
for it.

### Who Reads Email Sent to cncf-kubevela-security@lists.cncf.io?

Only a restricted and carefully chosen group of KubeVela maintainers will have access to
material sent to the cncf-kubevela-security@lists.cncf.io address. No outside users can
subscribe to this list.

### What to Send to cncf-kubevela-security@lists.cncf.io

Name the skill, file and line, the client you used, and whether the agent ran in the MCP
or the CLI-assisted profile. Include the transcript excerpt that shows the unsafe step and
the exact text in the skill or in retrieved content that caused it. Redact secret values
and customer data before sending.

## How We Respond

Email sent to cncf-kubevela-security@lists.cncf.io is read and acknowledged with a
non-automated response within three working days. For issues that are complicated and
require significant attention, we will open an investigation and will provide you with a
mechanism to check the status of our progress at any time.
