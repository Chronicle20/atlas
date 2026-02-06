package messenger

import (
	"atlas-messengers/kafka/message/messenger"
	"atlas-messengers/kafka/producer"
	"atlas-messengers/rest"
	"net/http"
	"strconv"

	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/server"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
)

func InitResource(si jsonapi.ServerInformation) server.RouteInitializer {
	return func(router *mux.Router, l logrus.FieldLogger) {
		registerGet := rest.RegisterHandler(l)(si)
		r := router.PathPrefix("/messengers").Subrouter()
		r.HandleFunc("", registerGet("get_messenger_by_member_id", handleGetMessengers)).Queries("filter[members.id]", "{memberId}").Methods(http.MethodGet)
		r.HandleFunc("", registerGet("get_messengers", handleGetMessengers)).Methods(http.MethodGet)
		r.HandleFunc("/{messengerId}", registerGet("get_messenger", handleGetMessenger)).Methods(http.MethodGet)
		r.HandleFunc("/{messengerId}/members", registerGet("get_messenger_members", handleGetMessengerMembers)).Methods(http.MethodGet)
		r.HandleFunc("/{messengerId}/relationships/members", registerGet("get_messenger_members", handleGetMessengerMembers)).Methods(http.MethodGet)
		r.HandleFunc("/{messengerId}/members", rest.RegisterInputHandler[MemberRestModel](l)(si)("create_messenger_member", handleCreateMessengerMember)).Methods(http.MethodPost)
		r.HandleFunc("/{messengerId}/members/{memberId}", registerGet("get_messenger_member", handleGetMessengerMember)).Methods(http.MethodGet)
		r.HandleFunc("/{messengerId}/members/{memberId}", rest.RegisterHandler(l)(si)("remove_messenger_member", handleRemoveMessengerMember)).Methods(http.MethodDelete)
	}
}

func handleGetMessengers(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
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

		proc := NewProcessor(d.Logger(), d.Context())
		ps, err := proc.GetSlice(filters...)
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

func handleGetMessenger(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseMessengerId(d.Logger(), func(messengerId uint32) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			proc := NewProcessor(d.Logger(), d.Context())
			p, err := proc.GetById(messengerId)
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

func handleGetMessengerMembers(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseMessengerId(d.Logger(), func(messengerId uint32) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			proc := NewProcessor(d.Logger(), d.Context())
			p, err := proc.GetById(messengerId)
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

func handleCreateMessengerMember(d *rest.HandlerDependency, _ *rest.HandlerContext, i MemberRestModel) http.HandlerFunc {
	return rest.ParseMessengerId(d.Logger(), func(messengerId uint32) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			ep := producer.ProviderImpl(d.Logger())(d.Context())
			err := ep(messenger.EnvCommandTopic)(joinCommandProvider(uuid.New(), messengerId, i.Id))
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			w.WriteHeader(http.StatusAccepted)
		}
	})
}

func handleGetMessengerMember(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseMessengerId(d.Logger(), func(messengerId uint32) http.HandlerFunc {
		return rest.ParseMemberId(d.Logger(), func(memberId uint32) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				proc := NewProcessor(d.Logger(), d.Context())
				p, err := proc.GetById(messengerId)
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				m, err := model.FirstProvider(model.FixedProvider(p.Members()), model.Filters[MemberModel](func(m MemberModel) bool {
					return m.Id() == memberId
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

func handleRemoveMessengerMember(d *rest.HandlerDependency, _ *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseMessengerId(d.Logger(), func(messengerId uint32) http.HandlerFunc {
		return rest.ParseMemberId(d.Logger(), func(memberId uint32) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				ep := producer.ProviderImpl(d.Logger())(d.Context())
				err := ep(messenger.EnvCommandTopic)(leaveCommandProvider(uuid.New(), messengerId, memberId))
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				w.WriteHeader(http.StatusAccepted)
			}
		})
	})
}
