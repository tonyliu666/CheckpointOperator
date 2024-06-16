package main

import (
	"context"
	"fmt"
	"os"

	"github.com/containers/buildah"
	"github.com/containers/image/v5/copy"
	"github.com/containers/image/v5/signature"
	"github.com/containers/image/v5/transports/alltransports"
	"github.com/containers/image/v5/types"
	"github.com/containers/storage"
	is "github.com/containers/image/v5/storage"
)

const (
	// tmpDir         = "/home/vagrant/"
	checkpointPath = "/home/vagrant/redis-test"
	checkpointName = "redis-test"
	path           = "/home/vagrant/redis"
	containerName  = "redis-test"
	tempdir        = "/home/vagrant"
)

// func main() {
// 	if reexec.Init() {
// 		return
// 	}
// 	sysCtx := &types.SystemContext{}
// 	policy, err := signature.DefaultPolicy(sysCtx)
// 	if err != nil {
// 		fmt.Println("obtaining default signature policy:", err)
// 	}
// 	policyContext, err := signature.NewPolicyContext(policy)
// 	if err != nil {
// 		fmt.Println("creating new signature policy context:", err)
// 	}

// 	checkpointName := filepath.Base(path)
// 	destinationImageName := fmt.Sprintf("containers-storage:localhost/%s:latest", checkpointName)
// 	destinationRef, err := alltransports.ParseImageName(destinationImageName)
// 	if err != nil {
// 		fmt.Println("destination error", err)
// 	}
// 	if destinationRef == nil {
// 		fmt.Println("destinationRef is nil")
// 	}
// 	fmt.Println("destinationRef: ", destinationRef)
// 	sourceImageName := fmt.Sprintf("oci-archive://%s/%s", tmpDir, checkpointName)
// 	sourceRef, err := alltransports.ParseImageName(sourceImageName)
// 	if err != nil {
// 		fmt.Println("source reference", err)
// 	}
// 	fmt.Println("sourceRef: ", sourceRef)

// 	destinationRef.DeleteImage(context.Background(), sysCtx)
// 	if err != nil {
// 		fmt.Println("WARNING: could not delete last image. Continuing...", err)
// 	}

//		copyOpts := &copy.Options{
//			DestinationCtx: sysCtx,
//			// SourceCtx:      sysCtx,
//		}
//		manifest, err := copy.Image(context.TODO(), policyContext, destinationRef, sourceRef, copyOpts)
//		if err != nil {
//			fmt.Println("copy image error", err)
//		}
//		fmt.Println("manifest: ", string(manifest))
//	}
func main() {
	checkpointPrefix := "cr"
	checkpointImageName := fmt.Sprintf("%s/%s:latest", checkpointPrefix, checkpointName)
	buildStoreOptions, err := storage.DefaultStoreOptions()
	ctx := context.TODO()
	if err != nil {
		fmt.Println("storage.DefaultStoreOptions")
		fmt.Println(err)
	}
	buildStore, err := storage.GetStore(buildStoreOptions)

	if err != nil {
		fmt.Println("storage.GetStore")
		fmt.Println(err)
	}
	defer buildStore.Shutdown(false)
	builderOpts := buildah.BuilderOptions{
		FromImage: "scratch",
	}
	builder, err := buildah.NewBuilder(ctx, buildStore, builderOpts)

	if err != nil {
		fmt.Println("buildah.NewBuilder")
		fmt.Println(err)
	}
	defer builder.Delete()

	fmt.Println("create Image: builder + store setup complete")
	err = builder.Add("/", true, buildah.AddAndCopyOptions{}, checkpointPath)

	if err != nil {
		fmt.Println("builder.Add")
		fmt.Println(err)
	}
	fmt.Println("create Image: added archive")
	builder.ImageAnnotations["io.kubernetes.cri-o.annotations.checkpoint.name"] = containerName
	imageRef, err := is.Transport.ParseStoreReference(buildStore, checkpointImageName)

	if err != nil {
		fmt.Println("is.Transport.ParseStoreReference")
		fmt.Println(err)
	}
	fmt.Println("create Image: generated store reference")
	imageId, _, _, err := builder.Commit(ctx, imageRef, buildah.CommitOptions{})

	fmt.Println("create Image: committed")
	if err != nil {
		fmt.Println("builder.Commit")
		fmt.Println(err)
	}

	sysCtx := &types.SystemContext{}
	policy, err := signature.DefaultPolicy(sysCtx)
	if err != nil {
		fmt.Println("obtaining default signature policy:", err)
	}
	policyContext, err := signature.NewPolicyContext(policy)
	if err != nil {
		fmt.Println("creating new signature policy context:", err)
	}
	copyOpts := &copy.Options{
		DestinationCtx: sysCtx,
	}

	fmt.Println("create Image: parsed policies")
	exportName := fmt.Sprintf("oci-archive://%s/%s", tempdir, checkpointName)
	destinationRef, err := alltransports.ParseImageName(exportName)
	if err != nil {
		fmt.Println("is.Transport.ParseStoreReference")
		fmt.Println(err)
	}
	fmt.Println("create Image: parsed image name")
	_, err = copy.Image(context.TODO(), policyContext, destinationRef, imageRef, copyOpts)

	fmt.Println("create Image: copied image")
	if err != nil {
		fmt.Println("copy.Image")
		fmt.Println(err)
	}
	err = os.Remove(checkpointPath)
	fmt.Println("create Image: removed archive")
	if err != nil {
		fmt.Println("error while deleting checkpoint archive")
		fmt.Println(err)
	}

	// return imageId, checkpointImageName, nil
	fmt.Println(imageId, checkpointImageName)
}
