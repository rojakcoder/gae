package gae

import (
	"fmt"
	"io"
	"io/ioutil"
	"strings"

	"cloud.google.com/go/storage"
	"google.golang.org/api/iterator"
	"google.golang.org/appengine/file"

	"golang.org/x/net/context"
)

// FolderSeparator is the slash ("/") that Google Cloud Storage uses to denote
// an object as a folder.
const FolderSeparator = "/"

// GCStorage utilises the API to access Google Cloud Storage.
type GCStorage struct {
	bucket     *storage.BucketHandle
	bucketName string
}

// RECEIVER definitions for GCStorage

// CreateFolder creates an empty folder in Cloud Storage. This is akin to the
// "mkdir" command in Bash.
//
// Note that in Cloud Storage, there is no concept of a folder. A folder is
// created when the name of the object ends with a slash ("/"). Therefore if
// the name does not end with a slash, an error is returned.
func (gcs *GCStorage) CreateFolder(ctx context.Context, name string) error {
	if gcs.bucket == nil {
		return NilError{
			Msg: "bucket is nil",
		}
	}
	if !strings.HasSuffix(name, FolderSeparator) {
		return InvalidError{
			Msg: fmt.Sprintf("object '%v' must end with a folder separator '%v'", name, FolderSeparator),
		}
	}
	wc := gcs.bucket.Object(name).NewWriter(ctx)
	if e := wc.Close(); e != nil {
		return e
	}
	return nil
}

// Delete deletes an object from Cloud Storage.
//
// This can delete both a file or "folder", noting that the concept of a
// folder does not exist in Cloud Storage other than through the name of the
// object.
func (gcs *GCStorage) Delete(ctx context.Context, objName string) error {
	if gcs.bucket == nil {
		return NilError{
			Msg: "bucket is nil",
		}
	}
	if e := gcs.bucket.Object(objName).Delete(ctx); e != nil {
		return e
	}
	return nil
}

// GetBucketName gets the name of the bucket
func (gcs *GCStorage) GetBucketName() string {
	return gcs.bucketName
}

// ListFiles lists the contents of a folder.
//
// The returned list of results contains the names of the objects in its full
// path. To read the names of the files less the directory, use
// `ListFilesAsString`.
//
// For the list of properties available with `ObjectAttrs`, see
// https://godoc.org/cloud.google.com/go/storage#ObjectAttrs
func (gcs *GCStorage) ListFiles(ctx context.Context, foldername string) ([]*storage.ObjectAttrs, error) {
	if gcs.bucket == nil {
		return nil, NilError{
			Msg: "bucket is nil",
		}
	}
	it := gcs.bucket.Objects(ctx, &storage.Query{
		Prefix: foldername,
	})
	results := make([]*storage.ObjectAttrs, 0)
	for {
		attrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		results = append(results, attrs)
	}
	return results, nil
}

// ListFilesAsString lists the file names inside a folder.
//
// The list of returned names is the canonical names of the files (i.e. less
// the path of the folder).
func (gcs *GCStorage) ListFilesAsString(ctx context.Context, foldername string) ([]string, error) {
	results, err := gcs.ListFiles(ctx, foldername)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(results))
	for _, res := range results {
		s := strings.TrimPrefix(res.Name, foldername)
		if len(s) > 0 {
			names = append(names, s)
		}
	}
	return names, nil
}

// ReadFile reads the contents of the object in Cloud Storage.
//
// Note that the full "path" of the object must be specified.
func (gcs *GCStorage) ReadFile(ctx context.Context, name string) ([]byte, error) {
	rc, err := gcs.bucket.Object(name).NewReader(ctx)
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	in, err := ioutil.ReadAll(rc)
	if err != nil {
		return nil, err
	}
	return in, nil
}

// WriteFile writes a file to Cloud Storage.
//
// It reads the bytes from the provided `src` Reader and writes them to the
// object in the bucket with the specified MIME type.
func (gcs *GCStorage) WriteFile(ctx context.Context, name string,
	src io.Reader, mime string) error {
	if gcs.bucket == nil {
		return NilError{
			Msg: "bucket is nil",
		}
	}
	wc := gcs.bucket.Object(name).NewWriter(ctx)
	wc.ContentType = mime
	buf, err := ioutil.ReadAll(src)
	if err != nil {
		return err
	}
	if _, e := wc.Write(buf); e != nil {
		return e
	}
	if e := wc.Close(); e != nil {
		return e
	}
	return nil
}

// GENERAL function definitions

// NewGCStorage creates a new Google Cloud Storage client.
//
// The client has to be created from the caller so that it may be closed on a
// per request basis.
func NewGCStorage(ctx context.Context, client *storage.Client,
	bucketName string) (GCStorage, error) {
	gcs := GCStorage{}
	if client == nil {
		return gcs, NilError{
			Msg: "client is nil",
		}
	}
	if bucketName == "" {
		bname, err := file.DefaultBucketName(ctx)
		if err != nil {
			return gcs, err
		}
		gcs.bucketName = bname
	} else {
		gcs.bucketName = bucketName
	}
	gcs.bucket = client.Bucket(gcs.bucketName)
	return gcs, nil
}
