Commit all changes, push, and create a git tag.

## Steps

1. Run `git status` to see what changed. If there are no changes, stop and tell the user.
2. Run `git diff --staged` and `git diff` to understand the changes.
3. Run `git log --oneline -5` to see recent commit style.
4. Stage all relevant files (`git add` specific files — never use `git add -A`). Do NOT stage files that look like secrets (.env, credentials, keys).
5. Create a commit with a concise message summarizing the changes. Do NOT include "Co-Authored-By" lines.
6. Determine the next version tag:
   - Run `git tag --sort=-v:refname | head -5` to see existing tags.
   - If the user provided a version as argument, use that (e.g. `/release v1.2.3`).
   - Otherwise, analyze all changes since the last tag (`git log <last-tag>..HEAD --oneline`) and pick the appropriate semver bump:
     - **Major** (X.0.0): breaking API changes, incompatible protocol changes, removed features.
     - **Minor** (x.Y.0): new features, new commands, new flags, significant enhancements.
     - **Patch** (x.y.Z): bug fixes, documentation updates, dependency bumps, refactors with no behavior change.
   - If no tags exist, start at `v0.1.0`.
7. Create an annotated tag: `git tag -a <version> -m "<version>"`.
8. Push the branch and tag: `git push && git push --tags`.
9. Report the commit hash, tag, and remote status to the user.
