package main

import (
	"bufio"
	"encoding/binary"
	"fmt"

	"github.com/Microsoft/go-winio"
)

const (
	AgentMaxMessageLength = 1<<14 - 1
)

// QueryAgent provides a way to query the named windows openssh agent pipe
func QueryAgent(pipeName string, buf []byte) (result []byte, err error) {
	if len(buf) > AgentMaxMessageLength {
		Logger.Error("Message too long")
		return nil, fmt.Errorf("Message too long")
	}

	conn, err := winio.DialPipe(pipeName, nil)
	if err != nil {
		Logger.Error("cannot connect to pipe %s: %w", pipeName, err)
		return nil, fmt.Errorf("cannot connect to pipe %s: %w", pipeName, err)
	}
	defer conn.Close()

	l, err := conn.Write(buf)
	if err != nil {
		Logger.Error("cannot write to pipe %s: %w", pipeName, err)
		return nil, fmt.Errorf("cannot write to pipe %s: %w", pipeName, err)
	}

	reader := bufio.NewReader(conn)

	// Magic numbers from the ssh-agent protocol specification.
	// <https://github.com/openssh/openssh-portable/blob/4e636cf/PROTOCOL.agent>
	// first 4 bytes are magic numbers related to the named pipe
	magic := make([]byte, 4)
	l, err = reader.Read(magic)
	if err != nil {
		Logger.Error("cannot read from pipe %s: %w", pipeName, err)
		return nil, fmt.Errorf("cannot read from pipe %s: %w", pipeName, err)
	}
	// next byte is the SSH2_AGENT_IDENTITIES_ANSWER
	sshHeader := make([]byte, 1)
	l, err = reader.Read(sshHeader)
	if err != nil {
		Logger.Error("cannot read from pipe %s: %w", pipeName, err)
		return nil, fmt.Errorf("cannot read from pipe %s: %w", pipeName, err)
	}
	// next 4 bytes (Uint32) is the number of keys
	keyCountSlice := make([]byte, 4)
	l, err = reader.Read(keyCountSlice)
	if err != nil {
		Logger.Error("cannot read from pipe %s: %w", pipeName, err)
		return nil, fmt.Errorf("cannot read from pipe %s: %w", pipeName, err)
	}
	// convert to Uint32
	keyCount := binary.BigEndian.Uint32(keyCountSlice)

	// set to max agent message length minus the previous 9 bytes
	res := make([]byte, AgentMaxMessageLength-9)
	// verify the key count is > 0, otherwise skip
	if keyCount > 0 {
		l, err = reader.Read(res)
		if err != nil {
			Logger.Error("cannot read from pipe %s: %w", pipeName, err)
			return nil, fmt.Errorf("cannot read from pipe %s: %w", pipeName, err)
		}
	} else {
		l = 0
	}

	// Concat all slices together
	concatRes := append(magic, sshHeader...)
	concatRes = append(concatRes, keyCountSlice...)
	concatRes = append(concatRes, res[0:l]...)

	return concatRes, nil
}
