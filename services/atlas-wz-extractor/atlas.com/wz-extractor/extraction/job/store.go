package job

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

// Store persists Jobs and Units in Redis. All methods are safe for concurrent
// use; correctness across pods is enforced by Redis primitives (HINCRBY,
// WATCH/MULTI/EXEC) inside the implementation.
type Store interface {
	Create(ctx context.Context, j Job, units []Unit, ttlSeconds int) error
	Get(ctx context.Context, jobId string) (Job, []Unit, error)

	MarkJobRunning(ctx context.Context, jobId string) error
	MarkUnitRunning(ctx context.Context, jobId, wzFile string) (claimed bool, err error)
	FinalizeUnit(ctx context.Context, jobId, wzFile string, terminal UnitStatus, runErr error) (Counters, error)
	MarkJobTerminal(ctx context.Context, jobId string, terminal JobStatus) (claimed bool, err error)
	MarkUnitsSkippedByStatus(ctx context.Context, jobId string, fromStatuses []UnitStatus) error

	Delete(ctx context.Context, jobId string) error
}

// ErrNotFound is returned by Get when the jobId does not exist.
var ErrNotFound = errors.New("job not found")

type storeImpl struct {
	client *goredis.Client
}

func NewStore(client *goredis.Client) Store {
	return &storeImpl{client: client}
}

type unitJSON struct {
	Status      string `json:"status"`
	StartedAt   string `json:"startedAt,omitempty"`
	CompletedAt string `json:"completedAt,omitempty"`
	Error       string `json:"error,omitempty"`
}

