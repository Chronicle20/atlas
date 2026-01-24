# REST Integration

This service does not expose a REST API. It acts as a REST client to other Atlas services.

## External Service Dependencies

### Account Service

Base URL: `ACCOUNTS` environment variable

#### GET /accounts

Retrieves all accounts.

**Response Model**

```go
type RestModel struct {
    Id             string `json:"id"`
    Name           string `json:"name"`
    Password       string `json:"password"`
    Pin            string `json:"pin"`
    Pic            string `json:"pic"`
    LoggedIn       byte   `json:"loggedIn"`
    LastLogin      uint64 `json:"lastLogin"`
    Gender         byte   `json:"gender"`
    Banned         bool   `json:"banned"`
    TOS            bool   `json:"tos"`
    Language       string `json:"language"`
    Country        string `json:"country"`
    CharacterSlots int16  `json:"characterSlots"`
}
```

#### GET /accounts?name={name}

Retrieves account by name.

**Parameters**

| Name | Type | Description |
|------|------|-------------|
| name | string | Account name |

#### GET /accounts/{id}

Retrieves account by ID.

**Parameters**

| Name | Type | Description |
|------|------|-------------|
| id | uint32 | Account ID |

#### PATCH /accounts/{id}

Updates account attributes.

**Parameters**

| Name | Type | Description |
|------|------|-------------|
| id | uint32 | Account ID |

**Request Model**

```go
type RestModel struct {
    Id             string `json:"id"`
    Name           string `json:"name"`
    Pin            string `json:"pin"`
    Pic            string `json:"pic"`
    LoggedIn       byte   `json:"loggedIn"`
    LastLogin      uint64 `json:"lastLogin"`
    Gender         byte   `json:"gender"`
    Banned         bool   `json:"banned"`
    TOS            bool   `json:"tos"`
    Language       string `json:"language"`
    Country        string `json:"country"`
    CharacterSlots int16  `json:"characterSlots"`
}
```

### Character Service

Base URL: `CHARACTERS` environment variable

#### GET /characters?accountId={accountId}&worldId={worldId}

Retrieves characters for an account in a world.

**Parameters**

| Name | Type | Description |
|------|------|-------------|
| accountId | uint32 | Account ID |
| worldId | byte | World ID |

#### GET /characters?name={name}

Retrieves characters by name.

**Parameters**

| Name | Type | Description |
|------|------|-------------|
| name | string | Character name |

#### GET /characters/{id}

Retrieves character by ID.

**Parameters**

| Name | Type | Description |
|------|------|-------------|
| id | uint32 | Character ID |

**Response Model**

```go
type RestModel struct {
    Id                 uint32 `json:"-"`
    AccountId          uint32 `json:"accountId"`
    WorldId            byte   `json:"worldId"`
    Name               string `json:"name"`
    Level              byte   `json:"level"`
    Experience         uint32 `json:"experience"`
    GachaponExperience uint32 `json:"gachaponExperience"`
    Strength           uint16 `json:"strength"`
    Dexterity          uint16 `json:"dexterity"`
    Intelligence       uint16 `json:"intelligence"`
    Luck               uint16 `json:"luck"`
    Hp                 uint16 `json:"hp"`
    MaxHp              uint16 `json:"maxHp"`
    Mp                 uint16 `json:"mp"`
    MaxMp              uint16 `json:"maxMp"`
    Meso               uint32 `json:"meso"`
    HpMpUsed           int    `json:"hpMpUsed"`
    JobId              uint16 `json:"jobId"`
    SkinColor          byte   `json:"skinColor"`
    Gender             byte   `json:"gender"`
    Fame               int16  `json:"fame"`
    Hair               uint32 `json:"hair"`
    Face               uint32 `json:"face"`
    Ap                 uint16 `json:"ap"`
    Sp                 string `json:"sp"`
    MapId              uint32 `json:"mapId"`
    SpawnPoint         uint32 `json:"spawnPoint"`
    Gm                 int    `json:"gm"`
    X                  int16  `json:"x"`
    Y                  int16  `json:"y"`
    Stance             byte   `json:"stance"`
}
```

