# Contributing to VCX

## Git Workflow

This project follows [GitHub Flow](https://docs.github.com/en/get-started/using-github/github-flow) with the following branch structure:

```
main       → Production-ready code (protected)
develop    → Integration branch (protected)
feature/*  → Feature branches
fix/*      → Bug fix branches
release/*  → Release preparation branches
```

### Branch Naming

| Type      | Pattern             | Example                       |
|-----------|---------------------|-------------------------------|
| Feature   | `feature/<name>`    | `feature/logging`             |
| Bug Fix   | `fix/<name>`        | `fix/hw-accel-detection`      |
| Release   | `release/v<X.Y.Z>`  | `release/v1.0.0`              |

### Development Flow

1. Create a branch from `develop`:
   ```bash
   git checkout develop
   git pull origin develop
   git checkout -b feature/my-feature
   ```

2. Make changes following [conventional commits](https://www.conventionalcommits.org/):
   ```bash
   git commit -m "feat: add logging system"
   git commit -m "fix: correct file path handling"
   ```

3. Push and create a Pull Request to `develop`

4. After approval, merge into `develop`

5. Release branches are created from `develop` and merged into `main`

## Commit Convention

All commits must follow the format:

```
<type>(<scope>): <description>

[optional body]
[optional footer]
```

**Types**: `feat`, `fix`, `docs`, `style`, `refactor`, `test`, `chore`, `ci`, `build`, `perf`

**Examples**:
```
feat: add OS-aware dependency checking
fix: handle filenames with spaces in subshell
docs: update README with installation guide
refactor: extract logging into separate module
```

## Setup Commit Hook

To enable commit message validation:

```bash
git config core.hooksPath .githooks
```

## Code Style

- Shell scripts: follow [Google Shell Style Guide](https://google.github.io/styleguide/shellguide.html)
- Use 4 spaces for indentation
- Functions documented with a comment block above
- Variables in uppercase for constants, lowercase for local
- Use `[[ ]]` instead of `[ ]` for tests
- Quote all variable expansions unless intentional

## Pull Request Process

1. Ensure all commits follow conventional commits format
2. Update documentation if applicable
3. Add or update tests if applicable
4. Request review from a maintainer
5. Squash merges are not used — preserve individual commits via merge commit
