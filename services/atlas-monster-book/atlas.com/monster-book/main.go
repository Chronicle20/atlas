package main

import (
	"atlas-monster-book/logger"
	"github.com/Chronicle20/atlas/libs/atlas-service"
)

const serviceName = "atlas-monster-book"

func main() {
	l := logger.CreateLogger(serviceName)
	l.Infoln("Starting main service.")
	tdm := service.GetTeardownManager()
	tdm.Wait()
	l.Infoln("Service shutdown.")
}
