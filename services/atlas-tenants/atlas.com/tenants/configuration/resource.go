package configuration

import (
	"atlas-tenants/rest"
	"encoding/json"
	"errors"
	"net/http"
	"sort"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server/paginate"
)

// GetAllRoutesHandler handles GET /tenants/{tenantId}/configurations/routes
func GetAllRoutesHandler(db *gorm.DB) func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseTenantId(d.Logger(), func(tenantId uuid.UUID) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				page, err := paginate.ParseParams(r.URL.Query(), paginate.DefaultPageSize, paginate.MaxPageSize)
				if err != nil {
					server.WriteBadRequest(d.Logger(), w, "invalid page[number]/page[size]")
					return
				}

				processor := NewProcessor(d.Logger(), d.Context(), db)

				routes, err := processor.GetAllRoutes(tenantId)
				if err != nil {
					if errors.Is(err, gorm.ErrRecordNotFound) {
						// If no routes exist, return an empty array instead of an error
						d.Logger().Info("No routes found for tenant, returning empty array")
						routes = []map[string]interface{}{}
					} else {
						d.Logger().WithError(err).Error("Failed to get routes")
						server.WriteErrorResponse(d.Logger())(w)(err)
						return
					}
				}

				restModels := make([]RouteRestModel, 0, len(routes))
				for _, route := range routes {
					rm, err := TransformRoute(tenantId, route)
					if err != nil {
						d.Logger().WithError(err).Error("Failed to transform route")
						server.WriteErrorResponse(d.Logger())(w)(err)
						return
					}
					restModels = append(restModels, rm)
				}

				// The route list materializes from one JSONB blob's "data"
				// array; sort by the unique id before paging so the response
				// order does not depend on how the blob happens to store them.
				sort.Slice(restModels, func(i, j int) bool {
					return restModels[i].Id < restModels[j].Id
				})
				paged := paginate.Slice(restModels, page)

				query := r.URL.Query()
				queryParams := jsonapi.ParseQueryFields(&query)
				server.MarshalPaginatedResponse[[]RouteRestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(paged.Items, paginate.EnvelopeFor(paged), r)
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

					rm, err := TransformRoute(tenantId, route)
					if err != nil {
						d.Logger().WithError(err).Error("Failed to transform route")
						server.WriteErrorResponse(d.Logger())(w)(err)
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
					server.WriteErrorResponse(d.Logger())(w)(err)
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
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}

				rm, err := TransformRoute(tenantId, createdRoute)
				if err != nil {
					d.Logger().WithError(err).Error("Failed to transform route")
					server.WriteErrorResponse(d.Logger())(w)(err)
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
						server.WriteErrorResponse(d.Logger())(w)(err)
						return
					}

					// Get the specific route that was just updated
					updatedRoute, err := processor.GetRouteById(tenantId, routeId)
					if err != nil {
						d.Logger().WithError(err).Error("Failed to get updated route")
						server.WriteErrorResponse(d.Logger())(w)(err)
						return
					}

					rm, err := TransformRoute(tenantId, updatedRoute)
					if err != nil {
						d.Logger().WithError(err).Error("Failed to transform route")
						server.WriteErrorResponse(d.Logger())(w)(err)
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
						server.WriteErrorResponse(d.Logger())(w)(err)
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
				page, err := paginate.ParseParams(r.URL.Query(), paginate.DefaultPageSize, paginate.MaxPageSize)
				if err != nil {
					server.WriteBadRequest(d.Logger(), w, "invalid page[number]/page[size]")
					return
				}

				processor := NewProcessor(d.Logger(), d.Context(), db)

				vessels, err := processor.GetAllVessels(tenantId)
				if err != nil {
					if errors.Is(err, gorm.ErrRecordNotFound) {
						// If no vessels exist, return an empty array instead of an error
						d.Logger().Info("No vessels found for tenant, returning empty array")
						vessels = []map[string]interface{}{}
					} else {
						d.Logger().WithError(err).Error("Failed to get vessels")
						server.WriteErrorResponse(d.Logger())(w)(err)
						return
					}
				}

				restModels := make([]VesselRestModel, 0, len(vessels))
				for _, vessel := range vessels {
					rm, err := TransformVessel(tenantId, vessel)
					if err != nil {
						d.Logger().WithError(err).Error("Failed to transform vessel")
						server.WriteErrorResponse(d.Logger())(w)(err)
						return
					}
					restModels = append(restModels, rm)
				}

				// The vessel list materializes from one JSONB blob's "data"
				// array; sort by the unique id before paging so the response
				// order does not depend on how the blob happens to store them.
				sort.Slice(restModels, func(i, j int) bool {
					return restModels[i].Id < restModels[j].Id
				})
				paged := paginate.Slice(restModels, page)

				query := r.URL.Query()
				queryParams := jsonapi.ParseQueryFields(&query)
				server.MarshalPaginatedResponse[[]VesselRestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(paged.Items, paginate.EnvelopeFor(paged), r)
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

					rm, err := TransformVessel(tenantId, vessel)
					if err != nil {
						d.Logger().WithError(err).Error("Failed to transform vessel")
						server.WriteErrorResponse(d.Logger())(w)(err)
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
					server.WriteErrorResponse(d.Logger())(w)(err)
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
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}

				rm, err := TransformVessel(tenantId, createdVessel)
				if err != nil {
					d.Logger().WithError(err).Error("Failed to transform vessel")
					server.WriteErrorResponse(d.Logger())(w)(err)
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
						server.WriteErrorResponse(d.Logger())(w)(err)
						return
					}

					// Get the specific vessel that was just updated
					updatedVessel, err := processor.GetVesselById(tenantId, vesselId)
					if err != nil {
						d.Logger().WithError(err).Error("Failed to get updated vessel")
						server.WriteErrorResponse(d.Logger())(w)(err)
						return
					}

					rm, err := TransformVessel(tenantId, updatedVessel)
					if err != nil {
						d.Logger().WithError(err).Error("Failed to transform vessel")
						server.WriteErrorResponse(d.Logger())(w)(err)
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
						server.WriteErrorResponse(d.Logger())(w)(err)
						return
					}

					w.WriteHeader(http.StatusNoContent)
				}
			})
		})
	}
}

// GetAllInstanceRoutesHandler handles GET /tenants/{tenantId}/configurations/instance-routes
func GetAllInstanceRoutesHandler(db *gorm.DB) func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseTenantId(d.Logger(), func(tenantId uuid.UUID) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				page, err := paginate.ParseParams(r.URL.Query(), paginate.DefaultPageSize, paginate.MaxPageSize)
				if err != nil {
					server.WriteBadRequest(d.Logger(), w, "invalid page[number]/page[size]")
					return
				}

				processor := NewProcessor(d.Logger(), d.Context(), db)

				routes, err := processor.GetAllInstanceRoutes(tenantId)
				if err != nil {
					if errors.Is(err, gorm.ErrRecordNotFound) {
						d.Logger().Info("No instance routes found for tenant, returning empty array")
						routes = []map[string]interface{}{}
					} else {
						d.Logger().WithError(err).Error("Failed to get instance routes")
						server.WriteErrorResponse(d.Logger())(w)(err)
						return
					}
				}

				restModels := make([]InstanceRouteRestModel, 0, len(routes))
				for _, route := range routes {
					rm, err := TransformInstanceRoute(tenantId, route)
					if err != nil {
						d.Logger().WithError(err).Error("Failed to transform instance route")
						server.WriteErrorResponse(d.Logger())(w)(err)
						return
					}
					restModels = append(restModels, rm)
				}

				// The instance-route list materializes from one JSONB blob's
				// "data" array; sort by the unique id before paging so the
				// response order does not depend on how the blob happens to
				// store them.
				sort.Slice(restModels, func(i, j int) bool {
					return restModels[i].Id < restModels[j].Id
				})
				paged := paginate.Slice(restModels, page)

				query := r.URL.Query()
				queryParams := jsonapi.ParseQueryFields(&query)
				server.MarshalPaginatedResponse[[]InstanceRouteRestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(paged.Items, paginate.EnvelopeFor(paged), r)
			}
		})
	}
}

