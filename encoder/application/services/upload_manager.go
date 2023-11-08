package services

import (
	"context"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
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

func (vu *VideoUpload) ProcessUpload(concurrency int, doneUpload chan string) error {

	//qual arquvio ele vai ler na posicao do slice
	in := make(chan int, runtime.NumCPU())
	//avisa quando tem problema
	returnChannel := make(chan string)

	err := vu.loadPaths()
	if err != nil {
		return err
	}

	uploadClient, ctx, err := getClientUpload()
	if err != nil {
		return err
	}

	//verifica quantos workers serão necessários
	for process := 0; process < concurrency; process++ {
		go vu.uploadWorker(in, returnChannel, uploadClient, ctx)
	}

	//criar uma goroutine para ficar escutando o canal de retorno
	go func() {
		for x := 0; x < len(vu.Paths); x++ {
			in <- x
		}
	}()

	for r := range returnChannel {
		if r != "" {
			doneUpload <- r
			break
		}
	}

	return nil
}

func (vu *VideoUpload) uploadWorker(in chan int, returnChan chan string, uploadClaint *storage.Client, ctx context.Context) {
	//range no canal de entrada
	for x := range in {
		error := vu.UploadObject(vu.Paths[x], uploadClaint, ctx)
		if error != nil {
			vu.Errors = append(vu.Errors, vu.Paths[x])
			log.Printf("error during the upload: %v. Error: %v", vu.Paths[x], error)
			returnChan <- error.Error()
		}

		returnChan <- ""
	}
}
