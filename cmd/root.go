package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"

	"github.com/matt-simons/ss/pkg"
	"github.com/spf13/cobra"
	"sigs.k8s.io/yaml"
)

func init() {
	viewCmd.Flags().StringVarP(&selector, "selector", "s", "", "The selector key/value pair used to match the SelectorSyncSet to Cluster(s)")
	viewCmd.Flags().StringVarP(&clusterName, "cluster-name", "c", "", "The cluster name used to match the SyncSet to a Cluster")
	viewCmd.Flags().StringVarP(&resources, "resources", "r", "", "The directory of resource manifest files to use")
	viewCmd.Flags().StringVarP(&patches, "patches", "p", "", "The directory of patch manifest files to use")
	viewCmd.Flags().StringVarP(&output, "output", "o", "json", fmt.Sprintf("Output format. One of: %v", outputPrinters))
	RootCmd.AddCommand(viewCmd)
}

var selector, clusterName, resources, patches, name, output string

var outputPrinters = []string{"json", "yaml"}

var RootCmd = &cobra.Command{
	Use:   "ss",
	Short: "SyncSet/SelectorSyncSet generator.",
	Long:  ``,
}

var viewCmd = &cobra.Command{
	Use:   "view",
	Short: "Parses a manifest directory and prints a SyncSet/SelectorSyncSet representation of the objects it contains.",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if selector == "" && clusterName == "" {
			return errors.New("one of --selector or --cluster-name must be specified")
		}
		if selector != "" && clusterName != "" {
			return errors.New("only one of --selector or --cluster-name can be specified")
		}
		if len(args) < 1 {
			return errors.New("name must be specified")
		}
		if !contains(outputPrinters, output) {
			return fmt.Errorf("unable to match a printer suitable for the output format %s allowed formats are: %v",
				output, outputPrinters)
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		prefix := "ss"
		create := func()interface{} { return pkg.CreateSyncSet(args[0], clusterName, resources, patches) }
		// override for selectorsyncsets
		if sss := (clusterName == ""); sss {
			prefix = "sss"
			create = func()interface{} { return pkg.CreateSelectorSyncSet(args[0], selector, resources, patches) }
		}

		secrets := pkg.TransformSecrets(args[0], prefix, resources)
		for _, s := range secrets {
			j, err := json.MarshalIndent(&s, "", "    ")
			if err != nil {
				log.Fatalf("error: %v", err)
			}
			fmt.Printf("%s\n", string(j))
		}
		ss := create()
		fmt.Printf("%s\n\n", getOutputStr(ss))
	},
}

// getOutputStr returns the output as json or yaml string
func getOutputStr(ss interface{}) string {
	var marshal func(interface{}) ([]byte, error)
	switch o := output; o {
	case "yaml":
		marshal = func(ss interface{}) ([]byte, error) { return yaml.Marshal(&ss) }
	default:
		marshal = func(ss interface{}) ([]byte, error) { return json.MarshalIndent(&ss, "", "    ") }
	}
	b, err := marshal(ss)
	if err != nil {
		log.Fatalf("error: %v", err)
	}
	return string(b)
}

// contains tells whether array a contains member x.
func contains(a []string, x string) bool {
	for _, n := range a {
		if x == n {
			return true
		}
	}
	return false
}
