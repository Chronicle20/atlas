package configuration

import (
	"atlas-tenants/rest"
	"errors"
	"github.com/Chronicle20/atlas-rest/server"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"net/http"
)

// GetAllRoutesHandler handles GET /tenants/{tenantId}/configurations/routes
func GetAllRoutesHandler(db *gorm.DB) func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseTenantId(d.Logger(), func(tenantId uuid.UUID) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				processor := NewProcessor(d.Logger(), d.Context(), db)

				routes, err := processor.GetAllRoutes(tenantId)
				if err != nil {
					if errors.Is(err, gorm.ErrRecordNotFound) {
						// If no routes exist, return an empty array instead of an error
						d.Logger().Info("No routes found for tenant, returning empty array")
						routes = []map[string]interface{}{}
					} else {
						d.Logger().WithError(err).Error("Failed to get routes")
						w.WriteHeader(http.StatusInternalServerError)
						return
					}
				}

				restModels := make([]RouteRestModel, 0, len(routes))
				for _, route := range routes {
					rm, err := TransformRoute(route)
					if err != nil {
						d.Logger().WithError(err).Error("Failed to transform route")
						w.WriteHeader(http.StatusInternalServerError)
						return
					}
					restModels = append(restModels, rm)
				}

				query := r.URL.Query()
				queryParams := jsonapi.ParseQueryFields(&query)
				server.MarshalResponse[[]RouteRestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(restModels)
			}
		})
	}
}

// GetRouteByIdHandler handles GET /tenants/{tenantId}/configurations/routes/{routeId}
func GetRouteByIdHandler(db *gorm.DB) func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseTenantId(d.Logger(), func(tenantId uuid.UUID) http.HandlerFunc {
			return rest.ParseRouteId(d.Logger(), func(routeId string) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					processor := NewProcessor(d.Logger(), d.Context(), db)

					route, err := processor.GetRouteById(tenantId, routeId)
					if err != nil {
						d.Logger().WithError(err).Error("Failed to get route")
						w.WriteHeader(http.StatusNotFound)
						return
					}

					rm, err := TransformRoute(route)
					if err != nil {
						d.Logger().WithError(err).Error("Failed to transform route")
						w.WriteHeader(http.StatusInternalServerError)
						return
					}

					query := r.URL.Query()
					queryParams := jsonapi.ParseQueryFields(&query)
					server.MarshalResponse[RouteRestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(rm)
				}
			})
		})
	}
}

// CreateRouteHandler handles POST /tenants/{tenantId}/configurations/routes
func CreateRouteHandler(db *gorm.DB) func(d *rest.HandlerDependency, c *rest.HandlerContext, model RouteRestModel) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext, model RouteRestModel) http.HandlerFunc {
		return rest.ParseTenantId(d.Logger(), func(tenantId uuid.UUID) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				route, err := ExtractRoute(model)
				if err != nil {
					d.Logger().WithError(err).Error("Failed to extract route data")
					w.WriteHeader(http.StatusBadRequest)
					return
				}

				processor := NewProcessor(d.Logger(), d.Context(), db)
				_, err = processor.CreateAndEmit(tenantId, route)
				if err != nil {
					d.Logger().WithError(err).Error("Failed to create route")
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				// Get the route ID from the created route
				routeId := ""
				if id, ok := route["id"].(string); ok {
					routeId = id
				}

				// Get the specific route that was just created
				createdRoute, err := processor.GetRouteById(tenantId, routeId)
				if err != nil {
					d.Logger().WithError(err).Error("Failed to get created route")
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				rm, err := TransformRoute(createdRoute)
				if err != nil {
					d.Logger().WithError(err).Error("Failed to transform route")
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				query := r.URL.Query()
				queryParams := jsonapi.ParseQueryFields(&query)
				w.WriteHeader(http.StatusCreated)
				server.MarshalResponse[RouteRestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(rm)
			}
		})
	}
}

// UpdateRouteHandler handles PATCH /tenants/{tenantId}/configurations/routes/{routeId}
func UpdateRouteHandler(db *gorm.DB) func(d *rest.HandlerDependency, c *rest.HandlerContext, model RouteRestModel) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext, model RouteRestModel) http.HandlerFunc {
		return rest.ParseTenantId(d.Logger(), func(tenantId uuid.UUID) http.HandlerFunc {
			return rest.ParseRouteId(d.Logger(), func(routeId string) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					route, err := ExtractRoute(model)
					if err != nil {
						d.Logger().WithError(err).Error("Failed to extract route data")
						w.WriteHeader(http.StatusBadRequest)
						return
					}

					processor := NewProcessor(d.Logger(), d.Context(), db)
					_, err = processor.UpdateAndEmit(tenantId, routeId, route)
					if err != nil {
						d.Logger().WithError(err).Error("Failed to update route")
						w.WriteHeader(http.StatusInternalServerError)
						return
					}

					// Get the specific route that was just updated
					updatedRoute, err := processor.GetRouteById(tenantId, routeId)
					if err != nil {
						d.Logger().WithError(err).Error("Failed to get updated route")
						w.WriteHeader(http.StatusInternalServerError)
						return
					}

					rm, err := TransformRoute(updatedRoute)
					if err != nil {
						d.Logger().WithError(err).Error("Failed to transform route")
						w.WriteHeader(http.StatusInternalServerError)
						return
					}

					query := r.URL.Query()
					queryParams := jsonapi.ParseQueryFields(&query)
					server.MarshalResponse[RouteRestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(rm)
				}
			})
		})
	}
}

// DeleteRouteHandler handles DELETE /tenants/{tenantId}/configurations/routes/{routeId}
func DeleteRouteHandler(db *gorm.DB) func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseTenantId(d.Logger(), func(tenantId uuid.UUID) http.HandlerFunc {
			return rest.ParseRouteId(d.Logger(), func(routeId string) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					processor := NewProcessor(d.Logger(), d.Context(), db)
					err := processor.DeleteAndEmit(tenantId, routeId)
					if err != nil {
						d.Logger().WithError(err).Error("Failed to delete route")
						w.WriteHeader(http.StatusInternalServerError)
						return
					}

					w.WriteHeader(http.StatusNoContent)
				}
			})
		})
	}
}

// RegisterRoutes registers the configuration routes
func RegisterRoutes(db *gorm.DB) func(si jsonapi.ServerInformation) server.RouteInitializer {
	return func(si jsonapi.ServerInformation) server.RouteInitializer {
		return func(r *mux.Router, l logrus.FieldLogger) {
			registerHandler := rest.RegisterHandler(l)(si)
			registerInputHandler := rest.RegisterInputHandler[RouteRestModel](l)(si)

			r.HandleFunc("/tenants/{tenantId}/configurations/routes", registerHandler("get_all_routes", GetAllRoutesHandler(db))).Methods(http.MethodGet)
			r.HandleFunc("/tenants/{tenantId}/configurations/routes/{routeId}", registerHandler("get_route_by_id", GetRouteByIdHandler(db))).Methods(http.MethodGet)
			r.HandleFunc("/tenants/{tenantId}/configurations/routes", registerInputHandler("create_route", CreateRouteHandler(db))).Methods(http.MethodPost)
			r.HandleFunc("/tenants/{tenantId}/configurations/routes/{routeId}", registerInputHandler("update_route", UpdateRouteHandler(db))).Methods(http.MethodPatch)
			r.HandleFunc("/tenants/{tenantId}/configurations/routes/{routeId}", registerHandler("delete_route", DeleteRouteHandler(db))).Methods(http.MethodDelete)
		}
	}
}
