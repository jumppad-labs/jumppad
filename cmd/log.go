package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	
	"github.com/TwinProduction/go-color"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/spf13/cobra"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	
	"github.com/shipyard-run/shipyard/pkg/clients"
	"github.com/shipyard-run/shipyard/pkg/clients/streams"
)

var logCmd = &cobra.Command{
	Use:   "log",
	Short: "Tails logs for the all containers of the currently active blueprint",
	Long:  `Tails logs for the all containers of the currently active blueprint`,
	Example: `
  shipyard log
	`,
	Args: cobra.ArbitraryArgs,
	Run: func(cmd *cobra.Command, args []string) {
		// to stop tailing logs on user interrupt
		closeLogs := make(chan os.Signal, 1)
		signal.Notify(closeLogs, os.Interrupt, syscall.SIGHUP,
			syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

		istream := streams.NewLogStreamI()

		// todo - set as cobra args
		if len(args) >= 1{
			kubernetesLogs(istream, closeLogs) // kubernetes
		}else {
			dockerLogs(istream, closeLogs) // docker
		}
	},
}

const FOLLOW = true
// logOptions defines the docker client's connection options
var dockerLogOptions = types.ContainerLogsOptions{
	ShowStdout: FOLLOW, // always true
	ShowStderr: FOLLOW, // can be false
	Follow:     FOLLOW, // always true, can stop with ctrl+c
}
var podLogOptions = v1.PodLogOptions{
	TypeMeta:   metav1.TypeMeta{},
	Container:  "",
	Follow:  FOLLOW, // always true, can stop with ctrl+c
}
var once sync.Once
var colorsStruct struct {
	colors         []string
	nextColorIndex int
}
// nextColor returns a color for the prefix
func nextColor() string {
	once.Do(func() {
		// one time initialization
		colorsStruct.colors = append(colorsStruct.colors, color.Blue)
		colorsStruct.colors = append(colorsStruct.colors, color.Green)
		colorsStruct.colors = append(colorsStruct.colors, color.Purple)
		colorsStruct.colors = append(colorsStruct.colors, color.Yellow)
		colorsStruct.colors = append(colorsStruct.colors, color.Bold)
		colorsStruct.colors = append(colorsStruct.colors, color.Gray)
		// excluded red as it is used later to denote *StdErr*
		colorsStruct.nextColorIndex = 0
	})
	// rotate index
	if colorsStruct.nextColorIndex == len(colorsStruct.colors) {
		colorsStruct.nextColorIndex = 0
	}
	c := colorsStruct.colors[colorsStruct.nextColorIndex]
	colorsStruct.nextColorIndex++
	return c
}
func dockerLogs(istream streams.Istream, closeLogs chan os.Signal) {
	ctx, cancel := context.WithCancel(context.TODO())
	client, containers, valid := GetDockerContainers(ctx, cancel)
	if !valid{
		fmt.Println("Could not get docker containers")
		os.Exit(1)
	}
	defer ctx.Done()
	for _, container := range containers {
		if logReader, err := client.ContainerLogs(ctx, container.ID, dockerLogOptions); err == nil{
			// colorize container's prefix
			sOutPrefix := color.Ize(nextColor(), container.Names[0][1:])
			if dockerLogOptions.ShowStderr {
				splitAndAddReaders(istream, sOutPrefix, ctx, logReader)
			} else {
				addReader(istream, sOutPrefix, logReader)
			}
		}
	}
	// start writing to the stream
	stream := istream.StartStream()
	
	// start reading from the stream until user interrupt
	for {
		select {
		case <-closeLogs:
			cleanup(cancel, stream.Cancel)
			fmt.Println("Closed logs")
			return // == os.exit
		case log := <-stream.OutputStream:
			_, _ = fmt.Fprintln(os.Stdout, string(log))
		case <-stream.Err:
			cleanup(cancel, stream.Cancel)
			return // == os.exit
		}
	}
}

func kubernetesLogs(istream streams.Istream, closeLogs chan os.Signal){
	clientSet, pl, done := kubeConfigGetPods()
	if !done {
		return
	}
	for _, pod := range pl.Items {
		if strings.Compare(string(pod.Status.Phase), "Running") == 0 &&
			strings.Contains(pod.Namespace, "default") { // todo replace strings.contains with label selector?
			// get the logs for this pod
			req := clientSet.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, &podLogOptions)
			if logReader, err := req.Stream(context.TODO()); err == nil {
				// colorize container's prefix
				sOutPrefix := color.Ize(nextColor(), pod.Name)
				// add logReader to iStream interface
				istream.AddStream(sOutPrefix, logReader)
			}
		}
	}
	// start reading
	stream := istream.StartStream()
	// start reading from the stream until user interrupt
	for {
		select {
		case <-closeLogs:
			stream.Cancel()
			fmt.Println("Closed logs")
			return // == os.exit
		case <-stream.Err:
			stream.Cancel()
			return // == os.exit
		case log := <-stream.OutputStream:
			_, _ = fmt.Fprintln(os.Stdout, string(log))
		}
	}
}
func addReader(istream streams.Istream, prefix string, logReader io.ReadCloser) {
	istream.AddStream(prefix, logReader)
}

