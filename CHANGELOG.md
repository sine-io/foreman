# Changelog

## 2026-03-27

### Repository Transition

- Renamed the active project direction to `Foreman`
- Split the repository away from the older shell-runtime/tooling line
- Removed the old Codex shell-runtime packaging direction from active development

### Design and Planning

- Added the approved Go design spec:
  - [docs/superpowers/specs/2026-03-27-foreman-go-design.md](/root/link/repo/docs/superpowers/specs/2026-03-27-foreman-go-design.md)
- Added the approved Go Phase 1 plan:
  - [docs/superpowers/plans/2026-03-27-foreman-go-phase-1.md](/root/link/repo/docs/superpowers/plans/2026-03-27-foreman-go-phase-1.md)

### Go Bootstrap

- Added `Foreman` Go module bootstrap
- Added config/bootstrap/runtime seam
- Added `cobra` root command and `serve`
- Added `zerolog` setup
