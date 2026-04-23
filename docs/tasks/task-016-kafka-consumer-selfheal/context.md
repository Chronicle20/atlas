# Context — task-016 Kafka Consumer Self-Healing + Visibility

This document captures the incident evidence and code audit that motivated the PRD. Reviewers should be able to verify the PRD's claims against these references without re-running the investigation.

---

## 1. The motivating incident

**What the user reported (2026-04-20):**

> I started quest 1034, which awards 10× item 4031792 on quest start. I forfeited the quest, and started it again. Which awarded 10× 4031792 again. In theory. But my inventory only showed 10 in the slot that had them before. I tried dropping the stack of 10, it generated a drop, but did not remove them from my inventory. My character was then stuck, and I had to relog in order to interact with anything. I still had 10 in my inventory, but did see the drop of 10 on the ground. I dropped the stack of 10 again, it generated a drop again, but did not remove it from my inventory. Again my character was stuck. I relogged. My inventory had none of those items in it, and the two stacks were still on the ground.

**Environment at the time of the incident:**
- Cluster-run: atlas-inventory, atlas-quest, atlas-drops, atlas-maps, etc. (standard `atlas` namespace pods).
- Locally-run: atlas-channel, atlas-saga-orchestrator, atlas-npc-conversations (GoLand debug build of atlas-channel, per the startup log: `<home>/.cache/JetBrains/GoLand2026.1/tmp/GoLand/___go_build_atlas_channel`).
- Character ID 11, tenant `ec876921-c363-4cc6-9c51-5bb8d57f9553`.

## 2. Server-side timeline (from cluster atlas-quest pod)

`atlas-quest-66994744bf-czpsr` consumed the following `EVENT_TOPIC_ASSET_STATUS` messages for this character during the second reproduction attempt:

```
00:48:27.519  transactionId=765d0650  assetId=57  tpl=4031792  slot=2  qty=10   type=CREATED
00:48:56.807  transactionId=cc30b3d2  assetId=57  tpl=4031792  slot=2  qty=20   type=QUANTITY_CHANGED
```

Interpretation:
- First start of quest 1034 at `00:48:27` created assetId 57 with quantity 10 (new stack).
- Second start of quest 1034 at `00:48:56` (after a forfeit at 20:48:48 local) merged the new +10 into the existing stack, bumping quantity to 20.
- atlas-inventory produced both messages; atlas-quest consumed them; server-side state was correct.

## 3. Client-side timeline (from local atlas-channel process)

Same time window, `atlas-channel` session `dc71e887-1d84-478f-97c4-061a3f43ae98` → `e87b5d93-8632-487a-8abe-eec6c1b2eb34` → `2f6b8bfa-0f22-4821-8ddc-bd08105e78b0` (three successive client logins as the user relogged to recover). Across the **entire** log window for character 11 spanning quest 1034 start / forfeit / re-start / drop / relog sequences, atlas-channel received:

- Messages from `EVENT_TOPIC_QUEST_STATUS` (STARTED, FORFEITED) ✓
- Messages from `EVENT_TOPIC_SAGA_STATUS` (quest_start COMPLETED) ✓
- Messages from `EVENT_TOPIC_CHARACTER_STATUS`, `EVENT_TOPIC_MAP_STATUS`, `EVENT_TOPIC_CASH_SHOP_STATUS`, `EVENT_TOPIC_ACCOUNT_SESSION_STATUS`, `EVENT_TOPIC_ACCOUNT_STATUS` ✓
- **Zero** messages from `EVENT_TOPIC_ASSET_STATUS`. ✗

The consumer **was** registered at startup. From the startup log:

```
20:52:15.451  "Creating topic consumer."  originator=EVENT_TOPIC_ASSET_STATUS
20:52:15.451  "Start consuming topic."    originator=EVENT_TOPIC_ASSET_STATUS
```

The subscription was present. The broker was reachable (other topics were delivering). The consumer group id (`Channel Service - e7fb1d7e-47b8-46bd-97dc-867d93530001`) was visible in `kafka-consumer-groups --describe` alongside the other working topics of the same group. The fetcher simply stopped delivering messages to the handler layer, without any error log, at some point between startup and the test run.

