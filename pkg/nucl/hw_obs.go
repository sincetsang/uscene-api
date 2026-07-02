package nucl

import (
	"github.com/huaweicloud/huaweicloud-sdk-go-obs/obs"
)

func (nu *Nucleus) initHwObs() error {
	if !nu.Options.HwObs {
		return nil
	}
	return nu.withLog("HwObs", func() error {
		client, err := obs.New(nu.Config.HwObs.Ak, nu.Config.HwObs.Sk, nu.Config.HwObs.EndPoint, obs.WithSignature(obs.SignatureObs))
		if err != nil {
			return err
		}

		nu.HwObsClient = client
		return nil
	})
}
