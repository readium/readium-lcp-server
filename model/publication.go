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

type (
	PublicationsCollection []*Publication
	// Publication struct defines a publication
	Publication struct {
		ID       int64    `json:"id" sql:"AUTO_INCREMENT" gorm:"primary_key"`
		UUID     string   `json:"uuid" sql:"NOT NULL" gorm:"size:36"` // uuid - max size 36
		Status   string   `json:"status" sql:"NOT NULL"`
		Title    string   `json:"title,omitempty" sql:"NOT NULL"`
		Files    []string `json:"-" gorm:"-"`
		RepoFile string   `gorm:"-"`
	}
)

// Implementation of gorm Tabler
func (p *Publication) TableName() string {
	return LUTPublicationTableName
}

// Implementation of GORM callback
func (p *Publication) BeforeSave() error {
	if p.ID == 0 {
		// Create uuid
		uid, errU := NewUUID()
		if errU != nil {
			return errU
		}
		p.UUID = uid.String()
	}
	return nil
}

// Add adds a new publication
// Encrypts a master File and sends the content to the LCP server
//
func (s publicationStore) Add(pub *Publication) error {
	return s.db.Create(pub).Error
}

// Update updates a publication
// Only the title is updated
//
func (s publicationStore) Update(changedPub *Publication) error {
	var result Publication
	err := s.db.Where(Publication{ID: changedPub.ID}).Find(&result).Error
	if err != nil {
		return err
	}
	return s.db.Model(&result).Updates(map[string]interface{}{
		"title":  changedPub.Title,
		"status": changedPub.Status,
	}).Error
}

// Delete deletes a publication, selected by its numeric id
//
func (s publicationStore) Delete(id int64) error {
	result := Transaction(s.db, func(tx txStore) error {
		// delete all purchases relative to this publication
		err := tx.Where("publication_id = ?", id).Delete(Purchase{}).Error
		if err != nil {
			return err
		}
		return tx.Where("id = ?", id).Delete(Publication{}).Error
	})
	return result
}

func (s publicationStore) Count() (int64, error) {
	var count int64
	return count, s.db.Model(Publication{}).Count(&count).Error
}

func (s publicationStore) FilterCount(paramLike string) (int64, error) {
	var count int64
	return count, s.db.Model(Publication{}).Where("title LIKE ? OR uuid LIKE ?", "%"+paramLike+"%", "%"+paramLike+"%").Count(&count).Error
}

func (s publicationStore) Filter(paramLike string, page, pageNum int64) (PublicationsCollection, error) {
	var result PublicationsCollection
	return result, s.db.Where("title LIKE ? OR uuid LIKE ?", "%"+paramLike+"%", "%"+paramLike+"%").Offset(pageNum * page).Limit(page).Where(&Publication{}).Order("title DESC").Find(&result).Error
}

// List lists publications within a given range
// Parameters: page = number of items per page; pageNum = page offset (0 for the first page)
//
func (s publicationStore) List(page, pageNum int64) (PublicationsCollection, error) {
	var result PublicationsCollection
	return result, s.db.Offset(pageNum * page).Limit(page).Where(&Publication{}).Order("title DESC").Find(&result).Error
}

// Get gets a publication by its ID
//
func (s publicationStore) Get(id int64) (*Publication, error) {
	var result Publication
	return &result, s.db.Where(Publication{ID: id}).Find(&result).Error
}

// GetByUUID returns a publication by its uuid
//
func (s publicationStore) GetByUUID(uuid string) (*Publication, error) {
	var result Publication
	return &result, s.db.Where(Publication{UUID: uuid}).Find(&result).Error
}

// CheckByTitle checks if the publication exists or not, by its title
//
func (s publicationStore) CheckByTitle(title string) (int64, error) {
	var result int64
	err := s.db.Model(Publication{}).Where("title = ?", title).Count(&result).Error
	if err != nil {
		return -1, err
	}
	return result, nil
}
