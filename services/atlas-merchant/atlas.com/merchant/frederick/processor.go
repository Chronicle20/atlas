package frederick

import (
	"context"
	"sync"
	"time"

	database "github.com/Chronicle20/atlas-database"
	"github.com/Chronicle20/atlas-model/model"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

const CleanupInterval = 6 * time.Hour
const CleanupAge = 100 * 24 * time.Hour

type Processor struct {
	l   logrus.FieldLogger
	ctx context.Context
	db  *gorm.DB
	t   tenant.Model
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) *Processor {
	return &Processor{
		l:   l,
		ctx: ctx,
		db:  db,
		t:   tenant.MustFromContext(ctx),
	}
}

type StoredItem struct {
	ItemId       uint32
	ItemType     byte
	Quantity     uint16
	ItemSnapshot []byte
}

// StoreItems moves unsold listing items into Frederick storage for a character.
func (p *Processor) StoreItems(characterId uint32, items []StoredItem) error {
	_, err := storeItems(p.t.Id(), characterId, items)(p.db.WithContext(p.ctx))()
	return err
}

// StoreMesos stores meso balance in Frederick storage for a character.
func (p *Processor) StoreMesos(characterId uint32, amount uint32) error {
	_, err := storeMesos(p.t.Id(), characterId, amount)(p.db.WithContext(p.ctx))()
	return err
}

// GetItems retrieves all items stored at Frederick for a character.
func (p *Processor) GetItems(characterId uint32) ([]ItemModel, error) {
	return model.SliceMap(MakeItem)(getItemsByCharacterId(characterId)(p.db.WithContext(p.ctx)))(model.ParallelMap())()
}

// GetMesos retrieves all meso records stored at Frederick for a character.
func (p *Processor) GetMesos(characterId uint32) ([]MesoModel, error) {
	return model.SliceMap(MakeMeso)(getMesosByCharacterId(characterId)(p.db.WithContext(p.ctx)))(model.ParallelMap())()
}

// ClearItems removes all items from Frederick storage for a character.
func (p *Processor) ClearItems(characterId uint32) error {
	_, err := clearItems(characterId)(p.db.WithContext(p.ctx))()
	return err
}

// ClearMesos removes all meso records from Frederick storage for a character.
func (p *Processor) ClearMesos(characterId uint32) error {
	_, err := clearMesos(characterId)(p.db.WithContext(p.ctx))()
	return err
}

// CreateNotification creates a Frederick notification record for a character.
func (p *Processor) CreateNotification(characterId uint32) error {
	_, err := createNotification(p.t, characterId)(p.db.WithContext(p.ctx))()
	return err
}

// ClearNotifications removes all notification records for a character.
func (p *Processor) ClearNotifications(characterId uint32) error {
	_, err := clearNotifications(characterId)(p.db.WithContext(p.ctx))()
	return err
}

// StartCleanupReaper starts a background goroutine that permanently deletes
// Frederick items and mesos older than 100 days.
func StartCleanupReaper(l logrus.FieldLogger, ctx context.Context, wg *sync.WaitGroup, db *gorm.DB) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		ticker := time.NewTicker(CleanupInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				l.Infoln("Frederick cleanup reaper shutting down.")
				return
			case <-ticker.C:
				reapExpired(l, ctx, db)
			}
		}
	}()
	l.Infoln("Frederick cleanup reaper started.")
}

func reapExpired(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) {
	noTenantCtx := database.WithoutTenantFilter(ctx)
	cutoff := time.Now().Add(-CleanupAge)

	rows, err := cleanupExpiredItems(cutoff)(db.WithContext(noTenantCtx))()
	if err != nil {
		l.WithError(err).Errorln("Error cleaning up expired Frederick items.")
	} else if rows > 0 {
		l.Infof("Cleaned up %d expired Frederick items.", rows)
	}

	rows, err = cleanupExpiredMesos(cutoff)(db.WithContext(noTenantCtx))()
	if err != nil {
		l.WithError(err).Errorln("Error cleaning up expired Frederick mesos.")
	} else if rows > 0 {
		l.Infof("Cleaned up %d expired Frederick meso records.", rows)
	}
}
