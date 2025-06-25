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
				_, err = processor.CreateRouteAndEmit(tenantId, route)
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
					_, err = processor.UpdateRouteAndEmit(tenantId, routeId, route)
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
					err := processor.DeleteRouteAndEmit(tenantId, routeId)
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

// GetAllVesselsHandler handles GET /tenants/{tenantId}/configurations/vessels
func GetAllVesselsHandler(db *gorm.DB) func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseTenantId(d.Logger(), func(tenantId uuid.UUID) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				processor := NewProcessor(d.Logger(), d.Context(), db)

				vessels, err := processor.GetAllVessels(tenantId)
				if err != nil {
					if errors.Is(err, gorm.ErrRecordNotFound) {
						// If no vessels exist, return an empty array instead of an error
						d.Logger().Info("No vessels found for tenant, returning empty array")
						vessels = []map[string]interface{}{}
					} else {
						d.Logger().WithError(err).Error("Failed to get vessels")
						w.WriteHeader(http.StatusInternalServerError)
						return
					}
				}

				restModels := make([]VesselRestModel, 0, len(vessels))
				for _, vessel := range vessels {
					rm, err := TransformVessel(vessel)
					if err != nil {
						d.Logger().WithError(err).Error("Failed to transform vessel")
						w.WriteHeader(http.StatusInternalServerError)
						return
					}
					restModels = append(restModels, rm)
				}

				query := r.URL.Query()
				queryParams := jsonapi.ParseQueryFields(&query)
				server.MarshalResponse[[]VesselRestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(restModels)
			}
		})
	}
}

// GetVesselByIdHandler handles GET /tenants/{tenantId}/configurations/vessels/{vesselId}
func GetVesselByIdHandler(db *gorm.DB) func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseTenantId(d.Logger(), func(tenantId uuid.UUID) http.HandlerFunc {
			return rest.ParseVesselId(d.Logger(), func(vesselId string) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					processor := NewProcessor(d.Logger(), d.Context(), db)

					vessel, err := processor.GetVesselById(tenantId, vesselId)
					if err != nil {
						d.Logger().WithError(err).Error("Failed to get vessel")
						w.WriteHeader(http.StatusNotFound)
						return
					}

					rm, err := TransformVessel(vessel)
					if err != nil {
						d.Logger().WithError(err).Error("Failed to transform vessel")
						w.WriteHeader(http.StatusInternalServerError)
						return
					}

					query := r.URL.Query()
					queryParams := jsonapi.ParseQueryFields(&query)
					server.MarshalResponse[VesselRestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(rm)
				}
			})
		})
	}
}

// CreateVesselHandler handles POST /tenants/{tenantId}/configurations/vessels
func CreateVesselHandler(db *gorm.DB) func(d *rest.HandlerDependency, c *rest.HandlerContext, model VesselRestModel) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext, model VesselRestModel) http.HandlerFunc {
		return rest.ParseTenantId(d.Logger(), func(tenantId uuid.UUID) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				vessel, err := ExtractVessel(model)
				if err != nil {
					d.Logger().WithError(err).Error("Failed to extract vessel data")
					w.WriteHeader(http.StatusBadRequest)
					return
				}

				processor := NewProcessor(d.Logger(), d.Context(), db)
				_, err = processor.CreateVesselAndEmit(tenantId, vessel)
				if err != nil {
					d.Logger().WithError(err).Error("Failed to create vessel")
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				// Get the vessel ID from the created vessel
				vesselId := ""
				if id, ok := vessel["id"].(string); ok {
					vesselId = id
				}

				// Get the specific vessel that was just created
				createdVessel, err := processor.GetVesselById(tenantId, vesselId)
				if err != nil {
					d.Logger().WithError(err).Error("Failed to get created vessel")
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				rm, err := TransformVessel(createdVessel)
				if err != nil {
					d.Logger().WithError(err).Error("Failed to transform vessel")
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				query := r.URL.Query()
				queryParams := jsonapi.ParseQueryFields(&query)
				w.WriteHeader(http.StatusCreated)
				server.MarshalResponse[VesselRestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(rm)
			}
		})
	}
}

// UpdateVesselHandler handles PATCH /tenants/{tenantId}/configurations/vessels/{vesselId}
func UpdateVesselHandler(db *gorm.DB) func(d *rest.HandlerDependency, c *rest.HandlerContext, model VesselRestModel) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext, model VesselRestModel) http.HandlerFunc {
		return rest.ParseTenantId(d.Logger(), func(tenantId uuid.UUID) http.HandlerFunc {
			return rest.ParseVesselId(d.Logger(), func(vesselId string) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					vessel, err := ExtractVessel(model)
					if err != nil {
						d.Logger().WithError(err).Error("Failed to extract vessel data")
						w.WriteHeader(http.StatusBadRequest)
						return
					}

					processor := NewProcessor(d.Logger(), d.Context(), db)
					_, err = processor.UpdateVesselAndEmit(tenantId, vesselId, vessel)
					if err != nil {
						d.Logger().WithError(err).Error("Failed to update vessel")
						w.WriteHeader(http.StatusInternalServerError)
						return
					}

					// Get the specific vessel that was just updated
					updatedVessel, err := processor.GetVesselById(tenantId, vesselId)
					if err != nil {
						d.Logger().WithError(err).Error("Failed to get updated vessel")
						w.WriteHeader(http.StatusInternalServerError)
						return
					}

					rm, err := TransformVessel(updatedVessel)
					if err != nil {
						d.Logger().WithError(err).Error("Failed to transform vessel")
						w.WriteHeader(http.StatusInternalServerError)
						return
					}

					query := r.URL.Query()
					queryParams := jsonapi.ParseQueryFields(&query)
					server.MarshalResponse[VesselRestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(rm)
				}
			})
		})
	}
}

