int __thiscall CField::OnModePacket(CField *this, CInPacket *a2)
{
  unsigned __int8 mode = CInPacket::Decode1(a2);  // mode
  switch ( mode )
  {
    case 1:
      CInPacket::Decode2(a2);                       // case1 payload
      break;
    case 2:
      CInPacket::Decode4(a2);                       // case2 payload
      break;
    case 3:
      break;                                        // empty case (label only, no read)
  }
  return mode;
}
