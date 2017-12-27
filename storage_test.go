package gae

import (
	"log"
	"strings"
	"testing"

	"cloud.google.com/go/storage"

	"google.golang.org/appengine/aetest"
)

const (
	ProjectID  = "even-dream-627"
	BucketName = "staging.even-dream-627.appspot.com"
)

func TestStorageWriteReadFolder(t *testing.T) {
	ctx, done, err := aetest.NewContext()
	if err != nil {
		t.Fatal(err)
	}
	defer done()

	client, err := storage.NewClient(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	gc1, err := NewGCStorage(ctx, client, BucketName)
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		folder        string
		file          string
		makeContents  bool
		wantFileSize  int
		wantFolder    bool
		wantFileCount int
	}{
		{
			//create a folder with an empty file in it
			folder:        "FB/",
			file:          "filetwo.txt",
			makeContents:  false,
			wantFileSize:  0,
			wantFolder:    false,
			wantFileCount: 1,
		},
		{
			//create a folder with a non-empty file in it
			folder:        "FC/",
			file:          "filethree.txt",
			makeContents:  true,
			wantFileSize:  13,
			wantFolder:    false,
			wantFileCount: 1,
		},
		{
			//create a non-empty file in an existing folder
			folder:        "precreated/",
			file:          "filefour.txt",
			makeContents:  true,
			wantFileSize:  12,
			wantFolder:    true,
			wantFileCount: 1,
		},
	}
	if e := gc1.CreateFolder(ctx, cases[2].folder); e != nil {
		t.Fatal(e)
	}
	for _, c := range cases {
		rdr := strings.NewReader("")
		if c.makeContents {
			rdr = strings.NewReader(c.file)
		}
		if e := gc1.WriteFile(ctx, c.folder+c.file, rdr, "text/plain"); e != nil {
			t.Fatal(e)
		}
		data, err := gc1.ReadFile(ctx, c.folder+c.file)
		if err != nil {
			t.Fatal(err)
		}
		if c.wantFileSize != len(data) {
			t.Errorf("expect size of '%v' to be %d; got %d",
				c.folder+c.file, c.wantFileSize, len(data))
		}
		//can only "read" the folder if it was explicitly created
		data, err = gc1.ReadFile(ctx, c.folder)
		if !c.wantFolder && err == nil {
			t.Errorf("expect folder '%v' to be not found; got nil",
				c.wantFolder)
		}
		if c.wantFolder && err != nil {
			t.Errorf("expect folder '%v' to be found; got error %v",
				c.wantFolder, err)
		}
		got, err := gc1.ListFilesAsString(ctx, c.folder)
		if err != nil {
			t.Fatal(err)
		}
		if c.wantFileCount != len(got) {
			t.Errorf("expect folder '%v' to have %d object; got %d",
				c.folder, c.wantFileCount, len(got))
		}
		if c.file != got[0] {
			t.Errorf("expect folder '%v' to contain file '%v'; got '%v'",
				c.folder, c.file, got[0])
		}
		if e := gc1.Delete(ctx, c.folder+c.file); e != nil {
			t.Fatal(e)
		}
	}
	//delete the explicitly created folder
	if e := gc1.Delete(ctx, cases[2].folder); e != nil {
		t.Fatal(e)
	}
	//create a folder
	folder1 := "FA/"
	//folder can be created as if creating an empty file named with a trailing slash
	if e := gc1.WriteFile(ctx, folder1, strings.NewReader(""), "text/plain"); e != nil {
		t.Fatal(e)
	}
	data, err := gc1.ReadFile(ctx, folder1)
	if err != nil {
		t.Fatal(err)
	}
	if 0 != len(data) {
		t.Errorf("expect size of empty folder to be 0; got %d", len(data))
	}
	got, err := gc1.ListFilesAsString(ctx, folder1)
	if err != nil {
		t.Fatal(err)
	}
	if 0 != len(got) {
		t.Errorf("expect folder '%v' to have 0 objects; got %d",
			folder1, len(got))
	}
	//delete the object
	if e := gc1.Delete(ctx, folder1); e != nil {
		t.Fatal(e)
	}
}

func TestStorageCreateFolder(t *testing.T) {
	ctx, done, err := aetest.NewContext()
	if err != nil {
		t.Fatal(err)
	}
	defer done()

	client, err := storage.NewClient(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	gc1, err := NewGCStorage(ctx, client, BucketName)
	if err != nil {
		t.Fatal(err)
	}
	inputsfail := []string{
		"/foldere",
		"//folderu",
	}
	for _, in := range inputsfail {
		log.Printf("> Attempting to create folder '%v'", in)
		if e := gc1.CreateFolder(ctx, in); e == nil {
			t.Errorf("expected error when creating non-folder object; got nil")
		} else {
			log.Printf("  correctly failed: %v", e)
		}
	}
	inputs := []string{
		"foldera/",
		"/foldere/",
		"//folderi/",
		"foldero//",
	}
	for _, in := range inputs {
		log.Printf("> Creating '%v'", in)
		if e := gc1.CreateFolder(ctx, in); e != nil {
			t.Fatalf("error creating %v", in)
		}
		log.Printf("  Done.")
	}
	outputs := map[string][]string{
		"foldera/": []string{},
		"/": []string{
			"/folderi/", "foldere/",
		},
		"foldero/": []string{
			"/",
		},
	}
	for k, want := range outputs {
		got, err := gc1.ListFilesAsString(ctx, k)
		if err != nil {
			t.Fatal(err)
		}
		if len(want) != len(got) {
			t.Errorf("expect for '%v' array of size %d; got %d\n\t%v\n\t%v",
				k, len(want), len(got), want, got)
		}
		for i := range got {
			if want[i] != got[i] {
				t.Errorf("expect for position %d of '%v' the array\n%v; got\n%v",
					i, k, want, got)
				break
			}
		}
	}
	for _, in := range inputs {
		log.Printf("> Deleting '%v'", in)
		if e := gc1.Delete(ctx, in); e != nil {
			t.Fatal(e)
		}
		log.Printf("  Done.")
	}
}
