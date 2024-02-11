package go_gorm

import (
	"context"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
)

func OpenConnection() *gorm.DB {
	dialect := mysql.Open("root@tcp(127.0.0.1:3306)/go_gorm?charset=utf8mb4&parseTime=True&loc=Local")
	db, err := gorm.Open(dialect, &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
		// tuning performance
		// SkipDefaultTransaction: true, //
		PrepareStmt: true,
	})
	if err != nil {
		panic(err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		panic(err)
	}

	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetConnMaxLifetime(30 * time.Minute)
	sqlDB.SetConnMaxIdleTime(5 * time.Minute)

	return db
}

var db = OpenConnection()

func TestOpenConnection(t *testing.T) {
	assert.NotNil(t, db)
}

func TestExecuteSQL(t *testing.T) {
	err := db.Exec("insert into sample(id, name) values(?,?)", "1", "rafli").Error
	assert.Nil(t, err)

	err = db.Exec("insert into sample(id, name) values(?,?)", "2", "muin").Error
	assert.Nil(t, err)

	err = db.Exec("insert into sample(id, name) values(?,?)", "3", "zaky").Error
	assert.Nil(t, err)

	err = db.Exec("insert into sample(id, name) values(?,?)", "4", "muhammad").Error
	assert.Nil(t, err)
}

type Sample struct {
	Id   string
	Name string
}

func TestRawSQL(t *testing.T) {
	var sample Sample
	err := db.Raw("select id, name from sample where id = ?", "1").Scan(&sample).Error
	assert.Nil(t, err)
	assert.Equal(t, "1", sample.Id)

	var samples []Sample
	err = db.Raw("select id, name from sample").Scan(&samples).Error
	assert.Nil(t, err)
	assert.Equal(t, 4, len(samples))
}

func TestRows(t *testing.T) {
	rows, err := db.Raw("select id, name from sample").Rows()
	assert.Nil(t, err)
	defer rows.Close()

	var samples []Sample
	for rows.Next() {
		var id string
		var name string

		err := rows.Scan(&id, &name)
		assert.Nil(t, err)

		samples = append(samples, Sample{
			Id:   id,
			Name: name,
		})
	}

	assert.Equal(t, 4, len(samples))
}

func TestScanRows(t *testing.T) {
	rows, err := db.Raw("select id, name from sample").Rows()
	assert.Nil(t, err)
	defer rows.Close()

	var samples []Sample
	for rows.Next() {
		err := db.ScanRows(rows, &samples)
		assert.Nil(t, err)
	}

	assert.Equal(t, 4, len(samples))
}

func TestCreateUser(t *testing.T) {
	user := User{
		ID:       "1",
		Password: "rahasia",
		Name: Name{
			FirstName:  "muhammad",
			MiddleName: "rafli",
			LastName:   "real",
		},
		Information: "ini akan di ignore",
	}

	response := db.Create(&user)
	assert.Nil(t, response.Error)
	assert.Equal(t, 1, int(response.RowsAffected))
}

func TestBatchInsert(t *testing.T) {
	var users []User
	for i := 1; i <= 2; i++ {
		users = append(users, User{
			ID: strconv.Itoa(i),
			Name: Name{
				FirstName: "User " + strconv.Itoa(i),
			},
			Password: "rahasia",
		})
	}

	result := db.Create(&users)
	assert.Nil(t, result.Error)
	assert.Equal(t, 8, int(result.RowsAffected))
}

func TestTransactionSuccess(t *testing.T) {
	err := db.Transaction(func(tx *gorm.DB) error {
		err := tx.Create(&User{ID: "10", Password: "rahasia", Name: Name{FirstName: "User 10"}}).Error
		if err != nil {
			return err
		}

		err = tx.Create(&User{ID: "11", Password: "rahasia", Name: Name{FirstName: "User 11"}}).Error
		if err != nil {
			return err
		}

		err = tx.Create(&User{ID: "12", Password: "rahasia", Name: Name{FirstName: "User 12"}}).Error
		if err != nil {
			return err
		}

		return nil
	})

	assert.Nil(t, err)
}

func TestTransactionError(t *testing.T) {
	err := db.Transaction(func(tx *gorm.DB) error {
		err := tx.Create(&User{ID: "13", Password: "rahasia", Name: Name{FirstName: "User 13"}}).Error
		if err != nil {
			return err
		}

		err = tx.Create(&User{ID: "11", Password: "rahasia", Name: Name{FirstName: "User 11"}}).Error
		if err != nil {
			return err
		}

		return nil
	})

	assert.Nil(t, err)
}

