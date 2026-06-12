int __thiscall CFoo::OnBar(CFoo *this, CInPacket *a2)
{
  int id = CInPacket::Decode4(a2);              // id
  CUIFadeYesNo::Create(a2);                      // denylisted UI helper (skip)
  StringPool::GetString(&s, id);                 // does not take a2 (skip)
  return id;
}
