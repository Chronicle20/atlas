package main

import (
	"atlas-compartment-transfer/logger"
	"atlas-compartment-transfer/service"
	"atlas-compartment-transfer/tracing"
)

const serviceName = "atlas-compartment-transfer"
const consumerGroupId = "Compartment Transfer Service"

func main() {
	l := logger.CreateLogger(serviceName)
	l.Infoln("Starting main service.")

	tdm := service.GetTeardownManager()

	tc, err := tracing.InitTracer(l)(serviceName)
	if err != nil {
		l.WithError(err).Fatal("Unable to initialize tracer.")
	}

	tdm.TeardownFunc(tracing.Teardown(l)(tc))

	tdm.Wait()
	l.Infoln("Service shutdown.")
}