// GetInstanceRouteByIdHandler handles GET /tenants/{tenantId}/configurations/instance-routes/{instanceRouteId}
func GetInstanceRouteByIdHandler(db *gorm.DB) func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseTenantId(d.Logger(), func(tenantId uuid.UUID) http.HandlerFunc {
			return rest.ParseInstanceRouteId(d.Logger(), func(instanceRouteId string) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					processor := NewProcessor(d.Logger(), d.Context(), db)

					route, err := processor.GetInstanceRouteById(tenantId, instanceRouteId)
					if err != nil {
						d.Logger().WithError(err).Error("Failed to get instance route")
						w.WriteHeader(http.StatusNotFound)
						return
					}

					rm, err := TransformInstanceRoute(tenantId, route)
					if err != nil {
						d.Logger().WithError(err).Error("Failed to transform instance route")
						server.WriteErrorResponse(d.Logger())(w)(err)
						return
					}

					query := r.URL.Query()
					queryParams := jsonapi.ParseQueryFields(&query)
					server.MarshalResponse[InstanceRouteRestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(rm)
				}
			})
		})
	}
}

// CreateInstanceRouteHandler handles POST /tenants/{tenantId}/configurations/instance-routes
func CreateInstanceRouteHandler(db *gorm.DB) func(d *rest.HandlerDependency, c *rest.HandlerContext, model InstanceRouteRestModel) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext, model InstanceRouteRestModel) http.HandlerFunc {
		return rest.ParseTenantId(d.Logger(), func(tenantId uuid.UUID) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				route, err := ExtractInstanceRoute(model)
				if err != nil {
					d.Logger().WithError(err).Error("Failed to extract instance route data")
					w.WriteHeader(http.StatusBadRequest)
					return
				}

				processor := NewProcessor(d.Logger(), d.Context(), db)
				_, err = processor.CreateInstanceRouteAndEmit(tenantId, route)
				if err != nil {
					d.Logger().WithError(err).Error("Failed to create instance route")
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}

				routeId := ""
				if id, ok := route["id"].(string); ok {
					routeId = id
				}

				createdRoute, err := processor.GetInstanceRouteById(tenantId, routeId)
				if err != nil {
					d.Logger().WithError(err).Error("Failed to get created instance route")
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}

				rm, err := TransformInstanceRoute(tenantId, createdRoute)
				if err != nil {
					d.Logger().WithError(err).Error("Failed to transform instance route")
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}

				query := r.URL.Query()
				queryParams := jsonapi.ParseQueryFields(&query)
				w.WriteHeader(http.StatusCreated)
				server.MarshalResponse[InstanceRouteRestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(rm)
			}
		})
	}
}

// UpdateInstanceRouteHandler handles PATCH /tenants/{tenantId}/configurations/instance-routes/{instanceRouteId}
func UpdateInstanceRouteHandler(db *gorm.DB) func(d *rest.HandlerDependency, c *rest.HandlerContext, model InstanceRouteRestModel) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext, model InstanceRouteRestModel) http.HandlerFunc {
		return rest.ParseTenantId(d.Logger(), func(tenantId uuid.UUID) http.HandlerFunc {
			return rest.ParseInstanceRouteId(d.Logger(), func(instanceRouteId string) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					route, err := ExtractInstanceRoute(model)
					if err != nil {
						d.Logger().WithError(err).Error("Failed to extract instance route data")
						w.WriteHeader(http.StatusBadRequest)
						return
					}

					processor := NewProcessor(d.Logger(), d.Context(), db)
					_, err = processor.UpdateInstanceRouteAndEmit(tenantId, instanceRouteId, route)
					if err != nil {
						d.Logger().WithError(err).Error("Failed to update instance route")
						server.WriteErrorResponse(d.Logger())(w)(err)
						return
					}

					updatedRoute, err := processor.GetInstanceRouteById(tenantId, instanceRouteId)
					if err != nil {
						d.Logger().WithError(err).Error("Failed to get updated instance route")
						server.WriteErrorResponse(d.Logger())(w)(err)
						return
					}

					rm, err := TransformInstanceRoute(tenantId, updatedRoute)
					if err != nil {
						d.Logger().WithError(err).Error("Failed to transform instance route")
						server.WriteErrorResponse(d.Logger())(w)(err)
						return
					}

					query := r.URL.Query()
					queryParams := jsonapi.ParseQueryFields(&query)
					server.MarshalResponse[InstanceRouteRestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(rm)
				}
			})
		})
	}
}

// DeleteInstanceRouteHandler handles DELETE /tenants/{tenantId}/configurations/instance-routes/{instanceRouteId}
func DeleteInstanceRouteHandler(db *gorm.DB) func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseTenantId(d.Logger(), func(tenantId uuid.UUID) http.HandlerFunc {
			return rest.ParseInstanceRouteId(d.Logger(), func(instanceRouteId string) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					processor := NewProcessor(d.Logger(), d.Context(), db)
					err := processor.DeleteInstanceRouteAndEmit(tenantId, instanceRouteId)
					if err != nil {
						d.Logger().WithError(err).Error("Failed to delete instance route")
						server.WriteErrorResponse(d.Logger())(w)(err)
						return
					}

					w.WriteHeader(http.StatusNoContent)
				}
			})
		})
	}
}

// GetAllRpsRewardsHandler handles GET /tenants/{tenantId}/configurations/rps-rewards
func GetAllRpsRewardsHandler(db *gorm.DB) func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseTenantId(d.Logger(), func(tenantId uuid.UUID) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				page, err := paginate.ParseParams(r.URL.Query(), paginate.DefaultPageSize, paginate.MaxPageSize)
				if err != nil {
					server.WriteBadRequest(d.Logger(), w, "invalid page[number]/page[size]")
					return
				}

				processor := NewProcessor(d.Logger(), d.Context(), db)

				rpsRewards, err := processor.GetAllRpsRewards(tenantId)
				if err != nil {
					if errors.Is(err, gorm.ErrRecordNotFound) {
						// If no rps-rewards exist, return an empty array instead of an error
						d.Logger().Info("No rps-rewards found for tenant, returning empty array")
						rpsRewards = []map[string]interface{}{}
					} else {
						d.Logger().WithError(err).Error("Failed to get rps-rewards")
						server.WriteErrorResponse(d.Logger())(w)(err)
						return
					}
				}

				restModels := make([]RpsRewardRestModel, 0, len(rpsRewards))
				for _, rpsReward := range rpsRewards {
					rm, err := TransformRpsReward(rpsReward)
					if err != nil {
						d.Logger().WithError(err).Error("Failed to transform rps-reward")
						server.WriteErrorResponse(d.Logger())(w)(err)
						return
					}
					restModels = append(restModels, rm)
				}

				// The rps-rewards list materializes from one JSONB blob's
				// "data" array; sort by the unique id before paging so the
				// response order does not depend on blob storage order.
				sort.Slice(restModels, func(i, j int) bool {
					return restModels[i].Id < restModels[j].Id
				})
				paged := paginate.Slice(restModels, page)

				query := r.URL.Query()
				queryParams := jsonapi.ParseQueryFields(&query)
				server.MarshalPaginatedResponse[[]RpsRewardRestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(paged.Items, paginate.EnvelopeFor(paged), r)
			}
		})
	}
}