A process restart of atlas-channel resolved the symptom. No code change, no config change.

## 4. Audit — silent-exit paths in `libs/atlas-kafka/consumer/manager.go`

File: `libs/atlas-kafka/consumer/manager.go` at the commit of this audit (monorepo HEAD 2026-04-20).

### 4.1 `io.EOF` treated as clean shutdown

Lines 179, 190–192:

```
179:            if err == io.EOF || errors.Is(err, context.Canceled) {
180:                return false, err
...
190:            if err == io.EOF || errors.Is(err, context.Canceled) {
191:                l.Infof("Reader closed, shutdown.")
192:                return
```

`kafka-go`'s `Reader.FetchMessage` can return `io.EOF` on a transient disconnect, a rebalance-induced fetcher restart, or any path where the underlying connection is torn down without the reader being `Close()`d. The code treats all of these as "reader was closed, exit cleanly." The outer goroutine then blocks at line 208 on `<-ctx.Done()` forever, and from the Manager's perspective the consumer is still registered. This matches the incident exactly: no error log, consumer silently dead.

### 4.2 Retry exhaustion exits the loop

Lines 188, 193–196:

```
188:            cfg := retry.DefaultConfig().WithMaxRetries(10).WithInitialDelay(100 * time.Millisecond).WithMaxDelay(10 * time.Second)
189:            err := retry.Try(readerCtx, cfg, readerFunc)
...
193:            } else if err != nil {
194:                l.WithError(err).Errorf("Could not successfully fetch message, exiting consumer loop.")
195:                return
196:            }
```

After 10 retries (total elapsed ~25 seconds given the backoff curve), the loop logs an `Errorf` and returns. The only observable signal is that one log line. The consumer is now dead with no recovery mechanism and no visibility beyond that one line.

### 4.3 Outer goroutine can't tell the inner died

Lines 171–173, 205, 208:

```
171:    done := make(chan struct{})
172:    go func() {
173:        defer close(done)
...
205:    }()
...
207:    l.Infof("Start consuming topic.")
208:    <-ctx.Done()
209:    l.Infof("Shutting down topic consumer.")
```

The fetcher runs on an inner goroutine that signals termination via `close(done)`. The outer `start` goroutine blocks on `<-ctx.Done()` alone — nothing reads `<-done` before `<-ctx.Done()` fires. If the inner goroutine exits early (for any reason from §4.1 or §4.2), the outer goroutine is oblivious until the parent context finally cancels, at which point `<-done` is already closed and the shutdown path no-ops. In the meantime, `Manager.consumers[topic]` still points to a `Consumer` struct whose reader is either closed or stuck; external observers (including a hypothetical health endpoint) would report "registered" with no distinction from a healthy consumer.

### 4.4 Handler-side panic recovery is correct

Lines 261–269:

```
261:    func (c *Consumer) safeHandle(h handler.Handler, l logrus.FieldLogger, ctx context.Context, msg kafka.Message) (cont bool, err error) {
262:        defer func() {
263:            if r := recover(); r != nil {
264:                cont = true
265:                err = fmt.Errorf("handler panicked: %v", r)
266:            }
267:        }()
268:        return h(l, ctx, msg)
269:    }
```

A handler panic is recovered, logged as a handler error, and the loop continues. This is not a silent-failure path. **No change is required here.**

### 4.5 Handler-deregistration path is orthogonal

Lines 244–249:

```
244:            cont, handlerErr := c.safeHandle(handle, handlerLogger, wctx, msg)
245:            if !cont {
246:                c.mu.Lock()
247:                delete(c.handlers, handleId)
248:                c.mu.Unlock()
249:            }
```

A handler that returns `cont=false` gets silently removed from the handler map. This is a conscious design — a handler can deregister itself by returning `cont=false`. It is not a failure mode, and it is orthogonal to the fetcher-death issue we're solving. **No change is required here.**

## 5. Kafka-broker state during the incident

