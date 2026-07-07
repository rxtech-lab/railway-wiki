package testutil

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/rxtech-lab/railway-wiki/internal/database"
	"github.com/stretchr/testify/suite"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// DBTestSuite is a test suite that provides a clean database for each test
// Usage:
//
//	type MyTestSuite struct {
//	    testutil.DBTestSuite
//	}
//
//	func TestMyTestSuite(t *testing.T) {
//	    suite.Run(t, new(MyTestSuite))
//	}
type DBTestSuite struct {
	suite.Suite
	DB           *gorm.DB
	DatabaseName string
}

// SetupTest runs before each test in the suite
func (s *DBTestSuite) SetupTest() {
	// Create a unique database name for this test
	dbName := fmt.Sprintf("test_db_%d_%d", time.Now().Unix(), rand.Intn(10000))
	s.DatabaseName = dbName

	// Create the test database
	err := s.createTestDatabase(dbName)
	s.Require().NoError(err)

	// Connect to test database
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		"localhost", 5432, "postgres", "postgres", dbName, "disable")

	db, err := database.NewConnection(dsn)
	s.Require().NoError(err)
	s.DB = db
}

// TearDownTest runs after each test in the suite
func (s *DBTestSuite) TearDownTest() {
	// Close the current database connection first
	if s.DB != nil {
		if sqlDB, err := s.DB.DB(); err == nil {
			sqlDB.Close()
		}
	}

	// Drop the test database
	s.dropTestDatabase(s.DatabaseName)
}

// createTestDatabase creates the test database
func (s *DBTestSuite) createTestDatabase(dbName string) error {
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		"localhost", 5432, "postgres", "postgres", "postgres", "disable")

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return fmt.Errorf("failed to connect to postgres: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get sql.DB: %w", err)
	}
	defer sqlDB.Close()

	// Create test database
	err = db.Exec(fmt.Sprintf("CREATE DATABASE %s", dbName)).Error
	if err != nil {
		return fmt.Errorf("failed to create test database: %w", err)
	}

	return nil
}

// dropTestDatabase drops the test database
func (s *DBTestSuite) dropTestDatabase(dbName string) {
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		"localhost", 5432, "postgres", "postgres", "postgres", "disable")

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return
	}

	sqlDB, err := db.DB()
	if err != nil {
		return
	}
	defer sqlDB.Close()

	db.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s", dbName))
}
