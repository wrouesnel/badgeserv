package main

import (
	"context"
	"fmt"
	"github.com/alecthomas/kong"
	"github.com/wrouesnel/badgeserv/api/v1"
	"github.com/wrouesnel/badgeserv/assets"
	"gocloud.dev/blob"
	_ "gocloud.dev/blob/azureblob"
	_ "gocloud.dev/blob/fileblob"
	_ "gocloud.dev/blob/gcsblob"
	_ "gocloud.dev/blob/s3blob"
	"io"
	"io/fs"
	"time"

	gap "github.com/muesli/go-app-paths"
	"go.uber.org/zap"
	"os"
	"path"
)

const name = "nmap-api"
const description = `nmap-api server frontend`

var Version string

var CLI struct {
	Logging struct {
		Level  string `help:"logging level" default:"info"`
		Format string `help:"logging format (${enum})" enum:"console,json" default:"json"`
	} `embed:"" prefix:"logging."`

	Debug struct {
		Assets struct {
			List struct {
			} `cmd:"" help:"list embedded files in the binary"`
			Cat struct {
				Filename string `arg:"" name:"filename" help:"embedded file to emit to stdout"`
			} `cmd:"" help:"output the specifid file to stdout"`
		} `cmd:""`
	} `cmd:""`

	Redis struct {
		Server    string        `help:"Redis server network addr" required:"true"`
		Username  string        `help:"Redis server network addr"`
		Password  string        `help:"Redis server network addr"`
		Db        int           `help:"Redis DB number" default:"1"`
		KeyPrefix string        `help:"Common key prefix to use for all keys from the application" default:"nmap-api"`
		RedisWait time.Duration `help:"Time to wait at startup for redis to become available" default:"30s"`
	} `embed:"" prefix:"redis."`
	ObjectStore struct {
		BucketUrl     string        `help:"bucket url to use e.g. s3://mybucket" required:"true"`
		BucketTimeout time.Duration `help:"Timeout before failing bucket connection" default:"5s"`
		BucketWait    time.Duration `help:"Time to wait for bucket to become usable" default:"30s"`
		BucketPrefix  string        `help:"fixed-prefix for all bucket values" default:"nmap-api/"`
	} `embed:"" prefix:"object-store."`

	Scheduling struct {
		HostLockTTL      time.Duration `help:"Default TTL before hosts locks are dropped" default:"1s"`
		RecoveryInterval time.Duration `help:"Interval between scheduling a recovery scan" default:"30s"`
	} `embed:"" prefix:"scheduling."`

	Api struct {
		Prefix string `help:"Prefix the API is bing served under, if any"`
		Port   int    `help:"Port to serve on" default:"8080"`
	} `cmd:""`

	Runner struct {
		Prefix string `help:"Prefix the API is being served under, if any"`
		Port   int    `help:"Port to serve on" default:"8080"`
	} `cmd:""`
}

func main() {
	exit_code := real_main(os.Stdout, os.Stderr)
	os.Exit(exit_code)
}

