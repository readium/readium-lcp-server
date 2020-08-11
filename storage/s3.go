// Copyright (c) 2016 Readium Foundation
//
// Redistribution and use in source and binary forms, with or without modification,
// are permitted provided that the following conditions are met:
//
// 1. Redistributions of source code must retain the above copyright notice, this
//    list of conditions and the following disclaimer.
// 2. Redistributions in binary form must reproduce the above copyright notice,
//    this list of conditions and the following disclaimer in the documentation and/or
//    other materials provided with the distribution.
// 3. Neither the name of the organization nor the names of its contributors may be
//    used to endorse or promote products derived from this software without specific
//    prior written permission
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND
// ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
// WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
// DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT OWNER OR CONTRIBUTORS BE LIABLE FOR
// ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
// (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
// LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND
// ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
// SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

package storage

import (
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

type s3store struct {
	bucket string
	client *s3.S3
}

type s3item struct {
	bucket string
	key    string
	store  *s3store
}

func (i s3item) Key() string {
	return i.key
}

func (i s3item) PublicURL() string {
	return fmt.Sprintf("http://%s/%s/%s", i.store.client.Endpoint, i.bucket, i.key)
}

func (i s3item) Contents() (io.ReadCloser, error) {
	resp, err := i.store.client.GetObject(&s3.GetObjectInput{
		Bucket: aws.String(i.store.bucket),
		Key:    aws.String(i.key),
	})

	return resp.Body, err
}

func (s *s3store) Add(key string, r io.ReadSeeker) (Item, error) {
	_, err := s.client.PutObject(&s3.PutObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
		Body:   r,
	})

	item := s3item{bucket: s.bucket, key: key, store: s}

	return item, err
}

func (s *s3store) Get(key string) (Item, error) {
	_, err := s.client.HeadObject(&s3.HeadObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	return s3item{bucket: s.bucket, key: key, store: s}, err
}

func (s *s3store) Remove(key string) error {
	_, err := s.client.DeleteObject(&s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})

	return err
}

func (s *s3store) List() ([]Item, error) {
	objects, err := s.client.ListObjects(&s3.ListObjectsInput{
		Bucket: aws.String(s.bucket),
	})

	if err != nil {
		return nil, err
	}

	var items []Item

	for _, o := range objects.Contents {
		items = append(items, s3item{bucket: s.bucket, key: *o.Key, store: s})
	}

	return items, nil
}

// S3Config structure
type S3Config struct {
	Bucket   string
	Endpoint string
	Region   string

	ID     string
	Secret string
	Token  string

	DisableSSL     bool
	ForcePathStyle bool
}

// S3 inits and S3 storage
func S3(config S3Config) (Store, error) {
	awsConfig := &aws.Config{
		DisableSSL:                    aws.Bool(config.DisableSSL),
		S3ForcePathStyle:              aws.Bool(config.ForcePathStyle),
		Region:                        aws.String(config.Region),
		Endpoint:                      aws.String(config.Endpoint)}

	// Credentials defaults to a chain of credential providers to search for credentials in environment
	// variables, shared credential file, and EC2 Instance Roles.
	// Therefore, we only explicitly define static credentials if these are present in config
	if config.ID != "" && config.Secret != "" {
		awsConfig.Credentials = credentials.NewStaticCredentials(config.ID, config.Secret, config.Token)
	}

	return &s3store{client: s3.New(session.New(awsConfig)), bucket: config.Bucket}, nil
}
