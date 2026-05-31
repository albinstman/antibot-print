# Install the antibot skill

Paste this file to an agent (e.g. Claude Code) and it will install the
`detecting-antibot-vendors` skill so the agent can use the `antibot` CLI.

---

Install the `detecting-antibot-vendors` agent skill for me. Steps:

1. Create the directory `~/.claude/skills/detecting-antibot-vendors/` (use
   `$HOME`, not a project-local `.claude/`, so the skill is available in every
   project).

2. Download the skill into it:

   ```sh
   mkdir -p ~/.claude/skills/detecting-antibot-vendors
   curl -fsSL https://raw.githubusercontent.com/albinstman/antibot-print/main/SKILL.md \
     -o ~/.claude/skills/detecting-antibot-vendors/SKILL.md
   ```

   If you already have this repo checked out locally, copy its `SKILL.md`
   instead of downloading:

   ```sh
   cp /path/to/antibot-print/SKILL.md ~/.claude/skills/detecting-antibot-vendors/SKILL.md
   ```

3. Verify the file exists and starts with YAML frontmatter containing
   `name: detecting-antibot-vendors`:

   ```sh
   head -5 ~/.claude/skills/detecting-antibot-vendors/SKILL.md
   ```

4. Tell me the skill is installed. It loads on the next session (or after
   `/reload` if the agent supports it). The skill itself will install the
   `antibot` binary on first use if it is not already present.
