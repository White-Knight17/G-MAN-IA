# Delta for jsonrpc-transport

## ADDED Requirements

### Requirement: Model Management RPC Methods

The system MUST provide JSON-RPC method `model.list` that returns available Ollama models and the currently active model. The system MUST provide JSON-RPC method `model.pull` that triggers `ollama pull <name>` with streaming progress notifications via `SendNotification`.

#### Scenario: List available models

- GIVEN the frontend sends `{"jsonrpc":"2.0","method":"model.list","id":1}`
- WHEN the sidecar processes the request
- THEN it queries Ollama `/api/tags` and returns `{"jsonrpc":"2.0","result":{"models":["llama3","codellama"],"active":"llama3"},"id":1}`

#### Scenario: Pull model with streaming progress

- GIVEN the frontend sends `{"jsonrpc":"2.0","method":"model.pull","params":{"name":"codellama"},"id":2}`
- WHEN the pull is in progress
- THEN the sidecar sends notification events like `{"jsonrpc":"2.0","method":"pull.progress","params":{"name":"codellama","percent":45}}`
- AND on completion sends `{"jsonrpc":"2.0","method":"pull.complete","params":{"name":"codellama"}}`
- AND finally returns `{"jsonrpc":"2.0","result":{"status":"success"},"id":2}`

#### Scenario: Pull non-existent model

- GIVEN the frontend sends `{"jsonrpc":"2.0","method":"model.pull","params":{"name":"nonexistent"},"id":3}`
- WHEN Ollama reports the model does not exist
- THEN the sidecar returns `{"jsonrpc":"2.0","error":{"code":-32001,"message":"model 'nonexistent' not found on registry"},"id":3}`

### Requirement: Configuration RPC Methods

The system MUST provide JSON-RPC method `config.set_api_key` that stores an API key for a given provider (`openai`, `anthropic`, `groq`) in the sidecar's config.json. The system MUST provide JSON-RPC method `config.get` that returns the current configuration including active provider and whether a key is set (without exposing the key value).

#### Scenario: Set API key

- GIVEN the frontend sends `{"jsonrpc":"2.0","method":"config.set_api_key","params":{"provider":"openai","key":"sk-xxx"},"id":4}`
- WHEN the sidecar processes the request
- THEN the key is written to config.json under the `openai` provider entry
- AND the response is `{"jsonrpc":"2.0","result":{"status":"ok"},"id":4}`

#### Scenario: Get config hides key value

- GIVEN an API key has been set for `openai`
- WHEN the frontend sends `{"jsonrpc":"2.0","method":"config.get","id":5}`
- THEN the response includes `{"provider":"openai","has_api_key":true}` but NOT the actual key value

#### Scenario: Set invalid provider

- GIVEN the frontend sends `{"jsonrpc":"2.0","method":"config.set_api_key","params":{"provider":"invalid","key":"xxx"},"id":6}`
- WHEN the sidecar processes the request
- THEN it returns `{"jsonrpc":"2.0","error":{"code":-32002,"message":"unsupported provider: invalid"},"id":6}`
