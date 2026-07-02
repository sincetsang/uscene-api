package nucl

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/rekognition"
)

type AwsConfig struct {
	AwsAccessKeyID       string `toml:"aws_access_key_id"`
	AwsSecretAccessKey   string `toml:"aws_secret_access_key"`
	AwsStorageBucketName string `toml:"aws_storage_bucket_name"`
	AwsS3RegionName      string `toml:"aws_s3_region_name"`
	AwsS3CustomDomain    string `toml:"aws_s3_custom_domain"`
}

func (nu *Nucleus) initAws() error {
	if !nu.Options.Aws {
		return nil
	}
	return nu.withLog("Aws", func() error {
		sess, err := session.NewSession(&aws.Config{
			Region: aws.String(nu.Config.Aws.AwsS3RegionName),
			Credentials: credentials.NewStaticCredentials(
				nu.Config.Aws.AwsAccessKeyID,     // AWS Access Key
				nu.Config.Aws.AwsSecretAccessKey, // AWS Secret Key
				""),
		})
		if err != nil {
			return err
		}
		client := rekognition.New(sess)
		nu.AwsClient = client
		nu.AwsSession = sess
		return nil
	})
}
