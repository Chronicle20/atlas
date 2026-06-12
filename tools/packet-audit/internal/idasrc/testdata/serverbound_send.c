void __thiscall CLogin::SendBar(CLogin *this, int id)
{
  COutPacket::COutPacket(&oPacket, 12);
  COutPacket::Encode4(&oPacket, id);            // [WANT]
  COutPacket::Encode1(&oPacket, 1);             // [WANT]
  CClientSocket::SendPacket(this, &oPacket);    // [IGNORE]
}
