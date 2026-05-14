package operatorcli

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"text/tabwriter"
	"time"

	"github.com/kevholditch/ghostwire/pkg/controlapi"
	"github.com/kevholditch/ghostwire/pkg/protocol"
)

type config struct {
	controlURL string
	apiToken   string
	output     string
}

func Run(ctx context.Context, args []string, getenv func(string) string, stdout, stderr io.Writer) int {
	if getenv == nil {
		getenv = os.Getenv
	}
	cfg, remaining, err := parseConfig(args, getenv, stderr)
	if err != nil {
		return fail(stderr, err)
	}
	if len(remaining) < 2 || remaining[0] != "nodes" {
		return fail(stderr, fmt.Errorf("usage: ghostwire [--control-url URL] [--api-token TOKEN] [--output table|json] nodes list|get|peers [node_id]"))
	}
	if cfg.controlURL == "" {
		return fail(stderr, fmt.Errorf("GHOSTWIRE_CONTROL_URL or --control-url is required"))
	}
	if cfg.apiToken == "" {
		return fail(stderr, fmt.Errorf("GHOSTWIRE_API_TOKEN or --api-token is required"))
	}

	client := controlapi.NewClient(cfg.controlURL, cfg.apiToken)
	switch remaining[1] {
	case "list":
		if len(remaining) != 2 {
			return fail(stderr, fmt.Errorf("usage: ghostwire nodes list"))
		}
		nodes, err := client.ListNodes(ctx)
		if err != nil {
			return fail(stderr, err)
		}
		return finish(stdout, stderr, cfg.output, nodes, nodes.Nodes)
	case "get":
		if len(remaining) != 3 {
			return fail(stderr, fmt.Errorf("usage: ghostwire nodes get <node_id>"))
		}
		node, err := client.GetNode(ctx, remaining[2])
		if err != nil {
			return fail(stderr, err)
		}
		return finish(stdout, stderr, cfg.output, node, []protocol.Node{node})
	case "peers":
		if len(remaining) != 3 {
			return fail(stderr, fmt.Errorf("usage: ghostwire nodes peers <node_id>"))
		}
		nodes, err := client.ListNodePeers(ctx, remaining[2])
		if err != nil {
			return fail(stderr, err)
		}
		return finish(stdout, stderr, cfg.output, nodes, nodes.Nodes)
	default:
		return fail(stderr, fmt.Errorf("unknown nodes command %q", remaining[1]))
	}
}

func parseConfig(args []string, getenv func(string) string, stderr io.Writer) (config, []string, error) {
	cfg := config{
		controlURL: getenv("GHOSTWIRE_CONTROL_URL"),
		apiToken:   getenv("GHOSTWIRE_API_TOKEN"),
		output:     "table",
	}
	flags := flag.NewFlagSet("ghostwire", flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.StringVar(&cfg.controlURL, "control-url", cfg.controlURL, "control-plane base URL")
	flags.StringVar(&cfg.apiToken, "api-token", cfg.apiToken, "API bearer token")
	flags.StringVar(&cfg.output, "output", cfg.output, "output format: table or json")
	if err := flags.Parse(args); err != nil {
		return config{}, nil, err
	}
	if cfg.output != "table" && cfg.output != "json" {
		return config{}, nil, fmt.Errorf("unsupported output %q", cfg.output)
	}
	return cfg, flags.Args(), nil
}

func finish(stdout, stderr io.Writer, output string, jsonValue any, tableRows []protocol.Node) int {
	if err := writeValue(stdout, output, jsonValue, tableRows); err != nil {
		return fail(stderr, err)
	}
	return 0
}

func fail(stderr io.Writer, err error) int {
	if _, writeErr := fmt.Fprintln(stderr, err); writeErr != nil {
		return 1
	}
	return 1
}

func writeValue(stdout io.Writer, output string, jsonValue any, tableRows []protocol.Node) error {
	if output == "json" {
		encoder := json.NewEncoder(stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(jsonValue); err != nil {
			return fmt.Errorf("write json output: %w", err)
		}
		return nil
	}
	return writeNodesTable(stdout, tableRows, time.Now())
}

func writeNodesTable(stdout io.Writer, nodes []protocol.Node, now time.Time) error {
	table := tabwriter.NewWriter(stdout, 0, 0, 2, ' ', 0)
	if _, err := fmt.Fprintln(table, "NODE ID\tHOSTNAME\tGHOSTWIRE IP\tENDPOINT\tSTATUS\tLAST SEEN"); err != nil {
		return fmt.Errorf("write table header: %w", err)
	}
	for _, node := range nodes {
		if _, err := fmt.Fprintf(table, "%s\t%s\t%s\t%s\t%s\t%s\n", node.NodeID, node.Hostname, node.GhostwireIP, node.Endpoint, node.Status, relativeAge(now.Sub(node.LastSeen))); err != nil {
			return fmt.Errorf("write table row: %w", err)
		}
	}
	if err := table.Flush(); err != nil {
		return fmt.Errorf("flush table output: %w", err)
	}
	return nil
}

func relativeAge(age time.Duration) string {
	if age < 0 {
		age = 0
	}
	switch {
	case age < time.Minute:
		return fmt.Sprintf("%ds ago", int(age.Seconds()))
	case age < time.Hour:
		return fmt.Sprintf("%dm ago", int(age.Minutes()))
	case age < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(age.Hours()))
	default:
		return fmt.Sprintf("%dd ago", int(age.Hours()/24))
	}
}