func real_main(stdout io.Writer, stderr io.Writer) int {
	appCtx, appCancel := context.WithCancel(context.Background())
	defer appCancel()

	deferred_logs := []string{}

	// Handle a sensible configuration loader path
	scope := gap.NewScope(gap.User, name)
	config_dirs, err := scope.ConfigDirs()
	if err != nil {
		deferred_logs = append(deferred_logs, err.Error())
	}
	config_paths := []string{}
	for _, config_dir := range config_dirs {
		config_paths = append(config_paths, path.Join(config_dir, fmt.Sprintf("%s.json", name)))
	}
	config_paths = append([]string{".nmap-api.json", path.Join(os.Getenv("HOME"), ".nmap-api.json")}, config_paths...)

	// Command line parsing can now happen
	ctx := kong.Parse(&CLI,
		kong.Description(description),
		kong.Configuration(kong.JSON, config_paths...))

	// Initialize logging as soon as possible
	// TODO: CLI configuration of logging
	logconfig := zap.NewProductionConfig()
	if err := logconfig.Level.UnmarshalText([]byte(CLI.Logging.Level)); err != nil {
		deferred_logs = append(deferred_logs, err.Error())
	}
	logconfig.Encoding = CLI.Logging.Format

	logger, err := logconfig.Build()
	if err != nil {
		// Error unhandled since this is a very early failure
		_, _ = io.WriteString(stderr, "Failure while building logger")
		return 1
	}

	// Install as the global logger
	zap.ReplaceGlobals(logger)

	// Emit deferred logs
	logger.Info("Using config paths", zap.Strings("config_paths", config_paths))
	for _, line := range deferred_logs {
		logger.Error(line)
	}

	logger.Info("Redis", zap.String("redis.server", CLI.Redis.Server))
	logger.Info("Object Store", zap.String("object_store.objectBucket-url", CLI.ObjectStore.BucketUrl))

	switch ctx.Command() {
	case "api", "runner":
		logger.Info("Opening redis connection")
		redisClient := redis.NewClient(&redis.Options{
			Addr:     CLI.Redis.Server,
			Username: CLI.Redis.Username,
			Password: CLI.Redis.Password,
			DB:       CLI.Redis.Db,
		})
		appCtx.Deadline()
		if _, err := CheckRedisConnection(appCtx, CLI.Redis.RedisWait, redisClient); err != nil {
			logger.Error("Redis ping failed", zap.Error(err))
			return 1
		}
		logger.Info("Redis appears usable")

		logger.Info("Initializing redis message queue client")
		redisMessageQueueClientErrorCh := make(chan error)
		rmqClient, err := rmq.OpenConnectionWithRedisClient(CLI.Redis.KeyPrefix, redisClient, redisMessageQueueClientErrorCh)
		if err != nil {
			logger.Error("Error establishing Redis Message Queue", zap.Error(err))
			return 1
		}
		go func() {
			// TODO: I'm unfamiliar with this library, so what we should do here other then collect messages as
			// metrics is not clear. Metrics is probably the best answer - lots of rejects or failures generally
			// means either redis is having a problem, or this client (or its host or whatever) is.
			for err := range redisMessageQueueClientErrorCh {
				zap.L().Error("Redis Message Queue Error", zap.Error(err))
			}
		}()

		logger.Info("Initializing redis lock client")
		redisLockClient := redislock.New(redisClient)

		logger.Info("Opening objectBucket connection")
		bucketAvailableCtx, bucketAvailableCancelFn := context.WithDeadline(appCtx, time.Now().Add(CLI.ObjectStore.BucketWait))
		defer bucketAvailableCancelFn()
		bucketTimeoutCtx, bucketTimeoutCtxCancelFn := context.WithDeadline(bucketAvailableCtx, time.Now().Add(CLI.ObjectStore.BucketTimeout))
		defer bucketTimeoutCtxCancelFn()
		objectBucket, err := blob.OpenBucket(bucketTimeoutCtx, CLI.ObjectStore.BucketUrl)
		if err != nil {
			logger.Error("Could not open object-storage objectBucket", zap.Error(err))
			return 1
		}

		logger.Info("Checking objectBucket connection")
		if connected, err := CheckBucketConnection(bucketAvailableCtx, CLI.ObjectStore.BucketTimeout, objectBucket); !connected {
			logger.Error("Bucket list object failed during startup - check objectBucket configuration", zap.Error(err))
			return 1
		}
		logger.Info("Bucket appears usable")

		doRecovery := false
		switch ctx.Command() {
		case "api":
			doRecovery = false
		case "runner":
			doRecovery = true
		}

		scanSchedulerConfig := scanscheduler.ScanScedulerConfig{
			RedisClient:      redisClient,
			RmqConnection:    rmqClient,
			RedisLocker:      redisLockClient,
			ObjectBucket:     objectBucket,
			RedisKeyPrefix:   CLI.Redis.KeyPrefix,
			ObjectKeyPrefix:  CLI.ObjectStore.BucketPrefix,
			LockTTL:          CLI.Scheduling.HostLockTTL,
			DoRecovery:       doRecovery,
			RecoveryInterval: CLI.Scheduling.RecoveryInterval,
		}

		logger.Info("Initializing scan scheduler client", zap.Bool("do_recoveries", doRecovery))
		scheduler := scanscheduler.NewScanSchedulerClient(appCtx, &scanSchedulerConfig)
		if scheduler == nil {
			logger.Error("Scheduler failed to initialize")
			return 1
		}

		switch ctx.Command() {
		case "api":
			logger.Info("Starting API server")
			// Create the API
			apiInstance := api.NewApi(scheduler, Version)
			if apiInstance == nil {
				logger.Error("API failed to initialize")
				return 1
			}

			// Start the API
			if err := apiServer(CLI.Api.Port, CLI.Api.Prefix, apiInstance); err != nil {
				logger.Error("Error from server", zap.Error(err))
				return 1
			}
		case "runner":
			logger.Info("Starting Runner server")
			// Start the API
			if err := runnerServer(CLI.Runner.Port, CLI.Runner.Prefix, scheduler); err != nil {
				logger.Error("Error from server", zap.Error(err))
				return 1
			}
		default:
			logger.Error("BUG: command was not implemented in common server path", zap.String("command", ctx.Command()))
			return 1
		}

	case "debug assets list":
		fs.WalkDir(assets.Assets, ".", func(path string, d fs.DirEntry, err error) error {
			_, _ = fmt.Fprintf(stdout, "%s\n", path)
			return nil
		})
	case "debug assets cat":
		if content, err := assets.Assets.ReadFile(CLI.Debug.Assets.Cat.Filename); err == nil {
			_, _ = stdout.Write(content)
		} else {
			logger.Error("Error reading embedded file", zap.Error(err))
		}

	default:
		logger.Error("Not implemented", zap.String("command", ctx.Command()))
		return 1
	}

	logger.Info("Exiting normally")
	return 0
}

// CheckBucketConnection repeatedly attempts to access a bucket. This allows us to handle delayed availability
// situations while still eventually exiting.
func CheckBucketConnection(parentCtx context.Context, operationTimeout time.Duration, bucket *blob.Bucket) (bool, error) {
	// Try and list the bucket initially - if we can't then the bucket probably isn't usable.
	for {
		ctx, cancelFn := context.WithDeadline(parentCtx, time.Now().Add(operationTimeout))

		select {
		case <-parentCtx.Done():
			cancelFn()
			return false, parentCtx.Err()
		default:
			bucket_iter := bucket.List(nil)
			if _, err := bucket_iter.Next(ctx); err != nil {
				if err != io.EOF {
					cancelFn()
					continue
				}
			}
			// Connection successful.
			cancelFn()
			return true, nil
		}
	}
}

// CheckRedisConnection checks if redis is working.
func CheckRedisConnection(parentCtx context.Context, operationTimeout time.Duration, rdb *redis.Client) (bool, error) {
	//
	for {
		ctx, cancelFn := context.WithDeadline(parentCtx, time.Now().Add(operationTimeout))

		select {
		case <-parentCtx.Done():
			cancelFn()
			return false, parentCtx.Err()
		default:
			pingResult := rdb.Ping(ctx)
			cancelFn()
			if pingResult.Err() == nil {
				return true, nil
			}
		}
	}
}
