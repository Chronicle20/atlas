# atlas-messages
Mushroom game messages Service

## Overview

Atlas Messages is a service that handles character messages and commands in the Mushroom game. It processes various types of messages including general chat, whispers, multi-recipient messages, messenger messages, and pet messages. The service also provides a command system that allows Game Masters (GMs) to execute administrative commands through the chat interface.

## Features

- Message processing for different chat types
- Command system for Game Masters
- Character data management
- Integration with Kafka for event streaming
- Distributed tracing with Jaeger

## Installation

### Prerequisites

- Go 1.16 or higher
- Kafka
- Jaeger (for distributed tracing)

## Environment Variables

- `JAEGER_HOST` - Jaeger [host]:[port] for distributed tracing
- `LOG_LEVEL` - Logging level (Panic / Fatal / Error / Warn / Info / Debug / Trace)
- `BASE_SERVICE_URL` - [scheme]://[host]:[port]/api/
- `BOOTSTRAP_SERVERS` - Kafka [host]:[port]
- `COMMAND_TOPIC_CHARACTER` - Kafka Topic for transmitting character commands
- `COMMAND_TOPIC_CHARACTER_GENERAL_CHAT` - Kafka Topic for transmitting message commands
- `EVENT_TOPIC_CHARACTER_GENERAL_CHAT` - Kafka Topic for transmitting message events

## Available Commands

The service supports the following GM commands:

### Character Commands

- `@award <target> experience <amount>` - Awards experience to a character
  - Example: `@award me experience 1000` - Awards 1000 experience to yourself
  - Example: `@award map experience 500` - Awards 500 experience to all characters in the current map
  - Example: `@award PlayerName experience 2000` - Awards 2000 experience to the player named "PlayerName"

- `@award <target> <amount> level` - Awards levels to a character
  - Example: `@award me 5 level` - Awards 5 levels to yourself
  - Example: `@award map 2 level` - Awards 2 levels to all characters in the current map
  - Example: `@award PlayerName 10 level` - Awards 10 levels to the player named "PlayerName"

- `@change <target> job <jobId>` - Changes a character's job
  - Example: `@change my job 110` - Changes your job to jobId 110
  - Example: `@change PlayerName job 120` - Changes the job of the player named "PlayerName" to jobId 120

- `@award <target> meso <amount>` - Awards meso (currency) to a character
  - Example: `@award me meso 10000` - Awards 10000 meso to yourself
  - Example: `@award map meso 5000` - Awards 5000 meso to all characters in the current map
  - Example: `@award PlayerName meso 20000` - Awards 20000 meso to the player named "PlayerName"

### Inventory Commands

- `@award <target> item <itemId> [<quantity>]` - Awards items to a character
  - Example: `@award me item 2000000` - Awards 1 of item 2000000 to yourself
  - Example: `@award me item 2000000 10` - Awards 10 of item 2000000 to yourself
  - Example: `@award map item 2000000 5` - Awards 5 of item 2000000 to all characters in the current map
  - Example: `@award PlayerName item 2000000 3` - Awards 3 of item 2000000 to the player named "PlayerName"

### Skill Commands

- `@skill max <skillId>` - Maximizes a skill level for the character issuing the command
  - Example: `@skill max 1000` - Maximizes skill 1000 for yourself

- `@skill reset <skillId>` - Resets a skill level to 0 for the character issuing the command
  - Example: `@skill reset 1000` - Resets skill 1000 to level 0 for yourself

### Map Commands

- `@warp <target> <mapId>` - Warps a character to a specific map
  - Example: `@warp me 100000000` - Warps yourself to map 100000000
  - Example: `@warp map 100000000` - Warps all characters in the current map to map 100000000
  - Example: `@warp PlayerName 100000000` - Warps the player named "PlayerName" to map 100000000

- `@query map` - Shows the ID of the character's current map
  - Example: `@query map` - Displays your current map ID

## Message Types

The service handles the following types of messages:

1. **General Chat** - Messages sent to all players in the same map
2. **Whisper** - Private messages sent to a specific player
3. **Multi-recipient** - Messages sent to multiple specific players
4. **Messenger** - Messages sent through the messenger system
5. **Pet** - Messages from pets