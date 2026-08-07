package chat

import (
	"atlas-messages/rest"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
)

// RestModel is the "chat-messages" resource, consumed by atlas-ban's report
// pipeline. Served at /api/chat/history, which IS routed through
// nginx/ingress (deploy/shared/routes.conf) and reachable at the ingress
// host WITHOUT AUTHENTICATION — this exposes captured player chat, including
// whispers, to anything that can reach that host. Accepted on the basis that
// API authentication is coming; see
// docs/tasks/task-145-player-reports/scope-amendment.md Amendment 2. Do not
// describe this endpoint as server-to-server only.
type RestModel struct {
	Id         string `json:"-"`
	Timestamp  int64  `json:"timestamp"`
	SenderId   uint32 `json:"senderId"`
	SenderName string `json:"senderName"`
	ChatType   string `json:"chatType"`
	Text       string `json:"text"`
}

func (r RestModel) GetName() string {
	return "chat-messages"
}

func (r RestModel) GetID() string {
	return r.Id
}

func (r *RestModel) SetID(idStr string) error {
	r.Id = idStr
	return nil
}

func Transform(index int, line Line) RestModel {
	return RestModel{
		Id:         strconv.Itoa(index),
		Timestamp:  line.Timestamp,
		SenderId:   line.SenderId,
		SenderName: line.SenderName,
		ChatType:   line.ChatType,
		Text:       line.Text,
	}
}

func parseCharacterIds(raw string) ([]uint32, error) {
	if raw == "" {
		return nil, errors.New("characterIds is required")
	}
	parts := strings.Split(raw, ",")
	ids := make([]uint32, 0, len(parts))
	for _, part := range parts {
		v, err := strconv.ParseUint(strings.TrimSpace(part), 10, 32)
		if err != nil {
			return nil, err
		}
		ids = append(ids, uint32(v))
	}
	return ids, nil
}

func InitResource(si jsonapi.ServerInformation) server.RouteInitializer {
	return func(router *mux.Router, l logrus.FieldLogger) {
		register := rest.RegisterHandler(l)(si)
		r := router.PathPrefix("/chat").Subrouter()
		r.HandleFunc("/history", register("get_chat_history", handleGetChatHistory)).Methods(http.MethodGet)
	}
}

func handleGetChatHistory(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ids, err := parseCharacterIds(r.URL.Query().Get("characterIds"))
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		lines, err := NewProcessor(d.Logger(), d.Context()).RecentInvolving(ids)
		if err != nil {
			d.Logger().WithError(err).Errorf("Unable to retrieve chat history.")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		res := make([]RestModel, 0, len(lines))
		for i, line := range lines {
			res = append(res, Transform(i, line))
		}

		query := r.URL.Query()
		queryParams := jsonapi.ParseQueryFields(&query)
		server.MarshalResponse[[]RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(res)
	}
}
