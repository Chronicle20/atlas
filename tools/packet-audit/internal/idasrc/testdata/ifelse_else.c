int __thiscall CWvsContext::OnSimpleResult(CWvsContext *this, CInPacket *a2)
{
  unsigned __int8 code = CInPacket::Decode1(a2);   // code
  if ( code == 1 )
  {
    CInPacket::Decode4(a2);                          // ok payload
  }
  else
  {
    CInPacket::Decode2(a2);                          // error payload
  }
  return code;
}
