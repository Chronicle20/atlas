# Cosmetic System Documentation

## Overview

The Atlas NPC Conversations service implements a comprehensive cosmetic change system for character appearance modifications (hair, face, skin). This system integrates with the saga orchestrator to ensure transactional integrity and event-driven coordination across microservices.

## Architecture

### System Flow

```
┌─────────────────────┐
│ NPC Conversation    │ 1. Player interacts with NPC
│ (atlas-npc-         │    Generates available styles
│  conversations)     │    Player selects or random picks
└──────────┬──────────┘
           │ 2. Emits Saga Command (Kafka)
           ▼
┌─────────────────────┐
│ Saga Orchestrator   │ 3. Creates saga with cosmetic step
│ (atlas-saga-        │    Routes to character processor
│  orchestrator)      │
└──────────┬──────────┘
           │ 4. Emits Character Command (Kafka)
           ▼
┌─────────────────────┐
│ Character Service   │ 5. Updates database (transaction)
│ (atlas-character)   │    Captures old cosmetic value
│                     │    Emits status events
└──────────┬──────────┘
           │ 6. Emits Status Events (Kafka)
           ▼
┌─────────────────────┐
│ Channel Service     │ 7. Broadcasts to client
│ (atlas-channel)     │    Player sees new cosmetic
└─────────────────────┘
```

## Components

### 1. State Types

#### askStyle

A dedicated state type for presenting cosmetic style selection interfaces.

**Structure**:
```json
{
  "id": "selectHairStyle",
  "type": "askStyle",
  "askStyle": {
    "text": "Choose your new hairstyle!",
    "stylesContextKey": "availableHairStyles",
    "contextKey": "selectedHair",
    "nextState": "checkHaircutCoupon"
  }
}
```

**Fields**:
- `text` (string, required): Prompt shown to player
- `stylesContextKey` (string, required): Context key containing the styles array (populated by local operations)
- `contextKey` (string, required): Context key where selected style ID will be stored
- `nextState` (string, required): Next state to transition to after selection

**Prerequisites**: The styles array must be populated in context before reaching an askStyle state, typically using `local:generate_hair_styles`, `local:generate_hair_colors`, or `local:generate_face_styles` operations.

### 2. Local Operations

These operations execute within the atlas-npc-conversations service without saga coordination.

#### local:generate_hair_styles

Generates available hair styles for the character based on configuration.

**Parameters**:
- `baseStyles` (string, required): Comma-separated list of hair base IDs (e.g., "30060,30140,30200")
- `genderFilter` (string, optional): Set to "true" to filter styles by character gender
- `preserveColor` (string, optional): Set to "true" to preserve character's current hair color
- `validateExists` (string, optional): Set to "true" to validate styles exist in WZ data
- `excludeEquipped` (string, optional): Set to "true" to exclude character's current hair
- `outputContextKey` (string, required): Context key to store the generated styles array

**Example**:
```json
{
  "type": "local:generate_hair_styles",
  "params": {
    "baseStyles": "30060,30140,30200,30210,30310,33040",
    "genderFilter": "true",
    "preserveColor": "true",
    "validateExists": "true",
    "excludeEquipped": "true",
    "outputContextKey": "availableHairStyles"
  }
}
```

#### local:generate_hair_colors

Generates available hair color variations for the character's current hairstyle.

**Parameters**:
- `colors` (string, required): Comma-separated list of color IDs (0-7)
- `validateExists` (string, optional): Set to "true" to validate colors exist
- `excludeEquipped` (string, optional): Set to "true" to exclude current color
- `outputContextKey` (string, required): Context key to store the generated colors array

**Example**:
```json
{
  "type": "local:generate_hair_colors",
  "params": {
    "colors": "0,1,2,3,4,5,6,7",
    "validateExists": "true",
    "excludeEquipped": "true",
    "outputContextKey": "availableHairColors"
  }
}
```

#### local:generate_face_styles

Generates available face styles for the character based on configuration.

**Parameters**:
- `baseStyles` (string, required): Comma-separated list of face base IDs (e.g., "20000,20001,20002")
- `genderFilter` (string, optional): Set to "true" to filter styles by character gender
- `validateExists` (string, optional): Set to "true" to validate styles exist in WZ data
- `excludeEquipped` (string, optional): Set to "true" to exclude character's current face
- `outputContextKey` (string, required): Context key to store the generated styles array

**Example**:
```json
{
  "type": "local:generate_face_styles",
  "params": {
    "baseStyles": "20000,20001,20002,20003,20004",
    "genderFilter": "true",
    "validateExists": "true",
    "excludeEquipped": "true",
    "outputContextKey": "availableFaceStyles"
  }
}
```

#### local:select_random_cosmetic

Randomly selects a cosmetic style from a pre-populated styles array.

**Parameters**:
- `stylesContextKey` (string, required): Context key containing the styles array
- `outputContextKey` (string, required): Context key to store the selected style ID

**Example**:
```json
{
  "type": "local:select_random_cosmetic",
  "params": {
    "stylesContextKey": "availableHairStyles",
    "outputContextKey": "selectedHair"
  }
}
```

