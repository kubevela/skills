
### Description of your changes

<!--

Briefly describe what this pull request changes and which skill, appendix, client template
or script it touches. Unless you are a maintainer, link the open issue this pull request
resolves; a pull request without one may not be picked up. Use "Fixes #500", or
"Related to #500" when it completes only part of the issue so the issue stays open.

-->

Fixes #

I have:

- [ ] Read and followed [CONTRIBUTING.md](../CONTRIBUTING.md), including the rules for what belongs in a skill.
- [ ] Written commit subjects as lowercase conventional commits, for example `feat(kubevela-sre): ...` or `fix: ...`. The scope is optional; where I used one, it is the skill directory or the area touched.
- [ ] Signed off every commit (DCO) with `git commit -s`.
- [ ] Run `python3 scripts/validate_skills.py`, `bash tests/validator/test.sh` and `bash tests/consumers/test.sh` and they pass.
- [ ] Changed the access policy or the approval rule in every skill, every appendix, the README and `eval/README.md` together, or did not touch them.
- [ ] Given every new or refreshed file under `references/` a `Source:` line, or regenerated it with `scripts/snapshot-references.sh`.

### How has this change been exercised

<!--

A skill is a procedure an agent follows literally. Say which client you ran it in
(Claude Code, Codex, Cursor, Gemini CLI), in which profile (MCP, CLI-assisted or Offline
authoring), and what the agent did at each stopping point. For the MCP and CLI-assisted
profiles, also give the KubeVela version you ran against; an Offline authoring run has
none, so leave it out. If you did not run it, say so.

-->


### Special notes for your reviewer

<!--

Point your reviewers at anything that needs special consideration: a new stopping point,
a changed boundary between servers, a guarantee the CLI fallback loses.

-->
