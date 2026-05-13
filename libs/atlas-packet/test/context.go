package test

import (
	"context"

	"github.com/google/uuid"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

type TenantVariant struct {
	Name          string
	Region        string
	MajorVersion  uint16
	MinorVersion  uint16
	ClientVariant string
}

var Variants = []TenantVariant{
	{Name: "GMS v28", Region: "GMS", MajorVersion: 28, MinorVersion: 1, ClientVariant: "modified"},
	{Name: "GMS v83", Region: "GMS", MajorVersion: 83, MinorVersion: 1, ClientVariant: "modified"},
	{Name: "GMS v95 modified", Region: "GMS", MajorVersion: 95, MinorVersion: 1, ClientVariant: "modified"},
	{Name: "JMS v185", Region: "JMS", MajorVersion: 185, MinorVersion: 1, ClientVariant: "modified"},
}

func CreateContext(region string, majorVersion uint16, minorVersion uint16) context.Context {
	return CreateContextWithVariant(region, majorVersion, minorVersion, "modified")
}

func CreateContextWithVariant(region string, majorVersion uint16, minorVersion uint16, variant string) context.Context {
	t, _ := tenant.CreateWithVariant(uuid.New(), region, majorVersion, minorVersion, variant)
	return tenant.WithContext(context.Background(), t)
}