func TestManualTransactionSuccess(t *testing.T) {
	tx := db.Begin()
	defer tx.Rollback()

	err := tx.Create(&User{ID: "15", Password: "rahasia", Name: Name{FirstName: "User 15"}}).Error
	assert.Nil(t, err)

	err = tx.Create(&User{ID: "14", Password: "rahasia", Name: Name{FirstName: "User 14"}}).Error
	assert.Nil(t, err)

	if err == nil {
		tx.Commit()
	}
}

func TestQuerySingleObject(t *testing.T) {
	user := User{}
	result := db.First(&user)
	assert.Nil(t, result.Error)
	assert.Equal(t, "1", user.ID)

	user = User{}
	result = db.Last(&user)
	assert.Nil(t, result.Error)
	assert.Equal(t, "9", user.ID)
}

func TestQuerySingleObjectInlineCondition(t *testing.T) {
	user := User{}
	err := db.Take(&user, "id = ?", "5").Error
	assert.Nil(t, err)
	assert.Equal(t, "5", user.ID)
	assert.Equal(t, "User 5", user.Name.FirstName)
}

func TestQueryAllObjects(t *testing.T) {
	var users []User
	result := db.Find(&users, "id in ?", []string{"1", "2", "3", "4"})
	assert.Nil(t, result.Error)
	assert.Equal(t, 4, len(users))
}

func TestQueryCondition(t *testing.T) {
	var users []User
	err := db.Where("first_name like ?", "%User %").Where("password = ?", "rahasia").Find(&users).Error
	assert.Nil(t, err)
	assert.Equal(t, 9, len(users))
}

func TestOrOperator(t *testing.T) {
	var users []User
	err := db.Where("first_name like ?", "%user %").Or("password = ?", "rahasia").Find(&users).Error
	assert.Nil(t, err)
	assert.Equal(t, 14, len(users))
}

func TestNotOperator(t *testing.T) {
	var users []User
	err := db.Not("first_name like ?", "%user %").Or("password = ?", "rahasia").Find(&users).Error
	assert.Nil(t, err)
	assert.Equal(t, 14, len(users))
}

func TestSelectField(t *testing.T) {
	var users []User
	err := db.Select("id", "first_name").Find(&users).Error
	assert.Nil(t, err)

	for _, user := range users {
		assert.NotNil(t, user.ID)
		assert.NotEqual(t, "", user.Name.FirstName)
	}

	assert.Equal(t, 14, len(users))
}

func TestStructCondition(t *testing.T) {
	userCondition := User{
		Name: Name{
			FirstName: "User 5",
		},
		Password: "rahasia",
	}

	var users []User
	err := db.Where(userCondition).Find(&users).Error
	assert.Nil(t, err)
	assert.Equal(t, 1, len(users))
}

func TestMapCondition(t *testing.T) {
	mapCondition := map[string]interface{}{
		"middle_name": "",
		"last_name":   "",
	}

	var users []User
	result := db.Where(mapCondition).Find(&users)
	assert.Nil(t, result.Error)
	assert.Equal(t, 13, len(users))
}

func TestOrderLimitOffset(t *testing.T) {
	var users []User
	result := db.Order("id asc, first_name desc").Limit(5).Offset(5).Find(&users)
	assert.Nil(t, result.Error)
	assert.Equal(t, 5, len(users))
}

type UserResponse struct {
	ID        string
	FirstName string
	LastName  string
}

func TestQueryNonModel(t *testing.T) {
	var users []UserResponse
	result := db.Model(&User{}).Select("id", "first_name", "last_name").Find(&users)
	assert.Nil(t, result.Error)
	assert.Equal(t, 14, len(users))
}

func TestUpdate(t *testing.T) {
	user := User{}
	err := db.Take(&user, "id = ?", "1").Error
	assert.Nil(t, err)

	user.Name.FirstName = "Eko"
	user.Name.MiddleName = ""
	user.Name.LastName = "Khannedy"
	user.Password = "rahasiapalsu"

	err = db.Save(&user).Error
	assert.Nil(t, err)
}

func TestUpdateSelectedColumns(t *testing.T) {
	err := db.Model(&User{}).Where("id = ?", "1").Updates(map[string]interface{}{
		"middle_name": "",
		"last_name":   "Morro",
	}).Error
	assert.Nil(t, err)

	err = db.Model(&User{}).Where("id = ?", "1").Update("password", "diubahlagi").Error
	assert.Nil(t, err)

	err = db.Where("id = ?", "1").Updates(User{
		Name: Name{
			FirstName: "Lord",
			LastName:  "",
		},
	}).Error
	assert.Nil(t, err)
}