**Use Case**: Random cosmetic NPCs like Brittany (NPC 1012104) that apply random styles instead of player selection.

### 3. Saga Operations

These operations create saga steps that coordinate distributed transactions.

#### change_hair

Changes the character's hair style via the saga orchestrator.

**Parameters**:
- `styleId` (string, required): Hair style ID (range: 30000-35000). Can use context references like `{context.selectedHair}`

**Example**:
```json
{
  "type": "change_hair",
  "params": {
    "styleId": "{context.selectedHair}"
  }
}
```

**Saga Flow**:
1. Creates `ChangeHair` saga action
2. Emits Kafka command to atlas-character
3. Character service updates database in transaction
4. Emits `HairChanged` and `StatChanged` events
5. Channel service broadcasts to client

#### change_face

Changes the character's face style via the saga orchestrator.

**Parameters**:
- `styleId` (string, required): Face style ID (range: 20000-25000). Can use context references like `{context.selectedFace}`

**Example**:
```json
{
  "type": "change_face",
  "params": {
    "styleId": "{context.selectedFace}"
  }
}
```

#### change_skin

Changes the character's skin color via the saga orchestrator.

**Parameters**:
- `styleId` (string, required): Skin color ID (range: 0-9). Can use context references like `{context.selectedSkin}`

**Example**:
```json
{
  "type": "change_skin",
  "params": {
    "styleId": "{context.selectedSkin}"
  }
}
```

## Implementation Patterns

### Pattern 1: Player-Selected Cosmetic

Used for NPCs where players choose their desired cosmetic (e.g., Hair Salon).

**Example Flow** (NPC 1012103 - Head Hair Salon):

```json
{
  "states": [
    {
      "id": "prepareHaircutStyles",
      "type": "genericAction",
      "genericAction": {
        "operations": [{
          "type": "local:generate_hair_styles",
          "params": {
            "baseStyles": "30060,30140,30200,30210",
            "genderFilter": "true",
            "preserveColor": "true",
            "validateExists": "true",
            "excludeEquipped": "true",
            "outputContextKey": "haircutStyles"
          }
        }],
        "outcomes": [{
          "conditions": [],
          "nextState": "selectHaircut"
        }]
      }
    },
    {
      "id": "selectHaircut",
      "type": "askStyle",
      "askStyle": {
        "text": "Choose your new hairstyle!",
        "stylesContextKey": "haircutStyles",
        "contextKey": "selectedHair",
        "nextState": "checkCoupon"
      }
    },
    {
      "id": "checkCoupon",
      "type": "genericAction",
      "genericAction": {
        "operations": [],
        "outcomes": [
          {
            "conditions": [{
              "type": "item",
              "operator": ">=",
              "value": "1",
              "referenceId": "5150001"
            }],
            "nextState": "applyHaircut"
          },
          {
            "conditions": [],
            "nextState": "noCoupon"
          }
        ]
      }
    },
    {
      "id": "applyHaircut",
      "type": "genericAction",
      "genericAction": {
        "operations": [
          {
            "type": "destroy_item",
            "params": {
              "itemId": "5150001",
              "quantity": "1"
            }
          },
          {
            "type": "change_hair",
            "params": {
              "styleId": "{context.selectedHair}"
            }
          }
        ],
        "outcomes": [{
          "conditions": [],
          "nextState": "success"
        }]
      }
    }
  ]
}
```

### Pattern 2: Random Cosmetic

Used for NPCs that apply random cosmetic changes (e.g., Random Hair Coupon NPCs).

**Example Flow** (NPC 1012104 - Brittany):

```json
{
  "states": [
    {
      "id": "prepareStyles",
      "type": "genericAction",
      "genericAction": {
        "operations": [{
          "type": "local:generate_hair_styles",
          "params": {
            "baseStyles": "30060,30140,30200",
            "genderFilter": "true",
            "preserveColor": "true",
            "validateExists": "true",
            "excludeEquipped": "true",
            "outputContextKey": "availableStyles"
          }
        }],
        "outcomes": [{
          "conditions": [],
          "nextState": "confirmRandom"
        }]
      }
    },
    {
      "id": "confirmRandom",
      "type": "dialogue",
      "dialogue": {
        "dialogueType": "sendYesNo",
        "text": "Your hair will change RANDOMLY. Continue?",
        "choices": [
          {
            "text": "Yes",
            "nextState": "selectRandom"
          },
          {
            "text": "No",
            "nextState": null
          },
          {
            "text": "Exit",
            "nextState": null
          }
        ]
      }
    },
    {
      "id": "selectRandom",
      "type": "genericAction",
      "genericAction": {
        "operations": [{
          "type": "local:select_random_cosmetic",
          "params": {
            "stylesContextKey": "availableStyles",
            "outputContextKey": "selectedHair"
          }
        }],
        "outcomes": [{
          "conditions": [{
            "type": "item",
            "operator": ">=",
            "value": "1",
            "referenceId": "5150000"
          }],
          "nextState": "applyRandom"
        }]
      }
    },
    {
      "id": "applyRandom",
      "type": "genericAction",
      "genericAction": {
        "operations": [
          {
            "type": "destroy_item",
            "params": {
              "itemId": "5150000",
              "quantity": "1"
            }
          },
          {
            "type": "change_hair",
            "params": {
              "styleId": "{context.selectedHair}"
            }
          }
        ],
        "outcomes": [{
          "conditions": [],
          "nextState": "success"
        }]
      }
    }
  ]
}
```

