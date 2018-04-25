package store_test

import (
	"github.com/abbot/go-http-auth"
	"github.com/dmgk/faker"
	"github.com/jinzhu/gorm"
	"github.com/readium/readium-lcp-server/store"
	"math/rand"
	"strconv"
	"testing"
	"time"
)

func TestUserStore_Add(t *testing.T) {
	r := rand.New(rand.NewSource(int64(time.Now().Unix())))
	salt := []byte(strconv.Itoa(r.Int()))
	magic := []byte("$" + "$")
	// create 100 different users. Password is the same as email
	for i := 0; i < 100; i++ {
		user := &store.User{}
		user.Email = faker.Internet().Email()
		user.Name = faker.Name().Name()
		user.Hint = faker.Name().FirstName()
		user.Password = string(auth.MD5Crypt([]byte(user.Email), salt, magic))
		err := stor.User().Add(user)
		if err != nil {
			t.Fatalf("Error creating user : %v", err)
		}
	}
	// create a user that we can find by email
	user := &store.User{}
	user.Email = "antone@bashirian.name"
	user.Name = faker.Name().Name()
	user.Hint = faker.Name().FirstName()
	user.Password = string(auth.MD5Crypt([]byte(user.Email), salt, magic))
	err := stor.User().Add(user)
	if err != nil {
		t.Fatalf("Error creating user : %v", err)
	}
}

func TestUserStore_ListUsers(t *testing.T) {
	users, err := stor.User().List(30, 0)
	if err != nil {
		t.Fatalf("Error retrieving users : %v", err)
	}
	for idx, user := range users {
		t.Logf("%d. %#v", idx, user)
	}

	users, err = stor.User().List(50, 1)
	if err != nil {
		t.Fatalf("Error retrieving users : %v", err)
	}
	for idx, user := range users {
		t.Logf("%d. %#v", idx, user)
	}
}

func TestUserStore_Get(t *testing.T) {
	user, err := stor.User().Get(1)
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			t.Fatalf("Error retrieving user by id : %v", err)
		} else {
			t.Skipf("You forgot to create user with id = 1 ?")
		}
	}
	t.Logf("Found user by id : %#v", user)
}

func TestUserStore_GetByEmail(t *testing.T) {
	user, err := stor.User().GetByEmail("antone@bashirian.name")
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			t.Fatalf("Error retrieving user by email : %v", err)
		} else {
			t.Skipf("You forgot to create user with email `antone@bashirian.name` ?")
		}
	}
	t.Logf("Found user by email : %#v", user)
}

func TestUserStore_Update(t *testing.T) {
	user, err := stor.User().GetByEmail("antone@bashirian.name")
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			t.Fatalf("Error retrieving user by email : %v", err)
		} else {
			t.Skipf("You forgot to create user with email `antone@bashirian.name` ?")
		}
	}
	user.Name = "Updated user"
	user.Hint = "Updated hint"
	err = stor.User().Update(user)
	if err != nil {
		t.Fatalf("Error updating user by email : %v", err)
	}
	// reading it once again
	updatedUser, err := stor.User().GetByEmail("antone@bashirian.name")
	if err != nil {
		t.Fatalf("Error retrieving user by email : %v", err)
	}
	if updatedUser.Name != "Updated user" || updatedUser.Hint != "Updated hint" || updatedUser.Password != user.Password {
		t.Fatal("Error : updated user is incorrect!\ngot %#v\nwant %#v", updatedUser, user)
	}
	t.Logf("Updated user : %#v", updatedUser)
}

func TestUserStore_DeleteUser(t *testing.T) {
	users, err := stor.User().List(1, 0)
	if err != nil {
		t.Fatalf("Error retrieving users : %v", err)
	}
	if len(users) != 1 {
		t.Fatalf("No users found. Create some users.")
	}
	// deleting the first user
	err = stor.User().Delete(users[0].ID)
	if err != nil {
		t.Fatalf("Error deleting user : %v", err)
	}
	t.Log("User deleted : %#v", users[0])
}
