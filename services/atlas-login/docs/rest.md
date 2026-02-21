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
    PinAttempts    int    `json:"pinAttempts"`
    PicAttempts    int    `json:"picAttempts"`
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
    PinAttempts    int    `json:"pinAttempts"`
    PicAttempts    int    `json:"picAttempts"`
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

#### POST /accounts/{id}/pin-attempts

Records a PIN verification attempt.

**Parameters**

| Name | Type | Description |
|------|------|-------------|
| id | uint32 | Account ID |

**Request Model**

```go
type PinAttemptInputRestModel struct {
    Id        string `json:"-"`
    Success   bool   `json:"success"`
    IpAddress string `json:"ipAddress"`
    HWID      string `json:"hwid"`
}
```

**Response Model**

```go
type PinAttemptOutputRestModel struct {
    Attempts     int  `json:"attempts"`
    LimitReached bool `json:"limitReached"`
}
```

#### POST /accounts/{id}/pic-attempts

Records a PIC verification attempt.

**Parameters**

| Name | Type | Description |
|------|------|-------------|
| id | uint32 | Account ID |

**Request Model**

```go
type PicAttemptInputRestModel struct {
    Id        string `json:"-"`
    Success   bool   `json:"success"`
    IpAddress string `json:"ipAddress"`
    HWID      string `json:"hwid"`
}
```

**Response Model**

```go
type PicAttemptOutputRestModel struct {
    Attempts     int  `json:"attempts"`
    LimitReached bool `json:"limitReached"`
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
| worldId | world.Id | World ID |

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
    Id                 uint32   `json:"-"`
    AccountId          uint32   `json:"accountId"`
    WorldId            world.Id `json:"worldId"`
    Name               string   `json:"name"`
    Level              byte     `json:"level"`
    Experience         uint32   `json:"experience"`
    GachaponExperience uint32   `json:"gachaponExperience"`
    Strength           uint16   `json:"strength"`
    Dexterity          uint16   `json:"dexterity"`
    Intelligence       uint16   `json:"intelligence"`
    Luck               uint16   `json:"luck"`
    Hp                 uint16   `json:"hp"`
    MaxHp              uint16   `json:"maxHp"`
    Mp                 uint16   `json:"mp"`
    MaxMp              uint16   `json:"maxMp"`
    Meso               uint32   `json:"meso"`
    HpMpUsed           int      `json:"hpMpUsed"`
    JobId              job.Id   `json:"jobId"`
    SkinColor          byte     `json:"skinColor"`
    Gender             byte     `json:"gender"`
    Fame               int16    `json:"fame"`
    Hair               uint32   `json:"hair"`
    Face               uint32   `json:"face"`
    Ap                 uint16   `json:"ap"`
    Sp                 string   `json:"sp"`
    MapId              _map.Id  `json:"mapId"`
    SpawnPoint         uint32   `json:"spawnPoint"`
    Gm                 int      `json:"gm"`
    X                  int16    `json:"x"`
    Y                  int16    `json:"y"`
    Stance             byte     `json:"stance"`
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
    Id           uint32   `json:"-"`
    AccountId    uint32   `json:"accountId"`
    WorldId      world.Id `json:"worldId"`
    Name         string   `json:"name"`
    Gender       byte     `json:"gender"`
    JobIndex     uint32   `json:"jobIndex"`
    SubJobIndex  uint32   `json:"subJobIndex"`
    Face         uint32   `json:"face"`
    Hair         uint32   `json:"hair"`
    HairColor    uint32   `json:"hairColor"`
    SkinColor    byte     `json:"skinColor"`
    Top          uint32   `json:"top"`
    Bottom       uint32   `json:"bottom"`
    Shoes        uint32   `json:"shoes"`
    Weapon       uint32   `json:"weapon"`
    Level        byte     `json:"level"`
    Strength     uint16   `json:"strength"`
    Dexterity    uint16   `json:"dexterity"`
    Intelligence uint16   `json:"intelligence"`
    Luck         uint16   `json:"luck"`
    Hp           uint16   `json:"hp"`
    Mp           uint16   `json:"mp"`
    MapId        _map.Id  `json:"mapId"`
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
    ExpRate            float64             `json:"expRate"`
    MesoRate           float64             `json:"mesoRate"`
    ItemDropRate       float64             `json:"itemDropRate"`
    QuestExpRate       float64             `json:"questExpRate"`
}
```

#### GET /worlds/{id}

Retrieves world by ID.

**Parameters**

| Name | Type | Description |
|------|------|-------------|
| id | world.Id | World ID |

### Channel Service

Base URL: `CHANNELS` environment variable

#### GET /worlds/{worldId}/channels

Retrieves channels for a world.

**Parameters**

