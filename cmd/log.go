package cmd

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"
	
	"github.com/TwinProduction/go-color"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/spf13/cobra"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	
	"github.com/shipyard-run/shipyard/pkg/clients"
	"github.com/shipyard-run/shipyard/pkg/config"
	"github.com/shipyard-run/shipyard/pkg/shipyard"
	"github.com/shipyard-run/shipyard/pkg/utils"
)

// print cli utility debug
const debug = false
var testKubeCfgFile = os.Getenv("KUBECONFIG")

// bpLogs defines the options for obtaining logs for a shipyard blueprint that is currently running
type bpLogs struct {
	// stdOud
	commonOut *os.File
	// print cli utility debug
	debugOn         bool
	// follow stdout, stdErr options for container, pods
	dockerLogsOpts types.ContainerLogsOptions
	podLogOptions v1.PodLogOptions
	// shipyard engine
	engineClients *shipyard.Clients
	// terminal colors
	once      sync.Once
	colors         []string
	nextColorIndex int
}

func logCmd(out *os.File, engine shipyard.Engine) *cobra.Command {
	return &cobra.Command{
		Use:   "log -- <command> ... ",
		Short: "Tails logs for the all containers of the currently active blueprint",
		Long:  `Tails logs for the all containers of the currently active blueprint`,
		Example: `
# Tail logs for either all docker containers or all cluster pods in a in the stack
shipyard log containers

# Tail logs for a kubernetes cluster
shipyard log kubernetes
	`,
		Args: cobra.ArbitraryArgs,
		Run: func(cmd *cobra.Command, args []string) {
			// initialise and set engine
			slog := InitSLogs(out)
			// Stop tailing logs on user interrupt
			closeLogs, ctx, cancel := catchInterrupt()
			// parseInput and start logging to StdOut
			parseInput(args, slog, engine, ctx)
			// cancel all streams on user interrupt
			<-closeLogs
			cancel()
			if debug {
				time.Sleep(500 * time.Millisecond) // ensures all close statements gets printed
			}
		},
	}
}

