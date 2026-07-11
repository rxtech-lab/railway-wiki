// Package testutil provides helpers for tests, notably an isolated in-memory
// libSQL/SQLite database per test.
package testutil

import (
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/rxtech-lab/railway-wiki/internal/models"
)

// DBTestSuite provides a clean in-memory database for each test.
//
//	type MyTestSuite struct{ testutil.DBTestSuite }
//	func TestMyTestSuite(t *testing.T) { suite.Run(t, new(MyTestSuite)) }
type DBTestSuite struct {
	suite.Suite
	DB *gorm.DB
}

// SetupTest opens a fresh in-memory database and runs migrations.
func (s *DBTestSuite) SetupTest() {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	s.Require().NoError(err)

	// A single connection keeps the in-memory database alive for the test.
	sqlDB, err := db.DB()
	s.Require().NoError(err)
	sqlDB.SetMaxOpenConns(1)

	s.Require().NoError(db.AutoMigrate(models.AllModels...))
	s.DB = db
}

// TearDownTest closes the database connection.
func (s *DBTestSuite) TearDownTest() {
	if s.DB != nil {
		if sqlDB, err := s.DB.DB(); err == nil {
			_ = sqlDB.Close()
		}
	}
}