// DeleteVesselHandler handles DELETE /tenants/{tenantId}/configurations/vessels/{vesselId}
func DeleteVesselHandler(db *gorm.DB) func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseTenantId(d.Logger(), func(tenantId uuid.UUID) http.HandlerFunc {
			return rest.ParseVesselId(d.Logger(), func(vesselId string) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					processor := NewProcessor(d.Logger(), d.Context(), db)
					err := processor.DeleteVesselAndEmit(tenantId, vesselId)
					if err != nil {
						d.Logger().WithError(err).Error("Failed to delete vessel")
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
			registerRouteInputHandler := rest.RegisterInputHandler[RouteRestModel](l)(si)
			registerVesselInputHandler := rest.RegisterInputHandler[VesselRestModel](l)(si)

			// Route endpoints
			r.HandleFunc("/tenants/{tenantId}/configurations/routes", registerHandler("get_all_routes", GetAllRoutesHandler(db))).Methods(http.MethodGet)
			r.HandleFunc("/tenants/{tenantId}/configurations/routes/{routeId}", registerHandler("get_route_by_id", GetRouteByIdHandler(db))).Methods(http.MethodGet)
			r.HandleFunc("/tenants/{tenantId}/configurations/routes", registerRouteInputHandler("create_route", CreateRouteHandler(db))).Methods(http.MethodPost)
			r.HandleFunc("/tenants/{tenantId}/configurations/routes/{routeId}", registerRouteInputHandler("update_route", UpdateRouteHandler(db))).Methods(http.MethodPatch)
			r.HandleFunc("/tenants/{tenantId}/configurations/routes/{routeId}", registerHandler("delete_route", DeleteRouteHandler(db))).Methods(http.MethodDelete)

			// Vessel endpoints
			r.HandleFunc("/tenants/{tenantId}/configurations/vessels", registerHandler("get_all_vessels", GetAllVesselsHandler(db))).Methods(http.MethodGet)
			r.HandleFunc("/tenants/{tenantId}/configurations/vessels/{vesselId}", registerHandler("get_vessel_by_id", GetVesselByIdHandler(db))).Methods(http.MethodGet)
			r.HandleFunc("/tenants/{tenantId}/configurations/vessels", registerVesselInputHandler("create_vessel", CreateVesselHandler(db))).Methods(http.MethodPost)
			r.HandleFunc("/tenants/{tenantId}/configurations/vessels/{vesselId}", registerVesselInputHandler("update_vessel", UpdateVesselHandler(db))).Methods(http.MethodPatch)
			r.HandleFunc("/tenants/{tenantId}/configurations/vessels/{vesselId}", registerHandler("delete_vessel", DeleteVesselHandler(db))).Methods(http.MethodDelete)
		}
	}
}
