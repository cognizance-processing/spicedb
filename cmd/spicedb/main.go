package main

import (
	"errors"
	"os"

	"github.com/rs/zerolog"
	"github.com/sercand/kuberesolver/v4"
	"github.com/spf13/cobra"
	"google.golang.org/grpc/balancer"
	_ "google.golang.org/grpc/xds"

	log "github.com/authzed/spicedb/internal/logging"
	consistentbalancer "github.com/authzed/spicedb/pkg/balancer"
	"github.com/authzed/spicedb/pkg/cmd"
	cmdutil "github.com/authzed/spicedb/pkg/cmd/server"
	"github.com/authzed/spicedb/pkg/cmd/testserver"
)

var errParsing = errors.New("parsing error")

func main() {
	// Enable Kubernetes gRPC resolver
	kuberesolver.RegisterInCluster()

	// Enable consistent hashring gRPC load balancer
	balancer.Register(consistentbalancer.NewConsistentHashringBuilder(cmdutil.ConsistentHashringPicker))

	log.SetGlobalLogger(zerolog.New(os.Stdout))

	// Create a root command
	rootCmd := cmd.NewRootCommand("spicedb")
	rootCmd.SetFlagErrorFunc(func(cmd *cobra.Command, err error) error {
		cmd.Println(err)
		cmd.Println(cmd.UsageString())
		return errParsing
	})
	cmd.RegisterRootFlags(rootCmd)

	// Add a version command
	versionCmd := cmd.NewVersionCommand(rootCmd.Use)
	cmd.RegisterVersionFlags(versionCmd)
	rootCmd.AddCommand(versionCmd)

	// Add migration commands
	migrateCmd := cmd.NewMigrateCommand(rootCmd.Use)
	cmd.RegisterMigrateFlags(migrateCmd)
	rootCmd.AddCommand(migrateCmd)

	// Add migration commands
	datastoreCmd, err := cmd.NewDatastoreCommand(rootCmd.Use)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to register datastore command")
	}

	cmd.RegisterDatastoreRootFlags(datastoreCmd)
	rootCmd.AddCommand(datastoreCmd)

	// Add head command.
	headCmd := cmd.NewHeadCommand(rootCmd.Use)
	cmd.RegisterHeadFlags(headCmd)
	rootCmd.AddCommand(headCmd)

	// Add server commands
	var serverConfig cmdutil.Config
	serveCmd := cmd.NewServeCommand(rootCmd.Use, &serverConfig)
	if err := cmd.RegisterServeFlags(serveCmd, &serverConfig); err != nil {
		log.Fatal().Err(err).Msg("failed to register server flags")
	}
	rootCmd.AddCommand(serveCmd)

	devtoolsCmd := cmd.NewDevtoolsCommand(rootCmd.Use)
	cmd.RegisterDevtoolsFlags(devtoolsCmd)
	rootCmd.AddCommand(devtoolsCmd)

	var testServerConfig testserver.Config
	testingCmd := cmd.NewTestingCommand(rootCmd.Use, &testServerConfig)
	cmd.RegisterTestingFlags(testingCmd, &testServerConfig)
	rootCmd.AddCommand(testingCmd)
	if err := rootCmd.Execute(); err != nil {
		if !errors.Is(err, errParsing) {
			log.Err(err).Msg("terminated with errors")
		}
		os.Exit(1)
	}
}
