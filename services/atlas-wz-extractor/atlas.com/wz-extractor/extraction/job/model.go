package job

import "time"

// JobStatus is the terminal/intermediate state of an extraction job.
type JobStatus string

const (
	JobPending             JobStatus = "pending"
	JobRunning             JobStatus = "running"
	JobCompleted           JobStatus = "completed"
	JobCompletedWithErrors JobStatus = "completed_with_errors"
	JobFailed              JobStatus = "failed"
)

// UnitStatus is per-WZ-file state.
type UnitStatus string

const (
	UnitPending   UnitStatus = "pending"
	UnitRunning   UnitStatus = "running"
	UnitSucceeded UnitStatus = "succeeded"
	UnitFailed    UnitStatus = "failed"
	UnitSkipped   UnitStatus = "skipped"
)

// Job is an immutable snapshot of one extraction job.
type Job struct {
	id             string
	tenantId       string
	region         string
	majorVersion   uint16
	minorVersion   uint16
	status         JobStatus
	unitsTotal     int
	unitsCompleted int
	unitsFailed    int
	xmlOnly        bool
	imagesOnly     bool
	createdAt      time.Time
	updatedAt      time.Time
	completedAt    time.Time
}

func (j Job) Id() string             { return j.id }
func (j Job) TenantId() string       { return j.tenantId }
func (j Job) Region() string         { return j.region }
func (j Job) MajorVersion() uint16   { return j.majorVersion }
func (j Job) MinorVersion() uint16   { return j.minorVersion }
func (j Job) Status() JobStatus      { return j.status }
func (j Job) UnitsTotal() int        { return j.unitsTotal }
func (j Job) UnitsCompleted() int    { return j.unitsCompleted }
func (j Job) UnitsFailed() int       { return j.unitsFailed }
func (j Job) XmlOnly() bool          { return j.xmlOnly }
func (j Job) ImagesOnly() bool       { return j.imagesOnly }
func (j Job) CreatedAt() time.Time   { return j.createdAt }
func (j Job) UpdatedAt() time.Time   { return j.updatedAt }
func (j Job) CompletedAt() time.Time { return j.completedAt }

// JobBuilder constructs an immutable Job.
type JobBuilder struct{ j Job }

func NewJobBuilder() *JobBuilder { return &JobBuilder{} }

func (b *JobBuilder) SetId(v string) *JobBuilder             { b.j.id = v; return b }
func (b *JobBuilder) SetTenantId(v string) *JobBuilder       { b.j.tenantId = v; return b }
func (b *JobBuilder) SetRegion(v string) *JobBuilder         { b.j.region = v; return b }
func (b *JobBuilder) SetMajorVersion(v uint16) *JobBuilder   { b.j.majorVersion = v; return b }
func (b *JobBuilder) SetMinorVersion(v uint16) *JobBuilder   { b.j.minorVersion = v; return b }
func (b *JobBuilder) SetStatus(v JobStatus) *JobBuilder      { b.j.status = v; return b }
func (b *JobBuilder) SetUnitsTotal(v int) *JobBuilder        { b.j.unitsTotal = v; return b }
func (b *JobBuilder) SetUnitsCompleted(v int) *JobBuilder    { b.j.unitsCompleted = v; return b }
func (b *JobBuilder) SetUnitsFailed(v int) *JobBuilder       { b.j.unitsFailed = v; return b }
func (b *JobBuilder) SetXmlOnly(v bool) *JobBuilder          { b.j.xmlOnly = v; return b }
func (b *JobBuilder) SetImagesOnly(v bool) *JobBuilder       { b.j.imagesOnly = v; return b }
func (b *JobBuilder) SetCreatedAt(v time.Time) *JobBuilder   { b.j.createdAt = v; return b }
func (b *JobBuilder) SetUpdatedAt(v time.Time) *JobBuilder   { b.j.updatedAt = v; return b }
func (b *JobBuilder) SetCompletedAt(v time.Time) *JobBuilder { b.j.completedAt = v; return b }
func (b *JobBuilder) Build() Job                             { return b.j }

// Unit is an immutable per-WZ-file record.
type Unit struct {
	wzFile      string
	status      UnitStatus
	startedAt   time.Time
	completedAt time.Time
	errMsg      string
}

func (u Unit) WzFile() string         { return u.wzFile }
func (u Unit) Status() UnitStatus     { return u.status }
func (u Unit) StartedAt() time.Time   { return u.startedAt }
func (u Unit) CompletedAt() time.Time { return u.completedAt }
func (u Unit) ErrorMessage() string   { return u.errMsg }

type UnitBuilder struct{ u Unit }

func NewUnitBuilder() *UnitBuilder                           { return &UnitBuilder{} }
func (b *UnitBuilder) SetWzFile(v string) *UnitBuilder       { b.u.wzFile = v; return b }
func (b *UnitBuilder) SetStatus(v UnitStatus) *UnitBuilder   { b.u.status = v; return b }
func (b *UnitBuilder) SetStartedAt(v time.Time) *UnitBuilder { b.u.startedAt = v; return b }
func (b *UnitBuilder) SetCompletedAt(v time.Time) *UnitBuilder {
	b.u.completedAt = v
	return b
}
func (b *UnitBuilder) SetErrorMessage(v string) *UnitBuilder { b.u.errMsg = v; return b }
func (b *UnitBuilder) Build() Unit                           { return b.u }

// Counters returned by FinalizeUnit; what the consumer needs to decide whether
// it's the "last one home" without a second Redis read.
type Counters struct {
	UnitsTotal     int
	UnitsCompleted int
	UnitsFailed    int
	AllDone        bool
	LockKey        string
}
