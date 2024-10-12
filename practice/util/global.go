package util

type MigrationInfo struct {
	PodName              string
	Deployment           string
	SourceNamespace      string
	DestinationNode      string
	DestinationNamespace string
	Specify              []string
}

func FillinGlobalVariables(PodName string, Deployment string, SourceNamespace string, DestinationNode string, DestinationNamespace string, Specify []string) {
	info := MigrationInfo{}
	info.Deployment = Deployment
	info.SourceNamespace = SourceNamespace
	info.DestinationNode = DestinationNode
	info.DestinationNamespace = DestinationNamespace
	info.Specify = Specify
	ProcessPodsMap[PodName] = info
}

// handle the pods that are being processed and remove the entry when the migration is done
// the value is MigrationInfo struct
var ProcessPodsMap = make(map[string]interface{})