func TestAutoIncrement(t *testing.T) {
	for i := 0; i < 10; i++ {
		userLogs := UserLogs{
			UserId: "1",
			Action: "Test Action",
		}
		result := db.Create(&userLogs)
		assert.Nil(t, result.Error)
		assert.NotEqual(t, 0, userLogs.ID)
		fmt.Println(userLogs.ID)
	}
}

func TestSaveOrUpdate(t *testing.T) {
	userLog := UserLogs{
		UserId: "1",
		Action: "Test Action",
	}
	result := db.Save(&userLog) // create
	assert.Nil(t, result.Error)

	userLog.UserId = "2"
	result = db.Save(&userLog) // update
	assert.Nil(t, result.Error)
}

func TestSaveOrUpdateNonAutoIncrement(t *testing.T) {
	user := User{
		ID: "99",
		Name: Name{
			FirstName: "User 99",
		},
	}

	err := db.Save(&user).Error // create
	assert.Nil(t, err)

	user.Name.FirstName = "User 99 Updated"
	err = db.Save(&user).Error // update
	assert.Nil(t, err)
}

func TestConflict(t *testing.T) {
	user := User{
		ID: "88",
		Name: Name{
			FirstName: "User 88",
		},
	}

	result := db.Clauses(clause.OnConflict{
		UpdateAll: true,
	}).Create(&user) // create

	assert.Nil(t, result.Error)
}

func TestDelete(t *testing.T) {
	var user User

	result := db.First(&user, "id = ?", "88")
	assert.Nil(t, result.Error)
	result = db.Delete(&user)
	assert.Nil(t, result.Error)

	result = db.Delete(&User{}, "id = ?", "99")
	assert.Nil(t, result.Error)

	result = db.Where("id = ?", "77").Delete(&User{})
	assert.Nil(t, result.Error)
}

func TestSoftDelete(t *testing.T) {
	todo := Todo{
		UserId:      "1",
		Title:       "Todo 1",
		Description: "Isi Todo 1",
	}
	result := db.Create(&todo)
	assert.Nil(t, result.Error)

	result = db.Delete(&todo)
	assert.Nil(t, result.Error)
	assert.NotNil(t, todo.DeletedAt)

	var todos []Todo
	result = db.Find(&todos)
	assert.Nil(t, result.Error)
	assert.Equal(t, 0, len(todos))
}

func TestUnscoped(t *testing.T) {
	var todo Todo
	result := db.Unscoped().First(&todo, "id = ?", "2")
	assert.Nil(t, result.Error)

	result = db.Unscoped().Delete(&todo)
	assert.Nil(t, result.Error)

	var todos []Todo
	result = db.Unscoped().Find(&todos)
	assert.Nil(t, result.Error)
	assert.Equal(t, 0, len(todos))
}

func TestLock(t *testing.T) {
	err := db.Transaction(func(tx *gorm.DB) error {
		var user User
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&user, "id = ?", "1").Error
		if err != nil {
			return err
		}

		user.Name.FirstName = "Joko"
		user.Name.LastName = "Morro"
		return tx.Save(&user).Error
	})

	assert.Nil(t, err)
}

func TestCreateWallet(t *testing.T) {
	wallet := Wallet{
		ID:      "1",
		UserId:  "1",
		Balance: 1000000,
	}
	err := db.Create(&wallet).Error
	assert.Nil(t, err)
}

func TestRetrieveRelation(t *testing.T) {
	var user User
	err := db.Model(&User{}).Preload("Wallet").First(&user).Error
	assert.Nil(t, err)

	assert.Equal(t, "1", user.ID)
	assert.Equal(t, "1", user.Wallet.ID)
}

func TestRetrieveRelationJoin(t *testing.T) {
	var user User
	err := db.Model(&User{}).Joins("Wallet").Take(&user, "users.id = ?", "1").Error
	assert.Nil(t, err)

	assert.Equal(t, "1", user.ID)
	assert.Equal(t, "1", user.Wallet.ID)
}

func TestAutoCreateUpdate(t *testing.T) {
	user := User{
		ID:       "20",
		Password: "rahasia",
		Name: Name{
			FirstName: "User 20",
		},
		Wallet: Wallet{
			ID:      "20",
			UserId:  "20",
			Balance: 1000000,
		},
	}

	err := db.Create(&user).Error
	assert.Nil(t, err)
}

