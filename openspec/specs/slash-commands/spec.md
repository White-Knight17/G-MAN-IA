# slash-commands Specification

## Purpose

Frontend input parser that intercepts slash-prefixed text and dispatches to command handlers, with backend RPC support for model and configuration operations.

## Requirements

### Requirement: Slash Command Parsing

The system MUST intercept input starting with `/` in the chat input field and parse it as a command rather than sending it to the AI. Command parsing MUST be case-insensitive. If the input matches a known command, the system MUST execute it; if not, it MUST display a helpful error message listing available commands.

#### Scenario: Valid command execution

- GIVEN the chat input field is focused
- WHEN the user types `/clear` and presses Enter
- THEN the command is executed (chat history cleared) and no message is sent to the AI

#### Scenario: Case-insensitive command

- GIVEN the chat input field is focused
- WHEN the user types `/HELP` or `/Help` and presses Enter
- THEN the `/help` command executes and displays available commands

#### Scenario: Unknown command error

- GIVEN the chat input field is focused
- WHEN the user types `/unknown` and presses Enter
- THEN an error message appears: "Unknown command. Type `/help` for available commands."

#### Scenario: Normal text with leading slash not a command

- GIVEN the chat input field is focused
- WHEN the user types `/etc/config` (not a known command) and presses Enter
- THEN an error message appears suggesting it is not a valid command

### Requirement: /model Command

The system MUST support `/model` to list available Ollama models and show the currently active model. The system MUST support `/model <name>` to switch the active model to the specified name.

#### Scenario: List models with current active

- GIVEN the user types `/model` and presses Enter
- WHEN the command executes
- THEN a list of available Ollama models appears with the current active model highlighted or marked

#### Scenario: Switch model

- GIVEN the user types `/model llama3` and presses Enter
- WHEN the command executes and the model exists
- THEN the active model switches to `llama3` and a confirmation message appears

#### Scenario: Switch to non-existent model

- GIVEN the user types `/model nonexistent` and presses Enter
- WHEN the command executes and the model does not exist
- THEN an error message appears: "Model 'nonexistent' not found. Use `/models <name>` to pull it."

### Requirement: /api Command

The system MUST support `/api <provider> <key>` to store an API key for a remote backend provider. Supported providers MUST include: `openai`, `anthropic`, `groq`.

#### Scenario: Set OpenAI API key

- GIVEN the user types `/api openai sk-xxx...` and presses Enter
- WHEN the command executes
- THEN the API key is stored securely, the backend switches to OpenAI, and a confirmation appears

#### Scenario: Set Anthropic API key

- GIVEN the user types `/api anthropic sk-ant-xxx...` and presses Enter
- WHEN the command executes
- THEN the API key is stored securely, the backend switches to Anthropic, and a confirmation appears

#### Scenario: Invalid provider

- GIVEN the user types `/api invalid key123` and presses Enter
- WHEN the command executes
- THEN an error message appears: "Unknown provider 'invalid'. Supported: openai, anthropic, groq."

### Requirement: /clear Command

The system MUST support `/clear` to clear the current chat history from the UI.

#### Scenario: Clear chat history

- GIVEN the chat has multiple messages
- WHEN the user types `/clear` and presses Enter
- THEN all messages are removed from the chat view and the input field is cleared

### Requirement: /models Command (Ollama Pull)

The system MUST support `/models <name>` (alias: `/ollamamodel <name>`) to trigger `ollama pull <name>` with streaming progress feedback to the user.

#### Scenario: Pull a model with progress

- GIVEN the user types `/models codellama` and presses Enter
- WHEN the pull starts
- THEN a progress indicator appears in the chat showing download percentage and status
- AND on completion, a success message appears: "Model 'codellama' is ready."

#### Scenario: Pull already-existing model

- GIVEN the user types `/models llama3` and the model already exists locally
- WHEN the command executes
- THEN a message appears: "Model 'llama3' is already available."

### Requirement: /help Command

The system MUST support `/help` to display all available slash commands with brief descriptions.

#### Scenario: Display help

- GIVEN the user types `/help` and presses Enter
- WHEN the command executes
- THEN a message appears listing: `/model`, `/models`, `/api`, `/clear`, `/help` with one-line descriptions
