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

import "fmt"

type (
	// Dashboard struct defines a publication
	Dashboard struct {
		PublicationCount int64 `json:"publicationCount"`
		UserCount        int64 `json:"userCount"`
		BuyCount         int64 `json:"buyCount"`
		LoanCount        int64 `json:"loanCount"`
		AverageDuration  int64 `json:"averageDuration"`
	}

	// BestSeller struct defines a best seller
	BestSeller struct {
		Title string `json:"title"`
		Count int64  `json:"count"`
	}
)

// GetDashboardInfos a publication for a given ID
func (s dashboardStore) GetDashboardInfos() (*Dashboard, error) {
	result := &Dashboard{}
	err := s.db.Model(&User{}).Count(&result.UserCount).Error
	if err != nil {
		return nil, err
	}

	err = s.db.Model(&Publication{}).Count(&result.PublicationCount).Error
	if err != nil {
		return nil, err
	}

	err = s.db.Model(&Purchase{}).Where("type = ?", "BUY").Count(&result.BuyCount).Error
	if err != nil {
		return nil, err
	}

	err = s.db.Model(&Purchase{}).Where("type = ?", "LOAN").Count(&result.BuyCount).Error
	if err != nil {
		return nil, err
	}

	err = s.db.Table(LUTPurchaseTableName).Select("ROUND(AVG(JULIANDAY(end_date) - JULIANDAY(start_date))) AS averageDuration").Where("type = ?", "LOAN").Scan(&result).Error
	if err != nil {
		return nil, err
	}
	return result, nil
}

// GetDashboardBestSellers a publication for a given ID
func (s dashboardStore) GetDashboardBestSellers() ([]BestSeller, error) {
	bestSellers := make([]BestSeller, 5)
	selectSQL := fmt.Sprintf("%s.title, COUNT(%s.id)", LUTPublicationTableName, LUTPublicationTableName)
	joinSQL := fmt.Sprintf("JOIN %s ON %s.publication_id = %s.id", LUTPublicationTableName, LUTPurchaseTableName, LUTPublicationTableName)
	groupSQL := fmt.Sprintf("%s.id", LUTPublicationTableName)
	orderSQL := fmt.Sprintf("COUNT(%s.id) DESC", LUTPurchaseTableName)
	err := s.db.Table(LUTPurchaseTableName).Select(selectSQL).Joins(joinSQL).Group(groupSQL).Order(orderSQL).Limit(5).Find(&bestSellers).Error
	return bestSellers, err
}
