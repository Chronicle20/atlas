/* line: 0, address: 0xa3f2e8 */ void __thiscall CWvsContext::OnFriendResult(CWvsContext *this, int *Index)
/* line: 1 */ {
/* line: 2 */   struct CInPacket *v2; // edi
/* line: 3 */   int v4; // ecx
/* line: 4 */   unsigned int v5; // eax
/* line: 5 */   struct CUIFadeYesNo *v6; // ebx
/* line: 6 */   struct CUIFadeYesNo *v7; // eax
/* line: 7 */   _DWORD *v8; // edx
/* line: 8 */   int v9; // ecx
/* line: 9 */   int v10; // edi
/* line: 10 */   int v11; // eax
/* line: 11 */   unsigned int *v12; // eax
/* line: 12 */   int v13; // ecx
/* line: 13 */   struct CUIFadeYesNo *v14; // eax
/* line: 14 */   _DWORD *v15; // eax
/* line: 15 */   int v16; // eax
/* line: 16 */   unsigned int *v17; // eax
/* line: 17 */   int v18; // ecx
/* line: 18 */   struct CUIFadeYesNo *v19; // eax
/* line: 19 */   _DWORD *v20; // eax
/* line: 20 */   int v21; // eax
/* line: 21 */   bool v22; // zf
/* line: 22 */   int v23; // ebx
/* line: 23 */   void **v24; // eax
/* line: 24 */   unsigned int *v25; // eax
/* line: 25 */   int v26; // ecx
/* line: 26 */   struct CUIFadeYesNo *v27; // eax
/* line: 27 */   struct CUIFadeYesNo *v28; // edi
/* line: 28 */   struct CUIFadeYesNo **v29; // ecx
/* line: 29 */   int v30; // ebx
/* line: 30 */   int v31; // ecx
/* line: 31 */   int v32; // ecx
/* line: 32 */   _DWORD *Instance; // eax
/* line: 33 */   StringPoolStrings v34; // [esp-18h] [ebp-40h]
/* line: 34 */   int v35; // [esp-14h] [ebp-3Ch] BYREF
/* line: 35 */   const wchar_t *v36; // [esp-10h] [ebp-38h]
/* line: 36 */   size_t v37; // [esp-Ch] [ebp-34h] BYREF
/* line: 37 */   size_t v38[2]; // [esp-4h] [ebp-2Ch] BYREF
/* line: 38 */   unsigned int *v39; // [esp+Ch] [ebp-1Ch]
/* line: 39 */   char v40[4]; // [esp+10h] [ebp-18h] BYREF
/* line: 40 */   struct CUIFadeYesNo *v41; // [esp+14h] [ebp-14h] BYREF
/* line: 41 */   struct CUIFadeYesNo *v42; // [esp+18h] [ebp-10h] BYREF
/* line: 42 */   int v43; // [esp+24h] [ebp-4h]
/* line: 43 */
/* line: 44, address: 0xa3f2f8 */   v2 = Index;
/* line: 45, address: 0xa3f313 */   switch ( CInPacket::Decode1(Index) )
/* line: 46 */   {
/* line: 47, address: 0xa3f313 */     case 7:
/* line: 48, address: 0xa3f313 */     case 0xA:
/* line: 49, address: 0xa3f313 */     case 0x12:
/* line: 50, address: 0xa3f321 */       CWvsContext::CFriend::Reset(*&this[1].m_Cookie.szCookie[4036], v2);
/* line: 51, address: 0xa3f326 */       goto LABEL_27;
/* line: 52, address: 0xa3f313 */     case 8:
/* line: 53, address: 0xa3f59a */       CWvsContext::CFriend::UpdateFriend(*&this[1].m_Cookie.szCookie[4036], v2, 1);
/* line: 54, address: 0xa3f59f */       goto LABEL_27;
/* line: 55, address: 0xa3f313 */     case 9:
/* line: 56, address: 0xa3f4ea */       v41 = 0;
/* line: 57, address: 0xa3f4ee */       v23 = CInPacket::Decode4(v2);
/* line: 58, address: 0xa3f4f0 */       Index = 0;
/* line: 59, address: 0xa3f4f4 */       v43 = 4;
/* line: 60, address: 0xa3f505 */       v24 = CInPacket::DecodeStr(v2, v40);
/* line: 61, address: 0xa3f50e */       LOBYTE(v43) = 5;
/* line: 62, address: 0xa3f512 */       ZXString<char>::operator=(&Index, v24);
/* line: 63, address: 0xa3f51a */       LOBYTE(v43) = 4;
/* line: 64, address: 0xa3f51e */       ZXString<char>::_Release(v40);
/* line: 65, address: 0xa3f52a */       sub_A40028(*&this[1].m_Cookie.szCookie[4036], v2);
/* line: 66, address: 0xa3f539 */       v25 = ZAllocEx<ZAllocAnonSelector>::Alloc(dword_BF0B00, 0x118u);
/* line: 67, address: 0xa3f53e */       v39 = v25;
/* line: 68, address: 0xa3f543 */       LOBYTE(v43) = 6;
/* line: 69, address: 0xa3f547 */       if ( v25 )
/* line: 70 */       {
/* line: 71, address: 0xa3f54b */         CUIFadeYesNo::CUIFadeYesNo(v25);
/* line: 72, address: 0xa3f550 */         v28 = v27;
/* line: 73 */       }
/* line: 74 */       else
/* line: 75 */       {
/* line: 76, address: 0xa3f554 */         v28 = 0;
/* line: 77 */       }
/* line: 78, address: 0xa3f556 */       LODWORD(v38[0]) = v23;
/* line: 79, address: 0xa3f557 */       HIDWORD(v37) = v26;
/* line: 80, address: 0xa3f55d */       v39 = &v37 + 1;
/* line: 81, address: 0xa3f561 */       LOBYTE(v43) = 4;
/* line: 82, address: 0xa3f565 */       sub_428211(&v37 + 1, &Index);
/* line: 83, address: 0xa3f56c */       CUIFadeYesNo::CreateFriendReg(v28, SHIDWORD(v37), v38[0]);
/* line: 84, address: 0xa3f574 */       CWvsContext::SetNewFadeWnd(this, v28);
/* line: 85, address: 0xa3f57c */       LOBYTE(v43) = 3;
/* line: 86, address: 0xa3f580 */       ZXString<char>::_Release(&Index);
/* line: 87, address: 0xa3f585 */       v43 = -1;
/* line: 88, address: 0xa3f589 */       v29 = &v41;
/* line: 89, address: 0xa3f58c */       goto LABEL_32;
/* line: 90, address: 0xa3f313 */     case 0xB:
/* line: 91, address: 0xa3f637 */       LODWORD(v38[0]) = 0;
/* line: 92, address: 0xa3f63a */       v37 = 0x100000000LL;
/* line: 93, address: 0xa3f63b */       v36 = 0;
/* line: 94, address: 0xa3f63c */       v35 = v4;
/* line: 95, address: 0xa3f63f */       Index = &v35;
/* line: 96, address: 0xa3f642 */       v34 = SP_720_YOUR_BUDDY_LIST_IS_FULL;
/* line: 97, address: 0xa3f647 */       goto LABEL_39;
/* line: 98, address: 0xa3f313 */     case 0xC:
/* line: 99, address: 0xa3f64b */       LODWORD(v38[0]) = 0;
/* line: 100, address: 0xa3f64e */       v37 = 0x100000000LL;
/* line: 101, address: 0xa3f64f */       v36 = 0;
/* line: 102, address: 0xa3f650 */       v35 = v4;
/* line: 103, address: 0xa3f653 */       Index = &v35;
/* line: 104, address: 0xa3f656 */       v34 = SP_721_THE_USERS_BUDDY_LIST_IS_FULL;
/* line: 105, address: 0xa3f65b */       goto LABEL_39;
/* line: 106, address: 0xa3f313 */     case 0xD:
/* line: 107, address: 0xa3f65f */       LODWORD(v38[0]) = 0;
/* line: 108, address: 0xa3f662 */       v37 = 0x100000000LL;
/* line: 109, address: 0xa3f663 */       v36 = 0;
/* line: 110, address: 0xa3f664 */       v35 = v4;
/* line: 111, address: 0xa3f667 */       Index = &v35;
/* line: 112, address: 0xa3f66a */       v34 = SP_722_THAT_CHARACTER_IS_ALREADY_REGISTERED_AS_YOUR_BUDDY;
/* line: 113, address: 0xa3f66f */       goto LABEL_39;
/* line: 114, address: 0xa3f313 */     case 0xE:
/* line: 115, address: 0xa3f687 */       LODWORD(v38[0]) = 0;
/* line: 116, address: 0xa3f68a */       v37 = 0x100000000LL;
/* line: 117, address: 0xa3f68b */       v36 = 0;
/* line: 118, address: 0xa3f68c */       v35 = v4;
/* line: 119, address: 0xa3f68f */       Index = &v35;
/* line: 120, address: 0xa3f692 */       v34 = SP_724_GAMEMASTER_IS_NOT_AVAILABLE_AS_A_BUDDY;
/* line: 121, address: 0xa3f692 */       goto LABEL_39;
/* line: 122, address: 0xa3f313 */     case 0xF:
/* line: 123, address: 0xa3f673 */       LODWORD(v38[0]) = 0;
/* line: 124, address: 0xa3f676 */       v37 = 0x100000000LL;
/* line: 125, address: 0xa3f677 */       v36 = 0;
/* line: 126, address: 0xa3f678 */       v35 = v4;
/* line: 127, address: 0xa3f67b */       Index = &v35;
/* line: 128, address: 0xa3f67e */       v34 = SP_723_THAT_CHARACTER_IS_NOT_REGISTERED;
/* line: 129, address: 0xa3f683 */       goto LABEL_39;
/* line: 130, address: 0xa3f313 */     case 0x10:
/* line: 131, address: 0xa3f313 */     case 0x11:
/* line: 132, address: 0xa3f313 */     case 0x13:
/* line: 133, address: 0xa3f313 */     case 0x16:
/* line: 134, address: 0xa3f5d6 */       if ( CInPacket::Decode1(v2) )
/* line: 135 */       {
/* line: 136, address: 0xa3f5ea */         CInPacket::DecodeStr(v2, &v42);
/* line: 137, address: 0xa3f5ef */         LODWORD(v38[0]) = 0;
/* line: 138, address: 0xa3f5f2 */         v37 = 0x100000000LL;
/* line: 139, address: 0xa3f5f3 */         v36 = 0;
/* line: 140, address: 0xa3f5f4 */         v35 = v32;
/* line: 141, address: 0xa3f5fa */         Index = &v35;
/* line: 142, address: 0xa3f5fe */         v43 = 7;
/* line: 143, address: 0xa3f605 */         sub_428211(&v35, &v42);
/* line: 144, address: 0xa3f60a */         CUtilDlg::Notice(v35, v36, v37, SHIDWORD(v37), v38[0]);
/* line: 145, address: 0xa3f612 */         v43 = -1;
/* line: 146, address: 0xa3f616 */         v29 = &v42;
/* line: 147 */ LABEL_32:
/* line: 148, address: 0xa3f619 */         ZXString<char>::_Release(v29);
/* line: 149 */       }
/* line: 150 */       else
/* line: 151 */       {
/* line: 152, address: 0xa3f623 */         LODWORD(v38[0]) = 0;
/* line: 153, address: 0xa3f626 */         v37 = 0x100000000LL;
/* line: 154, address: 0xa3f627 */         v36 = 0;
/* line: 155, address: 0xa3f628 */         v35 = v31;
/* line: 156, address: 0xa3f62b */         Index = &v35;
/* line: 157, address: 0xa3f62e */         v34 = SP_719_THE_REQUEST_WAS_DENIED_DUE_TO_AN_UNKNOWN_ERROR;
/* line: 158 */ LABEL_39:
/* line: 159, address: 0xa3f697 */         Instance = StringPool::GetInstance();
/* line: 160, address: 0xa3f69f */         StringPool::GetString(Instance, &v35, v34);
/* line: 161, address: 0xa3f6a4 */         CUtilDlg::Notice(v35, v36, v37, SHIDWORD(v37), v38[0]);
/* line: 162 */       }
/* line: 163, address: 0xa3f61e */       return;
/* line: 164, address: 0xa3f313 */     case 0x14:
/* line: 165, address: 0xa3f32d */       v5 = CInPacket::Decode4(v2);
/* line: 166, address: 0xa3f340 */       Index = CWvsContext::CFriend::FindIndex(*&this[1].m_Cookie.szCookie[4036], v5);
/* line: 167, address: 0xa3f343 */       if ( Index < 0 )
/* line: 168, address: 0xa3f343 */         return;
/* line: 169, address: 0xa3f354 */       v6 = CInPacket::Decode1(v2);
/* line: 170, address: 0xa3f356 */       v7 = CInPacket::Decode4(v2);
/* line: 171, address: 0xa3f35b */       v8 = *&this[1].m_Cookie.szCookie[4036];
/* line: 172, address: 0xa3f36a */       v42 = *(v8[1] + 4 * Index);
/* line: 173, address: 0xa3f371 */       v9 = 39 * Index;
/* line: 174, address: 0xa3f37a */       v41 = *(39 * Index + *v8 + 18);
/* line: 175, address: 0xa3f382 */       if ( v41 == v7 && v42 == v6 )
/* line: 176, address: 0xa3f382 */         return;
/* line: 177, address: 0xa3f38a */       v10 = Index;
/* line: 178, address: 0xa3f38d */       *(v9 + *v8 + 18) = v7;
/* line: 179, address: 0xa3f39a */       *(*(*&this[1].m_Cookie.szCookie[4036] + 4) + 4 * v10) = v6;
/* line: 180, address: 0xa3f3aa */       if ( v41 >= 0 || v7 < 0 )
/* line: 181 */       {
/* line: 182, address: 0xa3f46a */         if ( v41 != v7 && v7 == *&this[1].m_Cookie.szCookie[20] )
/* line: 183 */         {
/* line: 184, address: 0xa3f476 */           v17 = ZAllocEx<ZAllocAnonSelector>::Alloc(dword_BF0B00, 0x118u);
/* line: 185, address: 0xa3f47b */           v39 = v17;
/* line: 186, address: 0xa3f480 */           v43 = 2;
/* line: 187, address: 0xa3f487 */           if ( v17 )
/* line: 188 */           {
/* line: 189, address: 0xa3f48b */             CUIFadeYesNo::CUIFadeYesNo(v17);
/* line: 190, address: 0xa3f490 */             v41 = v19;
/* line: 191 */           }
/* line: 192 */           else
/* line: 193 */           {
/* line: 194, address: 0xa3f495 */             v41 = 0;
/* line: 195 */           }
/* line: 196, address: 0xa3f498 */           v43 = -1;
/* line: 197, address: 0xa3f49c */           LODWORD(v38[0]) = 1;
/* line: 198, address: 0xa3f49e */           HIDWORD(v37) = v18;
/* line: 199, address: 0xa3f4a1 */           v39 = &v37 + 1;
/* line: 200, address: 0xa3f4a4 */           LODWORD(v37) = -1;
/* line: 201, address: 0xa3f4a6 */           v36 = Index;
/* line: 202, address: 0xa3f4af */           v20 = sub_A402EC(&this[1].m_Cookie.szCookie[4032]);
/* line: 203, address: 0xa3f4b6 */           v21 = sub_A4046F(v20, v36);
/* line: 204, address: 0xa3f4c1 */           ZXString<char>::ZXString<char>(&v37 + 1, (v21 + 4), v37);
/* line: 205, address: 0xa3f4c9 */           CUIFadeYesNo::CreateUserAlarm(v41, SHIDWORD(v37), v38[0]);
/* line: 206, address: 0xa3f4d3 */           CWvsContext::SetNewFadeWnd(this, v41);
/* line: 207 */         }
/* line: 208 */       }
/* line: 209 */       else
/* line: 210 */       {
/* line: 211, address: 0xa3f3b6 */         v11 = **&this[1].m_Cookie.szCookie[4036];
/* line: 212, address: 0xa3f3bc */         LODWORD(v38[0]) = -1;
/* line: 213, address: 0xa3f3c2 */         v41 = 0;
/* line: 214, address: 0xa3f3c5 */         ZXString<char>::GetBuffer(&v41, this, (v9 + v11 + 4), v38[0]);
/* line: 215, address: 0xa3f3d0 */         v42 = v38;
/* line: 216, address: 0xa3f3d4 */         v43 = 0;
/* line: 217, address: 0xa3f3d7 */         LODWORD(v38[0]) = 0;
/* line: 218, address: 0xa3f3d9 */         ZXString<char>::operator=(v38, &v41);
/* line: 219, address: 0xa3f3e0 */         if ( !CWvsContext::GetGuildMemberIDByName(this, v38[0]) )
/* line: 220 */         {
/* line: 221, address: 0xa3f3f3 */           v12 = ZAllocEx<ZAllocAnonSelector>::Alloc(dword_BF0B00, 0x118u);
/* line: 222, address: 0xa3f3f8 */           v42 = v12;
/* line: 223, address: 0xa3f3fd */           LOBYTE(v43) = 1;
/* line: 224, address: 0xa3f401 */           if ( v12 )
/* line: 225 */           {
/* line: 226, address: 0xa3f405 */             CUIFadeYesNo::CUIFadeYesNo(v12);
/* line: 227, address: 0xa3f40a */             v42 = v14;
/* line: 228 */           }
/* line: 229 */           else
/* line: 230 */           {
/* line: 231, address: 0xa3f40f */             v42 = 0;
/* line: 232 */           }
/* line: 233, address: 0xa3f412 */           LOBYTE(v43) = 0;
/* line: 234, address: 0xa3f416 */           LODWORD(v38[0]) = 0;
/* line: 235, address: 0xa3f417 */           HIDWORD(v37) = v13;
/* line: 236, address: 0xa3f41a */           v39 = &v37 + 1;
/* line: 237, address: 0xa3f41d */           LODWORD(v37) = -1;
/* line: 238, address: 0xa3f41f */           v36 = Index;
/* line: 239, address: 0xa3f428 */           v15 = sub_A402EC(&this[1].m_Cookie.szCookie[4032]);
/* line: 240, address: 0xa3f42f */           v16 = sub_A4046F(v15, v36);
/* line: 241, address: 0xa3f43a */           ZXString<char>::ZXString<char>(&v37 + 1, (v16 + 4), v37);
/* line: 242, address: 0xa3f442 */           CUIFadeYesNo::CreateUserAlarm(v42, SHIDWORD(v37), v38[0]);
/* line: 243, address: 0xa3f44c */           CWvsContext::SetNewFadeWnd(this, v42);
/* line: 244 */         }
/* line: 245, address: 0xa3f451 */         v43 = -1;
/* line: 246, address: 0xa3f458 */         ZXString<char>::_Release(&v41);
/* line: 247 */       }
/* line: 248, address: 0xa3f4d8 */       v22 = dword_BED784 == 0;
/* line: 249, address: 0xa3f4de */       goto LABEL_28;
/* line: 250, address: 0xa3f313 */     case 0x15:
/* line: 251, address: 0xa3f5a1 */       v30 = *&this[1].m_Cookie.szCookie[116];
/* line: 252, address: 0xa3f5b1 */       *(v30 + 1295) = CInPacket::Decode1(v2);
/* line: 253 */ LABEL_27:
/* line: 254, address: 0xa3f5b7 */       v22 = dword_BED784 == 0;
/* line: 255 */ LABEL_28:
/* line: 256, address: 0xa3f5be */       if ( !v22 )
/* line: 257, address: 0xa3f5ca */         CUIUserList::ResetInfo(*&this[1].m_Cookie.szReserved[1444]);
/* line: 258, address: 0xa3f5cf */       break;
/* line: 259 */     default:
/* line: 260 */       return;
/* line: 261 */   }
/* line: 262 */ }