// GetRpsRewardByIdHandler handles GET /tenants/{tenantId}/configurations/rps-rewards/{rpsRewardId}
func GetRpsRewardByIdHandler(db *gorm.DB) func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseTenantId(d.Logger(), func(tenantId uuid.UUID) http.HandlerFunc {
			return rest.ParseRpsRewardId(d.Logger(), func(rpsRewardId string) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					processor := NewProcessor(d.Logger(), d.Context(), db)

					rpsReward, err := processor.GetRpsRewardById(tenantId, rpsRewardId)
					if err != nil {
						d.Logger().WithError(err).Error("Failed to get rps-reward")
						w.WriteHeader(http.StatusNotFound)
						return
					}

					rm, err := TransformRpsReward(rpsReward)
					if err != nil {
						d.Logger().WithError(err).Error("Failed to transform rps-reward")
						server.WriteErrorResponse(d.Logger())(w)(err)
						return
					}

					query := r.URL.Query()
					queryParams := jsonapi.ParseQueryFields(&query)
					server.MarshalResponse[RpsRewardRestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(rm)
				}
			})
		})
	}
}

// CreateRpsRewardHandler handles POST /tenants/{tenantId}/configurations/rps-rewards
func CreateRpsRewardHandler(db *gorm.DB) func(d *rest.HandlerDependency, c *rest.HandlerContext, model RpsRewardRestModel) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext, model RpsRewardRestModel) http.HandlerFunc {
		return rest.ParseTenantId(d.Logger(), func(tenantId uuid.UUID) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				rpsReward, err := ExtractRpsReward(model)
				if err != nil {
					d.Logger().WithError(err).Error("Failed to extract rps-reward data")
					w.WriteHeader(http.StatusBadRequest)
					return
				}

				processor := NewProcessor(d.Logger(), d.Context(), db)
				_, err = processor.CreateRpsRewardAndEmit(tenantId, rpsReward)
				if err != nil {
					d.Logger().WithError(err).Error("Failed to create rps-reward")
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}

				// Get the rps-reward ID from the created rps-reward
				rpsRewardId := ""
				if id, ok := rpsReward["id"].(string); ok {
					rpsRewardId = id
				}

				// Get the specific rps-reward that was just created
				createdRpsReward, err := processor.GetRpsRewardById(tenantId, rpsRewardId)
				if err != nil {
					d.Logger().WithError(err).Error("Failed to get created rps-reward")
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}

				rm, err := TransformRpsReward(createdRpsReward)
				if err != nil {
					d.Logger().WithError(err).Error("Failed to transform rps-reward")
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}

				query := r.URL.Query()
				queryParams := jsonapi.ParseQueryFields(&query)
				w.WriteHeader(http.StatusCreated)
				server.MarshalResponse[RpsRewardRestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(rm)
			}
		})
	}
}

// UpdateRpsRewardHandler handles PATCH /tenants/{tenantId}/configurations/rps-rewards/{rpsRewardId}
func UpdateRpsRewardHandler(db *gorm.DB) func(d *rest.HandlerDependency, c *rest.HandlerContext, model RpsRewardRestModel) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext, model RpsRewardRestModel) http.HandlerFunc {
		return rest.ParseTenantId(d.Logger(), func(tenantId uuid.UUID) http.HandlerFunc {
			return rest.ParseRpsRewardId(d.Logger(), func(rpsRewardId string) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					rpsReward, err := ExtractRpsReward(model)
					if err != nil {
						d.Logger().WithError(err).Error("Failed to extract rps-reward data")
						w.WriteHeader(http.StatusBadRequest)
						return
					}

					processor := NewProcessor(d.Logger(), d.Context(), db)
					_, err = processor.UpdateRpsRewardAndEmit(tenantId, rpsRewardId, rpsReward)
					if err != nil {
						d.Logger().WithError(err).Error("Failed to update rps-reward")
						server.WriteErrorResponse(d.Logger())(w)(err)
						return
					}

					// Get the specific rps-reward that was just updated
					updatedRpsReward, err := processor.GetRpsRewardById(tenantId, rpsRewardId)
					if err != nil {
						d.Logger().WithError(err).Error("Failed to get updated rps-reward")
						server.WriteErrorResponse(d.Logger())(w)(err)
						return
					}

					rm, err := TransformRpsReward(updatedRpsReward)
					if err != nil {
						d.Logger().WithError(err).Error("Failed to transform rps-reward")
						server.WriteErrorResponse(d.Logger())(w)(err)
						return
					}

					query := r.URL.Query()
					queryParams := jsonapi.ParseQueryFields(&query)
					server.MarshalResponse[RpsRewardRestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(rm)
				}
			})
		})
	}
}

// DeleteRpsRewardHandler handles DELETE /tenants/{tenantId}/configurations/rps-rewards/{rpsRewardId}
func DeleteRpsRewardHandler(db *gorm.DB) func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseTenantId(d.Logger(), func(tenantId uuid.UUID) http.HandlerFunc {
			return rest.ParseRpsRewardId(d.Logger(), func(rpsRewardId string) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					processor := NewProcessor(d.Logger(), d.Context(), db)
					err := processor.DeleteRpsRewardAndEmit(tenantId, rpsRewardId)
					if err != nil {
						d.Logger().WithError(err).Error("Failed to delete rps-reward")
						server.WriteErrorResponse(d.Logger())(w)(err)
						return
					}

					w.WriteHeader(http.StatusNoContent)
				}
			})
		})
	}
}

// GetMtsConfigHandler handles GET /tenants/{tenantId}/configurations/mts-configs
// and returns the single per-tenant MTS configuration. atlas-mts decodes this
// as a single JSON:API object (requests.GetRequest[RestModel]).
func GetMtsConfigHandler(db *gorm.DB) func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseTenantId(d.Logger(), func(tenantId uuid.UUID) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				processor := NewProcessor(d.Logger(), d.Context(), db)

				configs, err := processor.GetAllMtsConfigs(tenantId)
				if err != nil {
					if errors.Is(err, gorm.ErrRecordNotFound) {
						d.Logger().Info("No mts config found for tenant")
						w.WriteHeader(http.StatusNotFound)
						return
					}
					d.Logger().WithError(err).Error("Failed to get mts config")
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}

				if len(configs) == 0 {
					d.Logger().Info("No mts config found for tenant")
					w.WriteHeader(http.StatusNotFound)
					return
				}

				rm, err := TransformMtsConfig(configs[0])
				if err != nil {
					d.Logger().WithError(err).Error("Failed to transform mts config")
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}

				query := r.URL.Query()
				queryParams := jsonapi.ParseQueryFields(&query)
				server.MarshalResponse[MtsConfigRestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(rm)
			}
		})
	}
}

// GetMtsConfigByIdHandler handles GET /tenants/{tenantId}/configurations/mts-configs/{mtsConfigId}
func GetMtsConfigByIdHandler(db *gorm.DB) func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseTenantId(d.Logger(), func(tenantId uuid.UUID) http.HandlerFunc {
			return rest.ParseMtsConfigId(d.Logger(), func(mtsConfigId string) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					processor := NewProcessor(d.Logger(), d.Context(), db)

					config, err := processor.GetMtsConfigById(tenantId, mtsConfigId)
					if err != nil {
						d.Logger().WithError(err).Error("Failed to get mts config")
						w.WriteHeader(http.StatusNotFound)
						return
					}

					rm, err := TransformMtsConfig(config)
					if err != nil {
						d.Logger().WithError(err).Error("Failed to transform mts config")
						server.WriteErrorResponse(d.Logger())(w)(err)
						return
					}

					query := r.URL.Query()
					queryParams := jsonapi.ParseQueryFields(&query)
					server.MarshalResponse[MtsConfigRestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(rm)
				}
			})
		})
	}
}

