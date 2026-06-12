int __thiscall CX::OnFriendResult(CX *this, CInPacket *a1)
{
  switch ( CInPacket::Decode1(a1) )
  {
    case 7u:
      CInPacket::Decode1(a1);
      break;
    case 9u:
      CInPacket::Decode4(a1);
      CInPacket::DecodeStr(a1, &name);
      break;
    case 0x12u:
      CInPacket::Decode2(a1);
      break;
  }
}
