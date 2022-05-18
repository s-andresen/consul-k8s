package read

import (
	"sync"

	"github.com/hashicorp/consul-k8s/cli/common"
	"github.com/hashicorp/consul-k8s/cli/common/flag"
	"k8s.io/client-go/kubernetes"
)

type ClustersSubcommand struct {
	*common.BaseCommand

	kubernetes kubernetes.Interface

	set *flag.Sets

	flagPodName   string
	flagNamespace string
	flagJSON      bool

	flagKubeConfig  string
	flagKubeContext string

	once sync.Once
	help string
}
