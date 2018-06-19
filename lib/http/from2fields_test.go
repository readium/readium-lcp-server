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

package http

import (
	"github.com/readium/readium-lcp-server/model"
	"reflect"
	"testing"
)

func TestFormToFields(t *testing.T) {
	var payload *model.Purchase
	payloadType := reflect.TypeOf(payload)
	deserializeTo := reflect.New(payloadType.Elem())
	formValues := make(map[string][]string)
	formValues["ID"] = []string{"5"}
	formValues["UUID"] = []string{"109a4fee-ac84-4432-9f1d-4fe096d83fcc"}
	formValues["UserId"] = []string{"33"}
	formValues["PublicationId"] = []string{"12"}
	formValues["Status"] = []string{"Revoked"}
	formValues["Type"] = []string{"Loan"}
	formValues["StartDate"] = []string{"2018-06-21T02:00:00+10:00"}
	formValues["EndDate"] = []string{"2018-06-30T02:00:00+24:00"}
	FormToFields(deserializeTo, formValues)
	t.Logf("%#v", deserializeTo)
}
