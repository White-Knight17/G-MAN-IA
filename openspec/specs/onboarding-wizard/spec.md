# onboarding-wizard Specification

## Purpose

First-run wizard for configuring G-MAN: AI backend selection, workspace directory grants, and theme preference. Skips if config exists; re-triggerable from settings.

## Requirements

### Requirement: Onboarding Wizard

The system MUST present a 3-step wizard on first launch when `~/.config/gman/config.json` is absent. Step 1 SHALL let the user choose between Ollama (local) and API key backend, and validate connectivity and model availability. Step 2 SHALL present allowed directories as checkboxes (default: `~/.config`). Step 3 SHALL offer theme selection (dark/light/system). The wizard MUST save the completed config to `~/.config/gman/config.json`. If config already exists, the wizard SHALL be skipped. A "Re-run setup wizard" button in settings SHALL clear config and restart the wizard.

#### Scenario: First launch with Ollama ready

- GIVEN no `config.json` exists, Ollama is running, and `deepseek-r1:1.5b` is pulled
- WHEN the app launches for the first time
- THEN the wizard starts at Step 1; user selects Ollama; wizard verifies connectivity and confirms model available; user proceeds to Step 2 and selects directories; user proceeds to Step 3 and picks dark theme; config.json is saved with `{"backend":"ollama","model":"deepseek-r1:1.5b","directories":["~/.config"],"theme":"dark"}`

#### Scenario: Ollama not installed

- GIVEN no `config.json` exists and Ollama is not reachable
- WHEN the user selects Ollama backend and clicks Next
- THEN the wizard displays "Ollama is not installed" with the install command `curl -fsSL https://ollama.com/install.sh | sh` and a Retry button; user must resolve before proceeding

#### Scenario: Ollama running but no model

- GIVEN Ollama is running but no models are pulled
- WHEN the wizard validates model availability
- THEN it shows "No models found" with a "Pull model" button that runs `ollama pull deepseek-r1:1.5b` and reports progress

#### Scenario: Re-trigger from settings

- GIVEN `config.json` exists and the app is running normally
- WHEN the user opens Settings and clicks "Re-run setup wizard"
- THEN a confirmation dialog appears; on confirm, config.json is deleted and the app restarts into the full wizard
