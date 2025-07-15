package scheduler

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"atlas-family/database"
	"atlas-family/family"
	"atlas-family/kafka/message"
	familymsg "atlas-family/kafka/message/family"
	"atlas-family/kafka/producer"
	"github.com/sirupsen/logrus"
)

// ReputationResetJob handles the daily reputation reset scheduling
type ReputationResetJob struct {
	log          logrus.FieldLogger
	administrator family.Administrator
	producer     producer.Provider
	resetHour    int
	resetMinute  int
	timezone     *time.Location
}

// ReputationResetJobConfig contains configuration for the reputation reset job
type ReputationResetJobConfig struct {
	Log          logrus.FieldLogger
	Administrator family.Administrator
	Producer     producer.Provider
	ResetHour    int
	ResetMinute  int
	Timezone     *time.Location
}

// NewReputationResetJob creates a new reputation reset job
func NewReputationResetJob(config ReputationResetJobConfig) *ReputationResetJob {
	return &ReputationResetJob{
		log:          config.Log,
		administrator: config.Administrator,
		producer:     config.Producer,
		resetHour:    config.ResetHour,
		resetMinute:  config.ResetMinute,
		timezone:     config.Timezone,
	}
}

// NewReputationResetJobFromEnv creates a new reputation reset job from environment variables
func NewReputationResetJobFromEnv(log logrus.FieldLogger, administrator family.Administrator, producer producer.Provider) *ReputationResetJob {
	// Default to midnight UTC
	resetHour := 0
	resetMinute := 0
	timezone := time.UTC

	// Check for custom reset hour
	if hourStr, ok := os.LookupEnv("REPUTATION_RESET_HOUR"); ok {
		if hour, err := strconv.Atoi(hourStr); err == nil && hour >= 0 && hour <= 23 {
			resetHour = hour
		}
	}

	// Check for custom reset minute
	if minuteStr, ok := os.LookupEnv("REPUTATION_RESET_MINUTE"); ok {
		if minute, err := strconv.Atoi(minuteStr); err == nil && minute >= 0 && minute <= 59 {
			resetMinute = minute
		}
	}

	// Check for custom timezone
	if tzStr, ok := os.LookupEnv("REPUTATION_RESET_TIMEZONE"); ok {
		if tz, err := time.LoadLocation(tzStr); err == nil {
			timezone = tz
		}
	}

	return NewReputationResetJob(ReputationResetJobConfig{
		Log:          log,
		Administrator: administrator,
		Producer:     producer,
		ResetHour:    resetHour,
		ResetMinute:  resetMinute,
		Timezone:     timezone,
	})
}

// Start begins the reputation reset job scheduler
func (j *ReputationResetJob) Start(ctx context.Context) error {
	j.log.WithFields(logrus.Fields{
		"resetHour":   j.resetHour,
		"resetMinute": j.resetMinute,
		"timezone":    j.timezone.String(),
	}).Info("Starting reputation reset job scheduler")

	// Start the scheduling goroutine
	go j.scheduleResetJob(ctx)
	
	return nil
}

// scheduleResetJob runs the daily scheduling loop
func (j *ReputationResetJob) scheduleResetJob(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			j.log.Info("Reputation reset job scheduler stopped")
			return
		default:
			// Calculate next reset time
			nextReset := j.calculateNextResetTime()
			
			j.log.WithFields(logrus.Fields{
				"nextReset": nextReset.Format(time.RFC3339),
			}).Info("Next reputation reset scheduled")
			
			// Sleep until the next reset time
			sleepDuration := time.Until(nextReset)
			
			select {
			case <-ctx.Done():
				j.log.Info("Reputation reset job scheduler stopped")
				return
			case <-time.After(sleepDuration):
				// Execute the reset job
				if err := j.executeResetJob(ctx); err != nil {
					j.log.WithError(err).Error("Failed to execute reputation reset job")
				}
			}
		}
	}
}

// calculateNextResetTime calculates the next time the reset job should run
func (j *ReputationResetJob) calculateNextResetTime() time.Time {
	now := time.Now().In(j.timezone)
	
	// Calculate today's reset time
	todayReset := time.Date(now.Year(), now.Month(), now.Day(), j.resetHour, j.resetMinute, 0, 0, j.timezone)
	
	// If today's reset time has already passed, schedule for tomorrow
	if now.After(todayReset) {
		return todayReset.Add(24 * time.Hour)
	}
	
	return todayReset
}

// executeResetJob performs the actual reputation reset operation
func (j *ReputationResetJob) executeResetJob(ctx context.Context) error {
	j.log.Info("Starting daily reputation reset job")
	
	startTime := time.Now()
	
	// Create database connection
	db := database.Connect(j.log)
	if db == nil {
		return fmt.Errorf("failed to connect to database")
	}
	
	// Get SQL database instance for connection management
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get SQL database instance: %w", err)
	}
	defer sqlDB.Close()
	
	// Execute the reset operation using the administrator
	result, err := j.administrator.BatchResetDailyRep(db, j.log)()()
	if err != nil {
		// Emit error event
		j.emitResetErrorEvent(ctx, err)
		return fmt.Errorf("failed to reset daily reputation: %w", err)
	}
	
	duration := time.Since(startTime)
	
	j.log.WithFields(logrus.Fields{
		"affectedMembers": result.AffectedCount,
		"duration":        duration.String(),
		"resetTime":       result.ResetTime.Format(time.RFC3339),
	}).Info("Daily reputation reset completed successfully")
	
	// Emit success event for audit trail
	j.emitResetSuccessEvent(ctx, result, duration)
	
	return nil
}

// emitResetSuccessEvent emits an event for successful reputation reset
func (j *ReputationResetJob) emitResetSuccessEvent(ctx context.Context, result family.BatchResetResult, duration time.Duration) {
	err := message.Emit(j.producer)(func(buf *message.Buffer) error {
		// Emit a global reputation reset event
		return buf.Put(familymsg.TopicFamilyReputation, family.RepResetEventProvider(0, 0, uint32(result.AffectedCount)))
	})
	
	if err != nil {
		j.log.WithError(err).Error("Failed to emit reputation reset success event")
	}
}

// emitResetErrorEvent emits an event for failed reputation reset
func (j *ReputationResetJob) emitResetErrorEvent(ctx context.Context, resetErr error) {
	err := message.Emit(j.producer)(func(buf *message.Buffer) error {
		// Emit a global reputation reset error event
		return buf.Put(familymsg.TopicFamilyErrors, family.RepErrorEventProvider(0, 0, "RESET_FAILED", resetErr.Error(), 0))
	})
	
	if err != nil {
		j.log.WithError(err).Error("Failed to emit reputation reset error event")
	}
}

// Stop gracefully stops the reputation reset job
func (j *ReputationResetJob) Stop() {
	j.log.Info("Stopping reputation reset job")
}