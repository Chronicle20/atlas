package party

import (
	"atlas-parties/character"
	"atlas-parties/invite"
	"atlas-parties/kafka/producer"
	"context"
	"errors"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const StartPartyId = uint32(1000000000)

var ErrNotFound = errors.New("not found")
var ErrAtCapacity = errors.New("at capacity")
var ErrAlreadyIn = errors.New("already in party")
var ErrNotIn = errors.New("not in party")
var ErrNotAsBeginner = errors.New("not as beginner")
var ErrNotAsGm = errors.New("not as gm")

func allProvider(ctx context.Context) model.Provider[[]Model] {
	return func() ([]Model, error) {
		t := tenant.MustFromContext(ctx)
		return GetRegistry().GetAll(t), nil
	}
}

func byIdProvider(ctx context.Context) func(partyId uint32) model.Provider[Model] {
	return func(partyId uint32) model.Provider[Model] {
		return func() (Model, error) {
			t := tenant.MustFromContext(ctx)
			return GetRegistry().Get(t, partyId)
		}
	}
}

func MemberFilter(memberId uint32) model.Filter[Model] {
	return func(m Model) bool {
		for _, id := range m.members {
			if id == memberId {
				return true
			}
		}
		return false
	}
}

func GetSlice(ctx context.Context) func(filters ...model.Filter[Model]) ([]Model, error) {
	return func(filters ...model.Filter[Model]) ([]Model, error) {
		return model.FilteredProvider(allProvider(ctx), model.Filters[Model](filters...))()
	}
}

func GetById(ctx context.Context) func(partyId uint32) (Model, error) {
	return func(partyId uint32) (Model, error) {
		return byIdProvider(ctx)(partyId)()
	}
}

func GetByCharacter(ctx context.Context) func(characterId uint32) (Model, error) {
	return func(characterId uint32) (Model, error) {
		t := tenant.MustFromContext(ctx)
		return GetRegistry().GetPartyByCharacter(t, characterId)
	}
}

// Efficient provider for character-to-party lookup
func byCharacterProvider(ctx context.Context) func(characterId uint32) model.Provider[Model] {
	return func(characterId uint32) model.Provider[Model] {
		return func() (Model, error) {
			t := tenant.MustFromContext(ctx)
			return GetRegistry().GetPartyByCharacter(t, characterId)
		}
	}
}

// Efficient batch character-to-party lookup
func GetPartiesByCharacters(ctx context.Context) func(characterIds []uint32) ([]Model, error) {
	return func(characterIds []uint32) ([]Model, error) {
		t := tenant.MustFromContext(ctx)
		parties := make([]Model, 0, len(characterIds))
		seen := make(map[uint32]bool)
		
		for _, characterId := range characterIds {
			if party, err := GetRegistry().GetPartyByCharacter(t, characterId); err == nil {
				if !seen[party.Id()] {
					parties = append(parties, party)
					seen[party.Id()] = true
				}
			}
		}
		
		return parties, nil
	}
}

// Check if character is in any party (efficient O(1) lookup)
func IsCharacterInParty(ctx context.Context) func(characterId uint32) bool {
	return func(characterId uint32) bool {
		t := tenant.MustFromContext(ctx)
		_, err := GetRegistry().GetPartyByCharacter(t, characterId)
		return err == nil
	}
}

// Cache management functions
func GetCacheStats(ctx context.Context) (hits, misses uint64, hitRate float64) {
	t := tenant.MustFromContext(ctx)
	return GetRegistry().GetCacheStats(t)
}

func ClearCache(ctx context.Context) {
	t := tenant.MustFromContext(ctx)
	GetRegistry().ClearCache(t)
}

func GetCacheSize(ctx context.Context) int {
	t := tenant.MustFromContext(ctx)
	return GetRegistry().GetCacheSize(t)
}

func CleanupStaleCache(ctx context.Context) {
	t := tenant.MustFromContext(ctx)
	GetRegistry().CleanupStaleCache(t)
}

// RemoveCharacterFromParty removes a character from any party they belong to
// This is used for character deletion events and other cleanup scenarios
func RemoveCharacterFromParty(l logrus.FieldLogger) func(ctx context.Context) func(characterId uint32) (Model, error) {
	return func(ctx context.Context) func(characterId uint32) (Model, error) {
		return func(characterId uint32) (Model, error) {
			t := tenant.MustFromContext(ctx)
			
			// Find the party containing the character
			party, err := GetByCharacter(ctx)(characterId)
			if err != nil {
				if err == ErrNotFound {
					l.Debugf("Character [%d] not found in any party, nothing to remove.", characterId)
					return Model{}, nil
				}
				l.WithError(err).Errorf("Error finding party for character [%d].", characterId)
				return Model{}, err
			}
			
			partyId := party.Id()
			l.Debugf("Character [%d] found in party [%d], removing from party.", characterId, partyId)
			
			// Check if character is the leader
			isLeader := party.LeaderId() == characterId
			
			// Remove the character from the party
			updatedParty, err := GetRegistry().Update(t, partyId, func(m Model) Model { 
				return Model.RemoveMember(m, characterId) 
			})
			if err != nil {
				l.WithError(err).Errorf("Unable to remove character [%d] from party [%d].", characterId, partyId)
				return Model{}, err
			}
			
			// Handle party state after member removal
			if len(updatedParty.Members()) == 0 {
				// Get character info for event emission before disbanding
				char, charErr := character.GetById(l)(ctx)(characterId)
				if charErr != nil {
					l.WithError(charErr).Warnf("Unable to get character [%d] info for disband event emission.", characterId)
				}
				
				// Emit disband event before removing party
				if charErr == nil {
					err = producer.ProviderImpl(l)(ctx)(EnvEventStatusTopic)(disbandEventProvider(characterId, partyId, char.WorldId(), party.Members()))
					if err != nil {
						l.WithError(err).Warnf("Unable to emit disband event for party [%d].", partyId)
						// Don't return error as the disbanding will still proceed
					} else {
						l.Debugf("Emitted disband event for party [%d] due to character [%d] deletion.", partyId, characterId)
					}
				}
				
				// Party is empty, disband it
				GetRegistry().Remove(t, partyId)
				l.Debugf("Party [%d] disbanded after removing character [%d] (last member).", partyId, characterId)
				
				return Model{}, nil
			} else if isLeader {
				// Character was the leader, elect a new leader with enhanced handling
				newLeaderId, err := electNewLeader(l, ctx, updatedParty, characterId)
				if err != nil {
					l.WithError(err).Errorf("Unable to elect new leader for party [%d] after removing character [%d].", partyId, characterId)
					return Model{}, err
				}
				
				// Update party with new leader
				updatedParty, err = GetRegistry().Update(t, partyId, func(m Model) Model { 
					return Model.SetLeader(m, newLeaderId) 
				})
				if err != nil {
					l.WithError(err).Errorf("Unable to set new leader [%d] for party [%d] after removing character [%d].", newLeaderId, partyId, characterId)
					return Model{}, err
				}
				
				l.Infof("Character [%d] was leader of party [%d], elected new leader [%d] due to character deletion.", 
					characterId, partyId, newLeaderId)
				
				// Emit leader change event for remaining party members
				err = emitLeaderChangeEvent(l, ctx, updatedParty, characterId, newLeaderId)
				if err != nil {
					l.WithError(err).Warnf("Unable to emit leader change event for party [%d].", partyId)
					// Don't return error as the leader change was successful
				}
			} else {
				// Character was not the leader, just emit a left event
				char, charErr := character.GetById(l)(ctx)(characterId)
				if charErr != nil {
					l.WithError(charErr).Warnf("Unable to get character [%d] info for left event emission.", characterId)
				} else {
					err = producer.ProviderImpl(l)(ctx)(EnvEventStatusTopic)(leftEventProvider(characterId, partyId, char.WorldId()))
					if err != nil {
						l.WithError(err).Warnf("Unable to emit left event for character [%d] from party [%d].", characterId, partyId)
						// Don't return error as the removal was successful
					} else {
						l.Debugf("Emitted left event for character [%d] from party [%d] due to deletion.", characterId, partyId)
					}
				}
			}
			
			l.Debugf("Successfully removed character [%d] from party [%d].", characterId, partyId)
			return updatedParty, nil
		}
	}
}

// electNewLeader selects a new leader for the party using enhanced logic
func electNewLeader(l logrus.FieldLogger, ctx context.Context, party Model, deletedCharacterId uint32) (uint32, error) {
	members := party.Members()
	if len(members) == 0 {
		return 0, errors.New("no members available for leader election")
	}
	
	// Find the best candidate for leadership
	// Priority: 1. Online members, 2. Highest level, 3. Longest party member (first in list)
	var bestCandidate uint32
	var bestCandidateLevel byte = 0
	var bestCandidateOnline bool = false
	
	for _, memberId := range members {
		if memberId == deletedCharacterId {
			continue // Skip the deleted character
		}
		
		// Get character info to make informed decision
		char, err := character.GetById(l)(ctx)(memberId)
		if err != nil {
			l.WithError(err).Warnf("Unable to get character [%d] info for leader election, skipping.", memberId)
			// If we can't get character info, still consider them as a fallback
			if bestCandidate == 0 {
				bestCandidate = memberId
			}
			continue
		}
		
		isOnline := char.Online()
		level := char.Level()
		
		// If this is the first candidate, set as best
		if bestCandidate == 0 {
			bestCandidate = memberId
			bestCandidateLevel = level
			bestCandidateOnline = isOnline
			continue
		}
		
		// Prefer online members over offline ones
		if isOnline && !bestCandidateOnline {
			bestCandidate = memberId
			bestCandidateLevel = level
			bestCandidateOnline = isOnline
		} else if isOnline == bestCandidateOnline {
			// If online status is the same, prefer higher level
			if level > bestCandidateLevel {
				bestCandidate = memberId
				bestCandidateLevel = level
				bestCandidateOnline = isOnline
			}
		}
	}
	
	if bestCandidate == 0 {
		return 0, errors.New("no suitable candidate found for leader election")
	}
	
	l.Debugf("Elected character [%d] as new leader (level: %d, online: %t) for party [%d].", 
		bestCandidate, bestCandidateLevel, bestCandidateOnline, party.Id())
	
	return bestCandidate, nil
}

// emitLeaderChangeEvent emits a leader change event for character deletion scenarios
func emitLeaderChangeEvent(l logrus.FieldLogger, ctx context.Context, party Model, oldLeaderId, newLeaderId uint32) error {
	// Get the new leader's character info for world context
	newLeader, err := character.GetById(l)(ctx)(newLeaderId)
	if err != nil {
		l.WithError(err).Errorf("Unable to get new leader [%d] character info for event emission.", newLeaderId)
		return err
	}
	
	// Emit the leader change event
	// Use the old leader ID as the actor since they initiated the change (through deletion)
	err = producer.ProviderImpl(l)(ctx)(EnvEventStatusTopic)(changeLeaderEventProvider(oldLeaderId, party.Id(), newLeader.WorldId(), newLeaderId))
	if err != nil {
		l.WithError(err).Errorf("Unable to emit leader change event for party [%d].", party.Id())
		return err
	}
	
	l.Debugf("Emitted leader change event for party [%d]: old leader [%d] -> new leader [%d].", 
		party.Id(), oldLeaderId, newLeaderId)
	
	return nil
}

func ValidateMembership(ctx context.Context) func(partyId uint32, characterId uint32) error {
	return func(partyId uint32, characterId uint32) error {
		party, err := GetById(ctx)(partyId)
		if err != nil {
			return err
		}
		
		if !IsMember(party, characterId) {
			return ErrNotIn
		}
		
		return nil
	}
}

func IsMember(party Model, characterId uint32) bool {
	for _, memberId := range party.Members() {
		if memberId == characterId {
			return true
		}
	}
	return false
}

func IsLeader(party Model, characterId uint32) bool {
	return party.LeaderId() == characterId
}

func CanRemoveMember(party Model, characterId uint32) error {
	if !IsMember(party, characterId) {
		return ErrNotIn
	}
	
	// Leader can only be removed if there are other members to elect as new leader
	if IsLeader(party, characterId) && len(party.Members()) <= 1 {
		return errors.New("cannot remove leader from single-member party")
	}
	
	return nil
}

func ValidatePartyIntegrity(party Model) error {
	// Check if party has members
	if len(party.Members()) == 0 {
		return errors.New("party has no members")
	}
	
	// Check if leader is a member
	if !IsMember(party, party.LeaderId()) {
		return errors.New("leader is not a member of the party")
	}
	
	// Check for duplicate members
	memberSet := make(map[uint32]bool)
	for _, memberId := range party.Members() {
		if memberSet[memberId] {
			return errors.New("duplicate member found in party")
		}
		memberSet[memberId] = true
	}
	
	return nil
}

func Create(l logrus.FieldLogger) func(ctx context.Context) func(leaderId uint32) (Model, error) {
	return func(ctx context.Context) func(leaderId uint32) (Model, error) {
		return func(leaderId uint32) (Model, error) {
			t := tenant.MustFromContext(ctx)
			c, err := character.GetById(l)(ctx)(leaderId)
			if err != nil {
				l.WithError(err).Errorf("Error getting character [%d].", leaderId)
				err = producer.ProviderImpl(l)(ctx)(EnvEventStatusTopic)(errorEventProvider(leaderId, 0, c.WorldId(), EventPartyStatusErrorUnexpected, ""))
				if err != nil {
					l.WithError(err).Errorf("Unable to announce party create error to [%d].", leaderId)
				}
				return Model{}, err
			}

			if c.PartyId() != 0 {
				l.Errorf("Character [%d] already in party. Cannot create another one.", leaderId)
				err = producer.ProviderImpl(l)(ctx)(EnvEventStatusTopic)(errorEventProvider(leaderId, 0, c.WorldId(), EventPartyStatusErrorTypeAlreadyJoined1, ""))
				if err != nil {
					l.WithError(err).Errorf("Unable to announce party create error to [%d].", leaderId)
				}
				return Model{}, ErrAlreadyIn
			}

			if c.IsBeginner() {
				l.Errorf("Character [%d] is a beginner, cannot create parties.", leaderId)
				err = producer.ProviderImpl(l)(ctx)(EnvEventStatusTopic)(errorEventProvider(leaderId, 0, c.WorldId(), EventPartyStatusErrorTypeBeginnerCannotCreate, ""))
				if err != nil {
					l.WithError(err).Errorf("Unable to announce party create error to [%d].", leaderId)
				}
				return Model{}, ErrNotAsBeginner
			}

			if c.GM() > 0 {
				l.Errorf("Character [%d] is a GM, cannot create parties.", leaderId)
				err = producer.ProviderImpl(l)(ctx)(EnvEventStatusTopic)(errorEventProvider(leaderId, 0, c.WorldId(), EventPartyStatusErrorTypeGmCannotCreate, ""))
				if err != nil {
					l.WithError(err).Errorf("Unable to announce party create error to [%d].", leaderId)
				}
				return Model{}, ErrNotAsGm
			}

			p := GetRegistry().Create(t, leaderId)

			l.Debugf("Created party [%d] for leader [%d].", p.Id(), leaderId)

			err = producer.ProviderImpl(l)(ctx)(EnvEventStatusTopic)(createdEventProvider(leaderId, p.Id(), c.WorldId()))
			if err != nil {
				l.WithError(err).Errorf("Unable to announce the party [%d] was created.", leaderId)
				err = producer.ProviderImpl(l)(ctx)(EnvEventStatusTopic)(errorEventProvider(leaderId, 0, c.WorldId(), EventPartyStatusErrorUnexpected, ""))
				if err != nil {
					l.WithError(err).Errorf("Unable to announce party [%d] error.", p.Id())
				}
				return Model{}, err
			}

			err = character.JoinParty(l)(ctx)(leaderId, p.Id())
			if err != nil {
				l.WithError(err).Errorf("Unable to have character [%d] join party [%d]", leaderId, p.Id())
				err = producer.ProviderImpl(l)(ctx)(EnvEventStatusTopic)(errorEventProvider(leaderId, 0, c.WorldId(), EventPartyStatusErrorUnexpected, ""))
				if err != nil {
					l.WithError(err).Errorf("Unable to announce party [%d] error.", p.Id())
				}
				return Model{}, err
			}

			return p, nil
		}
	}
}

func Join(l logrus.FieldLogger) func(ctx context.Context) func(partyId uint32, characterId uint32) (Model, error) {
	return func(ctx context.Context) func(partyId uint32, characterId uint32) (Model, error) {
		return func(partyId uint32, characterId uint32) (Model, error) {
			t := tenant.MustFromContext(ctx)
			c, err := character.GetById(l)(ctx)(characterId)
			if err != nil {
				l.WithError(err).Errorf("Error getting character [%d].", characterId)
				err = producer.ProviderImpl(l)(ctx)(EnvEventStatusTopic)(errorEventProvider(characterId, partyId, c.WorldId(), EventPartyStatusErrorUnexpected, ""))
				if err != nil {
					l.WithError(err).Errorf("Unable to announce party [%d] error.", partyId)
				}
				return Model{}, err
			}

			if c.PartyId() != 0 {
				l.Errorf("Character [%d] already in party. Cannot create another one.", characterId)
				err = producer.ProviderImpl(l)(ctx)(EnvEventStatusTopic)(errorEventProvider(characterId, c.PartyId(), c.WorldId(), EventPartyStatusErrorTypeAlreadyJoined2, ""))
				if err != nil {
					l.WithError(err).Errorf("Unable to announce party [%d] error.", partyId)
				}
				return Model{}, ErrAlreadyIn
			}

			p, err := GetRegistry().Get(t, partyId)
			if err != nil {
				l.WithError(err).Errorf("Unable to retrieve party [%d].", partyId)
				err = producer.ProviderImpl(l)(ctx)(EnvEventStatusTopic)(errorEventProvider(characterId, partyId, c.WorldId(), EventPartyStatusErrorUnexpected, ""))
				if err != nil {
					l.WithError(err).Errorf("Unable to announce party [%d] error.", partyId)
				}
				return Model{}, err
			}

			if len(p.Members()) >= 6 {
				l.Errorf("Party [%d] already at capacity.", partyId)
				err = producer.ProviderImpl(l)(ctx)(EnvEventStatusTopic)(errorEventProvider(characterId, partyId, c.WorldId(), EventPartyStatusErrorTypeAtCapacity, ""))
				if err != nil {
					l.WithError(err).Errorf("Unable to announce party [%d] error.", partyId)
				}
				return Model{}, ErrAtCapacity
			}

			fn := func(m Model) Model { return Model.AddMember(m, characterId) }
			p, err = GetRegistry().Update(t, partyId, fn)
			if err != nil {
				l.WithError(err).Errorf("Unable to join party [%d].", partyId)
				err = producer.ProviderImpl(l)(ctx)(EnvEventStatusTopic)(errorEventProvider(characterId, partyId, c.WorldId(), EventPartyStatusErrorUnexpected, ""))
				if err != nil {
					l.WithError(err).Errorf("Unable to announce party [%d] error.", partyId)
				}
				return Model{}, err
			}
			err = character.JoinParty(l)(ctx)(characterId, partyId)
			if err != nil {
				l.WithError(err).Errorf("Unable to join party [%d].", partyId)
				p, err = GetRegistry().Update(t, partyId, func(m Model) Model { return Model.RemoveMember(m, characterId) })
				if err != nil {
					l.WithError(err).Errorf("Unable to clean up party [%d], when failing to add member [%d].", partyId, characterId)
				}
				err = producer.ProviderImpl(l)(ctx)(EnvEventStatusTopic)(errorEventProvider(characterId, partyId, c.WorldId(), EventPartyStatusErrorUnexpected, ""))
				if err != nil {
					l.WithError(err).Errorf("Unable to announce party [%d] error.", partyId)
				}
				return Model{}, err
			}

			l.Debugf("Character [%d] joined party [%d].", characterId, partyId)
			err = producer.ProviderImpl(l)(ctx)(EnvEventStatusTopic)(joinedEventProvider(characterId, p.Id(), c.WorldId()))
			if err != nil {
				l.WithError(err).Errorf("Unable to announce the party [%d] was created.", partyId)
				err = producer.ProviderImpl(l)(ctx)(EnvEventStatusTopic)(errorEventProvider(characterId, partyId, c.WorldId(), EventPartyStatusErrorUnexpected, ""))
				if err != nil {
					l.WithError(err).Errorf("Unable to announce party [%d] error.", partyId)
				}
				return Model{}, err
			}

			return p, nil
		}
	}
}

func Expel(l logrus.FieldLogger) func(ctx context.Context) func(actorId uint32, partyId uint32, characterId uint32) (Model, error) {
	return func(ctx context.Context) func(actorId uint32, partyId uint32, characterId uint32) (Model, error) {
		return func(actorId uint32, partyId uint32, characterId uint32) (Model, error) {
			t := tenant.MustFromContext(ctx)
			c, err := character.GetById(l)(ctx)(characterId)
			if err != nil {
				l.WithError(err).Errorf("Error getting character [%d].", characterId)
				err = producer.ProviderImpl(l)(ctx)(EnvEventStatusTopic)(errorEventProvider(actorId, partyId, c.WorldId(), EventPartyStatusErrorUnexpected, ""))
				if err != nil {
					l.WithError(err).Errorf("Unable to announce party [%d] error.", partyId)
				}
				return Model{}, err
			}

			if c.PartyId() != partyId {
				l.Errorf("Character [%d] not in party.", characterId)
				err = producer.ProviderImpl(l)(ctx)(EnvEventStatusTopic)(errorEventProvider(actorId, partyId, c.WorldId(), EventPartyStatusErrorUnexpected, ""))
				if err != nil {
					l.WithError(err).Errorf("Unable to announce party [%d] error.", partyId)
				}
				return Model{}, ErrNotIn
			}

			p, err := GetRegistry().Get(t, partyId)
			if err != nil {
				l.WithError(err).Errorf("Unable to retrieve party [%d].", partyId)
				err = producer.ProviderImpl(l)(ctx)(EnvEventStatusTopic)(errorEventProvider(actorId, partyId, c.WorldId(), EventPartyStatusErrorUnexpected, ""))
				if err != nil {
					l.WithError(err).Errorf("Unable to announce party [%d] error.", partyId)
				}
				return Model{}, err
			}

			p, err = GetRegistry().Update(t, partyId, func(m Model) Model { return Model.RemoveMember(m, characterId) })
			if err != nil {
				l.WithError(err).Errorf("Unable to expel from party [%d].", partyId)
				err = producer.ProviderImpl(l)(ctx)(EnvEventStatusTopic)(errorEventProvider(actorId, partyId, c.WorldId(), EventPartyStatusErrorUnexpected, ""))
				if err != nil {
					l.WithError(err).Errorf("Unable to announce party [%d] error.", partyId)
				}
				return Model{}, err
			}
			err = character.LeaveParty(l)(ctx)(characterId)
			if err != nil {
				l.WithError(err).Errorf("Unable to expel from party [%d].", partyId)
				p, err = GetRegistry().Update(t, partyId, func(m Model) Model { return Model.AddMember(m, characterId) })
				if err != nil {
					l.WithError(err).Errorf("Unable to clean up party [%d], when failing to remove member [%d].", partyId, characterId)
				}
				err = producer.ProviderImpl(l)(ctx)(EnvEventStatusTopic)(errorEventProvider(actorId, partyId, c.WorldId(), EventPartyStatusErrorUnexpected, ""))
				if err != nil {
					l.WithError(err).Errorf("Unable to announce party [%d] error.", partyId)
				}
				return Model{}, err
			}

			l.Debugf("Character [%d] expelled from party [%d].", characterId, partyId)
			err = producer.ProviderImpl(l)(ctx)(EnvEventStatusTopic)(expelEventProvider(actorId, p.Id(), c.WorldId(), characterId))
			if err != nil {
				l.WithError(err).Errorf("Unable to announce the party [%d] was left.", partyId)
				err = producer.ProviderImpl(l)(ctx)(EnvEventStatusTopic)(errorEventProvider(actorId, partyId, c.WorldId(), EventPartyStatusErrorUnexpected, ""))
				if err != nil {
					l.WithError(err).Errorf("Unable to announce party [%d] error.", partyId)
				}
				return Model{}, err
			}

			return p, nil
		}
	}
}

func Leave(l logrus.FieldLogger) func(ctx context.Context) func(partyId uint32, characterId uint32) (Model, error) {
	return func(ctx context.Context) func(partyId uint32, characterId uint32) (Model, error) {
		return func(partyId uint32, characterId uint32) (Model, error) {
			t := tenant.MustFromContext(ctx)
			c, err := character.GetById(l)(ctx)(characterId)
			if err != nil {
				l.WithError(err).Errorf("Error getting character [%d].", characterId)
				err = producer.ProviderImpl(l)(ctx)(EnvEventStatusTopic)(errorEventProvider(characterId, partyId, c.WorldId(), EventPartyStatusErrorUnexpected, ""))
				if err != nil {
					l.WithError(err).Errorf("Unable to announce party [%d] error.", partyId)
				}
				return Model{}, err
			}

			if c.PartyId() != partyId {
				l.Errorf("Character [%d] not in party.", characterId)
				err = producer.ProviderImpl(l)(ctx)(EnvEventStatusTopic)(errorEventProvider(characterId, partyId, c.WorldId(), EventPartyStatusErrorUnexpected, ""))
				if err != nil {
					l.WithError(err).Errorf("Unable to announce party [%d] error.", partyId)
				}
				return Model{}, ErrNotIn
			}

			p, err := GetRegistry().Get(t, partyId)
			if err != nil {
				l.WithError(err).Errorf("Unable to retrieve party [%d].", partyId)
				err = producer.ProviderImpl(l)(ctx)(EnvEventStatusTopic)(errorEventProvider(characterId, partyId, c.WorldId(), EventPartyStatusErrorUnexpected, ""))
				if err != nil {
					l.WithError(err).Errorf("Unable to announce party [%d] error.", partyId)
				}
				return Model{}, err
			}

			var disbandParty = p.LeaderId() == characterId

			p, err = GetRegistry().Update(t, partyId, func(m Model) Model { return Model.RemoveMember(m, characterId) })
			if err != nil {
				l.WithError(err).Errorf("Unable to leave party [%d].", partyId)
				err = producer.ProviderImpl(l)(ctx)(EnvEventStatusTopic)(errorEventProvider(characterId, partyId, c.WorldId(), EventPartyStatusErrorUnexpected, ""))
				if err != nil {
					l.WithError(err).Errorf("Unable to announce party [%d] error.", partyId)
				}
				return Model{}, err
			}
			err = character.LeaveParty(l)(ctx)(characterId)
			if err != nil {
				l.WithError(err).Errorf("Unable to leave party [%d].", partyId)
				p, err = GetRegistry().Update(t, partyId, func(m Model) Model { return Model.AddMember(m, characterId) })
				if err != nil {
					l.WithError(err).Errorf("Unable to clean up party [%d], when failing to remove member [%d].", partyId, characterId)
				}
				err = producer.ProviderImpl(l)(ctx)(EnvEventStatusTopic)(errorEventProvider(characterId, partyId, c.WorldId(), EventPartyStatusErrorUnexpected, ""))
				if err != nil {
					l.WithError(err).Errorf("Unable to announce party [%d] error.", characterId)
				}
				return Model{}, err
			}

			if disbandParty {
				for _, m := range p.Members() {
					err = character.LeaveParty(l)(ctx)(m)
					if err != nil {
						l.WithError(err).Errorf("Character [%d] stuck in party [%d].", m, partyId)
					}
				}

				GetRegistry().Remove(t, partyId)
				l.Debugf("Party [%d] has been disbanded.", partyId)
				err = producer.ProviderImpl(l)(ctx)(EnvEventStatusTopic)(disbandEventProvider(characterId, partyId, c.WorldId(), p.Members()))
				if err != nil {
					l.WithError(err).Errorf("Unable to announce the party [%d] was disbanded.", partyId)
					err = producer.ProviderImpl(l)(ctx)(EnvEventStatusTopic)(errorEventProvider(characterId, partyId, c.WorldId(), EventPartyStatusErrorUnexpected, ""))
					if err != nil {
						l.WithError(err).Errorf("Unable to announce party [%d] error.", partyId)
					}
					return Model{}, err
				}
			} else {
				l.Debugf("Character [%d] left party [%d].", characterId, partyId)
				err = producer.ProviderImpl(l)(ctx)(EnvEventStatusTopic)(leftEventProvider(characterId, partyId, c.WorldId()))
				if err != nil {
					l.WithError(err).Errorf("Unable to announce the party [%d] was left.", partyId)
					err = producer.ProviderImpl(l)(ctx)(EnvEventStatusTopic)(errorEventProvider(characterId, partyId, c.WorldId(), EventPartyStatusErrorUnexpected, ""))
					if err != nil {
						l.WithError(err).Errorf("Unable to announce party [%d] error.", partyId)
					}
					return Model{}, err
				}
			}

			return p, nil
		}
	}
}

func ChangeLeader(l logrus.FieldLogger) func(ctx context.Context) func(actorId uint32, partyId uint32, characterId uint32) (Model, error) {
	return func(ctx context.Context) func(actorId uint32, partyId uint32, characterId uint32) (Model, error) {
		return func(actorId uint32, partyId uint32, characterId uint32) (Model, error) {
			t := tenant.MustFromContext(ctx)
			a, err := character.GetById(l)(ctx)(characterId)
			if err != nil {
				l.WithError(err).Errorf("Error getting character [%d].", actorId)
				err = producer.ProviderImpl(l)(ctx)(EnvEventStatusTopic)(errorEventProvider(actorId, partyId, a.WorldId(), EventPartyStatusErrorUnexpected, ""))
				if err != nil {
					l.WithError(err).Errorf("Unable to announce party [%d] error.", partyId)
				}
				return Model{}, err
			}

			c, err := character.GetById(l)(ctx)(characterId)
			if err != nil {
				l.WithError(err).Errorf("Error getting character [%d].", characterId)
				err = producer.ProviderImpl(l)(ctx)(EnvEventStatusTopic)(errorEventProvider(characterId, partyId, c.WorldId(), EventPartyStatusErrorUnexpected, ""))
				if err != nil {
					l.WithError(err).Errorf("Unable to announce party [%d] error.", partyId)
				}
				return Model{}, err
			}

			if c.PartyId() != partyId {
				l.Errorf("Character [%d] not in party. Cannot become leader.", characterId)
				err = producer.ProviderImpl(l)(ctx)(EnvEventStatusTopic)(errorEventProvider(actorId, partyId, c.WorldId(), EventPartyStatusErrorUnexpected, ""))
				if err != nil {
					l.WithError(err).Errorf("Unable to announce party [%d] error.", partyId)
				}
				return Model{}, ErrNotIn
			}

			if a.PartyId() != c.PartyId() {
				l.Errorf("Character [%d] not in the same party as actor [%d]. Cannot become leader.", characterId, actorId)
				err = producer.ProviderImpl(l)(ctx)(EnvEventStatusTopic)(errorEventProvider(actorId, partyId, c.WorldId(), EventPartyStatusErrorUnexpected, ""))
				if err != nil {
					l.WithError(err).Errorf("Unable to announce party [%d] error.", partyId)
				}
				return Model{}, ErrNotIn
			}

			if a.WorldId() != c.WorldId() && a.ChannelId() != c.ChannelId() {
				l.Errorf("Character [%d] not in the same channel as actor [%d]. Cannot become leader.", characterId, actorId)
				err = producer.ProviderImpl(l)(ctx)(EnvEventStatusTopic)(errorEventProvider(actorId, partyId, c.WorldId(), EventPartyStatusErrorTypeNotInChannel, ""))
				if err != nil {
					l.WithError(err).Errorf("Unable to announce party [%d] error.", partyId)
				}
				return Model{}, ErrNotIn
			}

			if a.MapId() != c.MapId() {
				l.Errorf("Character [%d] not in the same map as actor [%d]. Cannot become leader.", characterId, actorId)
				err = producer.ProviderImpl(l)(ctx)(EnvEventStatusTopic)(errorEventProvider(actorId, partyId, c.WorldId(), EventPartyStatusErrorTypeUnableToInVicinity, ""))
				if err != nil {
					l.WithError(err).Errorf("Unable to announce party [%d] error.", partyId)
				}
				return Model{}, ErrNotIn
			}

			p, err := GetRegistry().Update(t, partyId, func(m Model) Model { return Model.SetLeader(m, characterId) })
			if err != nil {
				l.WithError(err).Errorf("Unable to join party [%d].", partyId)
				err = producer.ProviderImpl(l)(ctx)(EnvEventStatusTopic)(errorEventProvider(actorId, partyId, c.WorldId(), EventPartyStatusErrorUnexpected, ""))
				if err != nil {
					l.WithError(err).Errorf("Unable to announce party [%d] error.", partyId)
				}
			}

			l.Debugf("Character [%d] became leader of party [%d].", characterId, partyId)
			err = producer.ProviderImpl(l)(ctx)(EnvEventStatusTopic)(changeLeaderEventProvider(actorId, p.Id(), c.WorldId(), characterId))
			if err != nil {
				l.WithError(err).Errorf("Unable to announce leadership change in party [%d].", c.Id())
				err = producer.ProviderImpl(l)(ctx)(EnvEventStatusTopic)(errorEventProvider(actorId, partyId, c.WorldId(), EventPartyStatusErrorUnexpected, ""))
				if err != nil {
					l.WithError(err).Errorf("Unable to announce party [%d] error.", partyId)
				}
			}
			return p, nil
		}
	}
}

func RequestInvite(l logrus.FieldLogger) func(ctx context.Context) func(actorId uint32, characterId uint32) error {
	return func(ctx context.Context) func(actorId uint32, characterId uint32) error {
		return func(actorId uint32, characterId uint32) error {
			a, err := character.GetById(l)(ctx)(actorId)
			if err != nil {
				l.WithError(err).Errorf("Error getting character [%d].", actorId)
				err = producer.ProviderImpl(l)(ctx)(EnvEventStatusTopic)(errorEventProvider(actorId, 0, a.WorldId(), EventPartyStatusErrorUnexpected, ""))
				if err != nil {
					l.WithError(err).Errorf("Unable to announce party [%d] error.", 0)
				}
				return err
			}

			c, err := character.GetById(l)(ctx)(characterId)
			if err != nil {
				l.WithError(err).Errorf("Error getting character [%d].", characterId)
				err = producer.ProviderImpl(l)(ctx)(EnvEventStatusTopic)(errorEventProvider(actorId, a.PartyId(), c.WorldId(), EventPartyStatusErrorUnexpected, ""))
				if err != nil {
					l.WithError(err).Errorf("Unable to announce party [%d] error.", a.PartyId())
				}
				return err
			}

			if c.PartyId() != 0 {
				err = producer.ProviderImpl(l)(ctx)(EnvEventStatusTopic)(errorEventProvider(actorId, c.PartyId(), c.WorldId(), EventPartyStatusErrorTypeAlreadyJoined2, c.Name()))
				if err != nil {
					l.WithError(err).Errorf("Unable to announce party [%d] error.", a.PartyId())
				}
				return ErrAlreadyIn
			}

			var p Model
			if a.PartyId() == 0 {
				p, err = Create(l)(ctx)(actorId)
				if err != nil {
					l.WithError(err).Errorf("Unable to automatically create party [%d].", a.PartyId())
					return err
				}
			} else {
				p, err = GetById(ctx)(a.PartyId())
				if err != nil {
					return err
				}
			}

			if len(p.Members()) >= 6 {
				l.Errorf("Party [%d] already at capacity.", p.Id())
				err = producer.ProviderImpl(l)(ctx)(EnvEventStatusTopic)(errorEventProvider(actorId, p.Id(), c.WorldId(), EventPartyStatusErrorTypeAtCapacity, ""))
				if err != nil {
					l.WithError(err).Errorf("Unable to announce party [%d] error.", p.Id())
				}
				return ErrAtCapacity
			}

			err = invite.Create(l)(ctx)(actorId, a.WorldId(), p.Id(), characterId)
			if err != nil {
				l.WithError(err).Errorf("Unable to announce party [%d] invite.", p.Id())
				err = producer.ProviderImpl(l)(ctx)(EnvEventStatusTopic)(errorEventProvider(actorId, a.PartyId(), c.WorldId(), EventPartyStatusErrorUnexpected, ""))
				if err != nil {
					l.WithError(err).Errorf("Unable to announce party [%d] error.", a.PartyId())
				}
				return err
			}

			return nil
		}
	}
}
