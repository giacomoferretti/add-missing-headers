# Add Missing Headers

A Traefik plugin that adds missing HTTP headers to requests and responses without overriding existing headers. Supports both strict and loose header checking modes, explicit flushing control, and conditional bypass functionality.

**GitHub**: <https://github.com/giacomoferretti/add-missing-headers-traefik-plugin>

## Features

- ✅ Add missing request headers
- ✅ Add missing response headers
- ✅ Configurable header checking modes (strict/loose)
- ✅ Optional explicit flushing control
- ✅ Preserves existing headers (won't override)

## Configuration

### Static Configuration

#### YAML (traefik.yml)

```yaml
experimental:
  plugins:
    add-missing-headers:
      moduleName: github.com/giacomoferretti/add-missing-headers-traefik-plugin
      version: v0.1.1
```

#### TOML (traefik.toml)

```toml
[experimental.plugins]
  [experimental.plugins.add-missing-headers]
    moduleName = "github.com/giacomoferretti/add-missing-headers-traefik-plugin"
    version = "v0.1.1"
```

#### CLI

```bash
--experimental.plugins.add-missing-headers.modulename=github.com/giacomoferretti/add-missing-headers-traefik-plugin \
--experimental.plugins.add-missing-headers.version=v0.1.1
```

### Dynamic Configuration Examples

#### Basic Example

##### YAML

```yaml
http:
  middlewares:
    custom-headers:
      plugin:
        add-missing-headers:
          requestHeaders:
            X-Custom-RequestHeader: "CustomRequestValue"
          responseHeaders:
            X-Custom-ResponseHeader: "CustomResponseValue"
          # Bypass example
          bypassHeaders:
            X-Skip-Headers: ""
```

##### TOML

```toml
[http.middlewares]
  [http.middlewares.custom-headers]
    [http.middlewares.custom-headers.plugin]
      [http.middlewares.custom-headers.plugin.add-missing-headers]
        [http.middlewares.custom-headers.plugin.add-missing-headers.requestHeaders]
          X-Custom-RequestHeader = "CustomRequestValue"
        [http.middlewares.custom-headers.plugin.add-missing-headers.responseHeaders]
          X-Custom-ResponseHeader = "CustomResponseValue"
        # Bypass example
        [http.middlewares.custom-headers.plugin.add-missing-headers.bypassHeaders]
          X-Skip-Headers = ""
```

##### Docker Labels

```yaml
  labels:
    - "traefik.http.middlewares.custom-headers.plugin.add-missing-headers.requestHeaders.X-Custom-RequestHeader=CustomRequestValue"
    - "traefik.http.middlewares.custom-headers.plugin.add-missing-headers.responseHeaders.X-Custom-ResponseHeader=CustomResponseValue"
    - "traefik.http.middlewares.custom-headers.plugin.add-missing-headers.bypassHeaders.X-Skip-Headers="
```

## Configuration Options

| Option                 | Type                | Default | Description                                             |
| ---------------------- | ------------------- | ------- | ------------------------------------------------------- |
| `requestHeaders`       | `map[string]string` | `{}`    | Headers to add to incoming requests if missing          |
| `responseHeaders`      | `map[string]string` | `{}`    | Headers to add to outgoing responses if missing         |
| `strictHeaderCheck`    | `bool`              | `true`  | Header checking mode (see below)                        |
| `disableExplicitFlush` | `bool`              | `false` | Disable explicit flushing after response writes         |
| `bypassHeaders`        | `map[string]string` | `{}`    | Headers that bypass the middleware when present/matched |

### Bypass Headers

The `bypassHeaders` option allows you to completely skip the middleware when certain request headers are present or match specific values.

#### Bypass Modes

**Header Presence Check** - Set value to empty string `""`:

```yaml
bypassHeaders:
  X-Accel-Buffering: ""  # Bypass if header exists with any value
```

**Header Value Match** - Set specific value:

```yaml
bypassHeaders:
  X-Skip-Processing: "true"  # Bypass only if header equals "true"
```

#### Example Configuration

```yaml
http:
  middlewares:
    conditional-headers:
      plugin:
        add-missing-headers:
          # Normal header additions
          requestHeaders:
            X-Forwarded-Proto: "https"
          responseHeaders:
            Cache-Control: "max-age=3600"
          # Bypass conditions
          bypassHeaders:
            X-Accel-Buffering: ""        # Skip if present (any value)
            X-Skip-Headers: "true"       # Skip if exactly "true"
            X-Debug-Mode: "enabled"      # Skip if exactly "enabled"
```

##### TOML Format

```toml
[http.middlewares]
  [http.middlewares.conditional-headers]
    [http.middlewares.conditional-headers.plugin]
      [http.middlewares.conditional-headers.plugin.add-missing-headers]
        # Normal header additions
        [http.middlewares.conditional-headers.plugin.add-missing-headers.requestHeaders]
          X-Forwarded-Proto = "https"
        [http.middlewares.conditional-headers.plugin.add-missing-headers.responseHeaders]
          Cache-Control = "max-age=3600"
        # Bypass conditions
        [http.middlewares.conditional-headers.plugin.add-missing-headers.bypassHeaders]
          X-Accel-Buffering = ""        # Skip if present (any value)
          X-Skip-Headers = "true"       # Skip if exactly "true"
          X-Debug-Mode = "enabled"      # Skip if exactly "enabled"
```

##### Docker Labels Format

```yaml
labels:
  # Normal header additions
  - "traefik.http.middlewares.conditional-headers.plugin.add-missing-headers.requestHeaders.X-Forwarded-Proto=https"
  - "traefik.http.middlewares.conditional-headers.plugin.add-missing-headers.responseHeaders.Cache-Control=max-age=3600"
  # Bypass conditions
  - "traefik.http.middlewares.conditional-headers.plugin.add-missing-headers.bypassHeaders.X-Accel-Buffering="
  - "traefik.http.middlewares.conditional-headers.plugin.add-missing-headers.bypassHeaders.X-Skip-Headers=true"
  - "traefik.http.middlewares.conditional-headers.plugin.add-missing-headers.bypassHeaders.X-Debug-Mode=enabled"
```

### Header Checking Modes

#### Strict Mode (`strictHeaderCheck: true`) - Default

- Only adds headers if they **don't exist at all**
- Respects explicitly set empty headers
- More precise and safer behavior
- Example: Won't override `Content-Type: ""`

#### Loose Mode (`strictHeaderCheck: false`)

- Adds headers if they **don't exist OR are empty**
- Will overwrite explicitly empty headers  
- More aggressive header replacement
- Example: Will override `Content-Type: ""` with configured value
