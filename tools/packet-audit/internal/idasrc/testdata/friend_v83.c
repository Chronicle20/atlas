int __thiscall CWvsContext::OnFriendResult(CWvsContext *this, CInPacket *a2)
{
  unsigned __int8 mode = CInPacket::Decode1(a2);   // mode
  switch ( mode ) {
    case 9:
      CInPacket::Decode4(a2);          // friendId
      CInPacket::DecodeStr(a2, &name);// name
      CFriend::Insert(&rec, a2);       // GW_Friend(39)
      CInPacket::Decode1(a2);          // inShop
      break;
  }
  return mode;
}
