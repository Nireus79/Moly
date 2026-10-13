// probe plays the part of the browser: it starts moly.exe as a native messaging host and asks for the backend.
// Usage: probe.exe <path to moly.exe> <extension origin>. Prints one JSON line per reply; exit code 0 if "start" worked.
package main

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: probe <moly.exe> <extension origin>")
		os.Exit(2)
	}
	cmd := exec.Command(os.Args[1], os.Args[2])
	in, _ := cmd.StdinPipe()
	out, _ := cmd.StdoutPipe()
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		fmt.Println("cannot start the host:", err)
		os.Exit(1)
	}
	ok := false
	for _, action := range []string{"ping", "start", "status"} {
		data, _ := json.Marshal(map[string]string{"action": action, "id": action})
		frame := make([]byte, 4, 4+len(data))
		binary.LittleEndian.PutUint32(frame, uint32(len(data)))
		in.Write(append(frame, data...))
		var head [4]byte
		if _, err := io.ReadFull(out, head[:]); err != nil {
			fmt.Println("no reply to", action, err)
			os.Exit(1)
		}
		body := make([]byte, binary.LittleEndian.Uint32(head[:]))
		io.ReadFull(out, body)
		var reply map[string]any
		json.Unmarshal(body, &reply)
		if t, isStr := reply["token"].(string); isStr && len(t) > 6 {
			reply["token"] = t[:6] + "..."
		}
		line, _ := json.Marshal(reply)
		fmt.Println(string(line))
		if action == "start" {
			ok = reply["success"] == true
		}
	}
	in.Close()
	cmd.Wait()
	if !ok {
		os.Exit(1)
	}
}