func splitAndAddReaders(istream streams.Istream, prefix string, ctx context.Context, logReader io.ReadCloser) {
	// Create io pipes to split logReader to StdOut and StdErr
	stdOutReadr, dstOut := io.Pipe()
	stdErrReadr, dstErr := io.Pipe()
	
	// close io pipes when ctx is cancelled
	go waitOnClose(prefix, ctx, logReader, dstOut, dstErr, stdOutReadr, stdErrReadr)
	// de-multiplex logReader to stdout and stderr streams
	go deMuxStream(prefix, dstOut, dstErr, logReader)
	
	sOutPrefix := prefix
	sErrPrefix := color.Ize(color.Red, "*") + sOutPrefix + color.Ize(color.Red, "*")
	
	istream.AddStream(sOutPrefix, stdOutReadr)
	istream.AddStream(sErrPrefix, stdErrReadr)
}

/*
func kubernetesLogs(){
		// load the stack
	c := config.New()
	err := c.FromJSON(utils.StatePath())
	if err != nil {
		fmt.Println("Unable to load state", err)
		os.Exit(1)
	}
	for _, r := range c.Resources {
		fmt.Println(r.Info().Name, r.Info().Type)
		for _, res:= range r.Info().Config.Resources{
			fmt.Printf("%s - %s \n", res.Info().Name, res.Info().Type)
		}
	}
}
*/

// cleanup closes docker/kubernetes client and all associated log readers
func cleanup(cancel context.CancelFunc,streamCancel context.CancelFunc) {
	cancel()        // stop docker/kubernetes client
	streamCancel()  // stop reading from logReaders
}
// deMuxStream splits readCloser into StdErr and StdOut if readCloser was written
// to in this way
func deMuxStream(name string, dstOut *io.PipeWriter, dstErr *io.PipeWriter, logReader io.ReadCloser) {
	_, _ = stdcopy.StdCopy(dstOut, dstErr, logReader)
	fmt.Println("Stopped de-mux-ing log streams for ", name)
}
// waitOnClose waits for the context to be done then closes all reader and writers
func waitOnClose(name string, ctx context.Context, logReader io.ReadCloser, dstOut *io.PipeWriter,
	dstErr *io.PipeWriter, stdOutReadr *io.PipeReader, stdErrReadr *io.PipeReader) {
	<- ctx.Done()
	_ = logReader.Close()
	_ = dstOut.Close()
	_ = dstErr.Close()
	_ = stdOutReadr.Close()
	_ = stdErrReadr.Close()
	fmt.Println("Stopped reading log streams for ", name)
}
// GetDockerContainers returns a shipyard-docker client and list or running containers
// and a flag indicating whether these values are valid
func GetDockerContainers(ctx context.Context, cancel context.CancelFunc) (clients.Docker, []types.Container, bool) {
	client, err := clients.NewDocker()
	if err != nil {
		fmt.Println("Unable to connect to Docker daemon", err)
		return nil, nil, false
	}
	filter := filters.NewArgs()
	filter.Add("name", "shipyard")
	filter.Add("status", "running")
	containers, err := client.ContainerList(ctx, types.ContainerListOptions{
		Filters: filter,
	})
	if err != nil || len(containers) == 0 {
		cancel()
		return nil, nil, false
	}
	return client, containers, true
}
// kubeConfigGetPods returns the clientSet and a valid pod list
func kubeConfigGetPods() (*kubernetes.Clientset, *v1.PodList, bool) {
	// todo - load config from stack
	cfg, err := clientcmd.BuildConfigFromFlags("", "/Users/ishan/.shipyard/config/k3s/kubeconfig.yaml")
	if err != nil {
		fmt.Println(err.Error())
		return nil, nil, false
	}
	clientSet, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		panic(err.Error())
	}
	// todo - update list options to get only desired labels/namespaces
	lo := metav1.ListOptions{}
	pl, err := clientSet.CoreV1().Pods("").List(context.Background(), lo)
	if err != nil {
		fmt.Println(err.Error())
		return nil, nil, false
	}
	return clientSet, pl, true
}