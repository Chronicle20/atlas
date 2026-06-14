package test

import (
	"context"

	"github.com/google/uuid"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

type TenantVariant struct {
	Name         string
	Region       string
	MajorVersion uint16
	MinorVersion uint16
}

var Variants = []TenantVariant{
	{Name: "GMS v28", Region: "GMS", MajorVersion: 28, MinorVersion: 1},
	{Name: "GMS v83", Region: "GMS", MajorVersion: 83, MinorVersion: 1},
	{Name: "GMS v87", Region: "GMS", MajorVersion: 87, MinorVersion: 1},
	{Name: "GMS v95", Region: "GMS", MajorVersion: 95, MinorVersion: 1},
	{Name: "JMS v185", Region: "JMS", MajorVersion: 185, MinorVersion: 1},
	// v84/v86 are byte-identical to v83 for most packets (minor GMS bump).
	// EXCEPTION (task-088, IDA-confirmed): the serverbound summon ATTACK send
	// already carries the anti-hack envelope at v84 (CSummoned::TryDoingAttackManual
	// send block @0x7cafcd in GMS_v84.1), so that gate is MajorAtLeast(84), not
	// 87. Clientbound summon packets remain v83-shaped. Appended (not inserted)
	// so existing positional Variants[N] references stay valid.
	{Name: "GMS v84", Region: "GMS", MajorVersion: 84, MinorVersion: 1},
	{Name: "GMS v86", Region: "GMS", MajorVersion: 86, MinorVersion: 1},
}

func CreateContext(region string, majorVersion uint16, minorVersion uint16) context.Context {
	t, _ := tenant.Create(uuid.New(), region, majorVersion, minorVersion)
	return tenant.WithContext(context.Background(), t)
}
