package configuration

import (
	"atlas-tenants/rest"
	"encoding/json"
	"errors"
	"net/http"
	"sort"

	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server/paginate"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
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
					rm, err := TransformRoute(route)
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

					rm, err := TransformRoute(route)
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

				rm, err := TransformRoute(createdRoute)
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

					rm, err := TransformRoute(updatedRoute)
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
					rm, err := TransformVessel(vessel)
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

					rm, err := TransformVessel(vessel)
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

				rm, err := TransformVessel(createdVessel)
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

					rm, err := TransformVessel(updatedVessel)
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

// GetAllIncubatorRewardsHandler handles GET /tenants/{tenantId}/configurations/incubator-rewards
func GetAllIncubatorRewardsHandler(db *gorm.DB) func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseTenantId(d.Logger(), func(tenantId uuid.UUID) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				processor := NewProcessor(d.Logger(), d.Context(), db)

				rewards, err := processor.GetAllIncubatorRewards(tenantId)
				if err != nil {
					if errors.Is(err, gorm.ErrRecordNotFound) {
						// If no incubator rewards exist, return an empty array instead of an error
						d.Logger().Info("No incubator rewards found for tenant, returning empty array")
						rewards = []map[string]interface{}{}
					} else {
						d.Logger().WithError(err).Error("Failed to get incubator rewards")
						w.WriteHeader(http.StatusInternalServerError)
						return
					}
				}

				restModels := make([]IncubatorRewardRestModel, 0, len(rewards))
				for _, reward := range rewards {
					rm, err := TransformIncubatorReward(reward)
					if err != nil {
						d.Logger().WithError(err).Error("Failed to transform incubator reward")
						w.WriteHeader(http.StatusInternalServerError)
						return
					}
					restModels = append(restModels, rm)
				}

				query := r.URL.Query()
				queryParams := jsonapi.ParseQueryFields(&query)
				server.MarshalResponse[[]IncubatorRewardRestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(restModels)
			}
		})
	}
}

// GetIncubatorRewardByIdHandler handles GET /tenants/{tenantId}/configurations/incubator-rewards/{incubatorRewardId}
func GetIncubatorRewardByIdHandler(db *gorm.DB) func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseTenantId(d.Logger(), func(tenantId uuid.UUID) http.HandlerFunc {
			return rest.ParseIncubatorRewardId(d.Logger(), func(incubatorRewardId string) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					processor := NewProcessor(d.Logger(), d.Context(), db)

					reward, err := processor.GetIncubatorRewardById(tenantId, incubatorRewardId)
					if err != nil {
						d.Logger().WithError(err).Error("Failed to get incubator reward")
						w.WriteHeader(http.StatusNotFound)
						return
					}

					rm, err := TransformIncubatorReward(reward)
					if err != nil {
						d.Logger().WithError(err).Error("Failed to transform incubator reward")
						w.WriteHeader(http.StatusInternalServerError)
						return
					}

					query := r.URL.Query()
					queryParams := jsonapi.ParseQueryFields(&query)
					server.MarshalResponse[IncubatorRewardRestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(rm)
				}
			})
		})
	}
}

