package daemon

import (
	"context"
	"fmt"
	"os"
	"time"

	kataTypes "github.com/openshift/kata-operator/api/v1"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

// KataActions declares the possible actions the daemon can take.
type KataActions interface {
	Install(kataConfigResourceName string) error
	Upgrade() error
	Uninstall(kataConfigResourceName string) error
}

type updateStatus = func(a *kataTypes.KataConfigStatus)

func updateKataConfigStatus(kataClient client.Client, kataConfigResourceName string, us updateStatus) (err error) {
	var kataConfig kataTypes.KataConfig

	return wait.Poll(2*time.Second, 2*time.Minute, func() (bool, error) {
		err := kataClient.Get(context.Background(), client.ObjectKey{Name: kataConfigResourceName}, &kataConfig)
		if err != nil {
			if client.IgnoreNotFound(err) != nil {
				return false, errors.Wrapf(err, "Unable to get KataConfig %q", kataConfigResourceName)
			}
			return false, nil
		}

		us(&kataConfig.Status)

		err = kataClient.Status().Update(context.Background(), &kataConfig)
		if err != nil {
			return false, nil
		}

		return true, nil
	})
}

func getFailedNode(err error) (fn kataTypes.FailedNodeStatus, retErr error) {
	nodeName, hErr := getNodeName()
	if hErr != nil {
		return kataTypes.FailedNodeStatus{}, hErr
	}

	return kataTypes.FailedNodeStatus{
		Name:  nodeName,
		Error: fmt.Sprintf("%+v", err),
	}, nil
}

func getHostName() (string, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return "", err
	}
	return hostname, nil
}

func getNodeName() (string, error) {
	return getHostName()
}
