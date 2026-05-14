# api-key-config Specification

## Purpose

Secure storage and management of remote API keys (OpenAI, Anthropic, Groq) as alternative LLM backends, persisted via Go sidecar config rather than browser localStorage.

## Requirements

### Requirement: Secure API Key Storage

The system MUST store API keys securely via the Go sidecar's configuration file (config.json), NOT in browser localStorage. Keys MUST be stored per-provider with support for at least: `openai`, `anthropic`, `groq`.

#### Scenario: Store API key via sidecar

- GIVEN the user sets an API key via `/api openai sk-xxx`
- WHEN the key is saved
- THEN it is written to the Go sidecar's config.json, not to localStorage
- AND the key is associated with the `openai` provider

#### Scenario: Multiple provider keys

- GIVEN the user has set keys for `openai` and `anthropic`
- WHEN the config is read
- THEN both keys are present under their respective provider entries

### Requirement: Backend Switching on Key Set

The system MUST switch the active LLM backend from local Ollama to the specified remote provider when an API key is set. The system MUST revert to local Ollama when the API key for the active provider is cleared.

#### Scenario: Switch to remote backend

- GIVEN the app is currently using local Ollama
- WHEN the user sets an API key via `/api openai sk-xxx`
- THEN subsequent chat requests are sent to the OpenAI API instead of Ollama

#### Scenario: Revert to Ollama on key clear

- GIVEN the app is using OpenAI as the backend
- WHEN the user clears the OpenAI API key
- THEN subsequent chat requests revert to local Ollama

### Requirement: API Key Persistence

The system MUST persist API keys across application restarts. Keys MUST be loaded from config.json on sidecar startup.

#### Scenario: Keys survive restart

- GIVEN the user set an API key for `groq` and then closed the app
- WHEN the app restarts
- THEN the Groq API key is loaded from config.json and the backend is set to Groq

#### Scenario: No keys on first launch

- GIVEN this is the first launch with no config.json or no stored keys
- WHEN the app starts
- THEN the backend defaults to local Ollama

### Requirement: API Key Retrieval

The system MUST provide a JSON-RPC method `config.get` to retrieve the current configuration including the active provider and whether a key is set (without exposing the key value in responses).

#### Scenario: Get config without exposing key

- GIVEN the user has set an OpenAI API key
- WHEN `config.get` is called
- THEN the response includes `provider: "openai"` and `has_api_key: true` but NOT the actual key value
