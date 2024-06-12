package session

//
//import (
//	"atlas-clc/json"
//	"atlas-clc/rest"
//	"atlas-clc/socket/response/writer"
//	"github.com/gorilla/mux"
//	"github.com/opentracing/opentracing-go"
//	"github.com/sirupsen/logrus"
//	"net/http"
//	"strconv"
//)
//
//const (
//	GetSessions = "get_sessions"
//	LoginError  = "login_error"
//)
//
//func InitResource(router *mux.Router, l logrus.FieldLogger) {
//	sRouter := router.PathPrefix("/sessions").Subrouter()
//	sRouter.HandleFunc("", registerGetSessions(l)).Methods(http.MethodGet)
//	sRouter.HandleFunc("/{sessionId}/errors/{errorId}", registerLoginError(l)).Methods(http.MethodPost)
//}
//
//func registerLoginError(l logrus.FieldLogger) http.HandlerFunc {
//	return rest.RetrieveSpan(LoginError, func(span opentracing.Span) http.HandlerFunc {
//		return parseSessionId(l, func(sessionId uint32) http.HandlerFunc {
//			return parseErrorId(l, func(errorId byte) http.HandlerFunc {
//				return handleLoginError(l)(span)(sessionId)(errorId)
//			})
//		})
//	})
//}
//
//func registerGetSessions(l logrus.FieldLogger) http.HandlerFunc {
//	return rest.RetrieveSpan(GetSessions, func(span opentracing.Span) http.HandlerFunc {
//		return handleGetSessions(l)(span)
//	})
//}
//
//func handleGetSessions(l logrus.FieldLogger) func(span opentracing.Span) http.HandlerFunc {
//	return func(span opentracing.Span) http.HandlerFunc {
//		return func(w http.ResponseWriter, _ *http.Request) {
//			ss := GetRegistry().GetAll()
//			var response dataListContainer
//			response.Data = make([]dataBody, 0)
//			for _, x := range ss {
//				response.Data = append(response.Data, getSessionObject(x))
//			}
//
//			err := json.ToJSON(response, w)
//			if err != nil {
//				l.WithError(err).Errorf("Encoding response")
//				w.WriteHeader(http.StatusInternalServerError)
//			}
//		}
//	}
//}
//
//type sessionIdHandler func(sessionId uint32) http.HandlerFunc
//
//func parseSessionId(l logrus.FieldLogger, next sessionIdHandler) http.HandlerFunc {
//	return func(w http.ResponseWriter, r *http.Request) {
//		vars := mux.Vars(r)
//		value, err := strconv.Atoi(vars["sessionId"])
//		if err != nil {
//			l.Println("Error parsing id as integer")
//			w.WriteHeader(http.StatusBadRequest)
//			return
//		}
//		next(uint32(value))(w, r)
//	}
//}
//
//type errorIdHandler func(errorId byte) http.HandlerFunc
//
//func parseErrorId(l logrus.FieldLogger, next errorIdHandler) http.HandlerFunc {
//	return func(w http.ResponseWriter, r *http.Request) {
//		vars := mux.Vars(r)
//		value, err := strconv.Atoi(vars["errorId"])
//		if err != nil {
//			l.Println("Error parsing id as byte")
//			w.WriteHeader(http.StatusBadRequest)
//			return
//		}
//		next(byte(value))(w, r)
//	}
//}
//
//func handleLoginError(l logrus.FieldLogger) func(span opentracing.Span) func(sessionId uint32) func(errorId byte) http.HandlerFunc {
//	return func(span opentracing.Span) func(sessionId uint32) func(errorId byte) http.HandlerFunc {
//		return func(sessionId uint32) func(errorId byte) http.HandlerFunc {
//			return func(errorId byte) http.HandlerFunc {
//				return func(w http.ResponseWriter, r *http.Request) {
//					ses, ok := GetRegistry().Get(sessionId)
//					if !ok {
//						w.WriteHeader(http.StatusNotFound)
//						return
//					}
//
//					w.WriteHeader(http.StatusNoContent)
//					err := Announce(writer.WriteLoginFailed(l)(errorId))(ses)
//					if err != nil {
//						l.WithError(err).Errorf("Unable to issue login failed due to reason %d", errorId)
//					}
//				}
//			}
//
//		}
//	}
//}
//
//func getSessionObject(x Model) dataBody {
//	return dataBody{
//		Id:   strconv.Itoa(int(x.SessionId())),
//		Type: "Session",
//		Attributes: attributes{
//			AccountId: x.AccountId(),
//			WorldId:   x.WorldId(),
//			ChannelId: x.ChannelId(),
//		},
//	}
//}
