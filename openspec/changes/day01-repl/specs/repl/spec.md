# Delta for repl

## ADDED Requirements

### Requirement: Streaming multi-turn REPL

The agent SHALL provide a terminal REPL that streams model replies and preserves multi-turn context via an OpenAI-compatible `messages` history array.

#### Scenario: Continue after a reply
- **WHEN** the user sends a message and the model finishes streaming
- **THEN** the prompt `You ›` is shown again and the user can ask another question

#### Scenario: Multi-turn memory
- **GIVEN** the user said their name in a prior turn
- **WHEN** the user asks what their name is
- **THEN** the model answers using that prior turn from history

#### Scenario: Stream deltas
- **WHEN** the model generates content in chunks
- **THEN** each chunk is printed as it arrives without waiting for the full answer

### Requirement: Session slash commands

The agent SHALL support `/help`, `/reset`, and `/exit` slash commands in the REPL.

#### Scenario: Reset clears memory
- **WHEN** the user runs `/reset`
- **THEN** history is cleared and subsequent answers MUST NOT rely on prior turns

#### Scenario: Exit
- **WHEN** the user runs `/exit`
- **THEN** the process exits cleanly after a goodbye message

### Requirement: Model configuration from environment

The agent SHALL load `OPENAI_BASE_URL`, `OPENAI_API_KEY`, and `OPENAI_MODEL` from the environment and MUST exit with a clear error when the API key is missing.

#### Scenario: Missing API key
- **WHEN** `OPENAI_API_KEY` is empty or unset
- **THEN** the process exits with a non-zero status and prints a message telling the user to copy `.env.example` to `.env`
