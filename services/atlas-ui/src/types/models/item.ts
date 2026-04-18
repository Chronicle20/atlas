export type ItemType = "Equipment" | "Consumable" | "Setup" | "Etc" | "Cash" | "Pet" | "Unknown";

export function getItemType(itemId: string): ItemType {
  const id = parseInt(itemId, 10);
  if (isNaN(id)) return "Unknown";
  const prefix = Math.floor(id / 1000000);
  switch (prefix) {
    case 1: return "Equipment";
    case 2: return "Consumable";
    case 3: return "Setup";
    case 4: return "Etc";
    case 5: return "Cash";
    default: return "Unknown";
  }
}

export function getItemTypeBadgeVariant(type: ItemType): string {
  switch (type) {
    case "Equipment": return "bg-blue-100 text-blue-800";
    case "Consumable": return "bg-green-100 text-green-800";
    case "Setup": return "bg-purple-100 text-purple-800";
    case "Etc": return "bg-gray-100 text-gray-800";
    case "Cash": return "bg-yellow-100 text-yellow-800";
    case "Pet": return "bg-pink-100 text-pink-800";
    default: return "bg-gray-100 text-gray-800";
  }
}

export interface ItemSearchResult {
  id: string;
  name: string;
  type: ItemType;
}

// Equipment detail attributes
export interface EquipmentAttributes {
  strength: number;
  dexterity: number;
  intelligence: number;
  luck: number;
  hp: number;
  mp: number;
  weaponAttack: number;
  magicAttack: number;
  weaponDefense: number;
  magicDefense: number;
  accuracy: number;
  avoidability: number;
  speed: number;
  jump: number;
  slots: number;
  cash: boolean;
  price: number;
  timeLimited: boolean;
}

export interface EquipmentData {
  id: string;
  attributes: EquipmentAttributes;
}

// Consumable detail attributes
export interface ConsumableAttributes {
  price: number;
  unitPrice: number;
  slotMax: number;
  reqLevel: number;
  quest: boolean;
  tradeBlock: boolean;
  notSale: boolean;
  timeLimited: boolean;
  success: number;
  cursed: number;
  rechargeable: boolean;
  spec: Record<string, number>;
}

export interface ConsumableData {
  id: string;
  attributes: ConsumableAttributes;
}

// Setup detail attributes
export interface SetupAttributes {
  price: number;
  slotMax: number;
  recoveryHP: number;
  tradeBlock: boolean;
  notSale: boolean;
  reqLevel: number;
  timeLimited: boolean;
}

export interface SetupData {
  id: string;
  attributes: SetupAttributes;
}

// Etc detail attributes
export interface EtcAttributes {
  price: number;
  unitPrice: number;
  slotMax: number;
  timeLimited: boolean;
}

export interface EtcData {
  id: string;
  attributes: EtcAttributes;
}

// Cash item detail attributes
export interface CashItemAttributes {
  slotMax: number;
  spec: Record<string, number>;
  timeWindows?: Array<{ day: string; startHour: number; endHour: number }>;
}

export interface CashItemData {
  id: string;
  attributes: CashItemAttributes;
}

export type ItemDetailData = EquipmentData | ConsumableData | SetupData | EtcData | CashItemData;
