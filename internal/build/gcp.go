package build

import (
	"io"
	"os"

	"cloud.google.com/go/storage"
	"golang.org/x/net/context"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

type GCPOption struct {
	CredentialPath string
}

// GCPFileUpload upload file to GCP storage
func GCPFileUpload(path, bucket, name string, opt GCPOption) error {
	ctx := context.Background()
	client, err := storage.NewClient(ctx, option.WithCredentialsFile(opt.CredentialPath))
	if err != nil {
		return err
	}

	wc := client.Bucket(bucket).Object(name).NewWriter(ctx)

	in, err := os.Open(path)
	if err != nil {
		return err
	}
	defer in.Close()

	if _, err = io.Copy(wc, in); err != nil {
		return err
	}
	if err := wc.Close(); err != nil {
		return err
	}

	return nil
}

// GCPFileList list files from GCP storage
func GCPFileList(bucket string, opt GCPOption) ([]*storage.ObjectAttrs, error) {
	ctx := context.Background()
	client, err := storage.NewClient(ctx, option.WithCredentialsFile(opt.CredentialPath))
	if err != nil {
		return nil, err
	}

	var list []*storage.ObjectAttrs
	it := client.Bucket(bucket).Objects(ctx, nil)
	for {
		attrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		list = append(list, attrs)
	}

	return list, nil
}

// GCPFileDelete delete files from GCP storage
func GCPFileDelete(bucket string, objects []*storage.ObjectAttrs, opt GCPOption) error {
	ctx := context.Background()
	client, err := storage.NewClient(ctx, option.WithCredentialsFile(opt.CredentialPath))
	if err != nil {
		return err
	}

	for _, obj := range objects {
		if err := client.Bucket(bucket).Object(obj.Name).Delete(ctx); err != nil {
			return err
		}
	}

	return nil
}