| Name | Type | Description |
|------|------|-------------|
| worldId | world.Id | World ID |

#### GET /worlds/{worldId}/channels/{channelId}

Retrieves channel by world and channel ID.

**Parameters**

| Name | Type | Description |
|------|------|-------------|
| worldId | world.Id | World ID |
| channelId | channel.Id | Channel ID |

**Response Model**

```go
type RestModel struct {
    Id              uuid.UUID  `json:"-"`
    WorldId         world.Id   `json:"worldId"`
    ChannelId       channel.Id `json:"channelId"`
    IpAddress       string     `json:"ipAddress"`
    Port            int        `json:"port"`
    CurrentCapacity uint32     `json:"currentCapacity"`
    MaxCapacity     uint32     `json:"maxCapacity"`
    CreatedAt       time.Time  `json:"createdAt"`
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

**Compartment RestModel** (JSON:API included relation)

```go
type RestModel struct {
    Id            uuid.UUID         `json:"-"`
    InventoryType inventory.Type    `json:"type"`
    Capacity      uint32            `json:"capacity"`
    Assets        []asset.RestModel `json:"-"`
}
```

**Asset RestModel** (JSON:API included relation)

```go
type RestModel struct {
    Id             uint32     `json:"-"`
    Slot           int16      `json:"slot"`
    TemplateId     uint32     `json:"templateId"`
    Expiration     time.Time  `json:"expiration"`
    CreatedAt      time.Time  `json:"createdAt"`
    Quantity       uint32     `json:"quantity"`
    OwnerId        uint32     `json:"ownerId"`
    Flag           uint16     `json:"flag"`
    Rechargeable   uint64     `json:"rechargeable"`
    Strength       uint16     `json:"strength"`
    Dexterity      uint16     `json:"dexterity"`
    Intelligence   uint16     `json:"intelligence"`
    Luck           uint16     `json:"luck"`
    HP             uint16     `json:"hp"`
    MP             uint16     `json:"mp"`
    WeaponAttack   uint16     `json:"weaponAttack"`
    MagicAttack    uint16     `json:"magicAttack"`
    WeaponDefense  uint16     `json:"weaponDefense"`
    MagicDefense   uint16     `json:"magicDefense"`
    Accuracy       uint16     `json:"accuracy"`
    Avoidability   uint16     `json:"avoidability"`
    Hands          uint16     `json:"hands"`
    Speed          uint16     `json:"speed"`
    Jump           uint16     `json:"jump"`
    Slots          uint16     `json:"slots"`
    Locked         bool       `json:"locked"`
    Spikes         bool       `json:"spikes"`
    KarmaUsed      bool       `json:"karmaUsed"`
    Cold           bool       `json:"cold"`
    CanBeTraded    bool       `json:"canBeTraded"`
    LevelType      byte       `json:"levelType"`
    Level          byte       `json:"level"`
    Experience     uint32     `json:"experience"`
    HammersApplied uint32     `json:"hammersApplied"`
    EquippedSince  *time.Time `json:"equippedSince"`
    CashId         int64      `json:"cashId,string"`
    CommodityId    uint32     `json:"commodityId"`
    PurchaseBy     uint32     `json:"purchaseBy"`
    PetId          uint32     `json:"petId"`
}
```

### Guild Service

Base URL: `GUILDS` environment variable

#### GET /guilds?filter[members.id]={memberId}

Retrieves guilds by member character ID.

**Parameters**

| Name | Type | Description |
|------|------|-------------|
| memberId | uint32 | Member character ID |

**Response Model**

```go
type RestModel struct {
    Id                  uint32             `json:"-"`
    WorldId             world.Id           `json:"worldId"`
    Name                string             `json:"name"`
    Notice              string             `json:"notice"`
    Points              uint32             `json:"points"`
    Capacity            uint32             `json:"capacity"`
    Logo                uint16             `json:"logo"`
    LogoColor           byte               `json:"logoColor"`
    LogoBackground      uint16             `json:"logoBackground"`
    LogoBackgroundColor byte               `json:"logoBackgroundColor"`
    LeaderId            uint32             `json:"leaderId"`
    Members             []member.RestModel `json:"members"`
    Titles              []title.RestModel  `json:"titles"`
}
```

### Configuration Service

Base URL: `CONFIGURATIONS` environment variable

#### GET /configurations/services/{serviceId}

Retrieves service configuration by service ID.

**Parameters**

| Name | Type | Description |
|------|------|-------------|
| serviceId | uuid.UUID | Service ID |

#### GET /configurations/tenants/{tenantId}

Retrieves tenant-specific configuration including socket handlers and writers.

**Parameters**

| Name | Type | Description |
|------|------|-------------|
| tenantId | uuid.UUID | Tenant ID |
