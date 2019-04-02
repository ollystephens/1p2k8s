# 1p2k8s
## One Password to Kubernetes Secrets

This tool extracts secrets from 1password and injects them into a kubernetes cluster.

It is far from finished. It's very much an early PoC. I wouldn't advise using it in anger just yet.

## Usage:

It expects the 1password CLI (`op`) to be on your path, and that you are signed in.

```bash
% eval $(op signin)
% 1p2k8s --vault Secrets --item PostgresAccount \
         --namespace dblayer --secret postgres-credentials
```
Extract key value pairs from the `PostgresAccount` item in the `Secrets` vault of 1password,
and create a secret called `postgres-credentials` in the namespace `dblayer` which contains them.