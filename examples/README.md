# Examples

Runnable examples demonstrating operator-go usage.

## Environment Variables

| Variable             | Description                                      |
| -------------------- | ------------------------------------------------ |
| `QUALITHM_API_TOKEN` | Member API token (`qmt_...`). Required.          |
| `QUALITHM_API_URL`   | Management API base URL. Defaults to production. |

## Running Examples

```bash
QUALITHM_API_TOKEN=qmt_... go run ./examples/basic_usage
```

## Example Files

| Example                            | Description                                |
| ---------------------------------- | ------------------------------------------ |
| [basic_usage](basic_usage/main.go) | List devices via the management API client |
