# Core agent rules — felicia

Hard DO/DON'T for all agents. Loaded every session.

## DO

- Use `make <target>` for all task execution (`make check`, `make test`, `make build`).
- Get runtimes from **mise** (`mise install`); system tools from the **nix flake**
  (`nix develop`, or let the Makefile's `NIX_RUN` wrap them). Never install tools globally.
- At session start, load asobi context (`asobi show`).
- At session end, save state to asobi (`asobi` session truths and obs); record non-obvious decisions as
  `felicia:decision:*` ADRs.
- Stay in the current phase: design → spec → TDD → implementation. Don't write application
  code while in research/design.
- Keep solutions simple; prefer small interfaces at I/O seams; keep `internal/domain` pure.

## DON'T

- DON'T commit or push without explicit user confirmation.
- DON'T use `--no-verify` (hook-enforced) or otherwise bypass `make check` / `make validate`.
- DON'T `rm -rf`; remove specific paths only.
- DON'T overwrite authored fields from the importer — re-import is field-scoped (design §5).
- DON'T use plan mode for small, well-scoped tasks.

## Tool provisioning

- **mise** — `mise install`, runtimes (go, bun) shimmed onto PATH.
- **nix** — `nix develop` for the system-tool shell; `make` targets prefix nix tools with
  `NIX_RUN` so they work in or out of the shell.
