package server

import (
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"runtime"

	"github.com/creack/pty"

	"github.com/gofiber/websocket/v2"
)

type windowSize struct {
	Rows uint16 `json:"rows"`
	Cols uint16 `json:"cols"`
	X    uint16
	Y    uint16
}

func (a *API) terminalWebsocket(c *websocket.Conn) {
	workdir := c.Query("workdir", "/")
	user := c.Query("user", "root")
	target := c.Query("target", "local")
	shell := c.Query("shell", "/bin/bash")

	var cmd *exec.Cmd
	if target == "local" {
		if runtime.GOOS == "windows" {
			a.log.Debug("Connecting to local terminal windows")
			cmd = exec.Command("powershell.exe")
		} else {
			a.log.Debug("Connecting to local terminal linux")
			cmd = exec.Command("bash")
		}
	} else {
		a.log.Debug("Connecting to remote Docker container", "workdir", workdir, "user", user, "target", target, "shell", shell)
		cmd = exec.Command("docker", "exec", "-ti", "-w", workdir, "-u", user, target, shell)
	}
	cmd.Env = append(os.Environ(), "TERM=xterm")

	tty, err := pty.Start(cmd)
	if err != nil {
		_ = c.WriteMessage(websocket.TextMessage, []byte(err.Error()))

		a.log.Error("Unable to start pty/cmd", "error", err)
		return
	}
	defer func() {
		cmd.Process.Kill()
		cmd.Process.Wait()
		tty.Close()
		c.Close()
	}()

	go func() {
		for {
			buf := make([]byte, 1024)
			read, err := tty.Read(buf)
			if err != nil {
				_ = c.WriteMessage(websocket.TextMessage, []byte(err.Error()))

				a.log.Error("Unable to read from pty/cmd", "error", err)
				return
			}
			_ = c.WriteMessage(websocket.BinaryMessage, buf[:read])
		}
	}()

	for {
		_, reader, err := c.NextReader()
		if err != nil {
			a.log.Error("Unable to grab next reader", "error", err)
			return
		}

		dataTypeBuf := make([]byte, 1)
		read, err := reader.Read(dataTypeBuf)
		if err != nil {
			a.log.Error("Unable to read message type from reader", "error", err)
			_ = c.WriteMessage(websocket.TextMessage, []byte("Unable to read message type from reader"))
			return
		}

		if read != 1 {
			a.log.Error("Unexpected number of bytes read")
			return
		}

		switch dataTypeBuf[0] {
		case 0:
			copied, err := io.Copy(tty, reader)
			if err != nil {
				a.log.Error("Error after copying data", "bytes", copied, "error", err)
			}
		case 1:
			decoder := json.NewDecoder(reader)
			resizeMessage := windowSize{}
			err := decoder.Decode(&resizeMessage)
			if err != nil {
				_ = c.WriteMessage(websocket.TextMessage, []byte("Error decoding resize message: "+err.Error()))
				continue
			}

			a.log.Debug("Resizing terminal")
			// #nosec G103
			//_, _, errno := syscall.Syscall(
			//	syscall.SYS_IOCTL,
			//	tty.Fd(),
			//	syscall.TIOCSWINSZ,
			//	uintptr(unsafe.Pointer(&resizeMessage)),
			//)
			//if errno != 0 {
			//	a.log.Error("Unable to resize terminal")
			//}
		default:
			a.log.Error("Unknown data", "type", dataTypeBuf[0])
		}
	}
}