// CreateIncubatorRewardHandler handles POST /tenants/{tenantId}/configurations/incubator-rewards
func CreateIncubatorRewardHandler(db *gorm.DB) func(d *rest.HandlerDependency, c *rest.HandlerContext, model IncubatorRewardRestModel) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext, model IncubatorRewardRestModel) http.HandlerFunc {
		return rest.ParseTenantId(d.Logger(), func(tenantId uuid.UUID) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				reward, err := ExtractIncubatorReward(model)
				if err != nil {
					d.Logger().WithError(err).Error("Failed to extract incubator reward data")
					w.WriteHeader(http.StatusBadRequest)
					return
				}

				processor := NewProcessor(d.Logger(), d.Context(), db)
				_, err = processor.CreateIncubatorRewardAndEmit(tenantId, reward)
				if err != nil {
					d.Logger().WithError(err).Error("Failed to create incubator reward")
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				// Get the incubator reward ID from the created incubator reward
				incubatorRewardId := ""
				if id, ok := reward["id"].(string); ok {
					incubatorRewardId = id
				}

				// Get the specific incubator reward that was just created
				createdReward, err := processor.GetIncubatorRewardById(tenantId, incubatorRewardId)
				if err != nil {
					d.Logger().WithError(err).Error("Failed to get created incubator reward")
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				rm, err := TransformIncubatorReward(createdReward)
				if err != nil {
					d.Logger().WithError(err).Error("Failed to transform incubator reward")
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				query := r.URL.Query()
				queryParams := jsonapi.ParseQueryFields(&query)
				w.WriteHeader(http.StatusCreated)
				server.MarshalResponse[IncubatorRewardRestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(rm)
			}
		})
	}
}

// UpdateIncubatorRewardHandler handles PATCH /tenants/{tenantId}/configurations/incubator-rewards/{incubatorRewardId}
func UpdateIncubatorRewardHandler(db *gorm.DB) func(d *rest.HandlerDependency, c *rest.HandlerContext, model IncubatorRewardRestModel) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext, model IncubatorRewardRestModel) http.HandlerFunc {
		return rest.ParseTenantId(d.Logger(), func(tenantId uuid.UUID) http.HandlerFunc {
			return rest.ParseIncubatorRewardId(d.Logger(), func(incubatorRewardId string) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					reward, err := ExtractIncubatorReward(model)
					if err != nil {
						d.Logger().WithError(err).Error("Failed to extract incubator reward data")
						w.WriteHeader(http.StatusBadRequest)
						return
					}

					processor := NewProcessor(d.Logger(), d.Context(), db)
					_, err = processor.UpdateIncubatorRewardAndEmit(tenantId, incubatorRewardId, reward)
					if err != nil {
						d.Logger().WithError(err).Error("Failed to update incubator reward")
						w.WriteHeader(http.StatusInternalServerError)
						return
					}

					// Get the specific incubator reward that was just updated
					updatedReward, err := processor.GetIncubatorRewardById(tenantId, incubatorRewardId)
					if err != nil {
						d.Logger().WithError(err).Error("Failed to get updated incubator reward")
						w.WriteHeader(http.StatusInternalServerError)
						return
					}

					rm, err := TransformIncubatorReward(updatedReward)
					if err != nil {
						d.Logger().WithError(err).Error("Failed to transform incubator reward")
						w.WriteHeader(http.StatusInternalServerError)
						return
					}

					query := r.URL.Query()
					queryParams := jsonapi.ParseQueryFields(&query)
					server.MarshalResponse[IncubatorRewardRestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(rm)
				}
			})
		})
	}
}

// DeleteIncubatorRewardHandler handles DELETE /tenants/{tenantId}/configurations/incubator-rewards/{incubatorRewardId}
func DeleteIncubatorRewardHandler(db *gorm.DB) func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseTenantId(d.Logger(), func(tenantId uuid.UUID) http.HandlerFunc {
			return rest.ParseIncubatorRewardId(d.Logger(), func(incubatorRewardId string) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					processor := NewProcessor(d.Logger(), d.Context(), db)
					err := processor.DeleteIncubatorRewardAndEmit(tenantId, incubatorRewardId)
					if err != nil {
						d.Logger().WithError(err).Error("Failed to delete incubator reward")
						w.WriteHeader(http.StatusInternalServerError)
						return
					}

					w.WriteHeader(http.StatusNoContent)
				}
			})
		})
	}
}

