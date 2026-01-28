# Atlas Project TODO

This document tracks planned features and improvements for the Atlas MapleStory server project.

---

## Priority Summary

### Critical (Core Gameplay)
- [ ] **Character Attack Effects** - 27 unimplemented combat mechanics in `character_attack_common.go`
- [ ] **Stat Reset Logic** - Job advancement stat reset not implemented

### High Priority (Feature Incomplete)
- [ ] **TokenItem Purchasing** - Returns "not implemented" error in NPC shops
- [ ] **Damage Reduction Effects** - 10 defensive abilities missing
- [ ] **Drop Rate System** - Player bonuses, buff rates, card rates not applied
- [ ] **IP/MAC Banning** - Security feature missing in account service
- [ ] **Reactor Actions** - Boss weakening, environment manipulation, mass kill sagas

---

## Services

### Account Service
- [ ] Implement IP, MAC, and temporary banning practices (`processor.go:313`)
- [ ] Implement Terms of Service tracking (`processor.go:318`)

### Buddies Service
- [ ] Trigger channel request for target when adding buddy (`list/processor.go:216`)
- [ ] Trigger channel request for target when accepting buddy (`list/processor.go:386`)

### Cash Shop Service
- [ ] Refactor item resource handling (`item/resource.go:106`)

### Chairs Service
- [ ] Verify character has chair item before sitting (`chair/processor.go:73`)

### Chalkboards Service
- [ ] Ensure character is in a valid location for chalkboard (`chalkboard/processor.go:53`)
- [ ] Ensure character is alive before setting chalkboard (`chalkboard/processor.go:54`)

### Channel Service
- [ ] Cash Item Usage should verify inventory contains item being used
- [ ] Timing issue with loading pre-existing chalkboards
- [ ] Timing issue with loading pre-existing chairs
- [ ] Parties: Party Portals missing. Party member map, level, job, and name changes need to be considered
- [ ] Identify correct compartment type based on character job for cash shop (`cashshop/processor.go:105,150`)
- [ ] Block cash shop entry during: Vega scrolling, events, mini dungeons, already in shop (`cash_shop_entry.go:28-31`)
- [ ] Performance optimization for character queries (`character/processor.go:103,117`)
- [ ] Restrict skill targets to those in range based on bitmap (`skill/handler/common.go:19`)
- [ ] Consume summoning rock for Shadow Partner (`skill/handler/shadow_partner.go:29`)
- [ ] Pet lookup for movement processing (`movement/processor.go:79`)
- [ ] Optimize extra queries in pet consumer (`kafka/consumer/pet/consumer.go:236,274`)
- [ ] Pet skill and item writing (`socket/writer/character_info.go:32`)
- [ ] Query cash shop for whisper targets (`character_chat_whisper.go:72`)
- [ ] Remote channel lookup for whispers (`character_chat_whisper.go:83`)
- [ ] Send rejection to requester for declined invites (`kafka/consumer/invite/consumer.go:138`)
- [ ] Medal name retrieval (`kafka/consumer/message/consumer.go:213`)
- [ ] Server notice on map change failure (`socket/handler/map_change.go:42`)
- [ ] Verify alive and not in mini dungeon for channel change (`channel_change.go:23-24`)
- [ ] Send server notice on channel change failure (`channel_change.go:29`)
- [ ] Validate NPC has ability to move (`npc_action.go:24`)
- [ ] Handle quest-in-progress states in NPC conversations (`npc_continue_conversation.go:24,26,30,39`)
- [ ] Announce guild operation errors (`guild_operation.go:137`)
- [ ] Send buddy operation errors to requester (`buddy_operation.go:47`)

#### Character Attack System (27 unimplemented effects)
Location: `socket/handler/character_attack_common.go:93-119`
- [ ] Apply cooldown
- [ ] Cancel dark sight / wind walk
- [ ] Apply combo orbs (add or consume)
- [ ] Decrease HP from DragonKnight Sacrifice
- [ ] Apply attack effects (heal, MP consumption, dispel, cure all, combo reset)
- [ ] Destroy Chief Bandit exploded mesos
- [ ] Apply Pick Pocket
- [ ] Increase HP from Energy Drain, Vampire, or Drain
- [ ] Apply Bandit Steal
- [ ] Fire Demon ice weaken
- [ ] Ice Demon fire weaken
- [ ] Homing Beacon / Bullseye
- [ ] Flame Thrower
- [ ] Snow Charge
- [ ] Hamstring
- [ ] Slow
- [ ] Blind
- [ ] Paladin / White Knight charges
- [ ] Combo Drain
- [ ] Mortal Blow
- [ ] Three Snails consumption
- [ ] Heavens Hammer
- [ ] ComboTempest
- [ ] BodyPressure
- [ ] Monster Weapon Atk Reflect
- [ ] Monster Magic Atk Reflect
- [ ] Apply MPEater

