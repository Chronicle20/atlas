int __thiscall CX::OnPacket(CX *this, CInPacket *a2)
{
  unsigned __int8 mode = CInPacket::Decode1(a2);   // mode
  switch ( mode )
  {
    case 1:
      CInPacket::Decode4(a2);                        // a
      for ( i = 0; i < count; ++i )
      {
        CInPacket::Decode2(a2);                      // b
        if ( bad )
          break;
        CInPacket::Decode1(a2);                      // c
      }
      CInPacket::Decode8(a2);                         // d
      break;
  }
  return mode;
}
