package party

import (
	"atlas-parties/kafka/producer"
	"atlas-parties/rest"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/server"
	"github.com/gorilla/mux"
	"github.com/manyminds/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"net/http"
	"strconv"
)

const (
	GetParties           = "get_parties"
	GetPartiesByMemberId = "get_parties_by_member_id"
	CreateParty          = "create_party"
	GetParty             = "get_party"
	UpdateParty          = "update_party"
	GetPartyMembers      = "get_party_members"
	CreatePartyMember    = "create_party_member"
	GetPartyMember       = "get_party_member"
	RemovePartyMember    = "remove_party_member"
)

func InitResource(si jsonapi.ServerInformation) server.RouteInitializer {
	return func(router *mux.Router, l logrus.FieldLogger) {
		registerGet := rest.RegisterHandler(l)(si)
		r := router.PathPrefix("/parties").Subrouter()
		r.HandleFunc("", registerGet(GetPartiesByMemberId, handleGetParties)).Queries("filter[members.id]", "{memberId}").Methods(http.MethodGet)
		r.HandleFunc("", registerGet(GetParties, handleGetParties)).Methods(http.MethodGet)
		r.HandleFunc("", rest.RegisterInputHandler[RestModel](l)(si)(CreateParty, handleCreateParty)).Methods(http.MethodPost)
		r.HandleFunc("/{partyId}", registerGet(GetParty, handleGetParty)).Methods(http.MethodGet)
		r.HandleFunc("/{partyId}", rest.RegisterInputHandler[RestModel](l)(si)(UpdateParty, handleUpdateParty)).Methods(http.MethodPatch)
		r.HandleFunc("/{partyId}/members", registerGet(GetPartyMembers, handleGetPartyMembers)).Methods(http.MethodGet)
		r.HandleFunc("/{partyId}/relationships/members", registerGet(GetPartyMembers, handleGetPartyMembers)).Methods(http.MethodGet)
		r.HandleFunc("/{partyId}/members", rest.RegisterInputHandler[MemberRestModel](l)(si)(CreatePartyMember, handleCreatePartyMember)).Methods(http.MethodPost)
		r.HandleFunc("/{partyId}/members/{memberId}", registerGet(GetPartyMember, handleGetPartyMember)).Methods(http.MethodGet)
		r.HandleFunc("/{partyId}/members/{memberId}", rest.RegisterHandler(l)(si)(RemovePartyMember, handleRemovePartyMember)).Methods(http.MethodDelete)
	}
}

func handleGetParties(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var filters = make([]model.Filter[Model], 0)
		if memberFilter, ok := mux.Vars(r)["memberId"]; ok {
			memberId, err := strconv.Atoi(memberFilter)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			filters = append(filters, MemberFilter(uint32(memberId)))
		}

		ps, err := GetSlice(d.Context())(filters...)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		res, err := model.SliceMap(Transform(d.Logger())(d.Context()))(model.FixedProvider(ps))()()
		if err != nil {
			d.Logger().WithError(err).Errorf("Creating REST model.")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		server.Marshal[[]RestModel](d.Logger())(w)(c.ServerInformation())(res)
	}
}

func handleCreateParty(d *rest.HandlerDependency, _ *rest.HandlerContext, i RestModel) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if len(i.Members) != 1 {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		ep := producer.ProviderImpl(d.Logger())(d.Context())
		err := ep(EnvCommandTopic)(createCommandProvider(i.Members[0].Id))
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusAccepted)
	}
}

func handleGetParty(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParsePartyId(d.Logger(), func(partyId uint32) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			p, err := GetById(d.Context())(partyId)
			if err != nil {
				w.WriteHeader(http.StatusNotFound)
				return
			}

			res, err := model.Map(Transform(d.Logger())(d.Context()))(model.FixedProvider(p))()
			if err != nil {
				d.Logger().WithError(err).Errorf("Creating REST model.")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			server.Marshal[RestModel](d.Logger())(w)(c.ServerInformation())(res)
		}
	})
}

func handleUpdateParty(d *rest.HandlerDependency, c *rest.HandlerContext, i RestModel) http.HandlerFunc {
	return rest.ParsePartyId(d.Logger(), func(partyId uint32) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			p, err := GetById(d.Context())(partyId)
			if err != nil {
				w.WriteHeader(http.StatusNotFound)
				return
			}

			if p.LeaderId() != i.LeaderId {
				ep := producer.ProviderImpl(d.Logger())(d.Context())
				err := ep(EnvCommandTopic)(changeLeaderCommandProvider(0, partyId, i.LeaderId))
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
			}
			w.WriteHeader(http.StatusAccepted)
		}
	})
}

func handleGetPartyMembers(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParsePartyId(d.Logger(), func(partyId uint32) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			p, err := GetById(d.Context())(partyId)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			res, err := model.Map(Transform(d.Logger())(d.Context()))(model.FixedProvider(p))()
			if err != nil {
				d.Logger().WithError(err).Errorf("Creating REST model.")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			server.Marshal[[]MemberRestModel](d.Logger())(w)(c.ServerInformation())(res.Members)
		}
	})
}

func handleCreatePartyMember(d *rest.HandlerDependency, _ *rest.HandlerContext, i MemberRestModel) http.HandlerFunc {
	return rest.ParsePartyId(d.Logger(), func(partyId uint32) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			ep := producer.ProviderImpl(d.Logger())(d.Context())
			err := ep(EnvCommandTopic)(joinCommandProvider(partyId, i.Id))
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			w.WriteHeader(http.StatusAccepted)
		}
	})
}

func handleGetPartyMember(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParsePartyId(d.Logger(), func(partyId uint32) http.HandlerFunc {
		return rest.ParseMemberId(d.Logger(), func(memberId uint32) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				p, err := GetById(d.Context())(partyId)
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				m, err := model.FirstProvider(model.FixedProvider(p.members), model.Filters[uint32](func(id uint32) bool {
					return id == memberId
				}))()
				if err != nil {
					w.WriteHeader(http.StatusNotFound)
					return
				}

				res, err := model.Map(TransformMember(d.Logger())(d.Context()))(model.FixedProvider(m))()
				if err != nil {
					d.Logger().WithError(err).Errorf("Creating REST model.")
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				server.Marshal[MemberRestModel](d.Logger())(w)(c.ServerInformation())(res)
			}
		})
	})
}

func handleRemovePartyMember(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParsePartyId(d.Logger(), func(partyId uint32) http.HandlerFunc {
		return rest.ParseMemberId(d.Logger(), func(memberId uint32) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				ep := producer.ProviderImpl(d.Logger())(d.Context())
				err := ep(EnvCommandTopic)(leaveCommandProvider(partyId, memberId, false))
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				w.WriteHeader(http.StatusAccepted)
			}
		})
	})
}
