package adapters

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/h44z/wg-portal/internal/model"
	"gorm.io/driver/sqlite"

	"gorm.io/gorm"
)

func testDB() *gorm.DB {
	_ = os.Remove("wg_unittest.db")
	gormDb, err := gorm.Open(sqlite.Open("wg_unittest.db"), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
	if err != nil {
		panic(err)
	}
	_ = gormDb.AutoMigrate(&model.Interface{}, &model.User{})
	_ = gormDb.AutoMigrate(&model.Peer{})

	return gormDb
}

func Test_sqlRepo_SaveInterface(t *testing.T) {
	db := testDB()

	repo := NewSqlRepository(db)
	err := repo.SaveInterface(context.Background(), "wg0", func(in *model.Interface) (*model.Interface, error) {
		in.Addresses = append(in.Addresses, model.MustCidrFromString("1.1.1.2/24"))
		in.Addresses = append(in.Addresses, model.MustCidrFromString("fe80::1/64"))
		in.Mtu = 1500

		return in, nil
	})
	assert.NoError(t, err)
}
