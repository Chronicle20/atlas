int __thiscall CFoo::OnNonEq(CFoo *this, CInPacket *a2)
{
  unsigned __int8 v5 = CInPacket::Decode1(a2);  // discriminator
  if ( v5 < 5 )
  {
    CInPacket::Decode2(a2);                       // small payload
  }
  else if ( v5 & 0x10 )
  {
    CInPacket::Decode4(a2);                       // flag payload
  }
  else
  {
    CInPacket::Decode8(a2);                        // default payload
  }
  return v5;
}
