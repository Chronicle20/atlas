int __thiscall CLogin::OnCheckPasswordResult(CLogin *this, CInPacket *a2)
{
  unsigned __int8 result = CInPacket::Decode1(a2);  // result code
  if ( result == 2 )
  {
    CInPacket::Decode1(a2);                          // ban kind
    CInPacket::Decode8(a2);                          // ban until
  }
  else if ( result == 5 )
  {
    CInPacket::Decode4(a2);                          // ban reason
  }
  else
  {
    CInPacket::Decode2(a2);                          // generic message
  }
  return result;
}