#### Character Damage System (10 unimplemented effects)
Location: `socket/handler/character_damage.go:23-32`
- [ ] Process Mana Reflection
- [ ] Process Achilles
- [ ] Process Combo Barrier
- [ ] Process Body Pressure
- [ ] Process PowerGuard
- [ ] Process Paladin Divine Shield
- [ ] Process Aran High Defense
- [ ] Process MagicGuard
- [ ] Process MesoGuard
- [ ] Decrease battleship HP

#### Protocol/Version Compatibility
- [ ] Test buddy model with JMS before moving to library (`socket/model/buddy.go:28`)
- [ ] Proper temp stat encoding for GMS v12 (`socket/model/monster.go:205`)
- [ ] Complete skill ID list for skill_usage_info (`socket/model/skill_usage_info.go:65,123,166`)
- [ ] Battle Mage attack info handling (`socket/model/attack_info.go:139`)
- [ ] Look up actual buff values if riding mount (`socket/model/character.go:481`)
- [ ] Document GMS v83/v95 constants (`socket/writer/character_attack_common.go:41,50,58`)
- [ ] Wild Hunter swallow (`socket/writer/character_attack_common.go:117`)
- [ ] BlazeWizardSpellMastery handling (`socket/writer/character_attack_common.go:157,170`)
- [ ] Clean up character spawn code (`socket/writer/character_spawn.go:75`)
- [ ] Handle GMS-JMS ring encoding differences (`socket/writer/character_spawn.go:100`)
- [ ] Fix crash issues in character effects (`socket/writer/character_effect.go:265,276`)
- [ ] Write doors for party (`socket/writer/party_operation.go:31,190`)
- [ ] Test party operations with JMS (`socket/writer/party_operation.go:199`)
- [ ] JMS map codes for cash shop (`socket/writer/cash_shop_operation.go:128`)
- [ ] Load gifts in cash shop (`socket/writer/cash_shop_operation.go:131`)

### Character Service
- [ ] Determine appropriate drop type and mod (`processor.go:707`)
- [ ] Incorporate computed total intelligence with buffs/weapons (`processor.go:949`)
- [ ] Consider effective (temporary) Max HP (`processor.go:981`)
- [ ] Emit event when character dies (`processor.go:989`)
- [ ] Consider effective (temporary) Max MP (`processor.go:1037`)
- [ ] Define AP auto-assign range for Beginner/Noblesse/Legend (`processor.go:1078`)
- [ ] Pre-compute HP and MP to avoid loop cost (`processor.go:1094`)
- [ ] Account for 6 beginner skill levels (`processor.go:1146`)
- [ ] Award job change AP (Cygnus only?) (`processor.go:1268`)
- [ ] **Implement stat reset logic for job advancement** (`processor.go:1522-1527`)

### Consumables Service
- [ ] Consume Vega scroll (`consumable/processor.go:512`)
- [ ] Handle spikes/cursed property (`consumable/processor.go:515`)
- [ ] Improve HP/MP structure (`consumable/processor.go:631`)

### Data Service
- [ ] Player NPCs and CPQ support (`map/reader.go:114`)
- [ ] Validate skill reader logic (`skill/reader.go:160`)
- [ ] Handle map chairs (`skill/reader.go:164`)
- [ ] Handle LT in skills (`skill/reader.go:175`)
- [ ] Support mount types: SpaceShip, YetiMount1/2, Broomstick, BalrogMount (`skill/reader.go:196`)
- [ ] Space dash handling (`skill/reader.go:266`)
- [ ] Power explosion handling (`skill/reader.go:279`)
- [ ] Better naming for skill properties (`skill/reader.go:411`)

