package services

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"

	"cloud.google.com/go/storage"
)

type VideoUpload struct {
	Paths        []string
	VideoPath    string
	OutputBucket string
	Errors       []string
}

func NewVideoUpload() *VideoUpload {
	return &VideoUpload{}
}

func (vu *VideoUpload) UploadObject(objectPath string, client *storage.Client, ctx context.Context) error {

	path := strings.Split(objectPath, os.Getenv("localStorage")+"/")

	f, error := os.Open(objectPath)
	if error != nil {
		return error
	}

	defer f.Close()

	wc := client.Bucket(vu.OutputBucket).Object(path[1]).NewWriter(ctx)
	wc.ACL = []storage.ACLRule{{Entity: storage.AllUsers, Role: storage.RoleReader}} //permissão para usuários lerem o arquivo

	if _, error = io.Copy(wc, f); error != nil {
		return error
	}

	if error = wc.Close(); error != nil {
		return error
	}

	return nil
}

func (vu *VideoUpload) loadPaths() error {

	err := filepath.Walk(vu.VideoPath, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			vu.Paths = append(vu.Paths, path)
		}
		return nil
	})

	if err != nil {
		return err
	}

	return nil
}

func getClientUpload() (*storage.Client, context.Context, error) {
	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, nil, err
	}

	return client, ctx, nil
}
