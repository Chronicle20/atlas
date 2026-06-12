int __thiscall CWvsContext::OnFriendResult(CWvsContext *this, CInPacket *a2)
{
  int characterId = CInPacket::Decode4(a2);     // friendId
  CInPacket::DecodeStr(a2, &name);              // name
  CFriend::Insert(&friendRec, a2);             // GW_Friend struct
  unsigned __int8 inShop = CInPacket::Decode1(a2); // inShop
  return characterId;
}
