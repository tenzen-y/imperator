package cmd

import (
	"github.com/spf13/cobra"
	"github.com/tenzen-y/imperator/pkg/version"
)

type options struct {
	metricsAddr          string
	probeAddr            string
	enableLeaderElection bool
}

func NewRootCmd() (*cobra.Command, error) {
	o := &options{}
	rootCmd := &cobra.Command{
		Use:     "imperator",
		Version: version.Version,
		Short:   "imperator",
		Long:    `imperator`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return o.run()
		},
	}
	f := rootCmd.Flags()
	f.StringVar(&o.metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	f.StringVar(&o.probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	f.BoolVar(&o.enableLeaderElection, "leader-elect", true,
		"Enable leader election for controller manager. Enabling this will ensure there is only one active controller manager.")

	return rootCmd, nil
}
