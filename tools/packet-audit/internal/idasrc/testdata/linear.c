int __thiscall CLogin::OnFoo(CLogin *this, CInPacket *a2)
{
  unsigned __int8 result = CInPacket::Decode1(a2);   // resultCode
  int accountId = CInPacket::Decode4(a2);            // accountId
  CInPacket::DecodeStr(a2, &name);                    // name
  return result;
}
