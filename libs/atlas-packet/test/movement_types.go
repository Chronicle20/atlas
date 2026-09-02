package test

// MovementTypesV95 returns the IDB-derived GMS v95 movement "types" options
// table used by CMovePath::Decode/Encode (fname CUserRemote::OnMove @0x948a80,
// CMob::OnMove @0x6521e0, CPet::OnMove @0x69fb60, CNpc::OnMove @0x678060 — all
// four thunk to the single shared CMovePath::OnMovePacket @0x6683f0 ->
// CMovePath::Decode @0x667920, so one 37-entry table covers all four).
//
// This is the SAME 37-entry array task-191 derived and shipped for the
// serverbound mirror direction (CharacterMoveHandle/MonsterMovementHandle/
// PetMovementHandle/SummonMoveHandle/NPCActionHandle in
// template_gms_95_1.json, e.g. "opCode": "0x2C" CharacterMoveHandle) — full
// per-index citation in
// docs/tasks/task-191-v92-v95-movement-types/movement-types-derivation.md
// §3.1, independently re-confirmed for this task in
// docs/tasks/task-146-v95-packet-verification-batch/v95-option-tables.md.
//
// Reading this from one shared source (rather than each _test.go file
// defining its own copy) means a template edit that desyncs the array from
// the real client fails `go test`, not just a template lint — see
// character/clientbound/movement_test.go, monster/clientbound/movement_test.go,
// and pet/clientbound/movement_test.go, each of which asserts against index 36
// (the highest index in the array) resolving to its real Name/Type rather than
// NOT_FOUND/DEFAULT.
func MovementTypesV95() map[string]interface{} {
	return map[string]interface{}{
		"types": []interface{}{
			map[string]interface{}{"Name": "NORMAL", "Type": "NORMAL"},                   // 0
			map[string]interface{}{"Name": "UNKNOWN", "Type": "JUMP"},                    // 1
			map[string]interface{}{"Name": "UNKNOWN", "Type": "JUMP"},                    // 2
			map[string]interface{}{"Name": "UNKNOWN", "Type": "TELEPORT"},                // 3
			map[string]interface{}{"Name": "UNKNOWN", "Type": "TELEPORT"},                // 4
			map[string]interface{}{"Name": "UNKNOWN", "Type": "NORMAL"},                  // 5
			map[string]interface{}{"Name": "UNKNOWN", "Type": "TELEPORT"},                // 6
			map[string]interface{}{"Name": "UNKNOWN", "Type": "TELEPORT"},                // 7
			map[string]interface{}{"Name": "UNKNOWN", "Type": "TELEPORT"},                // 8
			map[string]interface{}{"Name": "STAT_CHANGE", "Type": "STAT_CHANGE"},         // 9
			map[string]interface{}{"Name": "UNKNOWN", "Type": "TELEPORT"},                // 10
			map[string]interface{}{"Name": "START_FALL_DOWN", "Type": "START_FALL_DOWN"}, // 11
			map[string]interface{}{"Name": "FALL_DOWN", "Type": "NORMAL"},                // 12
			map[string]interface{}{"Name": "UNKNOWN", "Type": "JUMP"},                    // 13
			map[string]interface{}{"Name": "UNKNOWN", "Type": "NORMAL"},                  // 14
			map[string]interface{}{"Name": "UNKNOWN", "Type": "DEFAULT"},                 // 15
			map[string]interface{}{"Name": "UNKNOWN", "Type": "JUMP"},                    // 16
			map[string]interface{}{"Name": "FLYING_BLOCK", "Type": "FLYING_BLOCK"},       // 17
			map[string]interface{}{"Name": "UNKNOWN", "Type": "JUMP"},                    // 18
			map[string]interface{}{"Name": "UNKNOWN", "Type": "DEFAULT"},                 // 19
			map[string]interface{}{"Name": "UNKNOWN", "Type": "DEFAULT"},                 // 20
			map[string]interface{}{"Name": "UNKNOWN", "Type": "DEFAULT"},                 // 21
			map[string]interface{}{"Name": "UNKNOWN", "Type": "DEFAULT"},                 // 22
			map[string]interface{}{"Name": "UNKNOWN", "Type": "DEFAULT"},                 // 23
			map[string]interface{}{"Name": "UNKNOWN", "Type": "DEFAULT"},                 // 24
			map[string]interface{}{"Name": "UNKNOWN", "Type": "DEFAULT"},                 // 25
			map[string]interface{}{"Name": "UNKNOWN", "Type": "DEFAULT"},                 // 26
			map[string]interface{}{"Name": "UNKNOWN", "Type": "DEFAULT"},                 // 27
			map[string]interface{}{"Name": "UNKNOWN", "Type": "DEFAULT"},                 // 28
			map[string]interface{}{"Name": "UNKNOWN", "Type": "DEFAULT"},                 // 29
			map[string]interface{}{"Name": "UNKNOWN", "Type": "DEFAULT"},                 // 30
			map[string]interface{}{"Name": "UNKNOWN", "Type": "JUMP"},                    // 31
			map[string]interface{}{"Name": "UNKNOWN", "Type": "JUMP"},                    // 32
			map[string]interface{}{"Name": "UNKNOWN", "Type": "JUMP"},                    // 33
			map[string]interface{}{"Name": "UNKNOWN", "Type": "JUMP"},                    // 34
			map[string]interface{}{"Name": "UNKNOWN", "Type": "NORMAL"},                  // 35
			map[string]interface{}{"Name": "UNKNOWN", "Type": "NORMAL"},                  // 36
		},
	}
}

