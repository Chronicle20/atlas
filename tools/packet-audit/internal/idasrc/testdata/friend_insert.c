// friend_insert.c — GW_Friend 39 bytes as primitives
int __thiscall CFriend::Insert(GW_Friend *r, CInPacket *a2)
{
  CInPacket::Decode4(a2);              // friendId (4)
  CInPacket::DecodeBuffer(a2, 13);     // name[13] (13)
  CInPacket::Decode1(a2);             // flag (1)
  CInPacket::Decode4(a2);             // ... (4)
  CInPacket::DecodeBuffer(a2, 17);     // group[17] (17)
  return 0;                            // total 39
}
