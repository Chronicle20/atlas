int __thiscall CFoo::OnBar(CFoo *this, CInPacket *a2)
{
  int id = CInPacket::Decode4(a2);             // id
  (*(void (__thiscall **)(CFoo *, CInPacket *))(*this + 4 * id))(this, a2); // vtable dispatch
  return id;
}
