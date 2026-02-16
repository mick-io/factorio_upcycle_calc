# Versioning Guide

This project should use **Semantic Versioning (SemVer)** and **Git tags** for releases.

## Version Source of Truth

Create a root `VERSION` file and keep the current release version there.

Example:

```text
1.4.0
```

## Branch and Release Flow

1. Develop on `feature/*` branches.
2. Merge features into `dev`.
3. Create a release PR from `dev` to `main`.
4. After merge, tag the release on `main`.

## Tagging a Release

```bash
git checkout main
git pull
git tag -a v1.4.0 -m "Release v1.4.0"
git push origin v1.4.0
```

## Container Image Tagging

For each release, publish image tags:

1. Exact version tag: `v1.4.0`
2. Minor stream tag: `v1.4`
3. `latest` (optional)

## Changelog

Maintain `CHANGELOG.md` and update it in the release PR.

Recommended format: Keep a Changelog style.

## SemVer Rules

- **MAJOR**: breaking changes
- **MINOR**: backward-compatible features
- **PATCH**: backward-compatible bug fixes
