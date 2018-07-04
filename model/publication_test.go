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

package model_test

import (
	"github.com/dmgk/faker"
	"github.com/jinzhu/gorm"
	"github.com/readium/readium-lcp-server/model"
	"testing"
)

func TestPublicationStore_Add(t *testing.T) {
	for i := 0; i < 100; i++ {
		pub := &model.Publication{
			Title: faker.Name().Name(),
		}
		err := stor.Publication().Add(pub)
		if err != nil {
			t.Fatalf("Error creating publication : %v", err)
		}
	}
}

func TestPublicationStore_Get(t *testing.T) {
	publication, err := stor.Publication().Get(1)
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			t.Fatalf("Error retrieving publication by id : %v", err)
		} else {
			t.Skipf("You forgot to create publication with id = 1 ?")
		}
	}
	t.Logf("Found publication with id 1 : %#v", publication)
}

func TestPublicationStore_GetByUUID(t *testing.T) {
	publication, err := stor.Publication().GetByUUID("")
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			t.Fatalf("Error retrieving publication by UUID : %v", err)
		} else {
			t.Skipf("You forgot to create publication with UUID `` ?")
		}
	}
	t.Logf("Found publication with UUID `` : %#v", publication)
}

func TestPublicationStore_CheckByTitle(t *testing.T) {
	checkedTitle := "Jillian Kovaceku"
	counter, err := stor.Publication().CheckByTitle(checkedTitle)
	if err != nil {
		t.Fatalf("Error checking publications : %v", err)
	}
	t.Logf("%#v counter with title %#v", counter, checkedTitle)
}

func TestPublicationStore_List(t *testing.T) {
	publications, err := stor.Publication().List(30, 0)
	if err != nil {
		t.Fatalf("Error retrieving publications : %v", err)
	}
	for idx, pub := range publications {
		t.Logf("%d. %#v", idx, pub)
	}

	publications, err = stor.Publication().List(50, 1)
	if err != nil {
		t.Fatalf("Error retrieving publications : %v", err)
	}
	for idx, pub := range publications {
		t.Logf("%d. %#v", idx, pub)
	}
}

func TestPublicationStore_Update(t *testing.T) {

}

func TestPublicationStore_Delete(t *testing.T) {
	pubs, err := stor.Publication().List(1, 0)
	if err != nil {
		t.Fatalf("Error retrieving publications : %v", err)
	}
	if len(pubs) != 1 {
		t.Fatalf("No users found. Create some publications.")
	}
	// deleting the first publication
	err = stor.Publication().Delete(pubs[0].ID)
	if err != nil {
		t.Fatalf("Error deleting Publication : %v", err)
	}
	first := pubs[0]
	t.Logf("Publication deleted : %#v", first)
}