func unitToJSON(u Unit) (string, error) {
	uj := unitJSON{Status: string(u.Status())}
	if !u.StartedAt().IsZero() {
		uj.StartedAt = u.StartedAt().UTC().Format(time.RFC3339)
	}
	if !u.CompletedAt().IsZero() {
		uj.CompletedAt = u.CompletedAt().UTC().Format(time.RFC3339)
	}
	uj.Error = u.ErrorMessage()
	b, err := json.Marshal(uj)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func unitFromJSON(wzFile, raw string) (Unit, error) {
	var uj unitJSON
	if err := json.Unmarshal([]byte(raw), &uj); err != nil {
		return Unit{}, err
	}
	b := NewUnitBuilder().SetWzFile(wzFile).SetStatus(UnitStatus(uj.Status))
	if uj.StartedAt != "" {
		if t, err := time.Parse(time.RFC3339, uj.StartedAt); err == nil {
			b = b.SetStartedAt(t)
		}
	}
	if uj.CompletedAt != "" {
		if t, err := time.Parse(time.RFC3339, uj.CompletedAt); err == nil {
			b = b.SetCompletedAt(t)
		}
	}
	if uj.Error != "" {
		b = b.SetErrorMessage(uj.Error)
	}
	return b.Build(), nil
}

func (s *storeImpl) Create(ctx context.Context, j Job, units []Unit, ttlSeconds int) error {
	jKey := jobKey(j.Id())
	uKey := unitsKey(j.Id())

	jobFields := map[string]interface{}{
		"tenantId":       j.TenantId(),
		"region":         j.Region(),
		"majorVersion":   strconv.Itoa(int(j.MajorVersion())),
		"minorVersion":   strconv.Itoa(int(j.MinorVersion())),
		"status":         string(j.Status()),
		"unitsTotal":     strconv.Itoa(j.UnitsTotal()),
		"unitsCompleted": "0",
		"unitsFailed":    "0",
		"xmlOnly":        strconv.FormatBool(j.XmlOnly()),
		"imagesOnly":     strconv.FormatBool(j.ImagesOnly()),
		"createdAt":      j.CreatedAt().UTC().Format(time.RFC3339),
		"updatedAt":      j.UpdatedAt().UTC().Format(time.RFC3339),
	}

	pipe := s.client.TxPipeline()
	pipe.HSet(ctx, jKey, jobFields)
	if ttlSeconds > 0 {
		pipe.Expire(ctx, jKey, time.Duration(ttlSeconds)*time.Second)
	}

	uMap := map[string]interface{}{}
	for _, u := range units {
		raw, err := unitToJSON(u)
		if err != nil {
			return err
		}
		uMap[u.WzFile()] = raw
	}
	if len(uMap) > 0 {
		pipe.HSet(ctx, uKey, uMap)
		if ttlSeconds > 0 {
			pipe.Expire(ctx, uKey, time.Duration(ttlSeconds)*time.Second)
		}
	}
	_, err := pipe.Exec(ctx)
	return err
}

func (s *storeImpl) Get(ctx context.Context, jobId string) (Job, []Unit, error) {
	fields, err := s.client.HGetAll(ctx, jobKey(jobId)).Result()
	if err != nil {
		return Job{}, nil, err
	}
	if len(fields) == 0 {
		return Job{}, nil, ErrNotFound
	}

	parseInt := func(v string) int {
		n, _ := strconv.Atoi(v)
		return n
	}
	parseTime := func(v string) time.Time {
		if v == "" {
			return time.Time{}
		}
		t, _ := time.Parse(time.RFC3339, v)
		return t
	}
	parseBool := func(v string) bool {
		b, _ := strconv.ParseBool(v)
		return b
	}

	jb := NewJobBuilder().
		SetId(jobId).
		SetTenantId(fields["tenantId"]).
		SetRegion(fields["region"]).
		SetMajorVersion(uint16(parseInt(fields["majorVersion"]))).
		SetMinorVersion(uint16(parseInt(fields["minorVersion"]))).
		SetStatus(JobStatus(fields["status"])).
		SetUnitsTotal(parseInt(fields["unitsTotal"])).
		SetUnitsCompleted(parseInt(fields["unitsCompleted"])).
		SetUnitsFailed(parseInt(fields["unitsFailed"])).
		SetXmlOnly(parseBool(fields["xmlOnly"])).
		SetImagesOnly(parseBool(fields["imagesOnly"])).
		SetCreatedAt(parseTime(fields["createdAt"])).
		SetUpdatedAt(parseTime(fields["updatedAt"])).
		SetCompletedAt(parseTime(fields["completedAt"]))
	j := jb.Build()

	uMap, err := s.client.HGetAll(ctx, unitsKey(jobId)).Result()
	if err != nil {
		return Job{}, nil, err
	}
	units := make([]Unit, 0, len(uMap))
	for wzFile, raw := range uMap {
		u, err := unitFromJSON(wzFile, raw)
		if err != nil {
			return Job{}, nil, err
		}
		units = append(units, u)
	}
	return j, units, nil
}

func (s *storeImpl) Delete(ctx context.Context, jobId string) error {
	_, err := s.client.Del(ctx, jobKey(jobId), unitsKey(jobId)).Result()
	return err
}

func (s *storeImpl) MarkJobRunning(ctx context.Context, jobId string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	pipe := s.client.TxPipeline()
	pipe.HSet(ctx, jobKey(jobId), "status", string(JobRunning), "updatedAt", now)
	_, err := pipe.Exec(ctx)
	return err
}
func (s *storeImpl) MarkUnitRunning(ctx context.Context, jobId, wzFile string) (bool, error) {
	uKey := unitsKey(jobId)

	var claimed bool
	txn := func(tx *goredis.Tx) error {
		raw, err := tx.HGet(ctx, uKey, wzFile).Result()
		if err != nil && err != goredis.Nil {
			return err
		}
		if err == goredis.Nil {
			return ErrNotFound
		}
		u, err := unitFromJSON(wzFile, raw)
		if err != nil {
			return err
		}
		if u.Status() == UnitSucceeded || u.Status() == UnitFailed || u.Status() == UnitSkipped {
			claimed = false
			return nil
		}
		nu := NewUnitBuilder().SetWzFile(wzFile).SetStatus(UnitRunning).
			SetStartedAt(time.Now().UTC()).Build()
		nraw, err := unitToJSON(nu)
		if err != nil {
			return err
		}
		_, err = tx.TxPipelined(ctx, func(p goredis.Pipeliner) error {
			p.HSet(ctx, uKey, wzFile, nraw)
			p.HSet(ctx, jobKey(jobId), "updatedAt", time.Now().UTC().Format(time.RFC3339))
			return nil
		})
		if err == nil {
			claimed = true
		}
		return err
	}

	for attempt := 0; attempt < 5; attempt++ {
		err := s.client.Watch(ctx, txn, uKey)
		if err == nil {
			return claimed, nil
		}
		if err == goredis.TxFailedErr {
			continue
		}
		return false, err
	}
	return false, errors.New("MarkUnitRunning: too many WATCH retries")
}
func (s *storeImpl) FinalizeUnit(ctx context.Context, jobId, wzFile string, terminal UnitStatus, runErr error) (Counters, error) {
	if terminal != UnitSucceeded && terminal != UnitFailed {
		return Counters{}, errors.New("FinalizeUnit: terminal must be Succeeded or Failed")
	}
	jKey := jobKey(jobId)
	uKey := unitsKey(jobId)

	var out Counters
	txn := func(tx *goredis.Tx) error {
		raw, err := tx.HGet(ctx, uKey, wzFile).Result()
		if err == goredis.Nil {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		u, err := unitFromJSON(wzFile, raw)
		if err != nil {
			return err
		}

		// Already terminal (redelivery): read counters, return them, no-op.
		if u.Status() == UnitSucceeded || u.Status() == UnitFailed || u.Status() == UnitSkipped {
			j, _, gerr := s.Get(ctx, jobId)
			if gerr != nil {
				return gerr
			}
			out = Counters{
				UnitsTotal:     j.UnitsTotal(),
				UnitsCompleted: j.UnitsCompleted(),
				UnitsFailed:    j.UnitsFailed(),
				AllDone:        (j.UnitsCompleted() + j.UnitsFailed()) == j.UnitsTotal(),
				LockKey:        LockKey(j.TenantId(), j.Region(), j.MajorVersion(), j.MinorVersion()),
			}
			return nil
		}

		nb := NewUnitBuilder().SetWzFile(wzFile).SetStatus(terminal).
			SetStartedAt(u.StartedAt()).
			SetCompletedAt(time.Now().UTC())
		if runErr != nil {
			nb = nb.SetErrorMessage(runErr.Error())
		}
		nraw, err := unitToJSON(nb.Build())
		if err != nil {
			return err
		}

		field := "unitsCompleted"
		if terminal == UnitFailed {
			field = "unitsFailed"
		}

		var totalCmd, completedCmd, failedCmd, tenantCmd, regionCmd, majCmd, minCmd *goredis.StringCmd
		var newCounter *goredis.IntCmd
		_, err = tx.TxPipelined(ctx, func(p goredis.Pipeliner) error {
			p.HSet(ctx, uKey, wzFile, nraw)
			newCounter = p.HIncrBy(ctx, jKey, field, 1)
			p.HSet(ctx, jKey, "updatedAt", time.Now().UTC().Format(time.RFC3339))
			totalCmd = p.HGet(ctx, jKey, "unitsTotal")
			completedCmd = p.HGet(ctx, jKey, "unitsCompleted")
			failedCmd = p.HGet(ctx, jKey, "unitsFailed")
			tenantCmd = p.HGet(ctx, jKey, "tenantId")
			regionCmd = p.HGet(ctx, jKey, "region")
			majCmd = p.HGet(ctx, jKey, "majorVersion")
			minCmd = p.HGet(ctx, jKey, "minorVersion")
			return nil
		})
		if err != nil {
			return err
		}
		_ = newCounter // increment already applied above; counters re-read post-EXEC

		total, _ := strconv.Atoi(totalCmd.Val())
		completed, _ := strconv.Atoi(completedCmd.Val())
		failed, _ := strconv.Atoi(failedCmd.Val())
		maj, _ := strconv.Atoi(majCmd.Val())
		min, _ := strconv.Atoi(minCmd.Val())
		out = Counters{
			UnitsTotal:     total,
			UnitsCompleted: completed,
			UnitsFailed:    failed,
			AllDone:        (completed + failed) == total,
			LockKey:        LockKey(tenantCmd.Val(), regionCmd.Val(), uint16(maj), uint16(min)),
		}
		return nil
	}

	for attempt := 0; attempt < 5; attempt++ {
		err := s.client.Watch(ctx, txn, uKey)
		if err == nil {
			return out, nil
		}
		if err == goredis.TxFailedErr {
			continue
		}
		return Counters{}, err
	}
	return Counters{}, errors.New("FinalizeUnit: too many WATCH retries")
}
func (s *storeImpl) MarkJobTerminal(ctx context.Context, jobId string, terminal JobStatus) (bool, error) {
	return false, errors.New("not implemented")
}
func (s *storeImpl) MarkUnitsSkippedByStatus(ctx context.Context, jobId string, fromStatuses []UnitStatus) error {
	return errors.New("not implemented")
}
