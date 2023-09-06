package db

import (
	"fmt"
	"os"

	"github.com/rs/xid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type database struct {
	db *gorm.DB
}

// DB is the object
var DB database

func InitDB() {
	dbURL := os.Getenv("DATABASE_URL")
	fmt.Printf("db url : %v", dbURL)

	if dbURL == "" {
		rdsHost := os.Getenv("RDS_HOSTNAME")
		rdsPort := os.Getenv("RDS_PORT")
		rdsDbName := os.Getenv("RDS_DB_NAME")
		rdsUsername := os.Getenv("RDS_USERNAME")
		rdsPassword := os.Getenv("RDS_PASSWORD")
		dbURL = fmt.Sprintf("postgres://%s:%s@%s:%s/%s", rdsUsername, rdsPassword, rdsHost, rdsPort, rdsDbName)
	}

	if dbURL == "" {
		panic("DB env vars not found")
	}

	var err error

	db, err := gorm.Open(postgres.New(postgres.Config{
		DSN:                  dbURL,
		PreferSimpleProtocol: true,
	}), &gorm.Config{})

	if err != nil {
		panic(err)
	}

	DB.db = db

	fmt.Println("db connected")

	// migrate table changes
	db.AutoMigrate(&Person{})
	db.AutoMigrate(&Channel{})
	db.AutoMigrate(&LeaderBoard{})
	db.AutoMigrate(&ConnectionCodes{})
	db.AutoMigrate(&Bounty{})
	db.AutoMigrate(&Organization{})
	db.AutoMigrate(&OrganizationUsers{})
	db.AutoMigrate(&BountyRoles{})
	db.AutoMigrate(&UserRoles{})
	db.AutoMigrate(&BountyBudget{})
	db.AutoMigrate(&BudgetHistory{})
	db.AutoMigrate(&PaymentHistory{})

	people := DB.GetAllPeople()
	for _, p := range people {
		if p.Uuid == "" {
			DB.AddUuidToPerson(p.ID, xid.New().String())
		}
	}

}

const (
	AddBounty      = "ADD BOUNTY"
	UpdateBounty   = "UPDATE BOUNTY"
	DeleteBounty   = "DELETE BOUNTY"
	PayBounty      = "PAY BOUNTY"
	AddUser        = "ADD USER"
	UpdateUser     = "UPDATE USER"
	DeleteUser     = "DELETE USER"
	AddRoles       = "ADD ROLES"
	AddBudget      = "ADD BUDGET"
	WithdrawBudget = "WITHDRAW BUDGET"
	ViewReport     = "VIEW REPORT"
)

var ConfigBountyRoles []BountyRoles = []BountyRoles{
	{
		Name: AddBounty,
	},
	{
		Name: UpdateBounty,
	},
	{
		Name: DeleteBounty,
	},
	{
		Name: PayBounty,
	},
	{
		Name: AddUser,
	},
	{
		Name: UpdateUser,
	},
	{
		Name: DeleteUser,
	},
	{
		Name: AddRoles,
	},
	{
		Name: AddBudget,
	},
	{
		Name: WithdrawBudget,
	},
	{
		Name: ViewReport,
	},
}

var Updatables = []string{
	"name", "description", "tags", "img",
	"owner_alias", "price_to_join", "price_per_message",
	"escrow_amount", "escrow_millis",
	"unlisted", "private", "deleted",
	"app_url", "bots", "feed_url", "feed_type",
	"owner_route_hint", "updated", "pin",
	"profile_filters",
}
var Botupdatables = []string{
	"name", "description", "tags", "img",
	"owner_alias", "price_per_use",
	"unlisted", "deleted",
	"owner_route_hint", "updated",
}
var Peopleupdatables = []string{
	"description", "tags", "img",
	"owner_alias",
	"unlisted", "deleted",
	"owner_route_hint",
	"price_to_meet", "updated",
	"extras",
}
var Channelupdatables = []string{
	"name", "deleted"}

func (db database) GetRolesCount() int64 {
	var count int64
	query := db.db.Model(&BountyRoles{})

	query.Count(&count)
	return count
}

func (db database) CreateRoles() {
	db.db.Create(&ConfigBountyRoles)
}

func (db database) DeleteRoles() {
	db.db.Exec("DELETE FROM bounty_roles")
}

func InitRoles() {
	count := DB.GetRolesCount()
	if count != int64(len(ConfigBountyRoles)) {
		// delete all the roles and insert again
		if count != 0 {
			DB.DeleteRoles()
		}
		DB.CreateRoles()
	}
}

func GetRolesMap() map[string]string {
	roles := map[string]string{}
	for _, v := range ConfigBountyRoles {
		roles[v.Name] = v.Name
	}
	return roles
}

func GetUserRolesMap(userRoles []UserRoles) map[string]string {
	roles := map[string]string{}
	for _, v := range userRoles {
		roles[v.Role] = v.Role
	}
	return roles
}

func RolesCheck(userRoles []UserRoles, check string) bool {
	hasRole := false
	rolesMap := GetRolesMap()
	userRolesMap := GetUserRolesMap(userRoles)

	// check if roles exists in config
	_, ok := rolesMap[check]
	_, ok1 := userRolesMap[check]

	// if any of the roles does not exists return false
	// if any of the roles does not exists user roles return false
	if !ok {
		hasRole = false
		return hasRole
	} else if !ok1 {
		hasRole = false
		return hasRole
	}

	hasRole = true
	return hasRole
}

func CheckUser(userRoles []UserRoles, pubkey string) bool {
	isUser := false
	for _, role := range userRoles {
		if role.OwnerPubKey == pubkey {
			isUser = true
			return isUser
		}
	}

	return isUser
}