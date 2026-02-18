# Configuration Guide

APIM Core is powered by a declarative `config.yaml` file. This allows you to manage your entire API infrastructure without directly interacting with a frontend or database.

## Configuration File Structure

The `config.yaml` is divided into four main sections: `gateway`, `server`, `products`, and `subscriptions`.

### Gateway Section

Defines the network settings for the API Gateway (the proxy).

```yaml
gateway:
  listen: ":8080" # The address the gateway will listen on
```

### Server Section

Defines the network settings for the management server (Admin API, Dev Portal, Metrics).

```yaml
server:
  listen: ":8081"
```

### Products and APIs

This is the heart of your configuration. APIs are grouped into "Products" for logical management and access control.

```yaml
products:
  - name: "Educational Services"
    slug: "edu"
    description: "Core educational management APIs"
    apis:
      - name: "EduCore"
        path_prefix: "/educore"
        target_url: "http://localhost:8082"
      - name: "IntegraCore"
        path_prefix: "/integra"
        target_url: "http://localhost:8083"
```

- `slug`: A unique identifier for the product.
- `path_prefix`: Requests matching this prefix will be routed to the `target_url`.

### Subscriptions and Keys

Manage access to your products using pre-defined keys.

```yaml
subscriptions:
  - developer_id: "default-dev"
    product_slug: "edu" # Must match a product slug
    plan: "premium"
    keys:
      - name: "Default Key"
        value: "dev-key-123"
```

- `value`: The actual string used in the `X-Api-Key` header.

## Dynamic Hot-Reloading

One of the most powerful features of APIM Core is **Hot-Reloading**. You can edit `config.yaml` while the server is running, and the changes will be applied automatically within seconds without a restart.

Try changing a `target_url` or adding a new key, save the file, and observe the logs:
`config file changed, reloading...`

## Environment Variables

You can override certain settings via environment variables:

- `APIM_CONFIG`: Path to the config file (default: `config.yaml`).
- `APIM_GATEWAY_LISTEN`: Override `gateway.listen`.
- `APIM_SERVER_LISTEN`: Override `server.listen`.
