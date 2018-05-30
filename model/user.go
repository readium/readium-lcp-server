/*
 * Copyright (c) 2016-2018 Readium Foundation
 *
 * Redistribution and use in source and binary forms, with or without modification,
 * are permitted provided that the following conditions are met:
 *
 *  1. Redistributions of source code must retain the above copyright notice, this
 *     list of conditions and the following disclaimer.
 *  2. Redistributions in binary form must reproduce the above copyright notice,
 *     this list of conditions and the following disclaimer in the documentation and/or
 *     other materials provided with the distribution.
 *  3. Neither the name of the organization nor the names of its contributors may be
 *     used to endorse or promote products derived from this software without specific
 *     prior written permission
 *
 *  THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND
 *  ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
 *  WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
 *  DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT OWNER OR CONTRIBUTORS BE LIABLE FOR
 *  ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
 *  (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
 *  LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND
 *  ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
 *  (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
 *  SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
 */

package model

import (
	"reflect"
	"strings"
)

type (
	UsersCollection []*User
	User            struct {
		ID        int64    `json:"id" sql:"AUTO_INCREMENT" gorm:"primary_key"`
		UUID      string   `json:"uuid" sql:"NOT NULL" gorm:"size:36"` // uuid - max size 36
		Name      string   `json:"name,omitempty"  gorm:"size:64"`
		Email     string   `json:"email,omitempty"  gorm:"size:64"`
		Password  string   `json:"password,omitempty"  gorm:"size:64"`
		Hint      string   `json:"hint"  gorm:"size:64"`
		Encrypted []string `json:"encrypted,omitempty" gorm:"-"` // TODO : never used. is this work in progress?
	}
)

func (u *User) getField(field string) reflect.Value {
	value := reflect.ValueOf(u).Elem()
	return value.FieldByName(strings.Title(field))
}

// Implementation of gorm Tabler
func (u *User) TableName() string {
	return LUTUserTableName
}

// Implementation of GORM callback
func (u *User) BeforeSave() error {
	if u.ID == 0 {
		// Create uuid for new user
		uid, errU := NewUUID()
		if errU != nil {
			return errU
		}
		u.UUID = uid.String()
	}
	return nil
}

func (s userStore) List(page int, pageNum int) (UsersCollection, error) {
	var result UsersCollection
	return result, s.db.Offset(pageNum * page).Limit(page).Order("email DESC").Find(&result).Error
}

func (s userStore) Get(id int64) (*User, error) {
	var result User
	return &result, s.db.Where(User{ID: id}).Find(&result).Error
}

func (s userStore) GetByEmail(email string) (*User, error) {
	var user User
	return &user, s.db.Where(User{Email: email}).Find(&user).Error
}

func (s userStore) Add(newUser *User) error {
	return s.db.Create(newUser).Error
}

func (s userStore) Update(changedUser *User) error {
	return s.db.Save(changedUser).Error
}

func (s userStore) Delete(userID int64) error {
	result := Transaction(s.db, func(tx txStore) error {
		err := tx.Where("user_id = ?", userID).Delete(Purchase{}).Error
		if err != nil {
			return err
		}
		return tx.Where("id = ?", userID).Delete(User{}).Error
	})
	return result
}
