void __thiscall CLogin::OnFoo(CLogin *this, CInPacket *a2)
{
  int result = CInPacket::Decode1(a2);          // resultCode  [WANT]
  int id = CInPacket::Decode4(a2);              // accountId   [WANT]
  COutPacket::COutPacket(&oPacket, 26);
  COutPacket::Encode4(&oPacket, id);            // outgoing     [IGNORE]
  COutPacket::EncodeStr(&oPacket, &name);       // outgoing     [IGNORE]
  CClientSocket::SendPacket(this, &oPacket);    // outgoing     [IGNORE]
}
