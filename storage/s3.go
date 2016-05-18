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

func (i s3item) PublicUrl() string {
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

type S3Config struct {
	Bucket   string
	Endpoint string
	Region   string

	Id     string
	Secret string
	Token  string

	DisableSSL     bool
	ForcePathStyle bool
}

func S3(config S3Config) (Store, error) {
	client := s3.New(session.New(&aws.Config{
		Credentials:      credentials.NewStaticCredentials(config.Id, config.Secret, config.Token),
		DisableSSL:       aws.Bool(config.DisableSSL),
		S3ForcePathStyle: aws.Bool(config.ForcePathStyle),
		Region:           aws.String(config.Region),
		Endpoint:         aws.String(config.Endpoint)}))
	return &s3store{client: client, bucket: config.Bucket}, nil
}
