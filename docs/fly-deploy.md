# Fly Deployment Setup

This repo is configured to deploy to Fly.io automatically on pushes to `main`.

Workflow file:

- `.github/workflows/fly-deploy.yml`

## Required GitHub Settings

1. Add repository secret:

- `FLY_API_TOKEN`:
  create with `fly auth token` from an account that can deploy the target app.

2. Add repository variable:

- `FLY_APP_NAME`:
  your Fly app name (for example: `lingering-cherry-5692`).

## Fly Config

`fly.toml` is included and points to `deploy/Dockerfile`.

Before first deploy:

1. Update `fly.toml`:
- replace `app = "replace-with-your-fly-app-name"` with your real app name.

2. If needed, create the app:

```bash
fly apps create <your-app-name>
```

## Deployment Trigger

- Automatic: push to `main`
- Manual: GitHub Actions `workflow_dispatch`
