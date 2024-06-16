package internal

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/containers/buildah"
	is "github.com/containers/image/v5/storage"
	"github.com/containers/storage"

	"github.com/containers/image/v5/copy"
	"github.com/containers/image/v5/signature"
	"github.com/containers/image/v5/transports/alltransports"
	"github.com/containers/image/v5/types"
)

const (
	tempdir = "/home/vagrant/checkpoint"
)

type Config struct {
	// have this label in the config struct
	// {\"Labels\":{\"io.buildah.version\":\"1.35.4\"}}
	Labels map[string]string `json:"labels"`
}

type RootFs struct {
	Type    string   `json:"type"`
	DiffIds []string `json:"diff_ids"`
}

type HistoryEntry struct {
	Created   time.Time `json:"created"`
	CreatedBy string    `json:"created_by"`
}

type ImageConfig struct {
	Created      time.Time      `json:"created"`
	Architecture string         `json:"architecture"`
	Variant      string         `json:"variant"`
	Os           string         `json:"os"`
	Config       Config         `json:"config"`
	RootFs       RootFs         `json:"rootfs"`
	History      []HistoryEntry `json:"history"`
}
type BlobReference struct {
	MediaType string `json:"mediaType"`
	Digest    string `json:"digest"`
	Size      int    `json:"size"`
}

type ImageManifest struct {
	SchemaVersion int               `json:"schemaVersion"`
	MediaType     string            `json:"mediaType"`
	Config        BlobReference     `json:"config"`
	Layers        []BlobReference   `json:"layers"`
	Annotations   map[string]string `json:"annotations"`
}
type Index struct {
	SchemaVersion int             `json:"schemaVersion"`
	Manifests     []BlobReference `json:"manifests"`
}