// CreateMtsConfigHandler handles POST /tenants/{tenantId}/configurations/mts-configs
func CreateMtsConfigHandler(db *gorm.DB) func(d *rest.HandlerDependency, c *rest.HandlerContext, model MtsConfigRestModel) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext, model MtsConfigRestModel) http.HandlerFunc {
		return rest.ParseTenantId(d.Logger(), func(tenantId uuid.UUID) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				config, err := ExtractMtsConfig(model)
				if err != nil {
					d.Logger().WithError(err).Error("Failed to extract mts config data")
					w.WriteHeader(http.StatusBadRequest)
					return
				}

				processor := NewProcessor(d.Logger(), d.Context(), db)
				_, err = processor.CreateMtsConfigAndEmit(tenantId, config)
				if err != nil {
					d.Logger().WithError(err).Error("Failed to create mts config")
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}

				// Get the config ID from the created config
				configId := ""
				if id, ok := config["id"].(string); ok {
					configId = id
				}

				// Get the specific config that was just created
				createdConfig, err := processor.GetMtsConfigById(tenantId, configId)
				if err != nil {
					d.Logger().WithError(err).Error("Failed to get created mts config")
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}

				rm, err := TransformMtsConfig(createdConfig)
				if err != nil {
					d.Logger().WithError(err).Error("Failed to transform mts config")
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}

				query := r.URL.Query()
				queryParams := jsonapi.ParseQueryFields(&query)
				w.WriteHeader(http.StatusCreated)
				server.MarshalResponse[MtsConfigRestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(rm)
			}
		})
	}
}

// UpdateMtsConfigHandler handles PATCH /tenants/{tenantId}/configurations/mts-configs/{mtsConfigId}
func UpdateMtsConfigHandler(db *gorm.DB) func(d *rest.HandlerDependency, c *rest.HandlerContext, model MtsConfigRestModel) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext, model MtsConfigRestModel) http.HandlerFunc {
		return rest.ParseTenantId(d.Logger(), func(tenantId uuid.UUID) http.HandlerFunc {
			return rest.ParseMtsConfigId(d.Logger(), func(mtsConfigId string) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					config, err := ExtractMtsConfig(model)
					if err != nil {
						d.Logger().WithError(err).Error("Failed to extract mts config data")
						w.WriteHeader(http.StatusBadRequest)
						return
					}

					processor := NewProcessor(d.Logger(), d.Context(), db)
					_, err = processor.UpdateMtsConfigAndEmit(tenantId, mtsConfigId, config)
					if err != nil {
						d.Logger().WithError(err).Error("Failed to update mts config")
						server.WriteErrorResponse(d.Logger())(w)(err)
						return
					}

					// Get the specific config that was just updated
					updatedConfig, err := processor.GetMtsConfigById(tenantId, mtsConfigId)
					if err != nil {
						d.Logger().WithError(err).Error("Failed to get updated mts config")
						server.WriteErrorResponse(d.Logger())(w)(err)
						return
					}

					rm, err := TransformMtsConfig(updatedConfig)
					if err != nil {
						d.Logger().WithError(err).Error("Failed to transform mts config")
						server.WriteErrorResponse(d.Logger())(w)(err)
						return
					}

					query := r.URL.Query()
					queryParams := jsonapi.ParseQueryFields(&query)
					server.MarshalResponse[MtsConfigRestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(rm)
				}
			})
		})
	}
}

// DeleteMtsConfigHandler handles DELETE /tenants/{tenantId}/configurations/mts-configs/{mtsConfigId}
func DeleteMtsConfigHandler(db *gorm.DB) func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseTenantId(d.Logger(), func(tenantId uuid.UUID) http.HandlerFunc {
			return rest.ParseMtsConfigId(d.Logger(), func(mtsConfigId string) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					processor := NewProcessor(d.Logger(), d.Context(), db)
					err := processor.DeleteMtsConfigAndEmit(tenantId, mtsConfigId)
					if err != nil {
						d.Logger().WithError(err).Error("Failed to delete mts config")
						server.WriteErrorResponse(d.Logger())(w)(err)
						return
					}

					w.WriteHeader(http.StatusNoContent)
				}
			})
		})
	}
}

// SeedMtsConfigsHandler handles POST /tenants/{tenantId}/configurations/mts-configs/seed
func SeedMtsConfigsHandler(db *gorm.DB) func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseTenantId(d.Logger(), func(tenantId uuid.UUID) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				processor := NewProcessor(d.Logger(), d.Context(), db)
				result, err := processor.SeedMtsConfigs(tenantId)
				if err != nil {
					d.Logger().WithError(err).Error("Failed to seed mts configs")
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(result)
			}
		})
	}
}

// GetTradeConfigHandler handles GET /tenants/{tenantId}/configurations/trade-configs
// and returns the single per-tenant trade configuration. atlas-trades decodes
// this as a single JSON:API object (requests.GetRequest[RestModel]) and falls
// back to its shipped defaults on the 404 below (FR-9.2).
func GetTradeConfigHandler(db *gorm.DB) func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseTenantId(d.Logger(), func(tenantId uuid.UUID) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				processor := NewProcessor(d.Logger(), d.Context(), db)

				configs, err := processor.GetAllTradeConfigs(tenantId)
				if err != nil {
					if errors.Is(err, gorm.ErrRecordNotFound) {
						d.Logger().Info("No trade config found for tenant")
						w.WriteHeader(http.StatusNotFound)
						return
					}
					d.Logger().WithError(err).Error("Failed to get trade config")
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}

				if len(configs) == 0 {
					d.Logger().Info("No trade config found for tenant")
					w.WriteHeader(http.StatusNotFound)
					return
				}

				rm, err := TransformTradeConfig(configs[0])
				if err != nil {
					d.Logger().WithError(err).Error("Failed to transform trade config")
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}

				query := r.URL.Query()
				queryParams := jsonapi.ParseQueryFields(&query)
				server.MarshalResponse[TradeConfigRestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(rm)
			}
		})
	}
}

// GetTradeConfigByIdHandler handles GET /tenants/{tenantId}/configurations/trade-configs/{tradeConfigId}
func GetTradeConfigByIdHandler(db *gorm.DB) func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseTenantId(d.Logger(), func(tenantId uuid.UUID) http.HandlerFunc {
			return rest.ParseTradeConfigId(d.Logger(), func(tradeConfigId string) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					processor := NewProcessor(d.Logger(), d.Context(), db)

					config, err := processor.GetTradeConfigById(tenantId, tradeConfigId)
					if err != nil {
						d.Logger().WithError(err).Error("Failed to get trade config")
						w.WriteHeader(http.StatusNotFound)
						return
					}

					rm, err := TransformTradeConfig(config)
					if err != nil {
						d.Logger().WithError(err).Error("Failed to transform trade config")
						server.WriteErrorResponse(d.Logger())(w)(err)
						return
					}

					query := r.URL.Query()
					queryParams := jsonapi.ParseQueryFields(&query)
					server.MarshalResponse[TradeConfigRestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(rm)
				}
			})
		})
	}
}