**JSON:API References**

| Type | Name |
|------|------|
| equipment | equipment |
| inventories | inventories |

#### DELETE /characters/{id}

Deletes a character by ID.

**Parameters**

| Name | Type | Description |
|------|------|-------------|
| id | uint32 | Character ID |

### Character Factory Service

Base URL: `CHARACTER_FACTORY` environment variable

#### POST /characters/seed

Creates a new character.

**Request Model**

```go
type RestModel struct {
    AccountId    uint32  `json:"accountId"`
    WorldId      byte    `json:"worldId"`
    Name         string  `json:"name"`
    Gender       byte    `json:"gender"`
    JobIndex     uint32  `json:"jobIndex"`
    SubJobIndex  uint32  `json:"subJobIndex"`
    Face         uint32  `json:"face"`
    Hair         uint32  `json:"hair"`
    HairColor    uint32  `json:"hairColor"`
    SkinColor    byte    `json:"skinColor"`
    Top          uint32  `json:"top"`
    Bottom       uint32  `json:"bottom"`
    Shoes        uint32  `json:"shoes"`
    Weapon       uint32  `json:"weapon"`
    Level        byte    `json:"level"`
    Strength     uint16  `json:"strength"`
    Dexterity    uint16  `json:"dexterity"`
    Intelligence uint16  `json:"intelligence"`
    Luck         uint16  `json:"luck"`
    Hp           uint16  `json:"hp"`
    Mp           uint16  `json:"mp"`
    MapId        uint32  `json:"mapId"`
}
```

**Response Model**

```go
type CreateCharacterResponse struct {
    TransactionId string `json:"transactionId"`
}
```

### World Service

Base URL: `WORLDS` environment variable

#### GET /worlds?include=channels

Retrieves all worlds with channel information.

**Response Model**

```go
type RestModel struct {
    Id                 string              `json:"-"`
    Name               string              `json:"name"`
    State              byte                `json:"state"`
    Message            string              `json:"message"`
    EventMessage       string              `json:"eventMessage"`
    Recommended        bool                `json:"recommended"`
    RecommendedMessage string              `json:"recommendedMessage"`
    CapacityStatus     uint16              `json:"capacityStatus"`
    Channels           []channel.RestModel `json:"-"`
}
```

#### GET /worlds/{id}

Retrieves world by ID.

**Parameters**

| Name | Type | Description |
|------|------|-------------|
| id | byte | World ID |

### Channel Service

Base URL: `CHANNELS` environment variable

#### GET /worlds/{worldId}/channels

Retrieves channels for a world.

**Parameters**

| Name | Type | Description |
|------|------|-------------|
| worldId | byte | World ID |

#### GET /worlds/{worldId}/channels/{channelId}

Retrieves channel by world and channel ID.

**Parameters**

| Name | Type | Description |
|------|------|-------------|
| worldId | byte | World ID |
| channelId | byte | Channel ID |

**Response Model**

```go
type RestModel struct {
    Id              uuid.UUID `json:"-"`
    WorldId         byte      `json:"worldId"`
    ChannelId       byte      `json:"channelId"`
    IpAddress       string    `json:"ipAddress"`
    Port            int       `json:"port"`
    CurrentCapacity uint32    `json:"currentCapacity"`
    MaxCapacity     uint32    `json:"maxCapacity"`
    CreatedAt       time.Time `json:"createdAt"`
}
```

### Inventory Service

Base URL: `INVENTORY` environment variable

#### GET /characters/{characterId}/inventory

Retrieves inventory for a character.

**Parameters**

| Name | Type | Description |
|------|------|-------------|
| characterId | uint32 | Character ID |

**Response Model**

```go
type RestModel struct {
    Id           uuid.UUID               `json:"-"`
    CharacterId  uint32                  `json:"characterId"`
    Compartments []compartment.RestModel `json:"-"`
}
```
