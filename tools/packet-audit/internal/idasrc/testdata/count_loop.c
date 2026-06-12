int __thiscall CParty::OnList(CParty *this, CInPacket *a2)
{
  int count = CInPacket::Decode4(a2);            // memberCount
  for ( i = 0; i < count; ++i )
  {
    CInPacket::Decode4(a2);                       // member id
    CInPacket::DecodeStr(a2, &name);             // member name
  }
  return count;
}