// CreateTradeConfigHandler handles POST /tenants/{tenantId}/configurations/trade-configs
func CreateTradeConfigHandler(db *gorm.DB) func(d *rest.HandlerDependency, c *rest.HandlerContext, model TradeConfigRestModel) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext, model TradeConfigRestModel) http.HandlerFunc {
		return rest.ParseTenantId(d.Logger(), func(tenantId uuid.UUID) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				config, err := ExtractTradeConfig(model)
				if err != nil {
					d.Logger().WithError(err).Error("Failed to extract trade config data")
					w.WriteHeader(http.StatusBadRequest)
					return
				}

				processor := NewProcessor(d.Logger(), d.Context(), db)
				_, err = processor.CreateTradeConfigAndEmit(tenantId, config)
				if err != nil {
					d.Logger().WithError(err).Error("Failed to create trade config")
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}

				// Get the config ID from the created config
				configId := ""
				if id, ok := config["id"].(string); ok {
					configId = id
				}

				// Get the specific config that was just created
				createdConfig, err := processor.GetTradeConfigById(tenantId, configId)
				if err != nil {
					d.Logger().WithError(err).Error("Failed to get created trade config")
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}

				rm, err := TransformTradeConfig(createdConfig)
				if err != nil {
					d.Logger().WithError(err).Error("Failed to transform trade config")
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}

				query := r.URL.Query()
				queryParams := jsonapi.ParseQueryFields(&query)
				w.WriteHeader(http.StatusCreated)
				server.MarshalResponse[TradeConfigRestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(rm)
			}
		})
	}
}

// UpdateTradeConfigHandler handles PATCH /tenants/{tenantId}/configurations/trade-configs/{tradeConfigId}
func UpdateTradeConfigHandler(db *gorm.DB) func(d *rest.HandlerDependency, c *rest.HandlerContext, model TradeConfigRestModel) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext, model TradeConfigRestModel) http.HandlerFunc {
		return rest.ParseTenantId(d.Logger(), func(tenantId uuid.UUID) http.HandlerFunc {
			return rest.ParseTradeConfigId(d.Logger(), func(tradeConfigId string) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					config, err := ExtractTradeConfig(model)
					if err != nil {
						d.Logger().WithError(err).Error("Failed to extract trade config data")
						w.WriteHeader(http.StatusBadRequest)
						return
					}

					processor := NewProcessor(d.Logger(), d.Context(), db)
					_, err = processor.UpdateTradeConfigAndEmit(tenantId, tradeConfigId, config)
					if err != nil {
						d.Logger().WithError(err).Error("Failed to update trade config")
						server.WriteErrorResponse(d.Logger())(w)(err)
						return
					}

					// Get the specific config that was just updated
					updatedConfig, err := processor.GetTradeConfigById(tenantId, tradeConfigId)
					if err != nil {
						d.Logger().WithError(err).Error("Failed to get updated trade config")
						server.WriteErrorResponse(d.Logger())(w)(err)
						return
					}

					rm, err := TransformTradeConfig(updatedConfig)
					if err != nil {
						d.Logger().WithError(err).Error("Failed to transform trade config")
						server.WriteErrorResponse(d.Logger())(w)(err)
						return
					}

					query := r.URL.Query()
					queryParams := jsonapi.ParseQueryFields(&query)
					server.MarshalResponse[TradeConfigRestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(rm)
				}
			})
		})
	}
}

// DeleteTradeConfigHandler handles DELETE /tenants/{tenantId}/configurations/trade-configs/{tradeConfigId}
func DeleteTradeConfigHandler(db *gorm.DB) func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseTenantId(d.Logger(), func(tenantId uuid.UUID) http.HandlerFunc {
			return rest.ParseTradeConfigId(d.Logger(), func(tradeConfigId string) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					processor := NewProcessor(d.Logger(), d.Context(), db)
					err := processor.DeleteTradeConfigAndEmit(tenantId, tradeConfigId)
					if err != nil {
						d.Logger().WithError(err).Error("Failed to delete trade config")
						server.WriteErrorResponse(d.Logger())(w)(err)
						return
					}

					w.WriteHeader(http.StatusNoContent)
				}
			})
		})
	}
}

// SeedTradeConfigsHandler handles POST /tenants/{tenantId}/configurations/trade-configs/seed
func SeedTradeConfigsHandler(db *gorm.DB) func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseTenantId(d.Logger(), func(tenantId uuid.UUID) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				processor := NewProcessor(d.Logger(), d.Context(), db)
				result, err := processor.SeedTradeConfigs(tenantId)
				if err != nil {
					d.Logger().WithError(err).Error("Failed to seed trade configs")
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode(result)
			}
		})
	}
}

// GetRankingsHandler handles GET /tenants/{tenantId}/configurations/rankings
func GetRankingsHandler(db *gorm.DB) func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseTenantId(d.Logger(), func(tenantId uuid.UUID) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				processor := NewProcessor(d.Logger(), d.Context(), db)

				rankings, err := processor.GetRankings(tenantId)
				if err != nil {
					if errors.Is(err, gorm.ErrRecordNotFound) {
						w.WriteHeader(http.StatusNotFound)
						return
					}
					d.Logger().WithError(err).Error("Failed to get rankings configuration")
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}

				rm, err := TransformRankings(rankings)
				if err != nil {
					d.Logger().WithError(err).Error("Failed to transform rankings configuration")
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}

				query := r.URL.Query()
				queryParams := jsonapi.ParseQueryFields(&query)
				server.MarshalResponse[RankingsRestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(rm)
			}
		})
	}
}

// CreateRankingsHandler handles POST /tenants/{tenantId}/configurations/rankings
func CreateRankingsHandler(db *gorm.DB) func(d *rest.HandlerDependency, c *rest.HandlerContext, model RankingsRestModel) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext, model RankingsRestModel) http.HandlerFunc {
		return rest.ParseTenantId(d.Logger(), func(tenantId uuid.UUID) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				rankings, err := ExtractRankings(model)
				if err != nil {
					d.Logger().WithError(err).Error("Failed to extract rankings data")
					w.WriteHeader(http.StatusBadRequest)
					return
				}

				processor := NewProcessor(d.Logger(), d.Context(), db)
				if _, err = processor.CreateRankingsAndEmit(tenantId, rankings); err != nil {
					d.Logger().WithError(err).Error("Failed to create rankings configuration")
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}

				created, err := processor.GetRankings(tenantId)
				if err != nil {
					d.Logger().WithError(err).Error("Failed to get created rankings configuration")
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}
				rm, err := TransformRankings(created)
				if err != nil {
					d.Logger().WithError(err).Error("Failed to transform rankings configuration")
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}

				query := r.URL.Query()
				queryParams := jsonapi.ParseQueryFields(&query)
				w.WriteHeader(http.StatusCreated)
				server.MarshalResponse[RankingsRestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(rm)
			}
		})
	}
}

// UpdateRankingsHandler handles PATCH /tenants/{tenantId}/configurations/rankings
func UpdateRankingsHandler(db *gorm.DB) func(d *rest.HandlerDependency, c *rest.HandlerContext, model RankingsRestModel) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext, model RankingsRestModel) http.HandlerFunc {
		return rest.ParseTenantId(d.Logger(), func(tenantId uuid.UUID) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				rankings, err := ExtractRankings(model)
				if err != nil {
					d.Logger().WithError(err).Error("Failed to extract rankings data")
					w.WriteHeader(http.StatusBadRequest)
					return
				}

				processor := NewProcessor(d.Logger(), d.Context(), db)
				if _, err = processor.UpdateRankingsAndEmit(tenantId, rankings); err != nil {
					if errors.Is(err, gorm.ErrRecordNotFound) {
						w.WriteHeader(http.StatusNotFound)
						return
					}
					d.Logger().WithError(err).Error("Failed to update rankings configuration")
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}

				updated, err := processor.GetRankings(tenantId)
				if err != nil {
					d.Logger().WithError(err).Error("Failed to get updated rankings configuration")
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}
				rm, err := TransformRankings(updated)
				if err != nil {
					d.Logger().WithError(err).Error("Failed to transform rankings configuration")
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}

				query := r.URL.Query()
				queryParams := jsonapi.ParseQueryFields(&query)
				server.MarshalResponse[RankingsRestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(rm)
			}
		})
	}
}

