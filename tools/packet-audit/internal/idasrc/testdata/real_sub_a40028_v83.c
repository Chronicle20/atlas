/* line: 0, address: 0xa40028 */ int __thiscall sub_A40028(_DWORD *this, CInPacket *a2)
/* line: 1 */ {
/* line: 2 */   int result; // eax
/* line: 3 */   _DWORD *v4; // edi
/* line: 4 */   _DWORD *v5; // ebx
/* line: 5 */   _DWORD *v6; // esi
/* line: 6 */   BOOL *v7; // [esp+4h] [ebp-4h]
/* line: 7 */
/* line: 8, address: 0xa4002f */   result = sub_A4026A(this);
/* line: 9, address: 0xa40036 */   if ( !result )
/* line: 10 */   {
/* line: 11, address: 0xa40048 */     v4 = sub_A402F0(this, -1);
/* line: 12, address: 0xa40054 */     v5 = ZArray<int>::InsertBefore(this + 1, -1);
/* line: 13, address: 0xa40060 */     v7 = ZArray<int>::InsertBefore(this + 2, -1);
/* line: 14, address: 0xa4006d */     v6 = ZArray<int>::InsertBefore(this + 3, -1);
/* line: 15, address: 0xa4006f */     sub_4E4427(v4, a2);
/* line: 16, address: 0xa4007f */     *v5 = CInPacket::Decode1(a2);
/* line: 17, address: 0xa40093 */     *v7 = sub_49EB5E(TSingleton<CConfig>::ms_pInstance, *v4, 0);
/* line: 18, address: 0xa4009f */     result = sub_49EB5E(TSingleton<CConfig>::ms_pInstance, *v4, 1);
/* line: 19, address: 0xa400a5 */     *v6 = result;
/* line: 20 */   }
/* line: 21, address: 0xa400a8 */   return result;
/* line: 22 */ }
