package server

import (
	"context"
	"net/http"

	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
)

func RegisterHandler(l logrus.FieldLogger) func(si jsonapi.ServerInformation) func(handlerName string, handler GetHandler) http.HandlerFunc {
	return func(si jsonapi.ServerInformation) func(handlerName string, handler GetHandler) http.HandlerFunc {
		return func(handlerName string, handler GetHandler) http.HandlerFunc {
			return RetrieveSpan(l, handlerName, context.Background(), func(sl logrus.FieldLogger, sctx context.Context) http.HandlerFunc {
				fl := sl.WithFields(logrus.Fields{"originator": handlerName, "type": "rest_handler"})
				return ParseEnvironment(fl, sctx, func(el logrus.FieldLogger, ectx context.Context) http.HandlerFunc {
					return ParseTenant(el, ectx, func(tl logrus.FieldLogger, tctx context.Context) http.HandlerFunc {
						return handler(&HandlerDependency{l: tl, ctx: tctx}, &HandlerContext{si: si})
					})
				})
			})
		}
	}
}

func RegisterInputHandler[M any](l logrus.FieldLogger) func(si jsonapi.ServerInformation) func(handlerName string, handler InputHandler[M]) http.HandlerFunc {
	return func(si jsonapi.ServerInformation) func(handlerName string, handler InputHandler[M]) http.HandlerFunc {
		return func(handlerName string, handler InputHandler[M]) http.HandlerFunc {
			return RetrieveSpan(l, handlerName, context.Background(), func(sl logrus.FieldLogger, sctx context.Context) http.HandlerFunc {
				fl := sl.WithFields(logrus.Fields{"originator": handlerName, "type": "rest_handler"})
				return ParseEnvironment(fl, sctx, func(el logrus.FieldLogger, ectx context.Context) http.HandlerFunc {
					return ParseTenant(el, ectx, func(tl logrus.FieldLogger, tctx context.Context) http.HandlerFunc {
						return ParseInput[M](&HandlerDependency{l: tl, ctx: tctx}, &HandlerContext{si: si}, handler)
					})
				})
			})
		}
	}
}

// RegisterOptionalInputHandler is RegisterInputHandler's sibling for a
// handler whose request body is OPTIONAL: an absent or `{}` body decodes to
// the zero value of M (see ParseOptionalInput). It composes the same
// RetrieveSpan -> ParseEnvironment -> ParseTenant chain, differing only in
// its final decode step.
func RegisterOptionalInputHandler[M any](l logrus.FieldLogger) func(si jsonapi.ServerInformation) func(handlerName string, handler InputHandler[M]) http.HandlerFunc {
	return func(si jsonapi.ServerInformation) func(handlerName string, handler InputHandler[M]) http.HandlerFunc {
		return func(handlerName string, handler InputHandler[M]) http.HandlerFunc {
			return RetrieveSpan(l, handlerName, context.Background(), func(sl logrus.FieldLogger, sctx context.Context) http.HandlerFunc {
				fl := sl.WithFields(logrus.Fields{"originator": handlerName, "type": "rest_handler"})
				return ParseEnvironment(fl, sctx, func(el logrus.FieldLogger, ectx context.Context) http.HandlerFunc {
					return ParseTenant(el, ectx, func(tl logrus.FieldLogger, tctx context.Context) http.HandlerFunc {
						return ParseOptionalInput[M](&HandlerDependency{l: tl, ctx: tctx}, &HandlerContext{si: si}, handler)
					})
				})
			})
		}
	}
}

func RegisterSimpleHandler(l logrus.FieldLogger) func(si jsonapi.ServerInformation) func(handlerName string, handler GetHandler) http.HandlerFunc {
	return func(si jsonapi.ServerInformation) func(handlerName string, handler GetHandler) http.HandlerFunc {
		return func(handlerName string, handler GetHandler) http.HandlerFunc {
			return RetrieveSpan(l, handlerName, context.Background(), func(sl logrus.FieldLogger, sctx context.Context) http.HandlerFunc {
				fl := sl.WithFields(logrus.Fields{"originator": handlerName, "type": "rest_handler"})
				return ParseEnvironment(fl, sctx, func(el logrus.FieldLogger, ectx context.Context) http.HandlerFunc {
					return handler(&HandlerDependency{l: el, ctx: ectx}, &HandlerContext{si: si})
				})
			})
		}
	}
}

func RegisterSimpleInputHandler[M any](l logrus.FieldLogger) func(si jsonapi.ServerInformation) func(handlerName string, handler InputHandler[M]) http.HandlerFunc {
	return func(si jsonapi.ServerInformation) func(handlerName string, handler InputHandler[M]) http.HandlerFunc {
		return func(handlerName string, handler InputHandler[M]) http.HandlerFunc {
			return RetrieveSpan(l, handlerName, context.Background(), func(sl logrus.FieldLogger, sctx context.Context) http.HandlerFunc {
				fl := sl.WithFields(logrus.Fields{"originator": handlerName, "type": "rest_handler"})
				return ParseEnvironment(fl, sctx, func(el logrus.FieldLogger, ectx context.Context) http.HandlerFunc {
					return ParseInput[M](&HandlerDependency{l: el, ctx: ectx}, &HandlerContext{si: si}, handler)
				})
			})
		}
	}
}

// RegisterSimpleOptionalInputHandler is RegisterSimpleInputHandler's sibling
// for a handler whose request body is OPTIONAL (see ParseOptionalInput). It
// omits ParseTenant exactly as RegisterSimpleInputHandler does.
func RegisterSimpleOptionalInputHandler[M any](l logrus.FieldLogger) func(si jsonapi.ServerInformation) func(handlerName string, handler InputHandler[M]) http.HandlerFunc {
	return func(si jsonapi.ServerInformation) func(handlerName string, handler InputHandler[M]) http.HandlerFunc {
		return func(handlerName string, handler InputHandler[M]) http.HandlerFunc {
			return RetrieveSpan(l, handlerName, context.Background(), func(sl logrus.FieldLogger, sctx context.Context) http.HandlerFunc {
				fl := sl.WithFields(logrus.Fields{"originator": handlerName, "type": "rest_handler"})
				return ParseEnvironment(fl, sctx, func(el logrus.FieldLogger, ectx context.Context) http.HandlerFunc {
					return ParseOptionalInput[M](&HandlerDependency{l: el, ctx: ectx}, &HandlerContext{si: si}, handler)
				})
			})
		}
	}
}