// DeleteRankingsHandler handles DELETE /tenants/{tenantId}/configurations/rankings
func DeleteRankingsHandler(db *gorm.DB) func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseTenantId(d.Logger(), func(tenantId uuid.UUID) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				processor := NewProcessor(d.Logger(), d.Context(), db)
				if err := processor.DeleteRankingsAndEmit(tenantId); err != nil {
					d.Logger().WithError(err).Error("Failed to delete rankings configuration")
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}
				w.WriteHeader(http.StatusNoContent)
			}
		})
	}
}

// GetKiteConfigHandler handles GET /tenants/{tenantId}/configurations/kite-configs
func GetKiteConfigHandler(db *gorm.DB) func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseTenantId(d.Logger(), func(tenantId uuid.UUID) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				processor := NewProcessor(d.Logger(), d.Context(), db)

				cfg, err := processor.GetKiteConfig(tenantId)
				if err != nil {
					if errors.Is(err, gorm.ErrRecordNotFound) {
						w.WriteHeader(http.StatusNotFound)
						return
					}
					d.Logger().WithError(err).Error("Failed to get kite-configs configuration")
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}

				rm, err := TransformKiteConfig(cfg)
				if err != nil {
					d.Logger().WithError(err).Error("Failed to transform kite-configs configuration")
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}

				query := r.URL.Query()
				queryParams := jsonapi.ParseQueryFields(&query)
				server.MarshalResponse[KiteConfigRestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(rm)
			}
		})
	}
}

// CreateKiteConfigHandler handles POST /tenants/{tenantId}/configurations/kite-configs
func CreateKiteConfigHandler(db *gorm.DB) func(d *rest.HandlerDependency, c *rest.HandlerContext, model KiteConfigRestModel) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext, model KiteConfigRestModel) http.HandlerFunc {
		return rest.ParseTenantId(d.Logger(), func(tenantId uuid.UUID) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				cfg, err := ExtractKiteConfig(model)
				if err != nil {
					d.Logger().WithError(err).Error("Failed to extract kite-configs data")
					w.WriteHeader(http.StatusBadRequest)
					return
				}

				processor := NewProcessor(d.Logger(), d.Context(), db)
				if _, err = processor.CreateKiteConfigAndEmit(tenantId, cfg); err != nil {
					d.Logger().WithError(err).Error("Failed to create kite-configs configuration")
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}

				created, err := processor.GetKiteConfig(tenantId)
				if err != nil {
					d.Logger().WithError(err).Error("Failed to get created kite-configs configuration")
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}
				rm, err := TransformKiteConfig(created)
				if err != nil {
					d.Logger().WithError(err).Error("Failed to transform kite-configs configuration")
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}

				query := r.URL.Query()
				queryParams := jsonapi.ParseQueryFields(&query)
				w.WriteHeader(http.StatusCreated)
				server.MarshalResponse[KiteConfigRestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(rm)
			}
		})
	}
}

// UpdateKiteConfigHandler handles PATCH /tenants/{tenantId}/configurations/kite-configs
func UpdateKiteConfigHandler(db *gorm.DB) func(d *rest.HandlerDependency, c *rest.HandlerContext, model KiteConfigRestModel) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext, model KiteConfigRestModel) http.HandlerFunc {
		return rest.ParseTenantId(d.Logger(), func(tenantId uuid.UUID) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				cfg, err := ExtractKiteConfig(model)
				if err != nil {
					d.Logger().WithError(err).Error("Failed to extract kite-configs data")
					w.WriteHeader(http.StatusBadRequest)
					return
				}

				processor := NewProcessor(d.Logger(), d.Context(), db)
				if _, err = processor.UpdateKiteConfigAndEmit(tenantId, cfg); err != nil {
					if errors.Is(err, gorm.ErrRecordNotFound) {
						w.WriteHeader(http.StatusNotFound)
						return
					}
					d.Logger().WithError(err).Error("Failed to update kite-configs configuration")
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}

				updated, err := processor.GetKiteConfig(tenantId)
				if err != nil {
					d.Logger().WithError(err).Error("Failed to get updated kite-configs configuration")
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}
				rm, err := TransformKiteConfig(updated)
				if err != nil {
					d.Logger().WithError(err).Error("Failed to transform kite-configs configuration")
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}

				query := r.URL.Query()
				queryParams := jsonapi.ParseQueryFields(&query)
				server.MarshalResponse[KiteConfigRestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(rm)
			}
		})
	}
}

// DeleteKiteConfigHandler handles DELETE /tenants/{tenantId}/configurations/kite-configs
func DeleteKiteConfigHandler(db *gorm.DB) func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseTenantId(d.Logger(), func(tenantId uuid.UUID) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				processor := NewProcessor(d.Logger(), d.Context(), db)
				if err := processor.DeleteKiteConfigAndEmit(tenantId); err != nil {
					d.Logger().WithError(err).Error("Failed to delete kite-configs configuration")
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}
				w.WriteHeader(http.StatusNoContent)
			}
		})
	}
}

// GetPlayerNpcConfigHandler handles GET /tenants/{tenantId}/configurations/player-npcs
func GetPlayerNpcConfigHandler(db *gorm.DB) func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseTenantId(d.Logger(), func(tenantId uuid.UUID) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				processor := NewProcessor(d.Logger(), d.Context(), db)

				cfg, err := processor.GetPlayerNpcConfig(tenantId)
				if err != nil {
					if errors.Is(err, gorm.ErrRecordNotFound) {
						w.WriteHeader(http.StatusNotFound)
						return
					}
					d.Logger().WithError(err).Error("Failed to get player-npcs configuration")
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}

				rm, err := TransformPlayerNpcConfig(cfg)
				if err != nil {
					d.Logger().WithError(err).Error("Failed to transform player-npcs configuration")
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}

				query := r.URL.Query()
				queryParams := jsonapi.ParseQueryFields(&query)
				server.MarshalResponse[PlayerNpcConfigRestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(rm)
			}
		})
	}
}

// CreatePlayerNpcConfigHandler handles POST /tenants/{tenantId}/configurations/player-npcs
func CreatePlayerNpcConfigHandler(db *gorm.DB) func(d *rest.HandlerDependency, c *rest.HandlerContext, model PlayerNpcConfigRestModel) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext, model PlayerNpcConfigRestModel) http.HandlerFunc {
		return rest.ParseTenantId(d.Logger(), func(tenantId uuid.UUID) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				cfg, err := ExtractPlayerNpcConfig(model)
				if err != nil {
					d.Logger().WithError(err).Error("Failed to extract player-npcs data")
					w.WriteHeader(http.StatusBadRequest)
					return
				}

				processor := NewProcessor(d.Logger(), d.Context(), db)
				if _, err = processor.CreatePlayerNpcConfigAndEmit(tenantId, cfg); err != nil {
					d.Logger().WithError(err).Error("Failed to create player-npcs configuration")
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}

				created, err := processor.GetPlayerNpcConfig(tenantId)
				if err != nil {
					d.Logger().WithError(err).Error("Failed to get created player-npcs configuration")
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}
				rm, err := TransformPlayerNpcConfig(created)
				if err != nil {
					d.Logger().WithError(err).Error("Failed to transform player-npcs configuration")
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}

				query := r.URL.Query()
				queryParams := jsonapi.ParseQueryFields(&query)
				w.WriteHeader(http.StatusCreated)
				server.MarshalResponse[PlayerNpcConfigRestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(rm)
			}
		})
	}
}

