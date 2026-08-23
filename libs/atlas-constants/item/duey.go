package item

// QuickDeliveryTicketId is the Quick Delivery Ticket (classification 533,
// ClassificationDueyCoupon), sold by Duey NPC 9010009's script. Holding one
// is the gate on the DUEY_ACTION SEND quick-delivery arm — the client
// itself pre-checks CWvsContext::IsExist(5330000) before letting the player
// send (task-241 design §9.5). Consuming it is FR-26: the ticket is spent at
// send time, not at open time.
//
// Typed uint32 (not Id) to match compartment.Model.FindFirstByItemId's
// signature at every call site without a conversion.
//
// task-241 Task 17 defined this ahead of Task 22 landing (per Task 17's
// brief: "if Task 22 has not landed, define it there and have Task 22
// remove the duplicate"). Task 22 landed the coupon-use handler
// (services/atlas-channel/atlas.com/channel/socket/handler/
// character_cash_item_use_duey.go) and reuses this constant rather than
// redefining it.
const QuickDeliveryTicketId = uint32(5330000)
