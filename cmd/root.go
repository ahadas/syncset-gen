package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"

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
	viewCmd.Flags().StringVarP(&input, "input", "i", "disk", fmt.Sprintf("Input source. One of: %v", inputSources))
	viewCmd.Flags().BoolVarP(&wait, "wait", "w", false, "Last resource needs to wait for custom resource definition to be exposed")
	RootCmd.AddCommand(viewCmd)
}

var selector, clusterName, resources, patches, name, output, input string
var wait bool

var outputPrinters = []string{"json", "yaml"}
var inputSources = []string{"disk", "stdin"}

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
		if !contains(inputSources, input) {
			return fmt.Errorf("unsupported input source %s allowed input sources are: %v",
				input, inputSources)
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		prefix := "ss"
		create := func()interface{} { return pkg.CreateSyncSet(args[0], clusterName, resources, patches) }
		create2 := func()interface{} { return nil }
		// override for selectorsyncsets
		if sss := (clusterName == ""); sss {
			prefix = "sss"
			var stdin []byte
			if input == "stdin" {
				stdin, _ = ioutil.ReadAll(os.Stdin)
				if wait {
					index := bytes.LastIndex(stdin, pkg.YAMLSeparator)
					cr := stdin[index:]
					create2 = func()interface{} { return pkg.CreateSelectorSyncSet(args[0]+"-cr", selector, resources, patches, cr) }
					stdin = stdin[:index]
				}
			}
			create = func()interface{} { return pkg.CreateSelectorSyncSet(args[0], selector, resources, patches, stdin) }
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
		fmt.Printf("%s", getOutputStr(ss))
		if ss2 := create2(); ss2 != nil {
			// assumed yaml output
			fmt.Printf("---\n%s", getOutputStr(ss2))
		}
		fmt.Printf("\n\n")
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