func TestSkipAutoCreateUpdate(t *testing.T) {
	user := User{
		ID:       "21",
		Password: "rahasia",
		Name: Name{
			FirstName: "User 21",
		},
		Wallet: Wallet{
			ID:      "21",
			UserId:  "21",
			Balance: 1000000,
		},
	}
	err := db.Omit(clause.Associations).Create(&user).Error
	assert.Nil(t, err)
}

func TestUserAndAddresses(t *testing.T) {
	user := User{
		ID:       "2",
		Password: "rahasia",
		Name: Name{
			FirstName: "User 50",
		},
		Wallet: Wallet{
			ID:      "2",
			UserId:  "2",
			Balance: 10000000,
		},
		Addresses: []Address{
			{
				UserId:  "2",
				Address: "Jalan A",
			},
			{
				UserId:  "2",
				Address: "Jalan B",
			},
		},
	}

	err := db.Save(&user).Error
	assert.Nil(t, err)
}

func TestPreloadJoinOneToMany(t *testing.T) {
	var users []User
	err := db.Model(&User{}).Preload("Addresses").Joins("Wallet").Find(&users).Error
	assert.Nil(t, err)
}

func TestTakePreloadJoinOneToMany(t *testing.T) {
	var user []User
	err := db.Model(&User{}).Preload("Addresses").Joins("Wallet").
		Take(&user, "users.id = ?", "50").Error
	assert.Nil(t, err)
}

func TestBelongsTo(t *testing.T) {
	fmt.Println("Preload")
	var addresses []Address
	err := db.Preload("User").Find(&addresses).Error
	assert.Nil(t, err)
	assert.Equal(t, 12, len(addresses))

	fmt.Println("Joins")
	addresses = []Address{}
	err = db.Joins("User").Find(&addresses).Error
	assert.Nil(t, err)
	assert.Equal(t, 12, len(addresses))
}

func TestBelongsToWallet(t *testing.T) {
	fmt.Println("Preload")
	var wallets []Wallet
	err := db.Preload("User").Find(&wallets).Error
	assert.Nil(t, err)

	fmt.Println("Joins")
	wallets = []Wallet{}
	err = db.Joins("User").Find(&wallets).Error
	assert.Nil(t, err)
}

func TestCreateManyToMany(t *testing.T) {
	product := Product{
		ID:    "P001",
		Name:  "Contoh Product",
		Price: 10000000,
	}
	err := db.Create(&product).Error
	assert.Nil(t, err)

	err = db.Table("user_like_product").Create(map[string]interface{}{
		"user_id":    "1",
		"product_id": "P001",
	}).Error
	assert.Nil(t, err)

	err = db.Table("user_like_product").Create(map[string]interface{}{
		"user_id":    "2",
		"product_id": "P001",
	}).Error
	assert.Nil(t, err)
}

func TestPreloadManyToMany(t *testing.T) {
	var product Product
	err := db.Preload("LikedByUsers").First(&product, "id = ?", "P001").Error
	assert.Nil(t, err)
	assert.Equal(t, 2, len(product.LikedByUsers))
}

func TestAssociationFind(t *testing.T) {
	var product Product
	err := db.First(&product, "id = ?", "P001").Error
	assert.Nil(t, err)

	var users []User
	err = db.Model(&product).Where("users.first_name LIKE ?", "User%").Association("LikedByUsers").Find(&users)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(users))
	fmt.Println(users)
}

func TestAssociationAppend(t *testing.T) {
	var user User
	err := db.First(&user, "id = ?", "3").Error
	assert.Nil(t, err)

	var product Product
	err = db.First(&product, "id = ?", "P001").Error
	assert.Nil(t, err)

	err = db.Model(&product).Association("LikedByUsers").Append(&user)
	assert.Nil(t, err)
}

func TestAssociationReplace(t *testing.T) {
	err := db.Transaction(func(tx *gorm.DB) error {
		var user User
		err := tx.First(&user, "id = ?", "1").Error
		assert.Nil(t, err)

		wallet := Wallet{
			ID:      "01",
			UserId:  user.ID,
			Balance: 10000000,
		}
		err = tx.Model(&user).Association("Wallet").Replace(&wallet)
		return err
	})
	assert.Nil(t, err)
}

func TestAssociationDelete(t *testing.T) {
	var user User
	err := db.First(&user, "id = ?", "3").Error
	assert.Nil(t, err)

	var product Product
	err = db.First(&product, "id = ?", "P001").Error
	assert.Nil(t, err)

	err = db.Model(&product).Association("LikedByUsers").Delete(&user)
	assert.Nil(t, err)
}

