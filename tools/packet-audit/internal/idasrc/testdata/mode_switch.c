int __thiscall CField::OnPacket(CField *this, CInPacket *a2)
{
  unsigned __int8 mode = CInPacket::Decode1(a2);  // mode
  switch ( mode )
  {
    case 0:
      CInPacket::Decode4(a2);                       // case0 id
      break;
    case 1:
      CInPacket::DecodeStr(a2, &s);                // case1 name
      CInPacket::Decode2(a2);                       // case1 qty
      break;
  }
  return mode;
}
