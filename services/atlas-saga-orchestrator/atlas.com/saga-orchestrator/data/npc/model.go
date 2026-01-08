package npc

type Model struct {
	id        uint32
	name      string
	trunkPut  int32
	trunkGet  int32
	storebank bool
}

func (n Model) Id() uint32 {
	return n.id
}

func (n Model) Name() string {
	return n.name
}

func (n Model) TrunkPut() int32 {
	return n.trunkPut
}

func (n Model) TrunkGet() int32 {
	return n.trunkGet
}

func (n Model) Storebank() bool {
	return n.storebank
}

// IsStorageNpc returns true if this NPC provides storage services
func (n Model) IsStorageNpc() bool {
	return n.storebank || n.trunkPut > 0 || n.trunkGet > 0
}

// GetDepositFee returns the fee for depositing an item
func (n Model) GetDepositFee() int32 {
	return n.trunkPut
}

// GetWithdrawFee returns the fee for withdrawing an item
func (n Model) GetWithdrawFee() int32 {
	return n.trunkGet
}
