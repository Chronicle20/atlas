int __thiscall CFoo::OnLeaf(CFoo *this, CInPacket *a2)
{
  CInPacket::Decode4(a2);          // id
  unsigned __int8 has = CInPacket::Decode1(a2);
  if ( has )
  {
    CInPacket::DecodeStr(a2, &s);  // optional name
  }
  return 0;
}
