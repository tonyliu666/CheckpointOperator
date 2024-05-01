manifest=$(curl -H "Accept: application/vnd.docker.distribution.manifest.v2+json" http://10.85.0.26:5000/v2/nginx/manifests/latest)
configDigest=$(echo "$manifest" | jq -r '.config.digest')
configBlob=$(curl "http:localhost:5001/v2/golang/blobs/$configDigest")
creationTime=$(echo "$configBlob" | jq -r '.created')
curl 10.85.0.26:5000/v2/checkpoint-image/tags/list
# get the manifests with the header which only accept oci format image
manifest=$(curl -H "Accept: application/vnd.oci.image.manifest.v1+json" http://10.85.0.26:5000/v2/checkpoint-image/manifests/latest | jq -r '.config.digest')
# configDigest=$(echo "$manifest" | jq -r '.config.digest')
# configBlob=$(curl "http://10.85.0.26:5000/v2/checkpoint-image/blobs/$configDigest")
configBlob=$(curl "http://10.85.0.26:5000/v2/checkpoint-image/blobs/$manifest" | jq '.created')

## skopeo can call this: return the json "Created": "2024-04-22T06:47:18.097636186Z",
skopeo inspect --tls-verify=false docker://10.85.0.26:5000/checkpoint-image:latest