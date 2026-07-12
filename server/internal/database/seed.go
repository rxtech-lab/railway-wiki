package database

import (
	"fmt"
	"log"

	"gorm.io/gorm"

	"github.com/rxtech-lab/railway-wiki/internal/models"
)

// Seed inserts a small, distinctly-named row for each core resource so an
// E2E-mode server has data to render immediately. It is idempotent (safe to run
// on every startup): each row is matched on a stable natural key via
// FirstOrCreate, so re-running never duplicates or mutates existing rows.
//
// Only intended for local SQLite end-to-end / UI testing.
func Seed(db *gorm.DB) error {
	strptr := func(s string) *string { return &s }

	var company models.Company
	if err := db.
		Where(models.Company{Name: "MTR"}).
		Attrs(models.Company{ShortName: strptr("MTR"), Country: strptr("Hong Kong")}).
		FirstOrCreate(&company).Error; err != nil {
		return fmt.Errorf("seed company: %w", err)
	}

	var station models.Station
	if err := db.
		Where(models.Station{Name: "Central"}).
		Attrs(models.Station{StationNumber: strptr("CEN"), NameEn: strptr("Central Station")}).
		FirstOrCreate(&station).Error; err != nil {
		return fmt.Errorf("seed station: %w", err)
	}

	var stationCode models.StationCode
	if err := db.
		Where(models.StationCode{Code: "TWL-01"}).
		Attrs(models.StationCode{StationId: station.ID, CompanyId: strptr(company.ID)}).
		FirstOrCreate(&stationCode).Error; err != nil {
		return fmt.Errorf("seed station code: %w", err)
	}

	// Platform has no natural unique column; match on its owning station + number.
	var platform models.Platform
	if err := db.
		Where("station_id = ? AND platform_number = ?", station.ID, "1").
		Attrs(models.Platform{StationId: station.ID, Name: strptr("Platform 1"), PlatformNumber: strptr("1")}).
		FirstOrCreate(&platform).Error; err != nil {
		return fmt.Errorf("seed platform: %w", err)
	}

	log.Printf("E2E seed ready: company=%q station=%q code=%q platform=%q",
		company.Name, station.Name, stationCode.Code, "Platform 1")
	return nil
}