// UpdatePlayerNpcConfigHandler handles PATCH /tenants/{tenantId}/configurations/player-npcs
func UpdatePlayerNpcConfigHandler(db *gorm.DB) func(d *rest.HandlerDependency, c *rest.HandlerContext, model PlayerNpcConfigRestModel) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext, model PlayerNpcConfigRestModel) http.HandlerFunc {
		return rest.ParseTenantId(d.Logger(), func(tenantId uuid.UUID) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				cfg, err := ExtractPlayerNpcConfig(model)
				if err != nil {
					d.Logger().WithError(err).Error("Failed to extract player-npcs data")
					w.WriteHeader(http.StatusBadRequest)
					return
				}

				processor := NewProcessor(d.Logger(), d.Context(), db)
				if _, err = processor.UpdatePlayerNpcConfigAndEmit(tenantId, cfg); err != nil {
					if errors.Is(err, gorm.ErrRecordNotFound) {
						w.WriteHeader(http.StatusNotFound)
						return
					}
					d.Logger().WithError(err).Error("Failed to update player-npcs configuration")
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}

				updated, err := processor.GetPlayerNpcConfig(tenantId)
				if err != nil {
					d.Logger().WithError(err).Error("Failed to get updated player-npcs configuration")
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}
				rm, err := TransformPlayerNpcConfig(updated)
				if err != nil {
					d.Logger().WithError(err).Error("Failed to transform player-npcs configuration")
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}

				query := r.URL.Query()
				queryParams := jsonapi.ParseQueryFields(&query)
				server.MarshalResponse[PlayerNpcConfigRestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(rm)
			}
		})
	}
}

// DeletePlayerNpcConfigHandler handles DELETE /tenants/{tenantId}/configurations/player-npcs
func DeletePlayerNpcConfigHandler(db *gorm.DB) func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseTenantId(d.Logger(), func(tenantId uuid.UUID) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				processor := NewProcessor(d.Logger(), d.Context(), db)
				if err := processor.DeletePlayerNpcConfigAndEmit(tenantId); err != nil {
					d.Logger().WithError(err).Error("Failed to delete player-npcs configuration")
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}
				w.WriteHeader(http.StatusNoContent)
			}
		})
	}
}