### Guilds Service
- [ ] Improve guild creation logic (`guild/processor.go:196`)
- [ ] Validate guild name (`guild/processor.go:236`)
- [ ] Respond with failure on guild errors (`guild/processor.go:319`)
- [ ] Proper error handling (`guild/processor.go:482,486`)
- [ ] Second query for party information (`party/rest.go:85`)

### Inventory Service
- [ ] Cash Shop integration for assets (`asset/processor.go:362,368`)
- [ ] Determine if creating Equip or Cash Equip (`asset/processor.go:525`)
- [ ] Asset owner validation for stacking (`compartment/processor.go:581`)
- [ ] Migrate TransactionId usage (5 locations in `kafka/consumer/compartment/consumer.go`)
- [ ] Add required fields to drop event (`kafka/consumer/drop/consumer.go:47,52`)

### Invite Service
- [ ] Character deletion should remove pending invites
- [ ] Invites should be able to be queued

### Login Service
- [ ] Implement error responses for character selection (multiple handlers)
- [ ] Terminate on too many PIN attempts (`after_login.go:98`)
- [ ] Clarify gender defaulting logic (`create_character.go:55`)
- [ ] Verify character is not a guild master before deletion (`delete_character.go:65`)
- [ ] Verify character is not engaged before deletion (`delete_character.go:66`)
- [ ] Verify character is not part of a family before deletion (`delete_character.go:67`)

### Monster Death Service
- [ ] Apply character's meso buff (`monster/drop/processor.go:38`)
- [ ] Determine drop type (`monster/processor.go:18`)
- [ ] Evaluate rates: channel rate, buff rate, card rate (`monster/processor.go:48-51`)
- [ ] Party drop distribution (`monster/processor.go:93`)
- [ ] Account for healing (`monster/processor.go:104`)

### Monsters Service
- [ ] More efficient mechanism for ID reuse (`monster/registry.go:59`)

### Notes Service
- [ ] Award fame when a note is discarded (`note/processor.go:216`)

### NPC Conversations Service
- [ ] Transmit stats in NPC conversations (`kafka/message/npc/kafka.go:78`)
- [ ] Integrate with WZ data registry for cosmetic validation (`cosmetic/validator.go:26`)
- [ ] Integrate with character equipment query for cosmetics (`cosmetic/validator.go:44`)

### NPC Shops Service
- [ ] Better transaction handling (`shops/processor.go:396,450`)
- [ ] **Implement TokenItem purchasing** (`shops/processor.go:427`)

### Pets Service
- [ ] Generate cashId if cashId == 0 (`pet/processor.go:199`)

### Portals Service
- [ ] Transmit stats in portal transitions (`character/kafka.go:18`)

### Reactor Actions Service
- [ ] Create saga action for boss weakening (`script/executor.go:230,244`)
- [ ] Create saga action for environment object manipulation (`script/executor.go:251,261`)
- [ ] Create saga action for mass monster killing (`script/executor.go:268,273`)

### Saga Orchestrator
- [ ] Player drop rate bonuses (hardcoded to 1.0) (`reactor/drop/processor.go:192,195,215`)

---

## Instance Based Transports
- [ ] Extend atlas-transports to support instance-based transport events
- [ ] Instance capacity management (e.g., "wagon is already full")
- [ ] On-demand warping (vs scheduled boarding windows)
- [ ] Use case: Kerning Square Train (NPC 1052007 selection 0)
  - Currently warps directly without capacity check
  - Original script used `em.startInstance(cm.getPlayer())` pattern

---

## NPC Conversations

### Pending Conversions
- NPCs requiring instance-based transports should be revisited after that feature is implemented

---

## Libraries

### atlas-constants
- [ ] Define HiddenStreet Nett's Pyramid battle room maps (926010100-926023500) (`map/model.go:434`)

---

## Notes

- Instance-based transports differ from scheduled transports:
  - **Scheduled transports**: Have boarding windows, departure times, and use `transportAvailable` condition (e.g., ships, subway to NLC)
  - **Instance-based transports**: Available on-demand with capacity limits, no fixed schedule (e.g., Kerning Square Train)

- **Total inline TODOs found**: 198 across the codebase
- **Most concentrated areas**: Channel Service (98), Character Service (11), Inventory Service (12)
