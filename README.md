# external-dns-cloudflare-tunnel-webhook

Read about how I implemented this [here](https://hsong.me/posts/creating-a-webhook-provider-for-external-dns/).

> [!WARNING]
> This provider is experimental

This is a provider for use with [external-dns](https://github.com/kubernetes-sigs/external-dns) via the webhook mechanism. It provides the ability to create public hostnames and backing DNS records for Cloudflare Tunnels.

> [!NOTE]
> Due to limitations of the external-dns webhook mechanism and my lack of brainpower, this provider only supports backing a single tunnel. To support more tunnels, deploy more instances of this provider.

## Deploying

You will need:

- A Kubernetes cluster
- Helm CLI installed
- A Cloudflare account with some form of authorization with scopes
  - All accounts - Cloudflare Tunnel:Edit
  - All zones - DNS:Edit

Ensure you have a secret with your Cloudflare credentials.

```shell
kubectl create secret generic cloudflare-credentials --from-literal=CLOUDFLARE_API_TOKEN=blah
```

Create a values file, see below for a minimum config.

```shell
cat <<EOF > ./values.yaml
logLevel: info
logFormat: json
interval: 1h
provider:
  name: webhook
  webhook:
    image:
      repository: docker.io/axatol/external-dns-cloudflare-tunnel-webhook
      tag: latest
    env:
      - name: CLOUDFLARE_API_TOKEN
        valueFrom:
          secretKeyRef:
            name: cloudflare-credentials
            key: CLOUDFLARE_API_TOKEN
EOF
```

Install the external-dns chart.

```shell
helm repo add external-dns https://kubernetes-sigs.github.io/external-dns/
helm repo update
helm upgrade external-dns-cloudflare-tunnel external-dns/external-dns \
  --install \
  --atomic \
  --create-namespace \
  --namespace external-dns \
  --values ./values.yaml
```

## Configuration

| Environment variable    | Type            | Default            | Notes |
| ----------------------- | --------------- | ------------------ | ----- |
| `LOG_LEVEL`             | `string`        | `"info"`           |       |
| `LOG_FORMAT`            | `string`        | `"json"`           |       |
| `CLOUDFLARE_API_KEY`    | `string`        | `""`               | ^1    |
| `CLOUDFLARE_API_EMAIL`  | `string`        | `""`               | ^1    |
| `CLOUDFLARE_API_TOKEN`  | `string`        | `""`               | ^1    |
| `CLOUDFLARE_ACCOUNT_ID` | `string`        |                    | ^2    |
| `CLOUDFLARE_TUNNEL_ID`  | `string`        |                    | ^2    |
| `CLOUDFLARE_SYNC_DNS`   | `bool`          | `"false"`          |       |
| `PORT`                  | `int64`         | `"8888"`           |       |
| `READ_TIMEOUT`          | `time.Duration` | `"5s"`             |       |
| `WRITE_TIMEOUT`         | `time.Duration` | `"10s"`            |       |
| `DRY_RUN`               | `bool`          | `"false"`          |       |
| `DOMAIN_FILTER`         | `[]string`      | `"" delimiter:","` | ^3    |

1. Must specify:
   - _both_ `CLOUDFLARE_API_KEY` and `CLOUDFLARE_API_EMAIL`
   - _or_ `CLOUDFLARE_API_TOKEN`
2. Required field
3. Specify multiple by delimiting with `,`
