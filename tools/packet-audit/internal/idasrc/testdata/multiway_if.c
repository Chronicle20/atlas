int __thiscall CFoo::OnMulti(CFoo *this, CInPacket *a2)
{
  unsigned __int8 v5 = CInPacket::Decode1(a2);
  if ( v5 == 1 )
  {
    CInPacket::Decode4(a2);
  }
  else if ( v5 == 2 )
  {
    CInPacket::Decode2(a2);
  }
  return v5;
}
