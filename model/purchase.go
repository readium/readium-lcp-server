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
	"database/sql"
	"fmt"
	"github.com/jinzhu/gorm"
	"time"
)

const (
	LoanType = "Loan"
	BuyType  = "Buy"
)

type (
	PurchaseCollection []*Purchase
	//Purchase struct defines a user in json and database
	//PurchaseType: BUY or LOAN
	Purchase struct {
		ID              int64       `json:"id,omitempty" sql:"AUTO_INCREMENT" gorm:"primary_key"`
		PublicationId   int64       `json:"-" sql:"NOT NULL"`
		UserId          int64       `json:"-" sql:"NOT NULL"`
		UUID            string      `json:"uuid" sql:"NOT NULL;UNIQUE_INDEX" gorm:"size:36"`
		Type            string      `json:"type" sql:"NOT NULL"`
		Status          Status      `json:"status" sql:"NOT NULL"`
		TransactionDate time.Time   `json:"transactionDate,omitempty" sql:"DEFAULT:current_timestamp;NOT NULL"`
		LicenseUUID     *NullString `json:"licenseUuid,omitempty" gorm:"size:36" sql:"DEFAULT NULL"`
		StartDate       *NullTime   `json:"startDate,omitempty" sql:"DEFAULT NULL"`
		EndDate         *NullTime   `json:"endDate,omitempty" sql:"DEFAULT NULL"`
		Publication     Publication `json:"publication" gorm:"foreignKey:PublicationId;save_associations:false"`
		User            User        `json:"user" gorm:"foreignKey:UserId;save_associations:false"`
		IsExternal      bool
		IsDelivered     bool
	}
)

func (p *Purchase) LicenseUUIDString() string {
	if p.LicenseUUID != nil && p.LicenseUUID.Valid {
		return p.LicenseUUID.String
	}
	return ""
}

// Implementation of gorm Tabler
func (p *Purchase) TableName() string {
	return LUTPurchaseTableName
}

// Implementation of GORM callback
func (p *Purchase) AfterFind() error {
	// cleanup for json to omit empty
	if p.LicenseUUID != nil && !p.LicenseUUID.Valid {
		p.LicenseUUID = nil
	}
	if p.StartDate != nil && !p.StartDate.Valid {
		p.StartDate = nil
	}
	if p.EndDate != nil && !p.EndDate.Valid {
		p.EndDate = nil
	}
	return nil
}

// Implementation of GORM callback
func (p *Purchase) BeforeSave() error {
	now := TruncatedNow()
	if p.TransactionDate.IsZero() {
		p.TransactionDate = now.Time
	}
	if p.UserId == 0 {
		if p.User.ID == 0 {
			return fmt.Errorf("User ID is zero. Must be set.")
		}
		p.UserId = p.User.ID
	}
	if p.Type == LoanType && p.StartDate == nil {
		p.StartDate = now
	}
	if p.UUID == "" && p.ID == 0 {
		// Create uuid
		uid, errU := NewUUID()
		if errU != nil {
			return errU
		}
		p.UUID = uid.String()
	}
	return nil
}

func (s purchaseStore) MarkAsDelivered(licenseID string) error {
	var result Purchase
	err := s.db.Where("license_uuid = ?", licenseID).Find(&result).Error
	if err != nil {
		return err
	}
	return s.db.Model(&result).Updates(map[string]interface{}{
		"is_delivered": true,
	}).Error
}

// Get a purchase using its id
//
func (s purchaseStore) Get(id int64) (*Purchase, error) {
	var result Purchase
	return &result, s.db.Where("id = ?", id).Preload("User").Preload("Publication").Find(&result).Error
}

// GetByLicenseID gets a purchase by the associated license id
//
func (s purchaseStore) GetByLicenseID(licenseID string) (*Purchase, error) {
	var result Purchase
	return &result, s.db.Where("license_uuid = ?", licenseID).Preload("User").Preload("Publication").Find(&result).Error
}

func (s purchaseStore) Count() (int64, error) {
	var count int64
	return count, s.db.Model(Purchase{}).Count(&count).Error
}

func (s purchaseStore) FilterCount(paramLike string) (int64, error) {
	var count int64
	return count, s.db.Model(Purchase{}).Where("uuid LIKE ?", "%"+paramLike+"%").Count(&count).Error
}

func (s purchaseStore) Filter(paramLike string, page, pageNum int64) (PurchaseCollection, error) {
	var result PurchaseCollection
	return result, s.db.Where("uuid LIKE ?", "%"+paramLike+"%").Offset(pageNum * page).Limit(page).Where(&Publication{}).Order("transaction_date DESC").Preload("User").Preload("Publication").Find(&result).Error
}