// parseInput parse cli inputs and starts streaming the logs.
// todo -> better cli parsing
func parseInput(args []string, slog *bpLogs, engine shipyard.Engine, ctx context.Context) {
	if len(args) == 0 {
		printStack(slog.commonOut)
		// `go test` throws error if os.exit() is called during a test case.
		// This check only ensure os.exit() is called only in case of StdOut
		if slog.commonOut == os.Stdout{
			os.Exit(0)
		}else {
			return
		}
	}
	slog.engineClients = engine.GetClients()

	if strings.Compare(args[0], "containers") == 0 { // `shipyard log containers`
		if !slog.dockerLogs(ctx, slog.engineClients.Docker) {
			fmt.Println("Could not get docker containers")
			os.Exit(1)
		}
	} else if strings.Compare(args[0], "kubernetes") == 0 { // `shipyard log kubernetes`
		if !slog.kubernetesLogs(ctx, slog.engineClients.Kubernetes) {
			fmt.Println("Could not load KubeConfig / Connect to Kubernetes")
			os.Exit(1)
		}
	} else {
		fmt.Println("Bad args")
		os.Exit(1)
	}
}
// printStack in case of `shipyard log`
// todo improve parsing
func printStack(out *os.File){
	c := config.New()
	err := c.FromJSON(utils.StatePath())
	if err != nil {
		fmt.Println("Unable to load state", err)
		os.Exit(1)
	}
	stack := make(map[string][]string)
	for _, r := range c.Resources {
		// if string(r.Info().Type) == "k8s_cluster" || string(r.Info().Type) == "container"{
			stack[string(r.Info().Type)] = append(stack[string(r.Info().Type)], r.Info().Name)
		// }
	}
	for typ, names := range stack{
		_, _ = fmt.Fprintln(out, fmt.Sprintf("%-8s\t%s", typ, names))
	}
}
// catchInterrupt sets up a os signal catch routine to stop all streams
// and exit the shipyard cli utility. It return a os.signal channel and
// contextWithCancel
func catchInterrupt() (chan os.Signal, context.Context, context.CancelFunc) {
	closeLogs := make(chan os.Signal, 1)
	signal.Notify(closeLogs, os.Interrupt, syscall.SIGHUP,
		syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	// to cancel engine.client log streams
	ctx, cancel := context.WithCancel(context.Background())
	return closeLogs, ctx, cancel
}
func printDebug(i ...interface{}) {
	if debug {
		fmt.Println(i...)
	}
}
// InitSLogs creates a bpLogs object
func InitSLogs(out *os.File) *bpLogs {
	return &bpLogs{
		commonOut:      out,
		debugOn:        debug,
		once:           sync.Once{},
		colors:         nil,
		nextColorIndex: 0,
		dockerLogsOpts: types.ContainerLogsOptions{
			ShowStdout: true, // always true
			ShowStderr: true, // can be false
			Follow:     true, // always true, can stop with ctrl+c
		},
		podLogOptions: v1.PodLogOptions{
			TypeMeta:  metav1.TypeMeta{},
			Container: "",
			Follow:    true, // always true, can stop with ctrl+c
		},
	}
}

func (slog *bpLogs)dockerLogs(ctx context.Context, client clients.Docker) bool {
	defer ctx.Done()
	containers, done := getDockerInfo(ctx, client)
	if !done {
		return false
	}
	for _, container := range containers {
		if logReader, err := client.ContainerLogs(ctx, container.ID, slog.dockerLogsOpts); err == nil {
			// colorize container's prefix
			sOutPrefix := color.Ize(slog.nextColor(), container.Names[0][1:])
			if slog.dockerLogsOpts.ShowStderr {
				// logReader has both stdout and stderr, de-mux them
				go slog.splitAndAddReaders(ctx, sOutPrefix, logReader)
			} else {
				slog.addReader(ctx, sOutPrefix, logReader)
			}
		}
	}
	return true
}

func (slog *bpLogs)kubernetesLogs(ctx context.Context, client clients.Kubernetes) bool {
	if _, err := os.Stat(testKubeCfgFile); err != nil{
		return false
		
	}
	config, err := client.SetConfig(testKubeCfgFile)
	if err != nil {
		printDebug(err.Error())
		return false
	}
	client = config
	pl, err := client.GetPods("")
	if err != nil {
		printDebug(err.Error())
		return false
	}
	logging := false
	for _, pod := range pl.Items {
		if strings.Compare(string(pod.Status.Phase), "Running") == 0 &&
			strings.Contains(pod.Namespace, "default") { // todo replace strings.contains with label selector?
			go func(ctx context.Context, pod v1.Pod) {
				if logReader, err := client.GetPodLogs(ctx, pod.Name, pod.Namespace); err == nil {
					sOutPrefix := color.Ize(slog.nextColor(), pod.Name)
					slog.addReader(ctx, sOutPrefix, logReader)
				}
			}(ctx, pod)
			logging = true // true if at least one pod is running
		}
	}
	return logging
}

// getDockerInfo returns a list of docker containers along with a flag to indicate any error
func getDockerInfo(ctx context.Context, client clients.Docker) ([]types.Container, bool) {
	filter := filters.NewArgs() // equivalent for k8 ?
	filter.Add("name", "shipyard")
	filter.Add("status", "running")
	containers, err := client.ContainerList(ctx, types.ContainerListOptions{
		Filters: filter,
	})
	if err != nil || len(containers) == 0 {
		return nil, false
	}
	return containers, true
}

// splitAndAddReaders splits a io.ReadCloser that is multiplexed as a common StdOut + StdErr
// into two separate io.ReadCloser. It then calls addReader for both streams
func (slog *bpLogs)splitAndAddReaders(ctx context.Context, prefix string, logReader io.ReadCloser) {
	// Create io pipes to split logReader to StdOut and StdErr
	stdOutReadr, dstOut := io.Pipe()
	stdErrReadr, dstErr := io.Pipe()

	// close io pipes when ctx is cancelled
	go waitClose(prefix, ctx, logReader, dstOut, dstErr, stdOutReadr, stdErrReadr)
	// de-multiplex logReader to stdout and stderr streams
	go deMuxStream(prefix, dstOut, dstErr, logReader)

	sOutPrefix := prefix
	sErrPrefix := sOutPrefix + color.Ize(color.Red, "*")
	slog.addReader(ctx, sOutPrefix, stdOutReadr)
	slog.addReader(ctx, sErrPrefix, stdErrReadr)
}
// addReader reads from the io.ReadCloser, adds the prefix and prints to the common out
func (slog *bpLogs)addReader(ctx context.Context, prefix string, logReader io.ReadCloser) {
	go func() {
		defer func(logReader io.ReadCloser) {
			_ = logReader.Close()
		}(logReader)
		scanner := bufio.NewScanner(logReader)
		printDebug("Added reader for", prefix)
		for {
			select {
			case <- ctx.Done():
				printDebug("stopped reader for", prefix)
				return
			default:
				for scanner.Scan() {
					log := "[" + prefix + "] " + scanner.Text()
					_, err := fmt.Fprintln(slog.commonOut, log)
					if err != nil {
						fmt.Println(err.Error())
					}
				}
			}
		}
	}()
}

// deMuxStream splits readCloser into StdErr and StdOut if readCloser was written
// to in this way
func deMuxStream(name string, dstOut *io.PipeWriter, dstErr *io.PipeWriter, logReader io.ReadCloser) {
	_, _ = stdcopy.StdCopy(dstOut, dstErr, logReader)
	printDebug("Stopped de-mux-ing log streams for ", name)
}

// waitClose waits for the context to finish then closes all reader and writers
func waitClose(name string, ctx context.Context, logReader io.ReadCloser, dstOut *io.PipeWriter,
	dstErr *io.PipeWriter, stdOutReadr *io.PipeReader, stdErrReadr *io.PipeReader) {
	<-ctx.Done()
	_ = logReader.Close()
	_ = dstOut.Close()
	_ = dstErr.Close()
	_ = stdOutReadr.Close()
	_ = stdErrReadr.Close()
	printDebug("Stopped reading log streams for ", name)
}

// nextColor returns a color for the prefix in the case of combined logs
func (slog *bpLogs)nextColor() string {
	slog.once.Do(func() {
		// one time initialization
		slog.colors = append(slog.colors, color.Blue)
		slog.colors = append(slog.colors, color.Green)
		slog.colors = append(slog.colors, color.Purple)
		slog.colors = append(slog.colors, color.Yellow)
		slog.colors = append(slog.colors, color.Bold)
		slog.colors = append(slog.colors, color.Gray)
		// excluded red as it is used later to denote *StdErr*
		slog.nextColorIndex = 0
	})
	// rotate index
	if slog.nextColorIndex == len(slog.colors) {
		slog.nextColorIndex = 0
	}
	c := slog.colors[slog.nextColorIndex]
	slog.nextColorIndex++
	return c
}