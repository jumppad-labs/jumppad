package providers

import (
	hclog "github.com/hashicorp/go-hclog"
	"github.com/shipyard-run/shipyard/pkg/clients"
)

// execute a command in a container
func execCommand(c clients.Docker, container string, command []string, l hclog.Logger) error {
	/*
		id, err := c.ContainerExecCreate(context.Background(), container, types.ExecConfig{
			Cmd:          command,
			WorkingDir:   "/",
			AttachStdout: true,
			AttachStderr: true,
		})

		if err != nil {
			return xerrors.Errorf("unable to create container exec: %w", err)
		}

		// get logs from an attach
		stream, err := c.ContainerExecAttach(context.Background(), id.ID, types.ExecStartCheck{})
		if err != nil {
			return xerrors.Errorf("unable to attach logging to exec process: %w", err)
		}
		defer stream.Close()

		// ensure that the log from the Docker exec command is copied to the default logger
		go func() {
			io.Copy(
				l.StandardWriter(&hclog.StandardLoggerOptions{}),
				stream.Reader,
			)
		}()

		err = c.ContainerExecStart(context.Background(), id.ID, types.ExecStartCheck{})
		if err != nil {
			return xerrors.Errorf("unable to start exec process: %w", err)
		}

		// loop until the container finishes execution
		for {
			i, err := c.ContainerExecInspect(context.Background(), id.ID)
			if err != nil {
				return xerrors.Errorf("unable to determine status of exec process: %w", err)
			}

			if !i.Running {
				if i.ExitCode == 0 {
					return nil
				}

				return xerrors.Errorf("container exec failed with exit code %d", i.ExitCode)
			}

			time.Sleep(1 * time.Second)
		}
	*/

	return nil
}
