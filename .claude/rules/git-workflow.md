# Git Workflow for db-go

## Commits
- Use Conventional Commits format: `type(scope): description`
- Types: `feat`, `fix`, `perf`, `refactor`, `docs`, `chore`, `style`, `test`
- Keep commit messages concise (imperative mood)
- Breaking changes: add `BREAKING CHANGE:` in commit body

## Branches
- Main branch: `master`
- Feature branches: `feat/short-description`
- Fix branches: `fix/short-description`

## CI/CD
- GitHub Actions runs Go build+test on push/PR to `master`
- semantic-release handles versioning and CHANGELOG.md automatically
- Do NOT manually edit CHANGELOG.md