// MovementTypesJMS185 returns the JMS v185.1 movement "types" options table —
// the 33-entry array the jms tenant template ships for the movement handlers
// (services/atlas-configurations/seed-data/templates/template_jms_185_1.json,
// "opCode": "0x20" CharacterMoveHandle and its MOB/PET/SUMMON/NPC siblings).
//
// A movement fixture MUST pass this rather than nil options: without it every
// element falls through resolveMovementPathAttr to the bare Element codec,
// which consumes the wrong number of bytes and hides a header misalignment
// behind a plausible-looking partial decode.
func MovementTypesJMS185() map[string]interface{} {
	return map[string]interface{}{
		"types": []interface{}{
			map[string]interface{}{"Name": "NORMAL", "Type": "NORMAL"},                   // 0
			map[string]interface{}{"Name": "JUMP", "Type": "JUMP"},                       // 1
			map[string]interface{}{"Name": "IMPACT", "Type": "JUMP"},                     // 2
			map[string]interface{}{"Name": "IMMEDIATE", "Type": "TELEPORT"},              // 3
			map[string]interface{}{"Name": "TELEPORT", "Type": "TELEPORT"},               // 4
			map[string]interface{}{"Name": "HANG_ON_BACK", "Type": "NORMAL"},             // 5
			map[string]interface{}{"Name": "UNKNOWN", "Type": "JUMP"},                    // 6
			map[string]interface{}{"Name": "ASSAULTER", "Type": "TELEPORT"},              // 7
			map[string]interface{}{"Name": "ASSASSINATION", "Type": "TELEPORT"},          // 8
			map[string]interface{}{"Name": "RUSH", "Type": "TELEPORT"},                   // 9
			map[string]interface{}{"Name": "STAT_CHANGE", "Type": "STAT_CHANGE"},         // 10
			map[string]interface{}{"Name": "SIT_DOWN", "Type": "TELEPORT"},               // 11
			map[string]interface{}{"Name": "UNKNOWN", "Type": "JUMP"},                    // 12
			map[string]interface{}{"Name": "UNKNOWN", "Type": "JUMP"},                    // 13
			map[string]interface{}{"Name": "START_FALL_DOWN", "Type": "START_FALL_DOWN"}, // 14
			map[string]interface{}{"Name": "FALL_DOWN", "Type": "NORMAL"},                // 15
			map[string]interface{}{"Name": "START_WINGS", "Type": "JUMP"},                // 16
			map[string]interface{}{"Name": "WINGS", "Type": "NORMAL"},                    // 17
			map[string]interface{}{"Name": "ARAN_ADJUST", "Type": "JUMP"},                // 18
			map[string]interface{}{"Name": "MOB_TOSS", "Type": "JUMP"},                   // 19
			map[string]interface{}{"Name": "DASH_SLIDE", "Type": "JUMP"},                 // 20
			map[string]interface{}{"Name": "UNKNOWN", "Type": "DEFAULT"},                 // 21
			map[string]interface{}{"Name": "UNKNOWN", "Type": "DEFAULT"},                 // 22
			map[string]interface{}{"Name": "UNKNOWN", "Type": "JUMP"},                    // 23
			map[string]interface{}{"Name": "FLYING_BLOCK", "Type": "FLYING_BLOCK"},       // 24
			map[string]interface{}{"Name": "UNKNOWN", "Type": "JUMP"},                    // 25
			map[string]interface{}{"Name": "UNKNOWN", "Type": "DEFAULT"},                 // 26
			map[string]interface{}{"Name": "UNKNOWN", "Type": "JUMP"},                    // 27
			map[string]interface{}{"Name": "UNKNOWN", "Type": "JUMP"},                    // 28
			map[string]interface{}{"Name": "UNKNOWN", "Type": "JUMP"},                    // 29
			map[string]interface{}{"Name": "UNKNOWN", "Type": "JUMP"},                    // 30
			map[string]interface{}{"Name": "MOB_ATK_RUSH", "Type": "NORMAL"},             // 31
			map[string]interface{}{"Name": "MOB_ATK_RUSH_STOP", "Type": "NORMAL"},        // 32
		},
	}
}