## Rollback and Error Handling

### Saga Compensation

The saga orchestrator includes compensation logic for failed cosmetic changes:

**Location**: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/compensator.go`

**Current Implementation**:
- Acknowledges compensation without reverting cosmetic changes
- Character retains the new cosmetic even if subsequent saga steps fail
- Logs the compensation attempt for audit purposes

**Rationale**:
- Cosmetic changes are non-critical (similar to `CreateCharacter` action)
- Partial rollback is acceptable for cosmetics
- Full rollback would require capturing old values in saga payload (future enhancement)

**Future Enhancement**:
To support full rollback, the system would need to:
1. Capture the character's current cosmetic value before creating the saga
2. Store the old value in the saga payload or metadata
3. In compensation, issue a reverse cosmetic change operation with the old value

### Error Scenarios

1. **Coupon Missing**: Conversation validates coupon before applying change
2. **Invalid Style ID**: Character service validates style exists before updating
3. **Saga Failure**: Compensator marks step as compensated, character keeps new cosmetic
4. **Database Error**: Transaction rolls back, no cosmetic change applied

## Example NPCs

### NPC 1012103 - Head Hair Salon

**Features**:
- Player-selected haircuts
- Player-selected hair colors
- Coupon validation (Regular Haircut Coupon #5150001#, Hair Color Coupon #5151001#)
- VIP coupon support (#5420002#)

**States**: 14 total
- Welcome screen with service selection
- Hair style generation and selection
- Hair color generation and selection
- Coupon validation
- Cosmetic application
- Success/failure messages

### NPC 1012104 - Brittany (Assistant)

**Features**:
- Random haircuts (REG coupon #5150000#)
- Random haircuts (EXP coupon #5150010#)
- Random hair colors (#5151000#)
- Confirmation prompts

**States**: 22 total
- Service selection
- Style generation for each coupon type
- Random selection
- Coupon validation
- Cosmetic application
- Success/failure messages

## Testing

### Unit Tests

Test local operations independently:
- `local:generate_hair_styles` with various configurations
- `local:generate_hair_colors` with color validation
- `local:generate_face_styles` with gender filtering
- `local:select_random_cosmetic` with different array sizes

### Integration Tests

Test complete cosmetic change flows:
1. Player selects style → Coupon consumed → Hair changed → Events emitted
2. Random selection → Coupon consumed → Hair changed → Events emitted
3. Missing coupon → Rejection message → No change
4. Saga failure → Compensation triggered → Character keeps cosmetic

### Manual Testing Scenarios

1. **Player Selection Flow**:
   - Talk to NPC 1012103
   - Select haircut service
   - Verify available styles match configuration
   - Select a style
   - Verify coupon is consumed
   - Verify hair changes in client

2. **Random Selection Flow**:
   - Talk to NPC 1012104
   - Select random haircut (REG)
   - Confirm random change
   - Verify coupon is consumed
   - Verify hair changes to random style

3. **Edge Cases**:
   - No styles available (all filtered out)
   - Already have all styles
   - Missing required coupon
   - Invalid style ID in context

## Migration Notes

### Deprecated Operations

- `local:apply_cosmetic`: **REMOVED** in favor of saga operations (`change_hair`, `change_face`, `change_skin`)

### Breaking Changes

None. Existing conversations using the old local:apply_cosmetic operation have been migrated.

### Migration Path

For NPCs using deprecated operations:

**Before** (deprecated):
```json
{
  "type": "local:apply_cosmetic",
  "params": {
    "cosmeticType": "hair",
    "styleId": "{context.selectedHair}"
  }
}
```

**After** (current):
```json
{
  "type": "change_hair",
  "params": {
    "styleId": "{context.selectedHair}"
  }
}
```

## References

### Related Services

- **atlas-saga-orchestrator**: Handles distributed transactions
  - `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/model.go` - Saga actions
  - `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/handler.go` - Action handlers
  - `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/compensator.go` - Rollback logic

- **atlas-character**: Manages character state
  - `services/atlas-character/atlas.com/character/character/processor.go` - Cosmetic change handlers
  - `services/atlas-character/atlas.com/character/character/producer.go` - Event producers
  - `services/atlas-character/atlas.com/character/kafka/consumer/character/consumer.go` - Command consumers

- **atlas-channel**: Broadcasts to clients
  - Consumes character status events
  - Sends cosmetic change packets to client

### Schema

- `services/atlas-npc-conversations/docs/npc_conversation_schema.json` - JSON schema for conversations

### Conversion Tool

- `.claude/commands/convert-npc.md` - NPC script conversion documentation