// SeedRpsRewardsHandler handles POST /tenants/{tenantId}/configurations/rps-rewards/seed
func SeedRpsRewardsHandler(db *gorm.DB) func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseTenantId(d.Logger(), func(tenantId uuid.UUID) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				processor := NewProcessor(d.Logger(), d.Context(), db)
				result, err := processor.SeedRpsRewards(tenantId)
				if err != nil {
					d.Logger().WithError(err).Error("Failed to seed rps-rewards")
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(result)
			}
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
			registerInstanceRouteInputHandler := rest.RegisterInputHandler[InstanceRouteRestModel](l)(si)
			registerRpsRewardInputHandler := rest.RegisterInputHandler[RpsRewardRestModel](l)(si)
			registerMtsConfigInputHandler := rest.RegisterInputHandler[MtsConfigRestModel](l)(si)
			registerTradeConfigInputHandler := rest.RegisterInputHandler[TradeConfigRestModel](l)(si)
			registerRankingsInputHandler := rest.RegisterInputHandler[RankingsRestModel](l)(si)
			registerKiteConfigInputHandler := rest.RegisterInputHandler[KiteConfigRestModel](l)(si)
			registerPlayerNpcConfigInputHandler := rest.RegisterInputHandler[PlayerNpcConfigRestModel](l)(si)

			// Route endpoints
			//
			// The path-scoped seed endpoint is gone (see configuration/seed);
			// without an explicit stand-in, POST "/routes/seed" would fall
			// through to the "/routes/{routeId}" pattern below (routeId=
			// "seed") and 405 instead of 404, since that pattern's GET/
			// PATCH/DELETE handlers still match the path. This stand-in is
			// scoped to POST only — like the removed route it replaces —
			// so GET/PATCH/DELETE on a route whose real id happens to be
			// "seed" still reach the CRUD {routeId} handlers below.
			r.HandleFunc("/tenants/{tenantId}/configurations/routes/seed", http.NotFound).Methods(http.MethodPost)
			r.HandleFunc("/tenants/{tenantId}/configurations/routes", registerHandler("get_all_routes", GetAllRoutesHandler(db))).Methods(http.MethodGet)
			r.HandleFunc("/tenants/{tenantId}/configurations/routes/{routeId}", registerHandler("get_route_by_id", GetRouteByIdHandler(db))).Methods(http.MethodGet)
			r.HandleFunc("/tenants/{tenantId}/configurations/routes", registerRouteInputHandler("create_route", CreateRouteHandler(db))).Methods(http.MethodPost)
			r.HandleFunc("/tenants/{tenantId}/configurations/routes/{routeId}", registerRouteInputHandler("update_route", UpdateRouteHandler(db))).Methods(http.MethodPatch)
			r.HandleFunc("/tenants/{tenantId}/configurations/routes/{routeId}", registerHandler("delete_route", DeleteRouteHandler(db))).Methods(http.MethodDelete)

			// Vessel endpoints
			// POST-only stand-in for the removed "/vessels/seed" endpoint —
			// see the routes-endpoints comment above for why this must not
			// shadow GET/PATCH/DELETE on a vessel whose id is "seed".
			r.HandleFunc("/tenants/{tenantId}/configurations/vessels/seed", http.NotFound).Methods(http.MethodPost)
			r.HandleFunc("/tenants/{tenantId}/configurations/vessels", registerHandler("get_all_vessels", GetAllVesselsHandler(db))).Methods(http.MethodGet)
			r.HandleFunc("/tenants/{tenantId}/configurations/vessels/{vesselId}", registerHandler("get_vessel_by_id", GetVesselByIdHandler(db))).Methods(http.MethodGet)
			r.HandleFunc("/tenants/{tenantId}/configurations/vessels", registerVesselInputHandler("create_vessel", CreateVesselHandler(db))).Methods(http.MethodPost)
			r.HandleFunc("/tenants/{tenantId}/configurations/vessels/{vesselId}", registerVesselInputHandler("update_vessel", UpdateVesselHandler(db))).Methods(http.MethodPatch)
			r.HandleFunc("/tenants/{tenantId}/configurations/vessels/{vesselId}", registerHandler("delete_vessel", DeleteVesselHandler(db))).Methods(http.MethodDelete)

			// Instance route endpoints
			// POST-only stand-in for the removed "/instance-routes/seed"
			// endpoint — see the routes-endpoints comment above for why this
			// must not shadow GET/PATCH/DELETE on an instance route whose id
			// is "seed".
			r.HandleFunc("/tenants/{tenantId}/configurations/instance-routes/seed", http.NotFound).Methods(http.MethodPost)
			r.HandleFunc("/tenants/{tenantId}/configurations/instance-routes", registerHandler("get_all_instance_routes", GetAllInstanceRoutesHandler(db))).Methods(http.MethodGet)
			r.HandleFunc("/tenants/{tenantId}/configurations/instance-routes/{instanceRouteId}", registerHandler("get_instance_route_by_id", GetInstanceRouteByIdHandler(db))).Methods(http.MethodGet)
			r.HandleFunc("/tenants/{tenantId}/configurations/instance-routes", registerInstanceRouteInputHandler("create_instance_route", CreateInstanceRouteHandler(db))).Methods(http.MethodPost)
			r.HandleFunc("/tenants/{tenantId}/configurations/instance-routes/{instanceRouteId}", registerInstanceRouteInputHandler("update_instance_route", UpdateInstanceRouteHandler(db))).Methods(http.MethodPatch)
			r.HandleFunc("/tenants/{tenantId}/configurations/instance-routes/{instanceRouteId}", registerHandler("delete_instance_route", DeleteInstanceRouteHandler(db))).Methods(http.MethodDelete)

			// RPS reward endpoints
			r.HandleFunc("/tenants/{tenantId}/configurations/rps-rewards/seed", registerHandler("seed_rps_rewards", SeedRpsRewardsHandler(db))).Methods(http.MethodPost)
			r.HandleFunc("/tenants/{tenantId}/configurations/rps-rewards", registerHandler("get_all_rps_rewards", GetAllRpsRewardsHandler(db))).Methods(http.MethodGet)
			r.HandleFunc("/tenants/{tenantId}/configurations/rps-rewards/{rpsRewardId}", registerHandler("get_rps_reward_by_id", GetRpsRewardByIdHandler(db))).Methods(http.MethodGet)
			r.HandleFunc("/tenants/{tenantId}/configurations/rps-rewards", registerRpsRewardInputHandler("create_rps_reward", CreateRpsRewardHandler(db))).Methods(http.MethodPost)
			r.HandleFunc("/tenants/{tenantId}/configurations/rps-rewards/{rpsRewardId}", registerRpsRewardInputHandler("update_rps_reward", UpdateRpsRewardHandler(db))).Methods(http.MethodPatch)
			r.HandleFunc("/tenants/{tenantId}/configurations/rps-rewards/{rpsRewardId}", registerHandler("delete_rps_reward", DeleteRpsRewardHandler(db))).Methods(http.MethodDelete)

			// MTS config endpoints
			r.HandleFunc("/tenants/{tenantId}/configurations/mts-configs/seed", registerHandler("seed_mts_configs", SeedMtsConfigsHandler(db))).Methods(http.MethodPost)
			r.HandleFunc("/tenants/{tenantId}/configurations/mts-configs", registerHandler("get_mts_config", GetMtsConfigHandler(db))).Methods(http.MethodGet)
			r.HandleFunc("/tenants/{tenantId}/configurations/mts-configs/{mtsConfigId}", registerHandler("get_mts_config_by_id", GetMtsConfigByIdHandler(db))).Methods(http.MethodGet)
			r.HandleFunc("/tenants/{tenantId}/configurations/mts-configs", registerMtsConfigInputHandler("create_mts_config", CreateMtsConfigHandler(db))).Methods(http.MethodPost)
			r.HandleFunc("/tenants/{tenantId}/configurations/mts-configs/{mtsConfigId}", registerMtsConfigInputHandler("update_mts_config", UpdateMtsConfigHandler(db))).Methods(http.MethodPatch)
			r.HandleFunc("/tenants/{tenantId}/configurations/mts-configs/{mtsConfigId}", registerHandler("delete_mts_config", DeleteMtsConfigHandler(db))).Methods(http.MethodDelete)

			// Trade config endpoints
			// "/seed" is registered BEFORE the "{tradeConfigId}" pattern so it
			// is not shadowed by a config whose id happens to be "seed".
			r.HandleFunc("/tenants/{tenantId}/configurations/trade-configs/seed", registerHandler("seed_trade_configs", SeedTradeConfigsHandler(db))).Methods(http.MethodPost)
			r.HandleFunc("/tenants/{tenantId}/configurations/trade-configs", registerHandler("get_trade_config", GetTradeConfigHandler(db))).Methods(http.MethodGet)
			r.HandleFunc("/tenants/{tenantId}/configurations/trade-configs/{tradeConfigId}", registerHandler("get_trade_config_by_id", GetTradeConfigByIdHandler(db))).Methods(http.MethodGet)
			r.HandleFunc("/tenants/{tenantId}/configurations/trade-configs", registerTradeConfigInputHandler("create_trade_config", CreateTradeConfigHandler(db))).Methods(http.MethodPost)
			r.HandleFunc("/tenants/{tenantId}/configurations/trade-configs/{tradeConfigId}", registerTradeConfigInputHandler("update_trade_config", UpdateTradeConfigHandler(db))).Methods(http.MethodPatch)
			r.HandleFunc("/tenants/{tenantId}/configurations/trade-configs/{tradeConfigId}", registerHandler("delete_trade_config", DeleteTradeConfigHandler(db))).Methods(http.MethodDelete)

			// Rankings endpoints
			r.HandleFunc("/tenants/{tenantId}/configurations/rankings", registerHandler("get_rankings_config", GetRankingsHandler(db))).Methods(http.MethodGet)
			r.HandleFunc("/tenants/{tenantId}/configurations/rankings", registerRankingsInputHandler("create_rankings_config", CreateRankingsHandler(db))).Methods(http.MethodPost)
			r.HandleFunc("/tenants/{tenantId}/configurations/rankings", registerRankingsInputHandler("update_rankings_config", UpdateRankingsHandler(db))).Methods(http.MethodPatch)
			r.HandleFunc("/tenants/{tenantId}/configurations/rankings", registerHandler("delete_rankings_config", DeleteRankingsHandler(db))).Methods(http.MethodDelete)

			// Kite config endpoints — one config per tenant (rankings shape:
			// no /seed endpoint, no id-addressed sub-resource).
			r.HandleFunc("/tenants/{tenantId}/configurations/kite-configs", registerHandler("get_kite_config", GetKiteConfigHandler(db))).Methods(http.MethodGet)
			r.HandleFunc("/tenants/{tenantId}/configurations/kite-configs", registerKiteConfigInputHandler("create_kite_config", CreateKiteConfigHandler(db))).Methods(http.MethodPost)
			r.HandleFunc("/tenants/{tenantId}/configurations/kite-configs", registerKiteConfigInputHandler("update_kite_config", UpdateKiteConfigHandler(db))).Methods(http.MethodPatch)
			r.HandleFunc("/tenants/{tenantId}/configurations/kite-configs", registerHandler("delete_kite_config", DeleteKiteConfigHandler(db))).Methods(http.MethodDelete)

			// Player NPC config endpoints — one config per tenant (rankings
			// shape: no /seed endpoint, no id-addressed sub-resource).
			r.HandleFunc("/tenants/{tenantId}/configurations/player-npcs", registerHandler("get_player_npc_config", GetPlayerNpcConfigHandler(db))).Methods(http.MethodGet)
			r.HandleFunc("/tenants/{tenantId}/configurations/player-npcs", registerPlayerNpcConfigInputHandler("create_player_npc_config", CreatePlayerNpcConfigHandler(db))).Methods(http.MethodPost)
			r.HandleFunc("/tenants/{tenantId}/configurations/player-npcs", registerPlayerNpcConfigInputHandler("update_player_npc_config", UpdatePlayerNpcConfigHandler(db))).Methods(http.MethodPatch)
			r.HandleFunc("/tenants/{tenantId}/configurations/player-npcs", registerHandler("delete_player_npc_config", DeletePlayerNpcConfigHandler(db))).Methods(http.MethodDelete)

			// Imprint config endpoints (FR-2.6 pending-change expiry) — see
			// imprint_handler.go.
			RegisterImprintConfigRoutes(db, si, l, r)
		}
	}
}
