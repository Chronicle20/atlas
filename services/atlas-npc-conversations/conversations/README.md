# NPC Conversations

This directory contains converted NPC conversation files in JSON state machine format.

## File Naming Convention

Files should be named: `npc_{NPC_ID}.json`

Example: `npc_9201000.json` for NPC ID 9201000 (Moony)

## Schema

All conversation files must conform to the schema defined in:
`../docs/npc_conversation_schema.json`

## Conversion

To convert a JavaScript NPC script to this format, use:

```bash
/convert-npc path/to/script.js
```

Or paste the script content:

```bash
/convert-npc
<JavaScript code>
```

## Structure

Each conversation file contains:
- `npcId`: The NPC's ID
- `startState`: ID of the initial state
- `states`: Array of state machine states (dialogue, genericAction, craftAction, listSelection)
