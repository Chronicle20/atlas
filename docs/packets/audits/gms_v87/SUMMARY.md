# Audit summary

| Packet | Verdict | Atlas file |
|---|---|---|
| [Disable](Disable.md) | ✅ | `libs/atlas-packet/ui/clientbound/disable.go` |
| [GiveResponse](GiveResponse.md) | ❌ | `libs/atlas-packet/fame/clientbound/response.go` |
| [ReceiveResponse](ReceiveResponse.md) | ✅ | `libs/atlas-packet/fame/clientbound/response.go` |
| [ErrorResponse](ErrorResponse.md) | ✅ | `libs/atlas-packet/fame/clientbound/response.go` |
| [Ping](Ping.md) | ✅ | `libs/atlas-packet/socket/clientbound/ping.go` |
| [Change](Change.md) | ✅ | `libs/atlas-packet/fame/serverbound/change.go` |
| [RegisterPin](RegisterPin.md) | ✅ | `libs/atlas-packet/account/serverbound/register_pin.go` |
| [SetGender](SetGender.md) | ✅ | `libs/atlas-packet/account/serverbound/set_gender.go` |
| [Changed](Changed.md) | ❌ | `libs/atlas-packet/stat/clientbound/changed.go` |
| [Lock](Lock.md) | ✅ | `libs/atlas-packet/ui/clientbound/lock.go` |
| [ErrorSimple](ErrorSimple.md) | ✅ | `libs/atlas-packet/merchant/clientbound/operation.go` |
| [ShopRename](ShopRename.md) | ✅ | `libs/atlas-packet/merchant/clientbound/operation.go` |
| [FreeFormNotice](FreeFormNotice.md) | ✅ | `libs/atlas-packet/merchant/clientbound/operation.go` |
| [ScriptProgress](ScriptProgress.md) | ✅ | `libs/atlas-packet/quest/clientbound/script_progress.go` |
| [Hello](Hello.md) | ❌ | `libs/atlas-packet/socket/clientbound/hello.go` |
| [ChannelChangeRequest](ChannelChangeRequest.md) | ✅ | `libs/atlas-packet/channel/serverbound/channel_change.go` |
| [ChannelChange](ChannelChange.md) | ❌ | `libs/atlas-packet/buddy/clientbound/channel_change.go` |
| [ShopSearch](ShopSearch.md) | ✅ | `libs/atlas-packet/merchant/clientbound/operation.go` |
| [RemoteShopWarp](RemoteShopWarp.md) | ✅ | `libs/atlas-packet/merchant/clientbound/operation.go` |
| [Action](Action.md) | ✅ | `libs/atlas-packet/quest/serverbound/action.go` |
| [ActionScriptStart](ActionScriptStart.md) | ✅ | `libs/atlas-packet/quest/serverbound/action_script_start.go` |
| [ActionScriptEnd](ActionScriptEnd.md) | ✅ | `libs/atlas-packet/quest/serverbound/action_script_end.go` |
| [StartError](StartError.md) | ✅ | `libs/atlas-packet/socket/serverbound/start_error.go` |
| [Open](Open.md) | ✅ | `libs/atlas-packet/ui/clientbound/open.go` |
| [OpenShop](OpenShop.md) | ✅ | `libs/atlas-packet/merchant/clientbound/operation.go` |
| [ConfirmManage](ConfirmManage.md) | ✅ | `libs/atlas-packet/merchant/clientbound/operation.go` |
| [ChannelConnect](ChannelConnect.md) | ❌ | `libs/atlas-packet/socket/serverbound/channel_connect.go` |
| [Pong](Pong.md) | ✅ | `libs/atlas-packet/socket/serverbound/pong.go` |
