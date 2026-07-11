//go:build e2e

package e2e

import (
	"bytes"
	"io"
	"net/http"
	"testing"

	"github.com/rxtech-lab/railway-wiki/e2e-tests/client"
)

// TestMediaUploadFlow exercises the real presign → PUT-to-storage → create-media
// flow against dockerized MinIO:
//
//  1. POST /api/management/media/upload-url -> presigned PUT target
//  2. HTTP PUT the bytes directly to the presigned uploadUrl (headers verbatim)
//  3. GET the canonical object url and verify the bytes round-trip
//  4. POST /api/management/media with that url -> Media record
//  5. GET /api/media/{id} (public) surfaces the new media
//
// Skipped unless S3 is configured (run `make test-e2e`, which boots MinIO).
func TestMediaUploadFlow(t *testing.T) {
	requireS3(t)
	admin := newAuthedClient(t)
	pub := newClient(t)

	payload := []byte("e2e-media-bytes: hello railway wiki\n")
	const contentType = "text/plain"

	// 1. Presign.
	presign, err := admin.AdminCreateMediaUploadUrlWithResponse(ctx(t), client.PresignMediaUploadRequest{
		FileName:    "e2e-note.txt",
		ContentType: contentType,
		SizeBytes:   int64Ptr(int64(len(payload))),
		MediaType:   strPtr("document"),
	})
	if err != nil {
		t.Fatalf("presign: %v", err)
	}
	if presign.StatusCode() != 201 || presign.JSON201 == nil {
		t.Fatalf("presign: expected 201, got %d (%s)", presign.StatusCode(), presign.Body)
	}
	target := presign.JSON201

	// 2. Upload the bytes to the presigned URL, sending signed headers verbatim.
	req, err := http.NewRequestWithContext(ctx(t), target.Method, target.UploadUrl, bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("build upload request: %v", err)
	}
	req.Header.Set("Content-Type", contentType)
	if target.Headers != nil {
		for k, v := range *target.Headers {
			req.Header.Set(k, v)
		}
	}
	putResp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("upload PUT: %v", err)
	}
	defer putResp.Body.Close()
	if putResp.StatusCode < 200 || putResp.StatusCode >= 300 {
		body, _ := io.ReadAll(putResp.Body)
		t.Fatalf("upload PUT: expected 2xx, got %d (%s)", putResp.StatusCode, body)
	}

	// 3. Fetch the canonical url and verify the bytes round-tripped through MinIO.
	getObj, err := http.Get(target.Url)
	if err != nil {
		t.Fatalf("GET object url: %v", err)
	}
	defer getObj.Body.Close()
	if getObj.StatusCode != 200 {
		t.Fatalf("GET object url: expected 200, got %d", getObj.StatusCode)
	}
	gotBytes, _ := io.ReadAll(getObj.Body)
	if !bytes.Equal(gotBytes, payload) {
		t.Fatalf("object bytes mismatch: got %q want %q", gotBytes, payload)
	}

	// 4. Create the Media record referencing the uploaded object.
	media, err := admin.AdminCreateMediaWithResponse(ctx(t), client.CreateMediaRequest{
		MediaType: "document",
		Url:       target.Url,
		Title:     strPtr("E2E uploaded note"),
	})
	if err != nil {
		t.Fatalf("create media: %v", err)
	}
	if media.StatusCode() != 201 || media.JSON201 == nil || media.JSON201.Id == nil {
		t.Fatalf("create media: expected 201 with id, got %d (%s)", media.StatusCode(), media.Body)
	}
	mediaID := *media.JSON201.Id
	t.Cleanup(func() { _, _ = admin.AdminDeleteMediaWithResponse(ctx(t), mediaID) })

	// 5. Public get by id surfaces it.
	pubGet, err := pub.GetMediaWithResponse(ctx(t), mediaID)
	if err != nil {
		t.Fatalf("public get media: %v", err)
	}
	if pubGet.StatusCode() != 200 || pubGet.JSON200 == nil {
		t.Fatalf("public get media: expected 200, got %d (%s)", pubGet.StatusCode(), pubGet.Body)
	}
	if pubGet.JSON200.Url != target.Url {
		t.Fatalf("media url mismatch: got %q want %q", pubGet.JSON200.Url, target.Url)
	}
}