`kubectl -n kafka exec kafka-broker-0 -- kafka-consumer-groups.sh --describe --group "Channel Service - e7fb1d7e-47b8-46bd-97dc-867d93530001"` during the incident showed `EVENT_TOPIC_ASSET_STATUS` with a consumer-id assigned to the atlas-channel host (`___go_build_atlas_channel@DESKTOP-KPLS7AL`) — i.e., from Kafka's perspective the consumer was still joined to the group. The issue was not that atlas-channel had left the group; it was that the goroutine responsible for fetching from the partition was no longer pulling. Both current-offset and log-end-offset were equal when checked after the incident, because the consumer had been restarted by then and had caught up.

This is the key evidence that distinguishes this from a "consumer never joined" problem: the group membership is fine, the fetcher is dead.

## 6. Ingress constraint for the debug route

`deploy/k8s/ingress.yaml` (ConfigMap `atlas-ingress-configmap`) uses nginx location rules like:

```
location ~ ^/api/merchants(/.*)?$ {
  proxy_pass http://atlas-merchant:8080;
}
location ~ ^/api/messengers(/.*)?$ {
  proxy_pass http://atlas-messengers:8080;
}
...
```

Each service owns a unique top-level `/api/<resource>` prefix. A naive `/api/debug/consumers` path would collide across the 49 consumer-owning services: nginx can only proxy that path to one backend. This is the reason the PRD explicitly **excludes** ingress exposure of the debug route in this task; access is cluster-internal only. If ingress exposure is pursued as a follow-up, a per-service path like `/api/debug/<service-slug>/consumers` would need to be added as a new nginx location rule per service (~49 rules). That's a mechanical follow-up, not a scope decision for this task.

## 7. Services without an existing REST server

`grep -l "server\.New\|AddRouteInitializer\|REST_PORT" services/*/atlas.com/*/main.go` identifies 8 services that do not own a REST server today:

- atlas-asset-expiration
- atlas-channel
- atlas-consumables
- atlas-expressions
- atlas-fame
- atlas-login
- atlas-messages
- atlas-monster-death

These are Kafka-consumer-only or socket-only services. Under this task, each gains a minimal `libs/atlas-rest/server.Builder` scaffold serving exactly `GET /api/debug/consumers`. The scaffold is ~5 lines of Go plus the `REST_PORT` / `containerPort` additions in their k8s manifests under `deploy/k8s/`. This is mechanical, reviewable at-a-glance, and keeps the debug surface uniform across the monorepo — critical because the service that motivated the task, atlas-channel, is one of these 8.

## 8. Related prior work

No prior task in `docs/tasks/` addresses consumer reliability or Kafka-consumer observability. The 15 existing tasks (`task-001-deploy-reorg` through `task-015-quest-start-reward-notices`) cover deployment, feature work, and data-model refactors. This is greenfield for this area.

`docs/TODO.md` has no entries for Kafka consumer infrastructure. The auto-memory `MEMORY.md` lists Kafka patterns as "message.Buffer for batching, message.Emit(p) for atomic emission, curried consumer registration InitConsumers(l)(cmf)(groupId)" — consistent with what the PRD preserves at the registration API layer.

## 9. Decisions that landed during design

- **Recovery over crash.** The user explicitly rejected a "panic on error → k8s restart" approach. Rationale: atlas-channel holds live game sessions; a process restart disconnects every connected player.
- **No staleness heuristics.** The user explicitly rejected "consumer silent for N seconds → alarm." Rationale: many topics are legitimately idle in a test system with no user activity. A time-based alarm would produce false positives on every non-peak hour.
- **Single REST server per service.** Initial proposal had a dedicated debug HTTP port on every service. User flagged this as a second REST server per service and pushed back. Revised to mount the debug route on the existing REST server, with a minimal REST server added to the 8 services that don't have one.
- **JSON:API payload for consistency.** Every other Atlas service's REST resource is JSON:API; the debug route follows suit.
- **No ingress exposure in this task.** Access is cluster-internal; ingress rules are a follow-up if the need arises.
- **No tenant scoping on the debug route.** Consumer groups are per-service-process, not per-tenant. The route reports service-level state.
- **Opt-in per service for the debug route.** Each `main.go` adds one line; there is no auto-wiring that would surprise a future service author.
