package destination

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	supconfig "github.com/supaback/supaback/internal/config"
)

type s3Dest struct {
	client *s3.Client
	bucket string
	prefix string
}

func newS3(cfg supconfig.S3Config) (*s3Dest, error) {
	region := cfg.Region
	if region == "" {
		region = "us-east-1"
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(context.Background(),
		awsconfig.WithRegion(region),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			cfg.AccessKeyID, cfg.SecretAccessKey, "",
		)),
	)
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}

	opts := []func(*s3.Options){
		func(o *s3.Options) { o.UsePathStyle = cfg.ForcePathStyle },
	}
	if cfg.Endpoint != "" {
		opts = append(opts, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(cfg.Endpoint)
		})
	}

	return &s3Dest{
		client: s3.NewFromConfig(awsCfg, opts...),
		bucket: cfg.Bucket,
		prefix: cfg.Prefix,
	}, nil
}

func (d *s3Dest) s3Key(key string) string {
	if d.prefix != "" {
		return d.prefix + "/" + key
	}
	return key
}

func (d *s3Dest) Write(ctx context.Context, key string, r io.Reader) (int64, error) {
	body, err := io.ReadAll(r)
	if err != nil {
		return 0, fmt.Errorf("read data: %w", err)
	}
	_, err = d.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(d.bucket),
		Key:           aws.String(d.s3Key(key)),
		Body:          bytes.NewReader(body),
		ContentLength: aws.Int64(int64(len(body))),
	})
	if err != nil {
		return 0, fmt.Errorf("s3 put object: %w", err)
	}
	return int64(len(body)), nil
}

func (d *s3Dest) List(ctx context.Context, prefix string) ([]string, error) {
	s3Prefix := ""
	if d.prefix != "" {
		s3Prefix = d.prefix + "/"
	}
	if prefix != "" {
		s3Prefix += prefix + "/"
	}

	out, err := d.client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket:    aws.String(d.bucket),
		Prefix:    aws.String(s3Prefix),
		Delimiter: aws.String("/"),
	})
	if err != nil {
		return nil, fmt.Errorf("s3 list: %w", err)
	}

	var names []string
	for _, cp := range out.CommonPrefixes {
		// cp.Prefix = "myprefix/2024-05-01/" → extract "2024-05-01"
		name := strings.TrimPrefix(aws.ToString(cp.Prefix), s3Prefix)
		name = strings.TrimSuffix(name, "/")
		if name != "" {
			names = append(names, name)
		}
	}
	return names, nil
}

func (d *s3Dest) Delete(ctx context.Context, key string) error {
	s3Prefix := d.s3Key(key) + "/"
	paginator := s3.NewListObjectsV2Paginator(d.client, &s3.ListObjectsV2Input{
		Bucket: aws.String(d.bucket),
		Prefix: aws.String(s3Prefix),
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return fmt.Errorf("s3 list for delete: %w", err)
		}
		if len(page.Contents) == 0 {
			continue
		}
		ids := make([]s3types.ObjectIdentifier, len(page.Contents))
		for i, obj := range page.Contents {
			ids[i] = s3types.ObjectIdentifier{Key: obj.Key}
		}
		if _, err := d.client.DeleteObjects(ctx, &s3.DeleteObjectsInput{
			Bucket: aws.String(d.bucket),
			Delete: &s3types.Delete{Objects: ids, Quiet: aws.Bool(true)},
		}); err != nil {
			return fmt.Errorf("s3 delete objects: %w", err)
		}
	}
	return nil
}

func (d *s3Dest) String() string {
	return fmt.Sprintf("s3://%s/%s", d.bucket, d.prefix)
}