func TestAssociationClear(t *testing.T) {
	var user User
	err := db.First(&user, "id = ?", "3").Error
	assert.Nil(t, err)

	var product Product
	err = db.First(&product, "id = ?", "P001").Error
	assert.Nil(t, err)

	err = db.Model(&product).Association("LikedByUsers").Clear()
	assert.Nil(t, err)
}

func TestPreloadingWithCondition(t *testing.T) {
	var user User
	err := db.Preload("Wallet", "balance > ?", 100).Take(&user, "id = ?", "1").Error
	assert.Nil(t, err)
	fmt.Println(user)
}

func TestNestedPreloading(t *testing.T) {
	var wallet Wallet
	err := db.Preload("User.Addresses").Find(&wallet, "wallets.id = ?", "2").Error
	assert.Nil(t, err)
	fmt.Println(wallet.User.Addresses)
}

func TestPreloadAll(t *testing.T) {
	var user User
	err := db.Preload(clause.Associations).Take(&user, "id = ?", "1").Error
	assert.Nil(t, err)
	fmt.Println(user)
}

func TestJoinQuery(t *testing.T) {
	var users []User
	err := db.Joins("join wallets on wallets.user_id = users.id").Find(&users).Error
	assert.Nil(t, err)
	assert.Equal(t, 4, len(users))

	users = []User{}
	err = db.Joins("Wallet").Find(&users).Error // using leftjoin
	assert.Nil(t, err)
	assert.Equal(t, 16, len(users))
}

func TestJoinQueryCondition(t *testing.T) {
	var users []User
	err := db.Joins("join wallets on wallets.user_id = users.id AND wallets.balance > ?", 500000).Find(&users).Error
	assert.Nil(t, err)
	assert.Equal(t, 4, len(users))

	users = []User{}
	err = db.Joins("Wallet").Where("Wallet.balance > ?", 500000).Find(&users).Error
	assert.Nil(t, err)
	assert.Equal(t, 4, len(users))
}

func TestCount(t *testing.T) {
	var count int64
	err := db.Model(&User{}).Joins("Wallet").Where("Wallet.balance > ?", 500000).Count(&count).Error
	assert.Nil(t, err)
	assert.Equal(t, int64(4), count)
}

type AggregationResult struct {
	TotalBalance int64
	MinBalance   int64
	MaxBalance   int64
	AvgBalance   float64
}

func TestAggregation(t *testing.T) {
	var result AggregationResult
	err := db.Model(&Wallet{}).Select("sum(balance) as total_balance", "min(balance) as min_balance", "max(balance) as max_balance", "avg(balance) as avg_balance").
		Take(&result).Error

	assert.Nil(t, err)
	assert.Equal(t, int64(22000000), result.TotalBalance)
	assert.Equal(t, int64(1000000), result.MinBalance)
	assert.Equal(t, int64(10000000), result.MaxBalance)
	assert.Equal(t, float64(5500000), result.AvgBalance)
}

func TestAggregationGroupByHaving(t *testing.T) {
	var result []AggregationResult
	err := db.Model(&Wallet{}).Select("sum(balance) as total_balance", "min(balance) as min_balance", "max(balance) as max_balance", "avg(balance) as avg_balance").
		Joins("User").Group("User.id").Having("sum(balance) > ?", 1000000).
		Find(&result).Error

	assert.Nil(t, err)
	assert.Equal(t, 2, len(result))
}

func TestContext(t *testing.T) {
	ctx := context.Background()

	var users []User
	err := db.WithContext(ctx).Find(&users).Error
	assert.Nil(t, err)
	assert.Equal(t, 16, len(users))
}

// scope
func BrokeWalletBalance(db *gorm.DB) *gorm.DB {
	return db.Where("balance = ?", 0)
}

func SultanWalletBalance(db *gorm.DB) *gorm.DB {
	return db.Where("balance > ?", 1000000)
}

func TestScopes(t *testing.T) {
	var wallets []Wallet
	err := db.Scopes(BrokeWalletBalance).Find(&wallets).Error
	assert.Nil(t, err)

	wallets = []Wallet{}
	err = db.Scopes(SultanWalletBalance).Find(&wallets).Error
	assert.Nil(t, err)
}

func TestMigrator(t *testing.T) {
	err := db.Migrator().AutoMigrate(&GuestBook{})
	assert.Nil(t, err)
}

func TestHook(t *testing.T) {
	user := User{
		Password: "secret",
		Name: Name{
			FirstName: "User 101",
		},
	}

	err := db.Create(&user).Error
	assert.Nil(t, err)
	assert.NotEqual(t, "", user.ID)

	fmt.Println(user.ID)
}