// SeedIncubatorRewardsHandler handles POST /tenants/{tenantId}/configurations/incubator-rewards/seed
func SeedIncubatorRewardsHandler(db *gorm.DB) func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseTenantId(d.Logger(), func(tenantId uuid.UUID) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				processor := NewProcessor(d.Logger(), d.Context(), db)
				result, err := processor.SeedIncubatorRewards(tenantId)
				if err != nil {
					d.Logger().WithError(err).Error("Failed to seed incubator rewards")
					w.WriteHeader(http.StatusInternalServerError)
					json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
					return
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(result)
			}
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
					rm, err := TransformInstanceRoute(route)
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

					rm, err := TransformInstanceRoute(route)
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

				rm, err := TransformInstanceRoute(createdRoute)
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

					rm, err := TransformInstanceRoute(updatedRoute)
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

// SeedRoutesHandler handles POST /tenants/{tenantId}/configurations/routes/seed
func SeedRoutesHandler(db *gorm.DB) func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseTenantId(d.Logger(), func(tenantId uuid.UUID) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				processor := NewProcessor(d.Logger(), d.Context(), db)
				result, err := processor.SeedRoutes(tenantId)
				if err != nil {
					d.Logger().WithError(err).Error("Failed to seed routes")
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

// SeedInstanceRoutesHandler handles POST /tenants/{tenantId}/configurations/instance-routes/seed
func SeedInstanceRoutesHandler(db *gorm.DB) func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseTenantId(d.Logger(), func(tenantId uuid.UUID) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				processor := NewProcessor(d.Logger(), d.Context(), db)
				result, err := processor.SeedInstanceRoutes(tenantId)
				if err != nil {
					d.Logger().WithError(err).Error("Failed to seed instance routes")
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

// SeedVesselsHandler handles POST /tenants/{tenantId}/configurations/vessels/seed
func SeedVesselsHandler(db *gorm.DB) func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseTenantId(d.Logger(), func(tenantId uuid.UUID) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				processor := NewProcessor(d.Logger(), d.Context(), db)
				result, err := processor.SeedVessels(tenantId)
				if err != nil {
					d.Logger().WithError(err).Error("Failed to seed vessels")
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

// RegisterRoutes registers the configuration routes
func RegisterRoutes(db *gorm.DB) func(si jsonapi.ServerInformation) server.RouteInitializer {
	return func(si jsonapi.ServerInformation) server.RouteInitializer {
		return func(r *mux.Router, l logrus.FieldLogger) {
			registerHandler := rest.RegisterHandler(l)(si)
			registerRouteInputHandler := rest.RegisterInputHandler[RouteRestModel](l)(si)
			registerVesselInputHandler := rest.RegisterInputHandler[VesselRestModel](l)(si)
			registerInstanceRouteInputHandler := rest.RegisterInputHandler[InstanceRouteRestModel](l)(si)
			registerIncubatorRewardInputHandler := rest.RegisterInputHandler[IncubatorRewardRestModel](l)(si)
			registerMtsConfigInputHandler := rest.RegisterInputHandler[MtsConfigRestModel](l)(si)

			// Route endpoints
			r.HandleFunc("/tenants/{tenantId}/configurations/routes/seed", registerHandler("seed_routes", SeedRoutesHandler(db))).Methods(http.MethodPost)
			r.HandleFunc("/tenants/{tenantId}/configurations/routes", registerHandler("get_all_routes", GetAllRoutesHandler(db))).Methods(http.MethodGet)
			r.HandleFunc("/tenants/{tenantId}/configurations/routes/{routeId}", registerHandler("get_route_by_id", GetRouteByIdHandler(db))).Methods(http.MethodGet)
			r.HandleFunc("/tenants/{tenantId}/configurations/routes", registerRouteInputHandler("create_route", CreateRouteHandler(db))).Methods(http.MethodPost)
			r.HandleFunc("/tenants/{tenantId}/configurations/routes/{routeId}", registerRouteInputHandler("update_route", UpdateRouteHandler(db))).Methods(http.MethodPatch)
			r.HandleFunc("/tenants/{tenantId}/configurations/routes/{routeId}", registerHandler("delete_route", DeleteRouteHandler(db))).Methods(http.MethodDelete)

			// Vessel endpoints
			r.HandleFunc("/tenants/{tenantId}/configurations/vessels/seed", registerHandler("seed_vessels", SeedVesselsHandler(db))).Methods(http.MethodPost)
			r.HandleFunc("/tenants/{tenantId}/configurations/vessels", registerHandler("get_all_vessels", GetAllVesselsHandler(db))).Methods(http.MethodGet)
			r.HandleFunc("/tenants/{tenantId}/configurations/vessels/{vesselId}", registerHandler("get_vessel_by_id", GetVesselByIdHandler(db))).Methods(http.MethodGet)
			r.HandleFunc("/tenants/{tenantId}/configurations/vessels", registerVesselInputHandler("create_vessel", CreateVesselHandler(db))).Methods(http.MethodPost)
			r.HandleFunc("/tenants/{tenantId}/configurations/vessels/{vesselId}", registerVesselInputHandler("update_vessel", UpdateVesselHandler(db))).Methods(http.MethodPatch)
			r.HandleFunc("/tenants/{tenantId}/configurations/vessels/{vesselId}", registerHandler("delete_vessel", DeleteVesselHandler(db))).Methods(http.MethodDelete)

			// Incubator reward endpoints
			r.HandleFunc("/tenants/{tenantId}/configurations/incubator-rewards/seed", registerHandler("seed_incubator_rewards", SeedIncubatorRewardsHandler(db))).Methods(http.MethodPost)
			r.HandleFunc("/tenants/{tenantId}/configurations/incubator-rewards", registerHandler("get_all_incubator_rewards", GetAllIncubatorRewardsHandler(db))).Methods(http.MethodGet)
			r.HandleFunc("/tenants/{tenantId}/configurations/incubator-rewards/{incubatorRewardId}", registerHandler("get_incubator_reward_by_id", GetIncubatorRewardByIdHandler(db))).Methods(http.MethodGet)
			r.HandleFunc("/tenants/{tenantId}/configurations/incubator-rewards", registerIncubatorRewardInputHandler("create_incubator_reward", CreateIncubatorRewardHandler(db))).Methods(http.MethodPost)
			r.HandleFunc("/tenants/{tenantId}/configurations/incubator-rewards/{incubatorRewardId}", registerIncubatorRewardInputHandler("update_incubator_reward", UpdateIncubatorRewardHandler(db))).Methods(http.MethodPatch)
			r.HandleFunc("/tenants/{tenantId}/configurations/incubator-rewards/{incubatorRewardId}", registerHandler("delete_incubator_reward", DeleteIncubatorRewardHandler(db))).Methods(http.MethodDelete)

			// Instance route endpoints
			r.HandleFunc("/tenants/{tenantId}/configurations/instance-routes/seed", registerHandler("seed_instance_routes", SeedInstanceRoutesHandler(db))).Methods(http.MethodPost)
			r.HandleFunc("/tenants/{tenantId}/configurations/instance-routes", registerHandler("get_all_instance_routes", GetAllInstanceRoutesHandler(db))).Methods(http.MethodGet)
			r.HandleFunc("/tenants/{tenantId}/configurations/instance-routes/{instanceRouteId}", registerHandler("get_instance_route_by_id", GetInstanceRouteByIdHandler(db))).Methods(http.MethodGet)
			r.HandleFunc("/tenants/{tenantId}/configurations/instance-routes", registerInstanceRouteInputHandler("create_instance_route", CreateInstanceRouteHandler(db))).Methods(http.MethodPost)
			r.HandleFunc("/tenants/{tenantId}/configurations/instance-routes/{instanceRouteId}", registerInstanceRouteInputHandler("update_instance_route", UpdateInstanceRouteHandler(db))).Methods(http.MethodPatch)
			r.HandleFunc("/tenants/{tenantId}/configurations/instance-routes/{instanceRouteId}", registerHandler("delete_instance_route", DeleteInstanceRouteHandler(db))).Methods(http.MethodDelete)

			// MTS config endpoints
			r.HandleFunc("/tenants/{tenantId}/configurations/mts-configs/seed", registerHandler("seed_mts_configs", SeedMtsConfigsHandler(db))).Methods(http.MethodPost)
			r.HandleFunc("/tenants/{tenantId}/configurations/mts-configs", registerHandler("get_mts_config", GetMtsConfigHandler(db))).Methods(http.MethodGet)
			r.HandleFunc("/tenants/{tenantId}/configurations/mts-configs/{mtsConfigId}", registerHandler("get_mts_config_by_id", GetMtsConfigByIdHandler(db))).Methods(http.MethodGet)
			r.HandleFunc("/tenants/{tenantId}/configurations/mts-configs", registerMtsConfigInputHandler("create_mts_config", CreateMtsConfigHandler(db))).Methods(http.MethodPost)
			r.HandleFunc("/tenants/{tenantId}/configurations/mts-configs/{mtsConfigId}", registerMtsConfigInputHandler("update_mts_config", UpdateMtsConfigHandler(db))).Methods(http.MethodPatch)
			r.HandleFunc("/tenants/{tenantId}/configurations/mts-configs/{mtsConfigId}", registerHandler("delete_mts_config", DeleteMtsConfigHandler(db))).Methods(http.MethodDelete)
		}
	}
}