// List purchases, with pagination
//
func (s purchaseStore) List(page, pageNum int64) (PurchaseCollection, error) {
	var result PurchaseCollection
	return result, s.db.Offset(pageNum * page).Limit(page).Order("transaction_date DESC").Preload("User").Preload("Publication").Find(&result).Error

}
func (s purchaseStore) CountByUser(userID int64) (int64, error) {
	var count int64
	return count, s.db.Model(Purchase{}).Where("user_id = ?", userID).Order("transaction_date DESC").Count(&count).Error
}

// ListByUser: list the purchases of a given user, with pagination
//
func (s purchaseStore) ListByUser(userID, page, pageNum int64) (PurchaseCollection, error) {
	var result PurchaseCollection
	return result, s.db.Where("user_id = ?", userID).Offset(pageNum * page).Limit(page).Order("transaction_date DESC").Preload("User").Preload("Publication").Find(&result).Error
}

// Add a purchase
//
func (s purchaseStore) Add(p *Purchase) error {
	return s.db.Create(p).Error
}

func (s purchaseStore) UpdateEndDate(id int64, date time.Time) error {
	var result Purchase
	err := s.db.Where(Purchase{ID: id}).Find(&result).Error
	if err != nil {
		return err
	}
	return s.db.Model(&result).Updates(map[string]interface{}{"end_date": date}).Error
}

func (s purchaseStore) LoadUser(p *Purchase) error {
	return s.db.Model(User{}).Where("id = ?", p.UserId).Find(&p.User).Error
}

func (s purchaseStore) LoadPublication(p *Purchase) error {
	return s.db.Model(User{}).Where("id = ?", p.PublicationId).Find(&p.Publication).Error
}

func (s purchaseStore) BulkAddOrUpdate(licenses LicensesCollection, statuses map[string]Status) error {
	var purchases PurchaseCollection
	// we need to save users and publications prior to saving purchases, becase inside transaction we need commit to get their ids
	for _, license := range licenses {
		if license.Content == nil {
			//ignore invalid licenses (because we can't save publication)
			continue
		}
		// save user if not already exist
		var userEntity User
		err := s.db.Find(&userEntity, "uuid = ?", license.UserId).Error
		switch err {
		case nil:
			// update user (nothing to update)
		case gorm.ErrRecordNotFound:
			// create user
			userEntity = User{
				UUID:       license.UserId,
				Name:       "Do not edit user",
				Email:      license.UserId + "@lcp.user",
				IsExternal: true,
			}
			err = s.db.Create(&userEntity).Error
			if err != nil {
				s.log.Errorf("Error creating user %q : %v", license.UserId, err)
				return err
			}
		default:
			return err
		}

		// save publication if not already exist
		var publicationEntity Publication
		err = s.db.Find(&publicationEntity, "uuid = ?", license.Content.Id).Error
		switch err {
		case nil:
			// update publication (nothing to update)
		case gorm.ErrRecordNotFound:
			// create publication
			publicationEntity = Publication{
				UUID:       license.Content.Id,
				Title:      license.Content.Location,
				IsExternal: true,
			}
			err = s.db.Create(&publicationEntity).Error
			if err != nil {
				s.log.Errorf("Error creating publication : %v", err)
				return err
			}
		default:
			return err
		}

		purchase := &Purchase{
			LicenseUUID:     &NullString{NullString: sql.NullString{String: license.Id, Valid: true}},
			UserId:          userEntity.ID,
			PublicationId:   publicationEntity.ID,
			Status:          statuses[license.Id],
			TransactionDate: license.Issued,
			IsExternal:      true,
			Type:            BuyType,
		}

		if license.Start != nil {
			purchase.StartDate = license.Start
		}
		// has an end date - so it's a loan
		if license.End != nil {
			purchase.EndDate = license.End
			purchase.Type = LoanType
		}

		var existingPurchase Purchase
		err = s.db.Find(&existingPurchase, "license_uuid = ?", license.Id).Error
		switch err {
		case nil:
			// entry exists - taking id, so we can update
			purchase.ID = existingPurchase.ID
			purchase.IsExternal = existingPurchase.IsExternal
		}
		purchases = append(purchases, purchase)
	}
	return Transaction(s.db, func(tx txStore) error {
		for _, purchase := range purchases {
			var err error
			if purchase.ID <= 0 {
				err = tx.Create(purchase).Error
				if err != nil {
					return err
				}
			} else {
				err = tx.Model(Purchase{}).Save(purchase).Error
				if err != nil {
					return err
				}
			}
		}
		return nil
	})
}

func (s purchaseStore) BulkDelete(ids []int64) error {
	return Transaction(s.db, func(tx txStore) error {
		for _, deletedId := range ids {
			err := tx.Where("id = ?", deletedId).Delete(Purchase{}).Error
			if err != nil {
				return err
			}
		}
		return nil
	})
}
