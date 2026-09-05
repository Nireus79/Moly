package main

import (
	"encoding/binary"
	"encoding/json"
	"log"
	"os"
)

// HandleNativeMessages listens for native host messages from Chrome extension
// This allows the extension to request backend startup
func HandleNativeMessages() {
	// Only listen if MOLY_NATIVE_HOST env var is set
	if os.Getenv("MOLY_NATIVE_HOST") != "1" {
		return
	}

	for {
		// Read message length (4 bytes, little-endian)
		lengthBytes := make([]byte, 4)
		_, err := os.Stdin.Read(lengthBytes)
		if err != nil {
			log.Printf("[Moly] Native host read error: %v", err)
			return
		}

		messageLength := binary.LittleEndian.Uint32(lengthBytes)

		// Read message
		messageBytes := make([]byte, messageLength)
		_, err = os.Stdin.Read(messageBytes)
		if err != nil {
			log.Printf("[Moly] Native host message read error: %v", err)
			return
		}

		var message map[string]interface{}
		if err := json.Unmarshal(messageBytes, &message); err != nil {
			log.Printf("[Moly] Failed to parse native message: %v", err)
			continue
		}

		// Process message
		if messageType, ok := message["type"].(string); ok {
			switch messageType {
			case "PING":
				// Extension is checking if we're alive
				sendNativeResponse(map[string]interface{}{
					"status": "alive",
					"port":   11436,
				})
			case "STATUS":
				// Extension wants status
				sendNativeResponse(map[string]interface{}{
					"status": "running",
					"port":   11436,
					"proxy":  11435,
				})
			default:
				log.Printf("[Moly] Unknown native message type: %v", messageType)
			}
		}
	}
}

// sendNativeResponse sends a response back to the extension
func sendNativeResponse(data interface{}) error {
	responseJSON, err := json.Marshal(data)
	if err != nil {
		return err
	}

	// Write length (4 bytes, little-endian)
	lengthBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(lengthBytes, uint32(len(responseJSON)))

	_, err = os.Stdout.Write(lengthBytes)
	if err != nil {
		return err
	}

	// Write message
	_, err = os.Stdout.Write(responseJSON)
	return err
}