func CreateCheckpointImage(ctx context.Context, checkpointPath string, containerName string, checkpointName string) (string, string, error) {
	checkpointPrefix := "cr"
	checkpointImageName := fmt.Sprintf("%s/%s:latest", checkpointPrefix, checkpointName)
	buildStoreOptions, err := storage.DefaultStoreOptions()

	if err != nil {
		fmt.Println("storage.DefaultStoreOptions")
		return "", "", err
	}
	buildStore, err := storage.GetStore(buildStoreOptions)

	if err != nil {
		fmt.Println("storage.GetStore")
		return "", "", err
	}
	defer buildStore.Shutdown(false)
	builderOpts := buildah.BuilderOptions{
		FromImage: "scratch",
	}
	builder, err := buildah.NewBuilder(ctx, buildStore, builderOpts)

	if err != nil {
		fmt.Println("buildah.NewBuilder")
		return "", "", err
	}
	defer builder.Delete()

	fmt.Println("create Image: builder + store setup complete")
	err = builder.Add("/", true, buildah.AddAndCopyOptions{}, checkpointPath)

	if err != nil {
		fmt.Println("builder.Add")
		return "", "", err
	}
	fmt.Println("create Image: added archive")
	builder.ImageAnnotations["io.kubernetes.cri-o.annotations.checkpoint.name"] = containerName
	imageRef, err := is.Transport.ParseStoreReference(buildStore, checkpointImageName)

	if err != nil {
		fmt.Println("is.Transport.ParseStoreReference")
		return "", "", err
	}
	fmt.Println("create Image: generated store reference")
	imageId, _, _, err := builder.Commit(ctx, imageRef, buildah.CommitOptions{})

	fmt.Println("create Image: committed")
	if err != nil {
		fmt.Println("builder.Commit")
		return "", "", err
	}

	sysCtx := &types.SystemContext{}
	policy, err := signature.DefaultPolicy(sysCtx)
	if err != nil {
		return "", "", fmt.Errorf("obtaining default signature policy: %w", err)
	}
	policyContext, err := signature.NewPolicyContext(policy)
	if err != nil {
		return "", "", fmt.Errorf("creating new signature policy context: %w", err)
	}
	copyOpts := &copy.Options{
		DestinationCtx: sysCtx,
	}

	fmt.Println("create Image: parsed policies")
	exportName := fmt.Sprintf("oci-archive://%s/%s", tempdir, checkpointName)
	destinationRef, err := alltransports.ParseImageName(exportName)
	if err != nil {
		fmt.Println("is.Transport.ParseStoreReference")
		return "", "", err
	}
	fmt.Println("create Image: parsed image name")
	_, err = copy.Image(context.TODO(), policyContext, destinationRef, imageRef, copyOpts)

	fmt.Println("create Image: copied image")
	if err != nil {
		fmt.Println("copy.Image")
		return "", "", err
	}
	err = os.Remove(checkpointPath)
	fmt.Println("create Image: removed archive")
	if err != nil {
		fmt.Println("error while deleting checkpoint archive")
		return "", "", err
	}

	return imageId, checkpointImageName, nil
}
func CreateOCIImage(fromPath string, containerName string, checkpointName string) error {
	checkpointFile, err := os.Create(filepath.Join(tempdir, checkpointName))
	fmt.Println("checkpointFile: ", filepath.Join(tempdir, checkpointName))
	if err != nil {
		return err
	}
	tarWriter := tar.NewWriter(checkpointFile)
	blobPath := "blobs/sha256/"

	err = tarWriter.WriteHeader(&tar.Header{
		Typeflag:   tar.TypeDir,
		Name:       "blobs/",
		Size:       0,
		Mode:       0755,
		ModTime:    time.Now(),
		AccessTime: time.Now(),
		ChangeTime: time.Now(),
	})
	err = tarWriter.WriteHeader(&tar.Header{
		Typeflag:   tar.TypeDir,
		Name:       blobPath,
		Size:       0,
		Mode:       0755,
		ModTime:    time.Now(),
		AccessTime: time.Now(),
		ChangeTime: time.Now(),
	})
	if err != nil {
		return err
	}

	OCILayout := []byte(GenerateOCILayout())
	//"oci-layout"--> header, write to the new tar file
	err = writeToTar(tarWriter, "oci-layout", OCILayout)
	if err != nil {
		return err
	}
	fmt.Println("frompath ", fromPath)
	from, err := os.Open(fromPath)
	if err != nil {
		fmt.Println(err)
		return err
	}
	defer from.Close()

	unzippedDigestHash := sha256.New()
	firstInputReader := io.TeeReader(from, unzippedDigestHash)

	zippedBuffer := &bytes.Buffer{}
	// transform the /var/lib/kubelet/checkpoint tar files to gzip files
	err = Gzip(firstInputReader, zippedBuffer)
	if err != nil {
		return err
	}
	unzippedDigest := fmt.Sprintf("%x", unzippedDigestHash.Sum(nil))

	zippedBytes := zippedBuffer.Bytes()
	zippedDigest := ComputeSha256(bytes.NewReader(zippedBytes))
	// the most critical part of the code
	err = writeToTar(tarWriter, blobPath+zippedDigest, zippedBytes)
	if err != nil {
		return err
	}
	imageConfig, err := GenerateImageConfig("sha256:" + unzippedDigest)
	if err != nil {
		return err
	}
	imageConfigDigest := ComputeSha256(bytes.NewReader(imageConfig))
	err = writeToTar(tarWriter, blobPath+imageConfigDigest, imageConfig)
	if err != nil {
		return err
	}
	manifest, err := GenerateImageManifest("sha256:"+imageConfigDigest, len(imageConfig), "sha256:"+zippedDigest, len(zippedBytes), containerName)
	if err != nil {
		return err
	}
	manifestDigest := ComputeSha256(bytes.NewReader(manifest))
	err = writeToTar(tarWriter, blobPath+manifestDigest, manifest)
	if err != nil {
		return err
	}
	index, err := GenerateIndex("sha256:"+manifestDigest, len(manifest))
	if err != nil {
		return err
	}
	err = writeToTar(tarWriter, "index.json", index)
	if err != nil {
		return err
	}

	err = tarWriter.Close()
	if err != nil {
		return err
	}

	return os.Remove(fromPath)
}
func GenerateOCILayout() string {
	return "{\"imageLayoutVersion\": \"1.0.0\"}"
}
func Gzip(reader io.Reader, target io.Writer) error {
	archiver := gzip.NewWriter(target)
	defer archiver.Close()

	_, err := io.Copy(archiver, reader)
	return err
}
func ComputeSha256(reader io.Reader) string {
	h := sha256.New()
	if _, err := io.Copy(h, reader); err != nil {
		log.Fatal(err)
	}

	return fmt.Sprintf("%x", h.Sum(nil))
}
func writeToTar(tarWriter *tar.Writer, filename string, bytes []byte) error {
	err := tarWriter.WriteHeader(&tar.Header{
		Typeflag:   tar.TypeReg,
		Name:       filename,
		Size:       int64(len(bytes)),
		Mode:       0644,
		ModTime:    time.Now(),
		AccessTime: time.Now(),
		ChangeTime: time.Now(),
	})
	if err != nil {
		return err
	}
	_, err = tarWriter.Write(bytes)
	return err
}
func GenerateImageConfig(diffId string) ([]byte, error) {
	conf := ImageConfig{
		Created:      time.Now(),
		Architecture: "amd64",
		// Variant:      "v7",
		Os: "linux",
		Config: Config{
			Labels: map[string]string{
				"io.buildah.version": "1.35.4",
			},
		},
		RootFs: RootFs{
			Type: "layers",
			DiffIds: []string{
				diffId,
			},
		},
		History: []HistoryEntry{
			{
				Created:   time.Now(),
				CreatedBy: "/bin/sh",
			},
		},
	}

	return json.Marshal(conf)
}
func GenerateImageManifest(imageConfigDigest string, imageConfigSize int, layerDigest string, layerSize int, containerName string) ([]byte, error) {
	manifest := ImageManifest{
		SchemaVersion: 2,
		MediaType:     "applications/vnd.oci.image.manifest.v1+json",
		Config: BlobReference{
			MediaType: "application/vnd.oci.image.config.v1+json",
			Digest:    imageConfigDigest,
			Size:      imageConfigSize,
		},
		Layers: []BlobReference{
			{
				MediaType: "application/vnd.oci.image.layer.v1.tar+gzip",
				Digest:    layerDigest,
				Size:      layerSize,
			},
		},
		Annotations: map[string]string{
			"io.kubernetes.cri-o.annotations.checkpoint.name": containerName,
			"org.opencontainers.image.base.digest":            "",
			"org.opencontainers.image.base.name":              "",
		},
	}

	return json.Marshal(manifest)
}
func GenerateIndex(imageManifestDigest string, imageManifestSize int) ([]byte, error) {
	manifest := Index{
		SchemaVersion: 2,
		Manifests: []BlobReference{
			{
				MediaType: "application/vnd.oci.image.manifest.v1+json",
				Digest:    imageManifestDigest,
				Size:      imageManifestSize,
			},
		},
	}
	return json.Marshal(manifest)
}

func ImportImage(path string) error {
	sysCtx := &types.SystemContext{}
	policy, err := signature.DefaultPolicy(sysCtx)
	if err != nil {
		return fmt.Errorf("obtaining default signature policy: %w", err)
	}
	policyContext, err := signature.NewPolicyContext(policy)
	if err != nil {
		return fmt.Errorf("creating new signature policy context: %w", err)
	}

	checkpointName := filepath.Base(path)
	destinationImageName := fmt.Sprintf("containers-storage:localhost/%s:latest", checkpointName)
	destinationRef, err := alltransports.ParseImageName(destinationImageName)
	if err != nil {
		return err
	}
	sourceImageName := fmt.Sprintf("oci-archive://%s/%s", tempdir, checkpointName)
	sourceRef, err := alltransports.ParseImageName(sourceImageName)
	if err != nil {
		return err
	}

	err = destinationRef.DeleteImage(context.Background(), sysCtx)
	if err != nil {
		fmt.Println("WARNING: could not delete last image. Continuing...")
	}

	copyOpts := &copy.Options{
		DestinationCtx: sysCtx,
	}
	_, err = copy.Image(context.TODO(), policyContext, destinationRef, sourceRef, copyOpts)
	if err != nil {
		return err
	}
	return nil
}
