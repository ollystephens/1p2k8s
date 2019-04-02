# 1p2k8s
## One Password to Kubernetes Secrets

This tool extracts secrets from 1password and injects them into a kubernetes cluster.

## Usage:

```bash
% 1p2k8s --vault Secrets --item PostgresAccount \
         --namespace dblayer --secret postgres-credentials
```
Extract key value pairs from the `PostgresAccount` item in the `Secrets` vault of 1password,
and create a secret called `postgres-credentials` in the namespace `dblayer` which contains them.